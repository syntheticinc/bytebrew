package agents

import (
	"context"
	"log/slog"

	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

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

// NewContextRewriterWithLogging creates a context rewriter with logging.
// maxContextTokens is the maximum context size in TOKENS (not characters).
// onContextSize is an optional callback invoked with the actual context size (in tokens)
// after each rewriter pass — both when within limit and after compression.
func NewContextRewriterWithLogging(maxContextTokens int, contextLogger *ContextLogger, onContextSize func(int)) react.MessageModifier {
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
			if onContextSize != nil {
				onContextSize(totalTokens)
			}
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

		// Report post-compression context size
		if onContextSize != nil {
			postChars := 0
			for _, msg := range result {
				postChars += messageChars(msg)
			}
			onContextSize(charsToTokens(postChars))
		}

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
	return NewContextRewriterWithLogging(maxContextTokens, nil, nil)
}


