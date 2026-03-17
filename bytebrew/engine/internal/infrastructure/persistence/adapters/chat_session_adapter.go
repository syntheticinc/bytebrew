package adapters

import (
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
)

// ChatSessionToModel converts domain ChatSession to persistence model
func ChatSessionToModel(session *domain.ChatSession) *models.ChatSession {
	if session == nil {
		return nil
	}

	var id uuid.UUID
	if session.ID != "" {
		id, _ = uuid.Parse(session.ID)
	}

	userID, _ := uuid.Parse(session.UserID)

	var projectID *uuid.UUID
	if session.ProjectID != nil {
		pid, _ := uuid.Parse(*session.ProjectID)
		projectID = &pid
	}

	return &models.ChatSession{
		ID:        id,
		UserID:    userID,
		ProjectID: projectID,
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
	}
}

// ChatSessionFromModel converts persistence model to domain ChatSession
func ChatSessionFromModel(model *models.ChatSession) (*domain.ChatSession, error) {
	if model == nil {
		return nil, nil
	}

	var projectID *string
	if model.ProjectID != nil {
		pid := model.ProjectID.String()
		projectID = &pid
	}

	session := &domain.ChatSession{
		ID:        model.ID.String(),
		UserID:    model.UserID.String(),
		ProjectID: projectID,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}

	return session, nil
}
