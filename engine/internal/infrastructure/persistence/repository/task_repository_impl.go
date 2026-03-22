package repository

import (
	"context"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/adapters"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/engine/pkg/errors"
	"gorm.io/gorm"
)

// TaskRepositoryImpl implements domain TaskRepository using GORM
type TaskRepositoryImpl struct {
	db *gorm.DB
}

// NewTaskRepositoryImpl creates a new TaskRepositoryImpl
func NewTaskRepositoryImpl(db *gorm.DB) *TaskRepositoryImpl {
	return &TaskRepositoryImpl{db: db}
}

// Create creates a new subtask
func (r *TaskRepositoryImpl) Create(ctx context.Context, subtask *domain.Subtask) error {
	model := adapters.SubtaskToRuntimeModel(subtask)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to create subtask")
	}
	return nil
}

// GetByID retrieves a subtask by ID
func (r *TaskRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.Subtask, error) {
	var model models.RuntimeSubtaskModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "subtask not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get subtask")
	}

	return adapters.RuntimeModelToSubtask(&model)
}

// Update updates a subtask
func (r *TaskRepositoryImpl) Update(ctx context.Context, subtask *domain.Subtask) error {
	model := adapters.SubtaskToRuntimeModel(subtask)
	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to update subtask")
	}
	return nil
}
