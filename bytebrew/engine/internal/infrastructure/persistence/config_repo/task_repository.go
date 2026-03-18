package config_repo

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// TaskFilter holds optional criteria for listing tasks.
type TaskFilter struct {
	Source       *domain.TaskSource
	AgentName    *string
	Status       *domain.EngineTaskStatus
	UserID       *string
	SessionID    *string
	ParentTaskID *uint
	Limit        int
	Offset       int
}

// GORMTaskRepository implements task persistence using GORM.
type GORMTaskRepository struct {
	db *gorm.DB
}

// NewGORMTaskRepository creates a new GORMTaskRepository.
func NewGORMTaskRepository(db *gorm.DB) *GORMTaskRepository {
	return &GORMTaskRepository{db: db}
}

// Create inserts a new task and populates the ID back into the domain entity.
func (r *GORMTaskRepository) Create(ctx context.Context, task *domain.EngineTask) error {
	m := toTaskModel(task)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	task.ID = m.ID
	return nil
}

// GetByID returns a single task by its primary key.
func (r *GORMTaskRepository) GetByID(ctx context.Context, id uint) (*domain.EngineTask, error) {
	var m models.TaskModel
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrEngineTaskNotFound
		}
		return nil, fmt.Errorf("get task %d: %w", id, err)
	}
	return toEngineTask(&m), nil
}

// List returns tasks matching the provided filter.
func (r *GORMTaskRepository) List(ctx context.Context, filter TaskFilter) ([]domain.EngineTask, error) {
	q := r.db.WithContext(ctx).Model(&models.TaskModel{})
	q = applyTaskFilter(q, filter)
	q = q.Order("created_at DESC")

	if filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		q = q.Offset(filter.Offset)
	}

	var rows []models.TaskModel
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	tasks := make([]domain.EngineTask, 0, len(rows))
	for i := range rows {
		tasks = append(tasks, *toEngineTask(&rows[i]))
	}
	return tasks, nil
}

// UpdateStatus transitions a task to a new status and optionally sets a result string.
func (r *GORMTaskRepository) UpdateStatus(ctx context.Context, id uint, status domain.EngineTaskStatus, result string) error {
	var m models.TaskModel
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.ErrEngineTaskNotFound
		}
		return fmt.Errorf("get task %d for status update: %w", id, err)
	}

	task := toEngineTask(&m)
	if err := task.Transition(status); err != nil {
		return err
	}
	task.Result = result

	updated := toTaskModel(task)
	if err := r.db.WithContext(ctx).Save(&updated).Error; err != nil {
		return fmt.Errorf("update task %d status: %w", id, err)
	}
	return nil
}

// Update saves all fields of the task.
func (r *GORMTaskRepository) Update(ctx context.Context, task *domain.EngineTask) error {
	m := toTaskModel(task)
	if err := r.db.WithContext(ctx).Save(&m).Error; err != nil {
		return fmt.Errorf("update task %d: %w", task.ID, err)
	}
	return nil
}

// GetSubTasks returns all direct children of the given parent task.
func (r *GORMTaskRepository) GetSubTasks(ctx context.Context, parentID uint) ([]domain.EngineTask, error) {
	var rows []models.TaskModel
	if err := r.db.WithContext(ctx).Where("parent_task_id = ?", parentID).Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("get subtasks for %d: %w", parentID, err)
	}

	tasks := make([]domain.EngineTask, 0, len(rows))
	for i := range rows {
		tasks = append(tasks, *toEngineTask(&rows[i]))
	}
	return tasks, nil
}

// GetPendingBySession returns all pending tasks for the given session.
func (r *GORMTaskRepository) GetPendingBySession(ctx context.Context, sessionID string) ([]domain.EngineTask, error) {
	var rows []models.TaskModel
	if err := r.db.WithContext(ctx).
		Where("session_id = ? AND status = ?", sessionID, string(domain.EngineTaskStatusPending)).
		Order("created_at ASC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("get pending tasks for session %s: %w", sessionID, err)
	}

	tasks := make([]domain.EngineTask, 0, len(rows))
	for i := range rows {
		tasks = append(tasks, *toEngineTask(&rows[i]))
	}
	return tasks, nil
}

