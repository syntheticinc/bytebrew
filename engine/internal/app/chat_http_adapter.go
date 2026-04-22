package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/google/uuid"

	pb "github.com/syntheticinc/bytebrew/engine/api/proto/gen"
	deliveryhttp "github.com/syntheticinc/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/agentregistry"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/flowregistry"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/engine/internal/service/sessionprocessor"
	pkgerrors "github.com/syntheticinc/bytebrew/engine/pkg/errors"
)

// schemaChatRepo narrows the schema repository to the operations the chat
// dispatcher needs: load for chat_enabled + entry_agent_id, and stamp
// chat_last_fired_at on the first message of a session. Defined consumer-side
// so the adapter can be unit-tested against a fake.
type schemaChatRepo interface {
	GetModelByID(ctx context.Context, id string) (*models.SchemaModel, error)
	MarkChatFired(ctx context.Context, id string) error
}

// chatSessionPersister persists chat sessions to the DB.
type chatSessionPersister interface {
	Create(ctx context.Context, session *models.SessionModel) error
	Update(ctx context.Context, id string, updates map[string]interface{}) error
}

// chatServiceHTTPAdapter bridges SessionRegistry + SessionProcessor to the
// deliveryhttp.ChatService interface for the REST chat endpoint.
type chatServiceHTTPAdapter struct {
	registry    *flowregistry.SessionRegistry
	processor   *sessionprocessor.Processor
	agents      *agentregistry.AgentRegistry  // non-nil in single-tenant (CE) mode
	registryMgr *agentregistry.Manager        // non-nil in multi-tenant (Cloud/EE) mode
	schemas     schemaChatRepo       // optional — nil in tests / no-DB mode
	sessions    chatSessionPersister // optional — nil when no DB
	chatEnabled bool                 // false when no LLM model configured
}

// resolveRegistry returns the AgentRegistry for the current request context.
// In single-tenant mode it returns a.agents directly; in multi-tenant mode it
// delegates to registryMgr.GetForContext to get the per-tenant registry.
func (a *chatServiceHTTPAdapter) resolveRegistry(ctx context.Context) (*agentregistry.AgentRegistry, error) {
	if a.agents != nil {
		return a.agents, nil
	}
	if a.registryMgr != nil {
		return a.registryMgr.GetForContext(ctx)
	}
	return nil, fmt.Errorf("no agent registry configured")
}

// Chat creates (or resumes) a session for the given schema, enqueues the
// user message, subscribes to events, and returns an SSEEvent channel that
// closes when processing stops.
//
// Resolution: schemaID → SchemaModel.entry_agent_id → agentregistry.GetByID →
// agent name used by SessionRegistry and Processor. Chat is allowed only when
// schemas.chat_enabled = true; if disabled, NotFound is returned so the route
// doesn't leak existence of chat-disabled schemas.
func (a *chatServiceHTTPAdapter) Chat(ctx context.Context, schemaID, message, userSub, sessionID string) (<-chan deliveryhttp.SSEEvent, error) {
	if userSub == "" {
		return nil, pkgerrors.InvalidInput("user_sub is required")
	}
	if a.schemas == nil {
		return nil, fmt.Errorf("schema repo not wired")
	}

	// Tenant-scoped schema lookup must happen before any other check so that
	// cross-tenant requests get NotFound (404) rather than leaking a 500.
	schema, err := a.schemas.GetModelByID(ctx, schemaID)
	if err != nil {
		return nil, fmt.Errorf("load schema: %w", err)
	}
	if schema == nil {
		return nil, pkgerrors.NotFound(fmt.Sprintf("schema not found: %s", schemaID))
	}
	if !schema.ChatEnabled {
		return nil, pkgerrors.NotFound(fmt.Sprintf("schema not found: %s", schemaID))
	}
	if schema.EntryAgentID == nil || *schema.EntryAgentID == "" {
		return nil, pkgerrors.InvalidInput("schema has no entry agent")
	}

	registry, err := a.resolveRegistry(ctx)
	if err != nil {
		return nil, fmt.Errorf("no agents configured: %w", err)
	}

	entryAgent, err := registry.GetByID(ctx, *schema.EntryAgentID)
	if err != nil {
		return nil, pkgerrors.NotFound(fmt.Sprintf("entry agent not found for schema %s", schemaID))
	}
	agentName := entryAgent.Record.Name

	isNewSession := sessionID == ""
	if isNewSession {
		sessionID = uuid.New().String()
	} else if _, err := uuid.Parse(sessionID); err != nil {
		return nil, pkgerrors.InvalidInput("session_id must be a valid UUID")
	}

	if !a.registry.HasSession(sessionID) {
		isNewSession = true
		a.registry.CreateSession(sessionID, "", userSub, "", "", agentName)
	}

	// On the first message stamp chat_last_fired_at and persist the session.
	// Non-fatal on errors — chat must still stream if bookkeeping hiccups.
	if isNewSession {
		if markErr := a.schemas.MarkChatFired(ctx, schemaID); markErr != nil {
			slog.WarnContext(ctx, "mark schema chat fired failed", "schema_id", schemaID, "error", markErr)
		}
		if a.sessions != nil {
			m := &models.SessionModel{
				ID:       sessionID,
				SchemaID: schemaID,
				UserSub:  userSub,
				Status:   "active",
				TenantID: domain.CETenantID,
			}
			if createErr := a.sessions.Create(ctx, m); createErr != nil {
				slog.WarnContext(ctx, "persist chat session failed", "session_id", sessionID, "error", createErr)
			}
		}
	}

	// Subscribe BEFORE enqueueing so we don't miss events.
	eventCh, cleanup := a.registry.Subscribe(sessionID)

	if err := a.registry.EnqueueMessage(sessionID, message); err != nil {
		cleanup()
		return nil, fmt.Errorf("enqueue message: %w", err)
	}

	a.processor.StartProcessing(ctx, sessionID)

	// Fan-out: read proto events, convert to SSE, close when processing stops.
	// Buffered channel avoids deadlock when the HTTP handler is slow to read.
	sseCh := make(chan deliveryhttp.SSEEvent, 64)
	go func() {
		defer close(sseCh)
		defer cleanup()

		for protoEvent := range eventCh {
			sseEvent := convertSessionEventToSSE(protoEvent, sessionID)
			if sseEvent == nil {
				continue
			}
			sseCh <- *sseEvent

			if sseEvent.Type == "done" {
				if a.sessions != nil {
					if updateErr := a.sessions.Update(context.Background(), sessionID, map[string]interface{}{"status": "completed"}); updateErr != nil {
						slog.WarnContext(context.Background(), "update chat session status failed", "session_id", sessionID, "error", updateErr)
					}
				}
				return
			}
		}
	}()

	return sseCh, nil
}

