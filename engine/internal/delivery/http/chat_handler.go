package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// ChatService handles agent chat sessions via SSE.
type ChatService interface {
	// Chat starts a chat session and streams events.
	// Returns a channel of SSE events and an error.
	Chat(ctx context.Context, agentName, message, userID, sessionID string) (<-chan SSEEvent, error)
}

// ChatHandler serves POST /api/v1/agents/{name}/chat with SSE streaming.
type ChatHandler struct {
	service          ChatService
	forwardHeadersFn func() []string // dynamic — returns current forward headers
}

// NewChatHandler creates a new ChatHandler.
// forwardHeadersFn returns the current union of all forward_headers across MCP server configs.
// It is called on every request so that config reloads take effect immediately.
func NewChatHandler(service ChatService, forwardHeadersFn func() []string) *ChatHandler {
	return &ChatHandler{service: service, forwardHeadersFn: forwardHeadersFn}
}

type chatRequest struct {
	Message   string `json:"message"`
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	Stream    *bool  `json:"stream,omitempty"` // default true
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

	ctx := h.buildRequestContext(r)
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

	// Get the underlying Flusher. chi wraps ResponseWriter, but
	// http.NewResponseController can unwrap it.
	rc := http.NewResponseController(w)

	// CRITICAL: WriteHeader MUST be called before any Write.
	// This commits the response headers immediately and prevents
	// Go from buffering the entire body to compute Content-Length.
	w.WriteHeader(http.StatusOK)

	// Write initial SSE comment and flush to send headers to client.
	_, _ = io.WriteString(w, ": ok\n\n")
	_ = rc.Flush()

	for event := range events {
		_, _ = io.WriteString(w, "event: "+event.Type+"\ndata: "+event.Data+"\n\n")
		_ = rc.Flush()
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
