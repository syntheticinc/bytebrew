package adapters

import (
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
)

// messageMetadata represents the full metadata stored in the database
type messageMetadata struct {
	UserMetadata map[string]string     `json:"user_metadata,omitempty"`
	ToolCalls    []domain.ToolCallInfo `json:"tool_calls,omitempty"`
	ToolCallID   string                `json:"tool_call_id,omitempty"`
	ToolName     string                `json:"tool_name,omitempty"`
}

// MessageToModel converts domain Message to RuntimeMessageModel
func MessageToModel(message *domain.Message) (*models.RuntimeMessageModel, error) {
	if message == nil {
		return nil, nil
	}

	id := message.ID
	if id == "" {
		id = uuid.New().String()
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

	return &models.RuntimeMessageModel{
		ID:          id,
		SessionID:   message.SessionID,
		MessageType: string(message.Type),
		Sender:      message.Sender,
		AgentID:     message.AgentID,
		Content:     message.Content,
		Metadata:    string(metadataJSON),
		CreatedAt:   message.CreatedAt,
		UpdatedAt:   message.CreatedAt,
	}, nil
}

// MessageFromModel converts RuntimeMessageModel to domain Message
func MessageFromModel(model *models.RuntimeMessageModel) (*domain.Message, error) {
	if model == nil {
		return nil, nil
	}

	message := &domain.Message{
		ID:        model.ID,
		SessionID: model.SessionID,
		Type:      domain.MessageType(model.MessageType),
		Sender:    model.Sender,
		AgentID:   model.AgentID,
		Content:   model.Content,
		Metadata:  make(map[string]string),
		CreatedAt: model.CreatedAt,
	}

	// Deserialize metadata if present
	if model.Metadata != "" {
		var meta messageMetadata
		if err := json.Unmarshal([]byte(model.Metadata), &meta); err == nil {
			if meta.UserMetadata != nil {
				message.Metadata = meta.UserMetadata
			}
			message.ToolCalls = meta.ToolCalls
			message.ToolCallID = meta.ToolCallID
			message.ToolName = meta.ToolName
		}
	}

	return message, nil
}
