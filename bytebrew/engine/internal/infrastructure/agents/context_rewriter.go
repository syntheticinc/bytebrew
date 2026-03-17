package agents

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/utils"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

// PlanProvider defines interface for getting active plan (consumer-side interface for context_rewriter)
type PlanProvider interface {
	GetActivePlan(ctx context.Context, sessionID string) (*domain.Plan, error)
}

// charsPerToken is the approximate ratio of characters to tokens for most LLMs
// This is a rough estimation: 1 token ≈ 4 characters for English text
const charsPerToken = 4

// charsToTokens converts character count to approximate token count
func charsToTokens(chars int) int {
	return chars / charsPerToken
}

// tokensToChars converts token count to approximate character count
func tokensToChars(tokens int) int {
	return tokens * charsPerToken
}

// messageChars estimates the total character size of a message including tool calls.
// This provides a more accurate estimate for token counting than Content alone.
func messageChars(msg *schema.Message) int {
	total := len(msg.Content)

	// Count tool calls (assistant messages)
	for _, tc := range msg.ToolCalls {
		total += len(tc.ID)
		total += len(tc.Type)
		total += len(tc.Function.Name)
		total += len(tc.Function.Arguments)
	}

	// Count tool result metadata (tool messages)
	total += len(msg.ToolCallID)
	total += len(msg.ToolName)
	total += len(msg.Name)

	return total
}

// NewContextRewriterWithLogging creates a context rewriter with logging
// maxContextTokens is the maximum context size in TOKENS (not characters)
func NewContextRewriterWithLogging(maxContextTokens int, contextLogger *ContextLogger, planManager PlanProvider, sessionID string) react.MessageModifier {
	// Convert token limit to character limit for internal calculations
	maxContextChars := tokensToChars(maxContextTokens)

	return func(ctx context.Context, input []*schema.Message) []*schema.Message {
		if len(input) == 0 {
			return input
		}

		// Note: Logging moved to MessageModifier to capture full context with system prompt

		beforeCount := len(input)

		// Calculate context size in characters
		totalChars := 0
		for _, msg := range input {
			totalChars += messageChars(msg)
		}
		totalTokens := charsToTokens(totalChars)

		// If context size is within limit, return as is
		if totalChars <= maxContextChars {
			slog.DebugContext(ctx, "context within limit, no compression needed",
				"tokens", totalTokens, "limit_tokens", maxContextTokens)
			return input
		}

		slog.DebugContext(ctx, "context exceeds limit, compressing",
			"tokens", totalTokens, "limit_tokens", maxContextTokens)

		// Compress context while preserving chronological order and message relationships
		// Strategy: Keep system prompt + ALL user messages + recent tool interactions
		// Important: Keep Assistant messages with tool calls together with their Tool results

		// Separate system prompt from conversation messages
		// CRITICAL: Preserve chronological order of ALL messages
		var systemPrompt *schema.Message
		var conversationMessages []*schema.Message // All non-system messages in order

		for _, msg := range input {
			if msg.Role == schema.System {
				systemPrompt = msg
			} else {
				conversationMessages = append(conversationMessages, msg)
			}
		}

		// NEW: If plan exists, compress completed steps into summaries
		if planManager != nil && sessionID != "" {
			plan, err := planManager.GetActivePlan(ctx, sessionID)
			if err == nil && plan != nil {
				conversationMessages = compressCompletedPlanSteps(ctx, conversationMessages, plan)
				completed, total := plan.Progress()
				slog.DebugContext(ctx, "compressed completed plan steps",
					"plan_id", plan.ID, "completed_steps", completed, "total_steps", total)
			}
		}

		// Calculate sizes
		systemSize := 0
		if systemPrompt != nil {
			systemSize = messageChars(systemPrompt)
		}

		// Calculate total size of ALL user messages (they are all preserved)
		userSize := 0
		for _, msg := range conversationMessages {
			if msg.Role == schema.User {
				userSize += messageChars(msg)
			}
		}

		// Reserve space for system prompt and ALL user messages
		reservedChars := systemSize + userSize
		if reservedChars > maxContextChars {
			slog.WarnContext(ctx, "system prompt + all user messages exceed limit",
				"system_tokens", charsToTokens(systemSize),
				"user_tokens", charsToTokens(userSize),
				"limit_tokens", maxContextTokens)
			// Still try to fit what we can from other messages
			reservedChars = maxContextChars
		}

		// Calculate remaining capacity for assistant/tool messages
		remainingCapacity := maxContextChars - reservedChars
		if remainingCapacity < 0 {
			remainingCapacity = 0
		}

		// Build a map of ToolCallID -> message index for proper pairing
		toolCallToAssistant := make(map[string]int)
		for i, msg := range conversationMessages {
			if msg.Role == schema.Assistant && len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					toolCallToAssistant[tc.ID] = i
				}
			}
		}

		// CRITICAL FIX: Process messages in chronological order, keeping interleaved structure
		// Strategy: Keep ALL user messages, and fit as many recent assistant/tool messages as possible
		// while maintaining proper message order (U1 -> A1 -> U2 -> A2, NOT U1 -> U2 -> A1 -> A2)

		// First pass: mark which non-user messages to keep (from most recent backwards)
		keepIndices := make(map[int]bool)
		skipIndices := make(map[int]bool)
		currentSize := 0
		var removedToolResults []string

		for i := len(conversationMessages) - 1; i >= 0; i-- {
			msg := conversationMessages[i]

			// Always keep user messages
			if msg.Role == schema.User {
				keepIndices[i] = true
				continue
			}

			// Skip if already processed
			if skipIndices[i] {
				continue
			}

			msgSize := messageChars(msg)

			// For tool messages, process as a pair with matching assistant
			if msg.Role == schema.Tool {
				assistantIdx := -1
				if msg.ToolCallID != "" {
					if idx, ok := toolCallToAssistant[msg.ToolCallID]; ok {
						assistantIdx = idx
					}
				}

				// Fallback: check previous message
				if assistantIdx == -1 && i > 0 {
					prevMsg := conversationMessages[i-1]
					if prevMsg.Role == schema.Assistant && len(prevMsg.ToolCalls) > 0 {
						assistantIdx = i - 1
					}
				}

				if assistantIdx >= 0 && !skipIndices[assistantIdx] {
					assistantMsg := conversationMessages[assistantIdx]
					pairSize := msgSize + messageChars(assistantMsg)

					if currentSize+pairSize > remainingCapacity {
						// Can't fit pair - mark both for removal
						removedToolResults = append(removedToolResults, msg.Name)
						if len(assistantMsg.ToolCalls) > 0 {
							removedToolResults = append(removedToolResults, assistantMsg.ToolCalls[0].Function.Name+" (call)")
						}
						skipIndices[assistantIdx] = true
						continue
					}

					// Keep both
					keepIndices[i] = true
					keepIndices[assistantIdx] = true
					currentSize += pairSize
					skipIndices[assistantIdx] = true
					continue
				}

				// Orphaned tool message
				removedToolResults = append(removedToolResults, msg.Name+" (orphaned)")
				continue
			}

			// For assistant messages with tool calls - will be processed with tool result
			if msg.Role == schema.Assistant && len(msg.ToolCalls) > 0 {
				// Check if any tool results exist
				hasToolResult := false
				for j := i + 1; j < len(conversationMessages) && j <= i+len(msg.ToolCalls)+1; j++ {
					if conversationMessages[j].Role == schema.Tool {
						hasToolResult = true
						break
					}
				}
				if !hasToolResult {
					// Orphaned
					if len(msg.ToolCalls) > 0 {
						removedToolResults = append(removedToolResults, msg.ToolCalls[0].Function.Name+" (orphaned call)")
					}
					continue
				}
				// Will be processed when we hit the tool result
				continue
			}

			// Regular assistant message
			if currentSize+msgSize > remainingCapacity {
				continue // Skip this one
			}

			keepIndices[i] = true
			currentSize += msgSize
		}

		// Second pass: build result maintaining chronological order
		var result []*schema.Message
		if systemPrompt != nil {
			result = append(result, systemPrompt)
		}

		// Add messages in original order, keeping only marked ones
		for i, msg := range conversationMessages {
			if keepIndices[i] {
				result = append(result, msg)
			}
		}

		afterCount := len(result)

		slog.DebugContext(ctx, "context compressed",
			"before", beforeCount,
			"after", afterCount,
			"removed", beforeCount-afterCount,
			"removed_tool_results", len(removedToolResults))

		// Log compression report if context logger is available
		if contextLogger != nil {
			contextLogger.LogCompressionReport(ctx, beforeCount, afterCount, removedToolResults)
		}

		// Log post-compression context (what LLM actually receives)
		if contextLogger != nil {
			contextLogger.LogContext(ctx, result, -1)
		}

		return result
	}
}

