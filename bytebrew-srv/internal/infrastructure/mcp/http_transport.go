package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// HTTPTransport connects to an MCP server via HTTP POST.
type HTTPTransport struct {
	url    string
	client *http.Client
}

// NewHTTPTransport creates a transport that communicates via HTTP POST requests.
func NewHTTPTransport(url string) *HTTPTransport {
	return &HTTPTransport{url: url, client: &http.Client{}}
}

func (t *HTTPTransport) Start(_ context.Context) error {
	return nil // HTTP is stateless
}

func (t *HTTPTransport) Send(ctx context.Context, req *Request) (*Response, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var resp Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &resp, nil
}

func (t *HTTPTransport) Notify(ctx context.Context, req *Request) {
	data, err := json.Marshal(req)
	if err != nil {
		return
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, bytes.NewReader(data))
	if err != nil {
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := t.client.Do(httpReq)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func (t *HTTPTransport) Close() error {
	return nil
}
