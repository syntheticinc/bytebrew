package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Compile-time interface check.
var _ model.ToolCallingChatModel = (*GeminiChatModel)(nil)

const defaultGeminiBaseURL = "https://generativelanguage.googleapis.com/v1beta"

// GeminiChatModel implements model.ToolCallingChatModel for the Google Gemini API.
type GeminiChatModel struct {
	apiKey     string
	modelName  string
	baseURL    string
	tools      []*schema.ToolInfo
	httpClient *http.Client
}

// GeminiOption configures a GeminiChatModel.
type GeminiOption func(*GeminiChatModel)

// WithGeminiBaseURL sets a custom base URL (useful for testing or proxies).
func WithGeminiBaseURL(url string) GeminiOption {
	return func(g *GeminiChatModel) {
		g.baseURL = strings.TrimRight(url, "/")
	}
}

// WithGeminiHTTPClient sets a custom HTTP client.
func WithGeminiHTTPClient(client *http.Client) GeminiOption {
	return func(g *GeminiChatModel) {
		g.httpClient = client
	}
}

// NewGeminiChatModel creates a new GeminiChatModel.
func NewGeminiChatModel(apiKey, modelName string, opts ...GeminiOption) *GeminiChatModel {
	g := &GeminiChatModel{
		apiKey:    apiKey,
		modelName: modelName,
		baseURL:   defaultGeminiBaseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// ---------- model.ToolCallingChatModel ----------

// Generate sends a non-streaming request to the Gemini API.
func (g *GeminiChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	body, err := g.buildRequestBody(input)
	if err != nil {
		return nil, fmt.Errorf("gemini build request: %w", err)
	}

	url := g.generateContentURL()
	resp, err := g.doRequest(ctx, url, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := g.checkResponseStatus(resp); err != nil {
		return nil, err
	}

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("gemini decode response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 {
		return nil, fmt.Errorf("gemini: empty candidates in response")
	}

	return geminiResponseToSchema(&geminiResp), nil
}

// Stream sends a streaming request to the Gemini API and returns a StreamReader.
func (g *GeminiChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	body, err := g.buildRequestBody(input)
	if err != nil {
		return nil, fmt.Errorf("gemini build request: %w", err)
	}

	url := g.streamContentURL()
	resp, err := g.doRequest(ctx, url, body)
	if err != nil {
		return nil, err
	}

	if err := g.checkResponseStatus(resp); err != nil {
		resp.Body.Close()
		return nil, err
	}

	sr, sw := schema.Pipe[*schema.Message](10)

	go func() {
		defer resp.Body.Close()
		defer sw.Close()

		if err := parseGeminiSSEStream(resp.Body, sw); err != nil {
			slog.ErrorContext(ctx, "gemini SSE stream error", "error", err)
			sw.Send(nil, fmt.Errorf("gemini stream: %w", err))
		}
	}()

	return sr, nil
}

// WithTools returns a copy of GeminiChatModel with the given tools attached.
func (g *GeminiChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	cp := *g
	cp.tools = make([]*schema.ToolInfo, len(tools))
	copy(cp.tools, tools)
	return &cp, nil
}

// ---------- HTTP ----------

func (g *GeminiChatModel) generateContentURL() string {
	return fmt.Sprintf("%s/models/%s:generateContent", g.baseURL, g.modelName)
}

func (g *GeminiChatModel) streamContentURL() string {
	return fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse", g.baseURL, g.modelName)
}

func (g *GeminiChatModel) doRequest(ctx context.Context, url string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gemini create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", g.apiKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini request: %w", err)
	}
	return resp, nil
}

func (g *GeminiChatModel) checkResponseStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	resp.Body.Close()

	return fmt.Errorf("gemini: HTTP %d: %s", resp.StatusCode, string(errBody))
}

// ---------- request body builder ----------

func (g *GeminiChatModel) buildRequestBody(input []*schema.Message) ([]byte, error) {
	contents, systemInstruction := schemaMessagesToGemini(input)

	reqBody := geminiRequest{
		Contents:          contents,
		SystemInstruction: systemInstruction,
	}

	if len(g.tools) > 0 {
		reqBody.Tools = schemaToolsToGemini(g.tools)
	}

	return json.Marshal(reqBody)
}

// ---------- SSE stream parser ----------

// parseGeminiSSEStream reads SSE events from the Gemini streaming endpoint
// and sends partial schema.Messages through the pipe writer.
func parseGeminiSSEStream(r io.Reader, sw *schema.StreamWriter[*schema.Message]) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return nil
		}

		var chunk geminiResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			slog.Warn("gemini SSE: skip malformed chunk", "error", err)
			continue
		}

		if len(chunk.Candidates) == 0 {
			continue
		}

		msg := geminiResponseToSchema(&chunk)
		if sw.Send(msg, nil) {
			return nil
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("gemini SSE scan: %w", err)
	}

	return nil
}