// Cancel cancels a task and all its non-terminal subtasks (cascading).
func (r *GORMTaskRepository) Cancel(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return cancelRecursive(tx, id)
	})
}

// cancelRecursive cancels a task and recursively cancels all non-terminal subtasks.
func cancelRecursive(tx *gorm.DB, id uint) error {
	var m models.TaskModel
	if err := tx.First(&m, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.ErrEngineTaskNotFound
		}
		return fmt.Errorf("get task %d for cancel: %w", id, err)
	}

	task := toEngineTask(&m)
	if task.IsTerminal() {
		return nil
	}

	if err := task.Transition(domain.EngineTaskStatusCancelled); err != nil {
		return fmt.Errorf("transition task %d to cancelled: %w", id, err)
	}

	if err := tx.Exec("UPDATE tasks SET status = $1, completed_at = NOW() WHERE id = $2",
		string(domain.EngineTaskStatusCancelled), id).Error; err != nil {
		return fmt.Errorf("save cancelled task %d: %w", id, err)
	}

	// Cancel subtasks
	var subtasks []models.TaskModel
	if err := tx.Where("parent_task_id = ?", id).Find(&subtasks).Error; err != nil {
		return fmt.Errorf("get subtasks for cancel %d: %w", id, err)
	}

	for _, sub := range subtasks {
		if err := cancelRecursive(tx, sub.ID); err != nil {
			return err
		}
	}

	return nil
}

// applyTaskFilter adds WHERE clauses based on non-nil filter fields.
func applyTaskFilter(q *gorm.DB, f TaskFilter) *gorm.DB {
	if f.Source != nil {
		q = q.Where("source = ?", string(*f.Source))
	}
	if f.AgentName != nil {
		q = q.Where("agent_name = ?", *f.AgentName)
	}
	if f.Status != nil {
		q = q.Where("status = ?", string(*f.Status))
	}
	if f.UserID != nil {
		q = q.Where("user_id = ?", *f.UserID)
	}
	if f.SessionID != nil {
		q = q.Where("session_id = ?", *f.SessionID)
	}
	if f.ParentTaskID != nil {
		q = q.Where("parent_task_id = ?", *f.ParentTaskID)
	}
	return q
}

// toTaskModel maps a domain EngineTask to a GORM TaskModel.
func toTaskModel(t *domain.EngineTask) models.TaskModel {
	return models.TaskModel{
		ID:           t.ID,
		Title:        t.Title,
		Description:  t.Description,
		AgentName:    t.AgentName,
		Source:       string(t.Source),
		SourceID:     t.SourceID,
		UserID:       t.UserID,
		SessionID:    strPtr(t.SessionID),
		ParentTaskID: t.ParentTaskID,
		Depth:        t.Depth,
		Status:       string(t.Status),
		Mode:         string(t.Mode),
		Result:       t.Result,
		Error:        t.Error,
		CreatedAt:    t.CreatedAt,
		StartedAt:    t.StartedAt,
		CompletedAt:  t.CompletedAt,
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// toEngineTask maps a GORM TaskModel to a domain EngineTask.
func toEngineTask(m *models.TaskModel) *domain.EngineTask {
	return &domain.EngineTask{
		ID:           m.ID,
		Title:        m.Title,
		Description:  m.Description,
		AgentName:    m.AgentName,
		Source:       domain.TaskSource(m.Source),
		SourceID:     m.SourceID,
		UserID:       m.UserID,
		SessionID:    derefStr(m.SessionID),
		ParentTaskID: m.ParentTaskID,
		Depth:        m.Depth,
		Status:       domain.EngineTaskStatus(m.Status),
		Mode:         domain.TaskMode(m.Mode),
		Result:       m.Result,
		Error:        m.Error,
		CreatedAt:    m.CreatedAt,
		StartedAt:    m.StartedAt,
		CompletedAt:  m.CompletedAt,
	}
}
