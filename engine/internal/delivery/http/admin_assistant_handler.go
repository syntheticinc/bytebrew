package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

const builderAssistantAgentName = "builder-assistant"

// AdminAssistantHandler serves POST /api/v1/admin/assistant/chat.
// Admin-only endpoint for the builder-assistant. Does NOT require a chat trigger.
type AdminAssistantHandler struct {
	service          ChatService
	forwardHeadersFn func() []string
}

// NewAdminAssistantHandler creates a new AdminAssistantHandler.
func NewAdminAssistantHandler(service ChatService, forwardHeadersFn func() []string) *AdminAssistantHandler {
	return &AdminAssistantHandler{service: service, forwardHeadersFn: forwardHeadersFn}
}

// Chat handles admin assistant chat — same logic as ChatHandler.Chat but fixed to builder-assistant.
func (h *AdminAssistantHandler) Chat(w http.ResponseWriter, r *http.Request) {
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

	events, err := h.service.Chat(ctx, builderAssistantAgentName, req.Message, req.UserID, req.SessionID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	// Non-streaming: collect all events and return JSON.
	if req.Stream != nil && !*req.Stream {
		h.handleNonStreaming(w, events)
		return
	}

	// Streaming: SSE.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})
	_ = rc.SetReadDeadline(time.Time{})

	flush := findFlusher(w)

	w.Header().Del("Content-Length")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)
	flush()

	_, _ = io.WriteString(w, ": ok\n\n")
	flush()

	for event := range events {
		_, _ = io.WriteString(w, "event: "+event.Type+"\ndata: "+event.Data+"\n\n")
		flush()
	}
}

// handleNonStreaming collects SSE events and returns a single JSON response.
func (h *AdminAssistantHandler) handleNonStreaming(w http.ResponseWriter, events <-chan SSEEvent) {
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
					if content != "" {
						message = content
					}
				} else {
					message += content
				}
			}
		case "tool_call":
			toolName, _ := data["tool"].(string)
			input, _ := data["content"].(string)
			lastTool = toolName
			toolCalls = append(toolCalls, toolCallEntry{Tool: toolName, Input: input})
		case "tool_result":
			output, _ := data["content"].(string)
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

	if message == "" && errMsg != "" {
		message = errMsg
	}

	resp := nonStreamResponse{
		SessionID: sessionID,
		Agent:     builderAssistantAgentName,
		Message:   message,
		Error:     errMsg,
		ToolCalls: toolCalls,
	}
	writeJSON(w, http.StatusOK, resp)
}

// buildRequestContext extracts configured forward headers from the HTTP request.
func (h *AdminAssistantHandler) buildRequestContext(r *http.Request) context.Context {
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

	return domain.WithRequestContext(ctx, &domain.RequestContext{Headers: headers})
}
