package repository

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/adapters"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/errors"
	"gorm.io/gorm"
)

type taskRepository struct {
	db *gorm.DB
}

// NewTaskRepository creates a new TaskRepository
func NewTaskRepository(db *gorm.DB) *taskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) Create(ctx context.Context, subtask *domain.Subtask) error {
	model := adapters.SubtaskToRuntimeModel(subtask)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to create subtask")
	}

	subtask.ID = model.ID
	return nil
}

func (r *taskRepository) GetByID(ctx context.Context, id string) (*domain.Subtask, error) {
	var model models.RuntimeSubtaskModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "subtask not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get subtask by id")
	}

	return adapters.RuntimeModelToSubtask(&model)
}

func (r *taskRepository) GetBySessionID(ctx context.Context, sessionID string) ([]*models.RuntimeSubtaskModel, error) {
	var subtasks []*models.RuntimeSubtaskModel
	if err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).Find(&subtasks).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get subtasks by session id")
	}
	return subtasks, nil
}

func (r *taskRepository) Update(ctx context.Context, subtask *domain.Subtask) error {
	model := adapters.SubtaskToRuntimeModel(subtask)

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to update subtask")
	}
	return nil
}

func (r *taskRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&models.RuntimeSubtaskModel{}, "id = ?", id).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to delete subtask")
	}
	return nil
}
