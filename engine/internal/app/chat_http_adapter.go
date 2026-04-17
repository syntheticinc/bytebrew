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
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/configrepo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/engine/internal/service/sessionprocessor"
	pkgerrors "github.com/syntheticinc/bytebrew/engine/pkg/errors"
)

// chatTriggerMarker narrows GORMTriggerRepository to the operations used by
// the chat dispatcher: look up the enabled chat trigger for an agent and
// stamp its last_fired_at on the first message of a new session (§4.1).
type chatTriggerMarker interface {
	FindEnabledChatTrigger(ctx context.Context, agentName string) (*models.TriggerModel, error)
	MarkFired(ctx context.Context, id string) error
}

// chatSessionPersister persists chat sessions to the DB.
// Narrowed consumer-side interface — only Create and Update are needed here.
type chatSessionPersister interface {
	Create(ctx context.Context, session *models.SessionModel) error
	Update(ctx context.Context, id string, updates map[string]interface{}) error
}

// chatServiceHTTPAdapter bridges SessionRegistry + SessionProcessor to the
// deliveryhttp.ChatService interface for the REST chat endpoint.
type chatServiceHTTPAdapter struct {
	registry    *flowregistry.SessionRegistry
	processor   *sessionprocessor.Processor
	agents      *agentregistry.AgentRegistry
	triggers    chatTriggerMarker    // optional — nil in tests / no-DB mode
	sessions    chatSessionPersister // optional — nil when no DB
	chatEnabled bool                 // false when no LLM model configured
}

// Chat creates (or resumes) a session, enqueues the message, subscribes to
// events, and returns an SSEEvent channel that closes when processing stops.
func (a *chatServiceHTTPAdapter) Chat(ctx context.Context, agentName, message, userID, sessionID string) (<-chan deliveryhttp.SSEEvent, error) {
	if a.agents == nil {
		return nil, fmt.Errorf("no agents configured")
	}

	if _, err := a.agents.Get(agentName); err != nil {
		return nil, pkgerrors.NotFound(fmt.Sprintf("agent not found: %s", agentName))
	}

	// Create a new session if none provided.
	isNewSession := sessionID == ""
	if isNewSession {
		sessionID = uuid.New().String()
	} else if _, err := uuid.Parse(sessionID); err != nil {
		return nil, pkgerrors.InvalidInput("session_id must be a valid UUID")
	}

	// Create session in registry (idempotent — reuses existing if already present).
	if !a.registry.HasSession(sessionID) {
		// Registry didn't know this session either — still "new" from the
		// trigger's perspective.
		isNewSession = true
		a.registry.CreateSession(sessionID, "", userID, "", "", agentName)
	}

	// §4.1: first message of a session → stamp the chat trigger's
	// last_fired_at and persist the session to DB using trigger.SchemaID.
	// Non-fatal on any error — chat must still work when trigger lookup fails.
	if isNewSession && a.triggers != nil {
		if trigger, lookupErr := a.triggers.FindEnabledChatTrigger(ctx, agentName); lookupErr == nil && trigger != nil {
			if markErr := a.triggers.MarkFired(ctx, trigger.ID); markErr != nil {
				slog.WarnContext(ctx, "mark chat trigger fired failed", "trigger_id", trigger.ID, "error", markErr)
			}
			if a.sessions != nil {
				m := &models.SessionModel{
					ID:       sessionID,
					SchemaID: trigger.SchemaID,
					Status:   "running",
					TenantID: domain.CETenantID,
				}
				if userID != "" {
					m.UserID = &userID
				}
				if createErr := a.sessions.Create(ctx, m); createErr != nil {
					slog.WarnContext(ctx, "persist chat session failed", "session_id", sessionID, "error", createErr)
				}
			}
		}
	}

	// Subscribe BEFORE enqueueing so we don't miss events.
	eventCh, cleanup := a.registry.Subscribe(sessionID)

	// Enqueue the user message.
	if err := a.registry.EnqueueMessage(sessionID, message); err != nil {
		cleanup()
		return nil, fmt.Errorf("enqueue message: %w", err)
	}

	// Start processing with the enriched context (carries RequestContext for MCP header forwarding).
	a.processor.StartProcessing(ctx, sessionID)

	// Fan-out: read proto events, convert to SSE, close when processing stops.
	// Buffered channel prevents deadlock: PublishEvent holds entry.mu.Lock while
	// sending to subscriber channel. If sseCh is unbuffered and the HTTP handler
	// is slow to read/flush, the fan-out goroutine blocks on sseCh send, which
	// blocks the subscriber channel read, which blocks PublishEvent, which holds
	// the lock and blocks ALL subsequent events — causing stream truncation.
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
						slog.Warn("update chat session status failed", "session_id", sessionID, "error", updateErr)
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
			// Try JSON first (new format: {"total_tokens":N,"context_tokens":N})
			var tokenData map[string]int
			if err := json.Unmarshal([]byte(content), &tokenData); err == nil {
				if t, ok := tokenData["total_tokens"]; ok && t > 0 {
					data["total_tokens"] = t
				}
				if c, ok := tokenData["context_tokens"]; ok && c > 0 {
					data["context_tokens"] = c
				}
			} else {
				// Legacy fallback: plain int format
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
		// PROCESSING_STARTED, USER_MESSAGE, PLAN_UPDATE, UNSPECIFIED — skip.
		return nil
	}
}

// sseEventJSON creates an SSEEvent with JSON-encoded data.
func sseEventJSON(eventType string, data map[string]interface{}) *deliveryhttp.SSEEvent {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		slog.Error("failed to marshal SSE event data", "type", eventType, "error", err)
		return nil
	}
	return &deliveryhttp.SSEEvent{
		Type: eventType,
		Data: string(jsonBytes),
	}
}

// chatTriggerCheckerAdapter implements deliveryhttp.ChatTriggerChecker.
type chatTriggerCheckerAdapter struct {
	repo *configrepo.GORMTriggerRepository
}

func (a *chatTriggerCheckerAdapter) HasEnabledChatTrigger(ctx context.Context, agentName string) (bool, error) {
	return a.repo.HasEnabledChatTrigger(ctx, agentName)
}
