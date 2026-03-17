package adapters

import (
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
)

// WorkflowStepToModel converts domain WorkflowStep to persistence model
func WorkflowStepToModel(step *domain.WorkflowStep) *models.WorkflowStep {
	if step == nil {
		return nil
	}

	var id uuid.UUID
	if step.ID != "" {
		id, _ = uuid.Parse(step.ID)
	}

	workflowID, _ := uuid.Parse(step.WorkflowID)
	agentTypeID, _ := uuid.Parse(step.AgentTypeID)

	return &models.WorkflowStep{
		ID:          id,
		WorkflowID:  workflowID,
		StepNumber:  step.StepNumber,
		AgentTypeID: agentTypeID,
		Status:      string(step.Status),
		CreatedAt:   step.CreatedAt,
		UpdatedAt:   step.UpdatedAt,
	}
}

// WorkflowStepFromModel converts persistence model to domain WorkflowStep
func WorkflowStepFromModel(model *models.WorkflowStep) (*domain.WorkflowStep, error) {
	if model == nil {
		return nil, nil
	}

	step := &domain.WorkflowStep{
		ID:          model.ID.String(),
		WorkflowID:  model.WorkflowID.String(),
		StepNumber:  model.StepNumber,
		AgentTypeID: model.AgentTypeID.String(),
		Status:      domain.WorkflowStepStatus(model.Status),
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}

	return step, nil
}
