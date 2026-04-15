package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/llm"
)

// propagateBYOK translates the http-layer BYOK context keys (set by
// BYOKMiddleware) into a single llm.BYOKCredentials value attached via
// llm.WithBYOKCredentials. The downstream turn executor factory reads
// from there to build an ad-hoc per-end-user ChatModel (V2 §5.8).
//
// No-op when no BYOK context is present — keeps the tenant-configured
// model in use.
func propagateBYOK(ctx context.Context) context.Context {
	provider, _ := ctx.Value(ContextKeyBYOKProvider).(string)
	apiKey, _ := ctx.Value(ContextKeyBYOKAPIKey).(string)
	if provider == "" || apiKey == "" {
		return ctx
	}
	model, _ := ctx.Value(ContextKeyBYOKModel).(string)
	baseURL, _ := ctx.Value(ContextKeyBYOKBaseURL).(string)
	return llm.WithBYOKCredentials(ctx, &llm.BYOKCredentials{
		Provider: provider,
		APIKey:   apiKey,
		Model:    model,
		BaseURL:  baseURL,
	})
}

// ChatService handles agent chat sessions via SSE.
type ChatService interface {
	// Chat starts a chat session and streams events.
	// Returns a channel of SSE events and an error.
	Chat(ctx context.Context, agentName, message, userID, sessionID string) (<-chan SSEEvent, error)
}

// ChatTriggerChecker checks whether an agent has an enabled chat trigger.
type ChatTriggerChecker interface {
	HasEnabledChatTrigger(ctx context.Context, agentName string) (bool, error)
}

// ChatHandler serves POST /api/v1/agents/{name}/chat with SSE streaming.
type ChatHandler struct {
	service        ChatService
	triggerChecker ChatTriggerChecker // nil = gate disabled (e.g. no DB)
	forwardHeadersFn func() []string // dynamic — returns current forward headers
}

// NewChatHandler creates a new ChatHandler.
// forwardHeadersFn returns the current union of all forward_headers across MCP server configs.
// It is called on every request so that config reloads take effect immediately.
func NewChatHandler(service ChatService, triggerChecker ChatTriggerChecker, forwardHeadersFn func() []string) *ChatHandler {
	return &ChatHandler{service: service, triggerChecker: triggerChecker, forwardHeadersFn: forwardHeadersFn}
}

type chatRequest struct {
	Message   string            `json:"message"`
	UserID    string            `json:"user_id"`
	SessionID string            `json:"session_id"`
	Stream    *bool             `json:"stream,omitempty"` // default true
	Headers   map[string]string `json:"headers,omitempty"` // optional headers forwarded to MCP tool calls
}

type nonStreamResponse struct {
	SessionID string          `json:"session_id,omitempty"`
	Agent     string          `json:"agent"`
	Message   string          `json:"message"`
	Error     string          `json:"error,omitempty"`
	ToolCalls []toolCallEntry `json:"tool_calls,omitempty"`
}

type toolCallEntry struct {
	Tool    string `json:"tool"`
	Input   string `json:"input,omitempty"`
	Output  string `json:"output,omitempty"`
}

// Chat handles SSE streaming or non-streaming chat.
func (h *ChatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	agentName := chi.URLParam(r, "name")
	if agentName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "agent name required"})
		return
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message required"})
		return
	}

	// Gate: agent must have an enabled chat trigger.
	if h.triggerChecker != nil {
		ok, err := h.triggerChecker.HasEnabledChatTrigger(r.Context(), agentName)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "trigger check failed"})
			return
		}
		if !ok {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "agent has no enabled chat trigger"})
			return
		}
	}

	ctx := h.buildRequestContext(r)
	// Lift BYOK context keys into the canonical llm.BYOKCredentials value
	// before handing off to the chat service. Downstream layers read them
	// from there to build an ad-hoc per-end-user ChatModel (V2 §5.8).
	ctx = propagateBYOK(ctx)
	if len(req.Headers) > 0 {
		existing := domain.GetRequestContext(ctx)
		merged := make(map[string]string, len(req.Headers))
		if existing != nil {
			for k, v := range existing.Headers {
				merged[k] = v
			}
		}
		for k, v := range req.Headers {
			merged[k] = v
		}
		ctx = domain.WithRequestContext(ctx, &domain.RequestContext{Headers: merged})
	}
	events, err := h.service.Chat(ctx, agentName, req.Message, req.UserID, req.SessionID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	// Non-streaming: collect all events → return JSON
	if req.Stream != nil && !*req.Stream {
		h.handleNonStreaming(w, agentName, events)
		return
	}

	// Streaming: SSE.
	//
	// CRITICAL: Go's net/http buffers small responses and sets Content-Length,
	// which breaks SSE streaming. To force chunked transfer encoding:
	// 1. Set headers
	// 2. Write initial comment
	// 3. Flush IMMEDIATELY — before any events arrive
	//
	// The Flush() call commits the headers with Transfer-Encoding: chunked
	// and sends the first chunk. Subsequent writes become additional chunks.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Disable Read/Write timeouts for SSE — long-running streams (multi-tool ReAct chains)
	// can exceed the server's default timeouts. The model may pause for extended periods
	// during tool calling or reasoning before producing the next token.
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{}) // zero = no deadline
	_ = rc.SetReadDeadline(time.Time{})  // clear read deadline too — prevents context cancellation during long model calls

	// Unwrap to find http.Flusher — chi middleware wraps ResponseWriter.
	flush := findFlusher(w)

	w.Header().Del("Content-Length")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)
	flush() // commit headers immediately — before any body

	_, _ = io.WriteString(w, ": ok\n\n")
	flush()

	for event := range events {
		_, _ = io.WriteString(w, "event: "+event.Type+"\ndata: "+event.Data+"\n\n")
		flush()
	}
}

