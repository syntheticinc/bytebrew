package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Sentinel errors returned by ProxyChatModel.
var (
	ErrQuotaExhausted = errors.New("proxy: quota exhausted")
	ErrRateLimited    = errors.New("proxy: rate limited")
)

// Compile-time interface check.
var _ model.ToolCallingChatModel = (*ProxyChatModel)(nil)

// ProxyChatModel implements model.ToolCallingChatModel by forwarding LLM
// requests to the Cloud API proxy endpoint. The proxy uses OpenAI-compatible
// request/response format with an additional "role" field for smart routing.
type ProxyChatModel struct {
	cloudAPIURL string
	accessToken string
	role        string // agent role: "supervisor", "coder", "reviewer", "tester"
	tools       []*schema.ToolInfo
	httpClient  *http.Client
}

// NewProxyChatModel creates a ProxyChatModel that sends requests to cloudAPIURL.
func NewProxyChatModel(cloudAPIURL, accessToken, role string) *ProxyChatModel {
	return &ProxyChatModel{
		cloudAPIURL: strings.TrimRight(cloudAPIURL, "/"),
		accessToken: accessToken,
		role:        role,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// ---------- model.ToolCallingChatModel ----------

// Generate sends a non-streaming chat completion request to the proxy.
func (p *ProxyChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	body, err := p.buildRequestBody(input, false)
	if err != nil {
		return nil, fmt.Errorf("proxy build request: %w", err)
	}

	resp, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := checkResponseStatus(resp); err != nil {
		return nil, err
	}

	var oaiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&oaiResp); err != nil {
		return nil, fmt.Errorf("proxy decode response: %w", err)
	}

	if len(oaiResp.Choices) == 0 {
		return nil, fmt.Errorf("proxy: empty choices in response")
	}

	return oaiMessageToSchema(&oaiResp.Choices[0].Message), nil
}

// Stream sends a streaming chat completion request to the proxy and returns a
// StreamReader that yields partial schema.Message chunks.
func (p *ProxyChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	body, err := p.buildRequestBody(input, true)
	if err != nil {
		return nil, fmt.Errorf("proxy build request: %w", err)
	}

	resp, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	if err := checkResponseStatus(resp); err != nil {
		resp.Body.Close()
		return nil, err
	}

	sr, sw := schema.Pipe[*schema.Message](10)

	go func() {
		defer resp.Body.Close()
		defer sw.Close()

		if err := parseSSEStream(resp.Body, sw); err != nil {
			slog.ErrorContext(ctx, "proxy SSE stream error", "error", err)
			sw.Send(nil, fmt.Errorf("proxy stream: %w", err))
		}
	}()

	return sr, nil
}

// WithTools returns a copy of ProxyChatModel with the given tools attached.
func (p *ProxyChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	cp := *p
	cp.tools = make([]*schema.ToolInfo, len(tools))
	copy(cp.tools, tools)
	return &cp, nil
}

// ---------- HTTP ----------

func (p *ProxyChatModel) doRequest(ctx context.Context, body []byte) (*http.Response, error) {
	url := p.cloudAPIURL + "/api/v1/proxy/llm"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("proxy create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.accessToken)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("proxy request: %w", err)
	}
	return resp, nil
}

// checkResponseStatus maps non-2xx HTTP codes to typed errors.
func checkResponseStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	// Read body for error detail (limit to 1 KiB).
	errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusPaymentRequired: // 402
		return fmt.Errorf("%w: %s", ErrQuotaExhausted, string(errBody))
	case http.StatusTooManyRequests: // 429
		return fmt.Errorf("%w: %s", ErrRateLimited, string(errBody))
	default:
		return fmt.Errorf("proxy: HTTP %d: %s", resp.StatusCode, string(errBody))
	}
}

// ---------- request body builder ----------

func (p *ProxyChatModel) buildRequestBody(input []*schema.Message, stream bool) ([]byte, error) {
	messages := make([]openAIMessage, 0, len(input))
	for _, msg := range input {
		messages = append(messages, schemaMessageToOpenAI(msg))
	}

	reqBody := openAIRequest{
		Messages: messages,
		Stream:   stream,
		Role:     p.role,
	}

	if len(p.tools) > 0 {
		reqBody.Tools = schemaToolsToOpenAI(p.tools)
	}

	return json.Marshal(reqBody)
}
