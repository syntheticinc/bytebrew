package repository

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/adapters"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/errors"
	"github.com/google/uuid"
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
	model := adapters.SubtaskToTaskModel(subtask)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to create task")
	}

	// Update domain entity with generated ID
	subtask.ID = model.ID.String()
	return nil
}

func (r *taskRepository) GetByID(ctx context.Context, id string) (*domain.Subtask, error) {
	taskID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidInput, "invalid task id")
	}

	var model models.Task
	if err := r.db.WithContext(ctx).First(&model, "id = ?", taskID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "task not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get task by id")
	}

	return adapters.TaskModelToSubtask(&model)
}

func (r *taskRepository) GetByCode(ctx context.Context, code string) (*models.Task, error) {
	var task models.Task
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "task not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get task by code")
	}
	return &task, nil
}

func (r *taskRepository) GetBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*models.Task, error) {
	var tasks []*models.Task
	if err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).Find(&tasks).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get tasks by session id")
	}
	return tasks, nil
}

func (r *taskRepository) GetSubTasks(ctx context.Context, parentTaskID uuid.UUID) ([]*models.Task, error) {
	var tasks []*models.Task
	if err := r.db.WithContext(ctx).Where("parent_task_id = ?", parentTaskID).Find(&tasks).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get subtasks")
	}
	return tasks, nil
}

func (r *taskRepository) Update(ctx context.Context, subtask *domain.Subtask) error {
	model := adapters.SubtaskToTaskModel(subtask)

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to update task")
	}
	return nil
}

func (r *taskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.Task{}, "id = ?", id).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to delete task")
	}
	return nil
}
