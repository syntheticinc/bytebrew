package repository

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type workflowExecutionRepository struct {
	db *gorm.DB
}

// NewWorkflowExecutionRepository creates a new WorkflowExecutionRepository
func NewWorkflowExecutionRepository(db *gorm.DB) *workflowExecutionRepository {
	return &workflowExecutionRepository{db: db}
}

func (r *workflowExecutionRepository) Create(ctx context.Context, execution *models.WorkflowExecution) error {
	if err := r.db.WithContext(ctx).Create(execution).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to create workflow execution")
	}
	return nil
}

func (r *workflowExecutionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
	var execution models.WorkflowExecution
	if err := r.db.WithContext(ctx).Preload("Steps").First(&execution, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "workflow execution not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get workflow execution by id")
	}
	return &execution, nil
}

func (r *workflowExecutionRepository) GetByTaskID(ctx context.Context, taskID uuid.UUID) (*models.WorkflowExecution, error) {
	var execution models.WorkflowExecution
	if err := r.db.WithContext(ctx).Preload("Steps").Where("task_id = ?", taskID).First(&execution).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "workflow execution not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get workflow execution by task id")
	}
	return &execution, nil
}

func (r *workflowExecutionRepository) Update(ctx context.Context, execution *models.WorkflowExecution) error {
	if err := r.db.WithContext(ctx).Save(execution).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to update workflow execution")
	}
	return nil
}

func (r *workflowExecutionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.WorkflowExecution{}, "id = ?", id).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to delete workflow execution")
	}
	return nil
}

type workflowStepRepository struct {
	db *gorm.DB
}

// NewWorkflowStepRepository creates a new WorkflowStepRepository
func NewWorkflowStepRepository(db *gorm.DB) *workflowStepRepository {
	return &workflowStepRepository{db: db}
}

func (r *workflowStepRepository) Create(ctx context.Context, step *models.WorkflowStep) error {
	if err := r.db.WithContext(ctx).Create(step).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to create workflow step")
	}
	return nil
}

func (r *workflowStepRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.WorkflowStep, error) {
	var step models.WorkflowStep
	if err := r.db.WithContext(ctx).Preload("AgentType").First(&step, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "workflow step not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get workflow step by id")
	}
	return &step, nil
}

func (r *workflowStepRepository) GetByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*models.WorkflowStep, error) {
	var steps []*models.WorkflowStep
	if err := r.db.WithContext(ctx).Preload("AgentType").Where("workflow_id = ?", workflowID).Order("step_number ASC").Find(&steps).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get workflow steps by workflow id")
	}
	return steps, nil
}

func (r *workflowStepRepository) GetCurrentStep(ctx context.Context, workflowID uuid.UUID) (*models.WorkflowStep, error) {
	var step models.WorkflowStep
	if err := r.db.WithContext(ctx).Preload("AgentType").Where("workflow_id = ? AND status = ?", workflowID, "in_progress").First(&step).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "current workflow step not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get current workflow step")
	}
	return &step, nil
}

func (r *workflowStepRepository) Update(ctx context.Context, step *models.WorkflowStep) error {
	if err := r.db.WithContext(ctx).Save(step).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to update workflow step")
	}
	return nil
}

func (r *workflowStepRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.WorkflowStep{}, "id = ?", id).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to delete workflow step")
	}
	return nil
}
