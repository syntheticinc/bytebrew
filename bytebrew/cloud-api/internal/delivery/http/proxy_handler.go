package http

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

	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/internal/delivery/http/middleware"
	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/internal/usecase/proxy_llm"
	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/pkg/errors"
)

// proxyAuthorizer authorizes proxy LLM requests.
type proxyAuthorizer interface {
	Authorize(ctx context.Context, userID, role, modelOverride string) (*proxy_llm.Result, error)
}

// proxyStepIncrementer increments proxy steps after successful request.
type proxyStepIncrementer interface {
	IncrementProxySteps(ctx context.Context, userID string) error
}

// ProxyHandler handles LLM proxy requests to DeepInfra.
type ProxyHandler struct {
	authorizer       proxyAuthorizer
	stepIncrementer  proxyStepIncrementer
	deepInfraAPIKey  string
	deepInfraBaseURL string
	httpClient       *http.Client
}

// NewProxyHandler creates a new ProxyHandler.
func NewProxyHandler(
	authorizer proxyAuthorizer,
	stepIncrementer proxyStepIncrementer,
	deepInfraAPIKey, deepInfraBaseURL string,
) *ProxyHandler {
	return &ProxyHandler{
		authorizer:       authorizer,
		stepIncrementer:  stepIncrementer,
		deepInfraAPIKey:  deepInfraAPIKey,
		deepInfraBaseURL: deepInfraBaseURL,
		httpClient:       &http.Client{},
	}
}

// proxyRequest represents the inbound LLM proxy request body.
// Extends the OpenAI chat completion format with our custom "role" field for smart routing.
type proxyRequest struct {
	Model    string `json:"model,omitempty"`
	Role     string `json:"role,omitempty"` // our custom field: supervisor, coder, reviewer, tester
	Stream   bool   `json:"stream,omitempty"`
	Messages []any  `json:"messages"`
}

// HandleProxy handles POST /api/v1/proxy/llm.
func (h *ProxyHandler) HandleProxy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	// Read and parse request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeProxyError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req proxyRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeProxyError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Authorize
	result, err := h.authorizer.Authorize(ctx, userID, req.Role, req.Model)
	if err != nil {
		h.handleAuthError(ctx, w, err)
		return
	}

	// Build outbound request body: replace model, remove our custom "role" field
	outboundBody, err := buildOutboundBody(body, result.TargetModel)
	if err != nil {
		slog.ErrorContext(ctx, "failed to build outbound body", "error", err)
		writeProxyError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Create outbound request to DeepInfra
	targetURL := strings.TrimRight(h.deepInfraBaseURL, "/") + "/chat/completions"
	outReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(outboundBody))
	if err != nil {
		slog.ErrorContext(ctx, "failed to create outbound request", "error", err)
		writeProxyError(w, http.StatusInternalServerError, "internal error")
		return
	}
	outReq.Header.Set("Content-Type", "application/json")
	outReq.Header.Set("Authorization", "Bearer "+h.deepInfraAPIKey)

	// Forward to DeepInfra
	resp, err := h.httpClient.Do(outReq)
	if err != nil {
		slog.ErrorContext(ctx, "deepinfra request failed", "error", err)
		writeProxyError(w, http.StatusBadGateway, "upstream request failed")
		return
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			slog.ErrorContext(ctx, "failed to close response body", "error", cerr)
		}
	}()

	// Non-2xx from DeepInfra: forward error as-is
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		h.forwardErrorResponse(ctx, w, resp)
		return
	}

	if req.Stream {
		h.handleStreamResponse(ctx, w, resp, userID)
		return
	}

	h.handleNonStreamResponse(ctx, w, resp, userID)
}

// handleStreamResponse pipes SSE chunks from DeepInfra to the client.
func (h *ProxyHandler) handleStreamResponse(ctx context.Context, w http.ResponseWriter, resp *http.Response, userID string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		slog.ErrorContext(ctx, "response writer does not support flushing")
		writeProxyError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		if _, err := fmt.Fprintln(w, line); err != nil {
			slog.ErrorContext(ctx, "failed to write SSE line", "error", err)
			return
		}
		flusher.Flush()

		// When stream ends, increment proxy steps
		if line == "data: [DONE]" {
			h.incrementSteps(ctx, userID)
			return
		}
	}

	if err := scanner.Err(); err != nil {
		slog.ErrorContext(ctx, "error reading SSE stream", "error", err)
	}

	// Stream ended without [DONE] -- still count the step
	h.incrementSteps(ctx, userID)
}

// handleNonStreamResponse forwards the JSON response and increments steps.
func (h *ProxyHandler) handleNonStreamResponse(ctx context.Context, w http.ResponseWriter, resp *http.Response, userID string) {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		slog.ErrorContext(ctx, "failed to copy response body", "error", err)
		return
	}

	h.incrementSteps(ctx, userID)
}

// forwardErrorResponse forwards a non-2xx response from DeepInfra to the client.
func (h *ProxyHandler) forwardErrorResponse(ctx context.Context, w http.ResponseWriter, resp *http.Response) {
	for key, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(key, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		slog.ErrorContext(ctx, "failed to forward error response", "error", err)
	}
}

// handleAuthError maps authorization errors to appropriate HTTP status codes.
func (h *ProxyHandler) handleAuthError(ctx context.Context, w http.ResponseWriter, err error) {
	code := errors.GetCode(err)
	switch code {
	case "RATE_LIMITED":
		writeProxyError(w, http.StatusTooManyRequests, errors.UserMessage(err))
	case "QUOTA_EXHAUSTED":
		writeProxyError(w, http.StatusPaymentRequired, errors.UserMessage(err))
	case errors.CodeForbidden:
		writeProxyError(w, http.StatusForbidden, errors.UserMessage(err))
	case errors.CodeInvalidInput:
		writeProxyError(w, http.StatusBadRequest, errors.UserMessage(err))
	default:
		slog.ErrorContext(ctx, "proxy authorization failed", "error", err)
		writeProxyError(w, http.StatusInternalServerError, "internal error")
	}
}

// incrementSteps increments the proxy steps counter. Logs error but does not fail the request.
func (h *ProxyHandler) incrementSteps(ctx context.Context, userID string) {
	if err := h.stepIncrementer.IncrementProxySteps(ctx, userID); err != nil {
		slog.ErrorContext(ctx, "failed to increment proxy steps", "user_id", userID, "error", err)
	}
}

// buildOutboundBody modifies the request body: sets the model and removes the custom "role" field.
func buildOutboundBody(originalBody []byte, targetModel string) ([]byte, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(originalBody, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal body: %w", err)
	}

	// Set model
	modelBytes, err := json.Marshal(targetModel)
	if err != nil {
		return nil, fmt.Errorf("marshal model: %w", err)
	}
	raw["model"] = modelBytes

	// Remove our custom "role" field (not part of OpenAI API)
	delete(raw, "role")

	return json.Marshal(raw)
}

// writeProxyError writes a JSON error response for the proxy endpoint.
func writeProxyError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"type":    "proxy_error",
		},
	}); err != nil {
		slog.Error("failed to write proxy error response", "error", err)
	}
}
