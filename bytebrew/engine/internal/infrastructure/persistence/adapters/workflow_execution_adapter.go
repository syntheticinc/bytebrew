package adapters

import (
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
)

// WorkflowExecutionToModel converts domain WorkflowExecution to persistence model
func WorkflowExecutionToModel(execution *domain.WorkflowExecution) *models.WorkflowExecution {
	if execution == nil {
		return nil
	}

	var id uuid.UUID
	if execution.ID != "" {
		id, _ = uuid.Parse(execution.ID)
	}

	taskID, _ := uuid.Parse(execution.TaskID)

	return &models.WorkflowExecution{
		ID:           id,
		TaskID:       taskID,
		WorkflowType: execution.WorkflowType,
		Status:       string(execution.Status),
		CreatedAt:    execution.CreatedAt,
		UpdatedAt:    execution.UpdatedAt,
	}
}

// WorkflowExecutionFromModel converts persistence model to domain WorkflowExecution
func WorkflowExecutionFromModel(model *models.WorkflowExecution) (*domain.WorkflowExecution, error) {
	if model == nil {
		return nil, nil
	}

	execution := &domain.WorkflowExecution{
		ID:           model.ID.String(),
		TaskID:       model.TaskID.String(),
		WorkflowType: model.WorkflowType,
		Status:       domain.WorkflowStatus(model.Status),
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}

	return execution, nil
}
