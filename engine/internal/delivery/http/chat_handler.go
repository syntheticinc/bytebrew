package http

import (
	"context"
	"encoding/json"
	"fmt"
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
	service        ChatService
	forwardHeaders []string // HTTP headers to capture into RequestContext
}

// NewChatHandler creates a new ChatHandler.
// forwardHeaders is the union of all forward_headers across MCP server configs.
func NewChatHandler(service ChatService, forwardHeaders []string) *ChatHandler {
	return &ChatHandler{service: service, forwardHeaders: forwardHeaders}
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
	// Go's ResponseWriter handles chunked encoding automatically when we
	// call Flush() without setting Content-Length. Do NOT set Transfer-Encoding
	// explicitly (Go docs: "It is an error to set Transfer-Encoding").
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		return
	}

	// Send initial SSE comment to start the stream and prevent middleware
	// from buffering. This also triggers Go to use chunked encoding.
	_, _ = fmt.Fprintf(w, ": ok\n\n")
	flusher.Flush()

	// Block on events — handler stays alive until channel closes (on "done" event).
	// The heartbeat keeps the connection alive during LLM processing.
	for event := range events {
		_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, event.Data)
		flusher.Flush()
	}
}

// handleNonStreaming collects SSE events and returns a single JSON response.
func (h *ChatHandler) handleNonStreaming(w http.ResponseWriter, agentName string, events <-chan SSEEvent) {
	var (
		message   string
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
					message = content // replace with final
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
		case "done":
			if sid, ok := data["session_id"].(string); ok {
				sessionID = sid
			}
		}
	}

	resp := nonStreamResponse{
		SessionID: sessionID,
		Agent:     agentName,
		Message:   message,
		ToolCalls: toolCalls,
	}
	writeJSON(w, http.StatusOK, resp)
}

// buildRequestContext extracts configured forward headers from the HTTP request
// and stores them in a domain.RequestContext within the request's Go context.
func (h *ChatHandler) buildRequestContext(r *http.Request) context.Context {
	ctx := r.Context()
	if len(h.forwardHeaders) == 0 {
		return ctx
	}

	headers := make(map[string]string)
	for _, name := range h.forwardHeaders {
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
