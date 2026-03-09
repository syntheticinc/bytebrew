package indexing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	DefaultOllamaURL = "http://localhost:11434"
	DefaultEmbedModel = "nomic-embed-text"
	DefaultDimension  = 768
	MaxTextLength     = 28000
	EmbedBatchSize    = 50
	maxRetries        = 3
)

// EmbeddingsClient calls the Ollama embeddings API over HTTP.
type EmbeddingsClient struct {
	baseURL   string
	model     string
	dimension int
	client    *http.Client
}

// NewEmbeddingsClient creates a client for the given Ollama instance.
func NewEmbeddingsClient(baseURL, model string, dimension int) *EmbeddingsClient {
	return &EmbeddingsClient{
		baseURL:   baseURL,
		model:     model,
		dimension: dimension,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Dimension returns the configured embedding dimension.
func (c *EmbeddingsClient) Dimension() int {
	return c.dimension
}

// Ping checks whether the Ollama server is reachable.
func (c *EmbeddingsClient) Ping(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// Embed generates an embedding for a single text.
func (c *EmbeddingsClient) Embed(ctx context.Context, text string) ([]float32, error) {
	results, err := c.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 || results[0] == nil {
		return nil, fmt.Errorf("empty embedding result")
	}
	return results[0], nil
}

// EmbedBatch generates embeddings for multiple texts.
// Returns nil entries for texts that failed to embed.
func (c *EmbeddingsClient) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// Truncate texts that are too long
	truncated := make([]string, len(texts))
	for i, t := range texts {
		if len(t) > MaxTextLength {
			truncated[i] = t[:MaxTextLength]
		} else {
			truncated[i] = t
		}
	}

	// Try native batch first
	results, err := c.embedBatchNative(ctx, truncated)
	if err == nil {
		return results, nil
	}

	slog.WarnContext(ctx, "native batch embed failed, falling back to individual requests", "error", err)
	return c.embedBatchIndividual(ctx, truncated)
}

// embedRequest is the JSON body for the Ollama embed endpoint.
type embedRequest struct {
	Model    string   `json:"model"`
	Input    []string `json:"input"`
	Truncate bool     `json:"truncate"`
}

// embedResponse is the JSON response from the Ollama embed endpoint.
type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// embedBatchNative sends all texts in a single API call.
func (c *EmbeddingsClient) embedBatchNative(ctx context.Context, texts []string) ([][]float32, error) {
	body, err := json.Marshal(embedRequest{
		Model:    c.model,
		Input:    texts,
		Truncate: true,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	respBody, err := c.doWithRetry(ctx, body)
	if err != nil {
		return nil, err
	}

	var resp embedResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(resp.Embeddings) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(resp.Embeddings))
	}

	// Convert float64 results to float32 (if needed, Ollama returns float64)
	results := make([][]float32, len(resp.Embeddings))
	for i, emb := range resp.Embeddings {
		results[i] = emb
	}
	return results, nil
}

// embedBatchIndividual sends one request per text as a fallback.
func (c *EmbeddingsClient) embedBatchIndividual(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))

	for i, text := range texts {
		body, err := json.Marshal(embedRequest{
			Model:    c.model,
			Input:    []string{text},
			Truncate: true,
		})
		if err != nil {
			continue
		}

		respBody, err := c.doWithRetry(ctx, body)
		if err != nil {
			slog.WarnContext(ctx, "embed individual failed", "index", i, "error", err)
			continue
		}

		var resp embedResponse
		if err := json.Unmarshal(respBody, &resp); err != nil {
			continue
		}

		if len(resp.Embeddings) > 0 {
			results[i] = resp.Embeddings[0]
		}
	}

	return results, nil
}

// doWithRetry performs the HTTP request with exponential backoff.
func (c *EmbeddingsClient) doWithRetry(ctx context.Context, body []byte) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/embed", bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http request: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusBadRequest {
			return nil, fmt.Errorf("bad request (400): %s", string(respBody))
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
			continue
		}

		if err != nil {
			lastErr = fmt.Errorf("read response: %w", err)
			continue
		}

		return respBody, nil
	}

	return nil, fmt.Errorf("embed failed after %d attempts: %w", maxRetries, lastErr)
}