// handleNonStreaming collects SSE events and returns a single JSON response.
func (h *ChatHandler) handleNonStreaming(w http.ResponseWriter, agentName string, events <-chan SSEEvent) {
	var (
		message   string
		errMsg    string
		toolCalls []toolCallEntry
		sessionID string
		lastTool  string
	)

	for event := range events {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(event.Data), &data); err != nil {
			continue
		}

		switch event.Type {
		case "message", "message_delta":
			if content, ok := data["content"].(string); ok {
				if event.Type == "message" {
					// Only replace if non-empty — the engine sends a trailing
					// "completion signal" ANSWER event with empty content after
					// streaming finishes; ignoring it preserves the real answer.
					if content != "" {
						message = content
					}
				} else {
					message += content // accumulate deltas
				}
			}
		case "tool_call":
			toolName, _ := data["tool"].(string)
			input, _ := data["content"].(string)
			lastTool = toolName
			toolCalls = append(toolCalls, toolCallEntry{Tool: toolName, Input: input})
		case "tool_result":
			output, _ := data["content"].(string)
			// Update last tool call with output
			for i := len(toolCalls) - 1; i >= 0; i-- {
				if toolCalls[i].Tool == lastTool && toolCalls[i].Output == "" {
					toolCalls[i].Output = output
					break
				}
			}
		case "error":
			if content, ok := data["content"].(string); ok && content != "" {
				errMsg = content
			} else if msg, ok := data["message"].(string); ok && msg != "" {
				errMsg = msg
			}
		case "done":
			if sid, ok := data["session_id"].(string); ok {
				sessionID = sid
			}
		}
	}

	// If there was an error and no message content, use the error as the message
	// so the client gets meaningful feedback instead of an empty response.
	if message == "" && errMsg != "" {
		message = errMsg
	}

	resp := nonStreamResponse{
		SessionID: sessionID,
		Agent:     agentName,
		Message:   message,
		Error:     errMsg,
		ToolCalls: toolCalls,
	}
	writeJSON(w, http.StatusOK, resp)
}

// buildRequestContext extracts configured forward headers from the HTTP request
// and stores them in a domain.RequestContext within the request's Go context.
func (h *ChatHandler) buildRequestContext(r *http.Request) context.Context {
	ctx := r.Context()
	forwardHeaders := h.forwardHeadersFn()
	if len(forwardHeaders) == 0 {
		return ctx
	}

	headers := make(map[string]string)
	for _, name := range forwardHeaders {
		if val := r.Header.Get(name); val != "" {
			headers[name] = val
		}
	}
	if len(headers) == 0 {
		return ctx
	}

	rc := &domain.RequestContext{Headers: headers}
	return domain.WithRequestContext(ctx, rc)
}

// findFlusher unwraps a ResponseWriter to find http.Flusher.
// Chi middleware wraps the ResponseWriter, hiding the Flusher interface.
// Returns a no-op function if Flusher is not available.
func findFlusher(w http.ResponseWriter) func() {
	type unwrapper interface{ Unwrap() http.ResponseWriter }
	for {
		if f, ok := w.(http.Flusher); ok {
			return f.Flush
		}
		if u, ok := w.(unwrapper); ok {
			w = u.Unwrap()
			continue
		}
		slog.Warn("[SSE] Flush not available — middleware may be wrapping ResponseWriter without Unwrap()")
		return func() {}
	}
}
