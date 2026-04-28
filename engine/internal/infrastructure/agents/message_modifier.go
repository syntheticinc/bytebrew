package agents

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/cloudwego/eino/schema"
)

// MessageModifier modifies messages before sending to the model
// It handles system prompt injection, urgency warnings, task reminders,
// and content recovery for streaming mode
type MessageModifier struct {
	systemPrompt     string
	urgencyWarning   string
	maxSteps         int
	stepContentStore StepContentStoreInterface
	contextLogger    ContextLoggerInterface
	stepCounter      int
	mu               sync.Mutex
}

// MessageModifierConfig holds configuration for MessageModifier
type MessageModifierConfig struct {
	SystemPrompt     string
	UrgencyWarning   string
	MaxSteps         int
	StepContentStore StepContentStoreInterface
	ContextLogger    ContextLoggerInterface
}

// NewMessageModifier creates a new MessageModifier
func NewMessageModifier(cfg MessageModifierConfig) *MessageModifier {
	return &MessageModifier{
		systemPrompt:     cfg.SystemPrompt,
		urgencyWarning:   cfg.UrgencyWarning,
		maxSteps:         cfg.MaxSteps,
		stepContentStore: cfg.StepContentStore,
		contextLogger:    cfg.ContextLogger,
		stepCounter:      0,
	}
}

// Modify modifies the input messages according to the current step and configuration
// Returns modified messages with system prompt, urgency warnings, and task reminders
func (m *MessageModifier) Modify(ctx context.Context, input []*schema.Message) []*schema.Message {
	m.mu.Lock()
	currentStep := m.stepCounter
	m.mu.Unlock()

	remainingSteps := m.maxSteps - currentStep

	// Build system prompt with urgency warning if approaching max steps
	currentSystemPrompt := m.systemPrompt
	if remainingSteps <= 3 && remainingSteps > 0 && m.urgencyWarning != "" {
		urgencyMsg := fmt.Sprintf(m.urgencyWarning, remainingSteps)
		currentSystemPrompt = currentSystemPrompt + urgencyMsg
	}

	// Find and extract LATEST user question for task reminder
	// In a conversation, the most recent user message is what needs to be answered
	var userQuestion string
	for _, msg := range input {
		if msg.Role == schema.User && msg.Content != "" {
			userQuestion = msg.Content
			// Don't break - continue to find the LAST user message
		}
	}

	// Add task reminder to system prompt after several steps
	// Uses the LATEST user message to keep focus on current question
	if currentStep >= 2 && userQuestion != "" {
		currentSystemPrompt += fmt.Sprintf("\n\n**CURRENT TASK (Step %d):** Answer the user's question: \"%s\"\nDo NOT get distracted - answer THIS question!", currentStep, userQuestion)
	}

	// Add system prompt at the beginning
	result := make([]*schema.Message, 0, len(input)+1)
	result = append(result, schema.SystemMessage(currentSystemPrompt))

	// CRITICAL FIX: Inject accumulated content into empty assistant messages
	// Eino's ReAct agent doesn't preserve content when there are tool_calls in streaming mode
	// We recover the content from our shared store
	var stepContent map[int]string
	if m.stepContentStore != nil {
		stepContent = m.stepContentStore.GetAll()
	}

	// Track which step each assistant message corresponds to
	assistantStepIdx := 0
	for _, msg := range input {
		if msg.Role == schema.Assistant {
			// If assistant message has tool_calls but empty content, try to fill it
			if msg.Content == "" && len(msg.ToolCalls) > 0 && stepContent != nil {
				if content, ok := stepContent[assistantStepIdx]; ok && content != "" {
					// Create a copy with filled content
					filledMsg := &schema.Message{
						Role:      msg.Role,
						Content:   content,
						ToolCalls: msg.ToolCalls,
						Name:      msg.Name,
					}
					result = append(result, filledMsg)
					slog.DebugContext(ctx, "filled empty assistant message with accumulated content",
						"step", assistantStepIdx, "content_length", len(content))
				} else {
					result = append(result, msg)
				}
			} else {
				result = append(result, msg)
			}
			assistantStepIdx++
		} else {
			result = append(result, msg)
		}
	}

	// Clean up old step content to prevent memory growth
	if m.stepContentStore != nil && currentStep > 2 {
		m.stepContentStore.ClearBefore(currentStep)
	}

	// Log the full context that will be sent to the model
	if m.contextLogger != nil {
		m.contextLogger.LogContext(ctx, result, currentStep)
		m.mu.Lock()
		m.stepCounter++
		m.mu.Unlock()
	}

	return result
}

// GetStep returns the current step counter (thread-safe)
func (m *MessageModifier) GetStep() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stepCounter
}

// ResetStep resets the step counter to 0
func (m *MessageModifier) ResetStep() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stepCounter = 0
}

// BuildModifierFunc returns a function suitable for use as AgentConfig.MessageModifier
func (m *MessageModifier) BuildModifierFunc() func(ctx context.Context, input []*schema.Message) []*schema.Message {
	return m.Modify
}
