package adapters

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
)

// messageMetadata represents the full metadata stored in the database
type messageMetadata struct {
	UserMetadata map[string]string      `json:"user_metadata,omitempty"`
	ToolCalls    []domain.ToolCallInfo  `json:"tool_calls,omitempty"`
	ToolCallID   string                 `json:"tool_call_id,omitempty"`
	ToolName     string                 `json:"tool_name,omitempty"`
}

// MessageToModel converts domain Message to persistence model
func MessageToModel(message *domain.Message) (*models.Message, error) {
	if message == nil {
		return nil, nil
	}

	var id uuid.UUID
	if message.ID == "" {
		id = uuid.New()
	} else {
		var err error
		id, err = uuid.Parse(message.ID)
		if err != nil {
			return nil, fmt.Errorf("parse message ID %q: %w", message.ID, err)
		}
	}
	sessionID, err := uuid.Parse(message.SessionID)
	if err != nil {
		return nil, fmt.Errorf("parse session ID %q: %w", message.SessionID, err)
	}

	// Serialize metadata including ToolCalls
	meta := messageMetadata{
		UserMetadata: message.Metadata,
		ToolCalls:    message.ToolCalls,
		ToolCallID:   message.ToolCallID,
		ToolName:     message.ToolName,
	}

	metadataJSON, err := json.Marshal(meta)
	if err != nil {
		slog.Error("failed to marshal message metadata", "message_id", message.ID, "error", err)
		metadataJSON = []byte("{}")
	}

	model := &models.Message{
		ID:          id,
		SessionID:   sessionID,
		MessageType: string(message.Type),
		Sender:      message.Sender,
		Content:     message.Content,
		Metadata:    metadataJSON,
		CreatedAt:   message.CreatedAt,
		UpdatedAt:   message.CreatedAt,
	}

	// Set AgentID if present (nullable field for backward compatibility)
	if message.AgentID != "" {
		agentID := message.AgentID
		model.AgentID = &agentID
	}

	return model, nil
}

// MessageFromModel converts persistence model to domain Message
func MessageFromModel(model *models.Message) (*domain.Message, error) {
	if model == nil {
		return nil, nil
	}

	message := &domain.Message{
		ID:        model.ID.String(),
		SessionID: model.SessionID.String(),
		Type:      domain.MessageType(model.MessageType),
		Sender:    model.Sender,
		Content:   model.Content,
		Metadata:  make(map[string]string),
		CreatedAt: model.CreatedAt,
	}

	// Deserialize metadata if present
	if len(model.Metadata) > 0 {
		var meta messageMetadata
		if err := json.Unmarshal(model.Metadata, &meta); err == nil {
			// Restore user metadata
			if meta.UserMetadata != nil {
				message.Metadata = meta.UserMetadata
			}
			// Restore tool-related fields
			message.ToolCalls = meta.ToolCalls
			message.ToolCallID = meta.ToolCallID
			message.ToolName = meta.ToolName
		}
	}

	// Restore AgentID if present
	if model.AgentID != nil {
		message.AgentID = *model.AgentID
	}

	return message, nil
}
