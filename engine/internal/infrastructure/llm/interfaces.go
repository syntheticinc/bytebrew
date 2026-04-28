package llm

import "context"

// GenerateRequest represents a generation request
type GenerateRequest struct {
	Prompt      string
	Model       string
	Temperature float32
	MaxTokens   int
	Stream      bool
}

// GenerateResponse represents a generation response
type GenerateResponse struct {
	Content string
	Done    bool
}

// ChatRequest represents a chat request
type ChatRequest struct {
	Messages    []ChatMessage
	Model       string
	Temperature float32
	MaxTokens   int
	Stream      bool
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string
	Content string
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Message ChatMessage
	Done    bool
}

// EmbeddingRequest represents an embedding request
type EmbeddingRequest struct {
	Input string
	Model string
}

// EmbeddingResponse represents an embedding response
type EmbeddingResponse struct {
	Embedding []float32
}

// BaseClient defines common operations for all LLM clients
type BaseClient interface {
	// Ping checks if the LLM provider is available
	Ping(ctx context.Context) error

	// Close closes the client and releases resources
	Close() error
}

// ChatClient defines interface for chat completions
type ChatClient interface {
	BaseClient

	// Chat performs a chat completion
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// ChatStream performs a chat completion with streaming
	ChatStream(ctx context.Context, req ChatRequest, streamFunc func(chunk ChatMessage) error) error
}

// GenerateClient defines interface for text generation
type GenerateClient interface {
	BaseClient

	// Generate generates text from a prompt
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)

	// GenerateStream generates text with streaming
	GenerateStream(ctx context.Context, req GenerateRequest, streamFunc func(chunk string) error) error
}

// EmbeddingClient defines interface for creating embeddings
type EmbeddingClient interface {
	BaseClient

	// CreateEmbedding creates embeddings for text
	CreateEmbedding(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error)
}

// Client is a unified interface combining all LLM capabilities
// Use specific interfaces (ChatClient, GenerateClient, EmbeddingClient) when possible
type Client interface {
	ChatClient
	GenerateClient
	EmbeddingClient
}
