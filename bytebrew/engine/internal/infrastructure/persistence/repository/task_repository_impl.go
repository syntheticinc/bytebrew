package repository

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/adapters"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/errors"
	"github.com/google/uuid"
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
	model := adapters.SubtaskToTaskModel(subtask)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to create task")
	}
	return nil
}

// GetByID retrieves a subtask by ID
func (r *TaskRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.Subtask, error) {
	taskID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidInput, "invalid task ID")
	}

	var model struct {
		ID          uuid.UUID `gorm:"type:uuid"`
		Code        string
		Title       string
		Description string
		TaskType    string
		SessionID   uuid.UUID `gorm:"type:uuid"`
		Status      string
		CreatedAt   string
		UpdatedAt   string
	}

	if err := r.db.WithContext(ctx).Table("task").Where("id = ?", taskID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "task not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get task")
	}

	subtask := &domain.Subtask{
		ID:          model.ID.String(),
		SessionID:   model.SessionID.String(),
		Description: model.Title,
		Status:      domain.SubtaskStatus(model.Status),
	}

	return subtask, nil
}

// Update updates a subtask
func (r *TaskRepositoryImpl) Update(ctx context.Context, subtask *domain.Subtask) error {
	model := adapters.SubtaskToTaskModel(subtask)
	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to update task")
	}
	return nil
}
