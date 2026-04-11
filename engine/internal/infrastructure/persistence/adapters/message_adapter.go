package adapters

import (
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
)

// EventToModel converts a domain Message (event) to RuntimeEventModel.
func EventToModel(event *domain.Message) (*models.RuntimeEventModel, error) {
	if event == nil {
		return nil, nil
	}

	id := event.ID
	if id == "" {
		id = uuid.New().String()
	}

	payload := event.Payload
	if payload == nil {
		payload = json.RawMessage("{}")
	}

	return &models.RuntimeEventModel{
		ID:        id,
		SessionID: event.SessionID,
		EventType: string(event.Type),
		AgentID:   event.AgentID,
		CallID:    event.CallID,
		Payload:   payload,
		CreatedAt: event.CreatedAt,
	}, nil
}

// EventFromModel converts a RuntimeEventModel to a domain Message (event).
func EventFromModel(model *models.RuntimeEventModel) (*domain.Message, error) {
	if model == nil {
		return nil, nil
	}

	payload := model.Payload
	if payload == nil {
		payload = json.RawMessage("{}")
	}

	return &domain.Message{
		ID:        model.ID,
		SessionID: model.SessionID,
		Type:      domain.MessageType(model.EventType),
		AgentID:   model.AgentID,
		CallID:    model.CallID,
		Payload:   payload,
		CreatedAt: model.CreatedAt,
	}, nil
}

// Legacy aliases for transition period (used by code that still references old names).

// MessageToModel wraps EventToModel for backward compatibility during refactor.
func MessageToModel(message *domain.Message) (*models.RuntimeEventModel, error) {
	slog.Warn("MessageToModel is deprecated, use EventToModel")
	return EventToModel(message)
}

// MessageFromModel wraps EventFromModel for backward compatibility during refactor.
func MessageFromModel(model *models.RuntimeEventModel) (*domain.Message, error) {
	slog.Warn("MessageFromModel is deprecated, use EventFromModel")
	return EventFromModel(model)
}
