package domain

import "context"

// LLMMessage represents a message in LLM conversation
type LLMMessage struct {
	Role    string
	Content string
}

// LLMChatRequest represents a chat request to LLM
type LLMChatRequest struct {
	Messages    []LLMMessage
	Model       string
	Temperature float32
	MaxTokens   int
}

// LLMChatResponse represents a chat response from LLM
type LLMChatResponse struct {
	Message LLMMessage
	Done    bool
}

// LLMProvider defines interface for LLM providers
type LLMProvider interface {
	// Chat performs a chat completion
	Chat(ctx context.Context, req LLMChatRequest) (*LLMChatResponse, error)

	// ChatStream performs a chat completion with streaming
	ChatStream(ctx context.Context, req LLMChatRequest, streamFunc func(chunk LLMMessage) error) error

	// Ping checks if the LLM provider is available
	Ping(ctx context.Context) error

	// Close closes the provider and releases resources
	Close() error
}
