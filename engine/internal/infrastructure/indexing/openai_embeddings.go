package indexing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// OpenAIEmbeddingsClient calls an OpenAI-compatible /embeddings endpoint.
// Supports: OpenAI, Azure OpenAI, Cohere (compat), vLLM, LocalAI, LiteLLM.
type OpenAIEmbeddingsClient struct {
	baseURL   string
	apiKey    string
	model     string
	dimension int
	client    *http.Client
}

// NewOpenAIEmbeddingsClient creates a client for an OpenAI-compatible embedding API.
func NewOpenAIEmbeddingsClient(baseURL, apiKey, model string, dimension int) *OpenAIEmbeddingsClient {
	// Ensure baseURL doesn't have trailing slash
	baseURL = strings.TrimRight(baseURL, "/")
	return &OpenAIEmbeddingsClient{
		baseURL:   baseURL,
		apiKey:    apiKey,
		model:     model,
		dimension: dimension,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Dimension returns the configured embedding dimension.
func (c *OpenAIEmbeddingsClient) Dimension() int {
	return c.dimension
}

// openaiEmbedRequest is the JSON body for POST /embeddings.
type openaiEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// openaiEmbedResponse is the JSON response from POST /embeddings.
type openaiEmbedResponse struct {
	Data  []openaiEmbedData `json:"data"`
	Error *openaiErrorBody  `json:"error,omitempty"`
}

type openaiEmbedData struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type openaiErrorBody struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// Embed generates an embedding for a single text.
func (c *OpenAIEmbeddingsClient) Embed(ctx context.Context, text string) ([]float32, error) {
	results, err := c.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 || results[0] == nil {
		return nil, fmt.Errorf("empty embedding result")
	}
	return results[0], nil
}

// EmbedBatch generates embeddings for multiple texts via the OpenAI-compatible API.
func (c *OpenAIEmbeddingsClient) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// Truncate texts that are too long (rough token limit)
	truncated := make([]string, len(texts))
	for i, t := range texts {
		if len(t) > MaxTextLength {
			truncated[i] = t[:MaxTextLength]
		} else {
			truncated[i] = t
		}
	}

	// Batch in groups of EmbedBatchSize
	results := make([][]float32, len(truncated))
	for start := 0; start < len(truncated); start += EmbedBatchSize {
		end := start + EmbedBatchSize
		if end > len(truncated) {
			end = len(truncated)
		}

		batch := truncated[start:end]
		embeddings, err := c.embedBatch(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("embed batch [%d:%d]: %w", start, end, err)
		}

		for i, emb := range embeddings {
			results[start+i] = emb
		}
	}

	return results, nil
}

// embedBatch sends a single batch request to the /embeddings endpoint.
func (c *OpenAIEmbeddingsClient) embedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody, err := json.Marshal(openaiEmbedRequest{
		Model: c.model,
		Input: texts,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

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

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/embeddings", bytes.NewReader(reqBody))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if c.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.apiKey)
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http request: %w", err)
			slog.WarnContext(ctx, "[OpenAIEmbed] request failed", "attempt", attempt+1, "error", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("read response: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
			slog.WarnContext(ctx, "[OpenAIEmbed] retryable error", "attempt", attempt+1, "status", resp.StatusCode)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("embedding API error (status %d): %s", resp.StatusCode, string(body))
		}

		var embedResp openaiEmbedResponse
		if err := json.Unmarshal(body, &embedResp); err != nil {
			return nil, fmt.Errorf("unmarshal response: %w", err)
		}

		if embedResp.Error != nil {
			return nil, fmt.Errorf("embedding API error: %s", embedResp.Error.Message)
		}

		if len(embedResp.Data) != len(texts) {
			return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(embedResp.Data))
		}

		// Sort by index (API may return in different order)
		results := make([][]float32, len(texts))
		for _, d := range embedResp.Data {
			if d.Index >= 0 && d.Index < len(results) {
				results[d.Index] = d.Embedding
			}
		}

		return results, nil
	}

	return nil, fmt.Errorf("embedding failed after %d attempts: %w", maxRetries, lastErr)
}