// NewContextRewriter creates a context rewriter that compresses context when it exceeds maxContextTokens
// maxContextTokens is the maximum context size in TOKENS (not characters)
func NewContextRewriter(maxContextTokens int) react.MessageModifier {
	return NewContextRewriterWithLogging(maxContextTokens, nil, nil, "")
}

// compressCompletedPlanSteps replaces tool results from completed steps with summaries
// This significantly reduces token usage while preserving critical information
func compressCompletedPlanSteps(ctx context.Context, messages []*schema.Message, plan *domain.Plan) []*schema.Message {
	// Build map: assistant message index -> plan step
	// Strategy: Map assistant messages to plan steps chronologically
	assistantToPlanStep := make(map[int]int)

	assistantIdx := 0
	completedStepIdx := 0

	// Find completed steps in order
	completedSteps := make([]*domain.PlanStep, 0)
	for _, step := range plan.Steps {
		if step.Status == domain.StepStatusCompleted {
			completedSteps = append(completedSteps, step)
		}
	}

	// Map assistant messages to completed steps
	for i, msg := range messages {
		if msg.Role == schema.Assistant {
			if completedStepIdx < len(completedSteps) {
				assistantToPlanStep[i] = completedSteps[completedStepIdx].Index
				completedStepIdx++
			}
			assistantIdx++
		}
	}

	// Compress tool results that belong to completed steps
	for assistantIdx, stepIdx := range assistantToPlanStep {
		step := plan.Steps[stepIdx]

		// Find tool results after this assistant message
		for i := assistantIdx + 1; i < len(messages) && i < assistantIdx+10; i++ {
			msg := messages[i]
			if msg.Role == schema.Tool {
				originalLen := len(msg.Content)

				// Only compress if content is substantial (>500 chars)
				if originalLen > 500 {
					// Use step result if available, otherwise truncate content
					summary := step.Result
					if summary == "" {
						summary = utils.Truncate(msg.Content, 200)
					}

					messages[i].Content = fmt.Sprintf(
						"[PLAN STEP %d COMPLETED: %s]\nTool: %s\nSummary: %s",
						stepIdx+1, step.Description, msg.Name, summary,
					)

					savings := originalLen - len(messages[i].Content)
					slog.DebugContext(ctx, "compressed tool result for plan step",
						"step", stepIdx+1,
						"tool", msg.Name,
						"original_chars", originalLen,
						"compressed_chars", len(messages[i].Content),
						"savings_chars", savings,
						"savings_tokens", charsToTokens(savings))
				}
			} else if msg.Role == schema.Assistant {
				// Stop when we hit next assistant message
				break
			}
		}
	}

	return messages
}

