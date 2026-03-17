package adapters

import (
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
)

// SubtaskToTaskModel converts domain Subtask to persistence Task model
func SubtaskToTaskModel(subtask *domain.Subtask) *models.Task {
	if subtask == nil {
		return nil
	}

	id, _ := uuid.Parse(subtask.ID)
	sessionID, _ := uuid.Parse(subtask.SessionID)

	return &models.Task{
		ID:          id,
		Code:        subtask.ID, // Using ID as code for now
		Title:       subtask.Description,
		Description: subtask.Description,
		TaskType:    "question",
		SessionID:   sessionID,
		Status:      string(subtask.Status),
		CreatedAt:   subtask.CreatedAt,
		UpdatedAt:   subtask.UpdatedAt,
	}
}

// TaskModelToSubtask converts persistence Task model to domain Subtask
func TaskModelToSubtask(model *models.Task) (*domain.Subtask, error) {
	if model == nil {
		return nil, nil
	}

	subtask := &domain.Subtask{
		ID:          model.ID.String(),
		SessionID:   model.SessionID.String(),
		Description: model.Title,
		Status:      domain.SubtaskStatus(model.Status),
		Result:      "", // Not stored in model
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}

	return subtask, nil
}