// convertSessionEventToSSE maps a pb.SessionEvent to an SSEEvent.
// Returns nil for event types that should not be forwarded over SSE.
func convertSessionEventToSSE(event *pb.SessionEvent, sessionID string) *deliveryhttp.SSEEvent {
	switch event.GetType() {
	case pb.SessionEventType_SESSION_EVENT_REASONING:
		return sseEventJSON("thinking", map[string]interface{}{
			"content": event.GetContent(),
		})

	case pb.SessionEventType_SESSION_EVENT_ANSWER_CHUNK:
		return sseEventJSON("message_delta", map[string]interface{}{
			"content": event.GetContent(),
		})

	case pb.SessionEventType_SESSION_EVENT_ANSWER:
		return sseEventJSON("message", map[string]interface{}{
			"content": event.GetContent(),
		})

	case pb.SessionEventType_SESSION_EVENT_TOOL_EXECUTION_START:
		data := map[string]interface{}{
			"tool":    event.GetToolName(),
			"call_id": event.GetCallId(),
		}
		if args := event.GetToolArguments(); len(args) > 0 {
			data["arguments"] = args
		}
		return sseEventJSON("tool_call", data)

	case pb.SessionEventType_SESSION_EVENT_TOOL_EXECUTION_END:
		return sseEventJSON("tool_result", map[string]interface{}{
			"tool":      event.GetToolName(),
			"call_id":   event.GetCallId(),
			"content":   event.GetContent(),
			"summary":   event.GetToolResultSummary(),
			"has_error": event.GetToolHasError(),
		})

	case pb.SessionEventType_SESSION_EVENT_ASK_USER:
		data := map[string]interface{}{
			"content": event.GetContent(),
			"call_id": event.GetCallId(),
		}
		if tn := event.GetToolName(); tn != "" {
			data["tool"] = tn
		}
		return sseEventJSON("confirmation", data)

	case pb.SessionEventType_SESSION_EVENT_PROCESSING_STOPPED:
		data := map[string]interface{}{
			"session_id": sessionID,
		}
		if content := event.GetContent(); content != "" {
			var tokenData map[string]int
			if err := json.Unmarshal([]byte(content), &tokenData); err == nil {
				if t, ok := tokenData["total_tokens"]; ok && t > 0 {
					data["total_tokens"] = t
				}
				if c, ok := tokenData["context_tokens"]; ok && c > 0 {
					data["context_tokens"] = c
				}
			} else {
				if tokens, err := strconv.Atoi(content); err == nil && tokens > 0 {
					data["total_tokens"] = tokens
				}
			}
		}
		return sseEventJSON("done", data)

	case pb.SessionEventType_SESSION_EVENT_ERROR:
		data := map[string]interface{}{
			"content": event.GetContent(),
		}
		if detail := event.GetErrorDetail(); detail != nil {
			data["code"] = detail.GetCode()
			data["message"] = detail.GetMessage()
		}
		return sseEventJSON("error", data)

	default:
		return nil
	}
}

// sseEventJSON creates an SSEEvent with JSON-encoded data.
func sseEventJSON(eventType string, data map[string]interface{}) *deliveryhttp.SSEEvent {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		slog.ErrorContext(context.Background(), "failed to marshal SSE event data", "type", eventType, "error", err)
		return nil
	}
	return &deliveryhttp.SSEEvent{
		Type: eventType,
		Data: string(jsonBytes),
	}
}
