package adapters

import (
	"encoding/json"
	"fmt"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/persistence/models"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

// SerializeSchemaMessages serializes []*schema.Message to JSON bytes
func SerializeSchemaMessages(messages []*schema.Message) ([]byte, error) {
	if messages == nil {
		messages = []*schema.Message{}
	}
	data, err := json.Marshal(messages)
	if err != nil {
		return nil, fmt.Errorf("serialize schema messages: %w", err)
	}
	return data, nil
}

// DeserializeSchemaMessages deserializes JSON bytes to []*schema.Message
func DeserializeSchemaMessages(data []byte) ([]*schema.Message, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var messages []*schema.Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("deserialize schema messages: %w", err)
	}
	return messages, nil
}

// AgentContextSnapshotToModel converts domain snapshot to DB model
func AgentContextSnapshotToModel(snapshot *domain.AgentContextSnapshot) *models.AgentContextSnapshot {
	if snapshot == nil {
		return nil
	}

	id, _ := uuid.Parse(snapshot.ID)
	sessionID, _ := uuid.Parse(snapshot.SessionID)

	return &models.AgentContextSnapshot{
		ID:            id,
		SessionID:     sessionID,
		AgentID:       snapshot.AgentID,
		FlowType:      string(snapshot.FlowType),
		SchemaVersion: snapshot.SchemaVersion,
		ContextData:   snapshot.ContextData,
		StepNumber:    snapshot.StepNumber,
		TokenCount:    snapshot.TokenCount,
		Status:        string(snapshot.Status),
		CreatedAt:     snapshot.CreatedAt,
		UpdatedAt:     snapshot.UpdatedAt,
	}
}

// AgentContextSnapshotFromModel converts DB model to domain snapshot
func AgentContextSnapshotFromModel(model *models.AgentContextSnapshot) *domain.AgentContextSnapshot {
	if model == nil {
		return nil
	}

	return &domain.AgentContextSnapshot{
		ID:            model.ID.String(),
		SessionID:     model.SessionID.String(),
		AgentID:       model.AgentID,
		FlowType:      domain.FlowType(model.FlowType),
		SchemaVersion: model.SchemaVersion,
		ContextData:   model.ContextData,
		StepNumber:    model.StepNumber,
		TokenCount:    model.TokenCount,
		Status:        domain.AgentContextStatus(model.Status),
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
	}
}
