package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// TaskStorage implements task persistence using GORM (PostgreSQL).
type TaskStorage struct {
	db *gorm.DB
}

// NewTaskStorage creates a new task storage.
func NewTaskStorage(db *gorm.DB) *TaskStorage {
	slog.Info("task storage initialized (PostgreSQL)")
	return &TaskStorage{db: db}
}

// Save persists a new task.
func (s *TaskStorage) Save(ctx context.Context, task *domain.Task) error {
	m, err := taskToModel(task)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	slog.DebugContext(ctx, "task saved", "task_id", task.ID, "session_id", task.SessionID)
	return nil
}

// Update updates an existing task.
func (s *TaskStorage) Update(ctx context.Context, task *domain.Task) error {
	criteriaJSON, err := json.Marshal(task.AcceptanceCriteria)
	if err != nil {
		return fmt.Errorf("marshal acceptance_criteria: %w", err)
	}

	result := s.db.WithContext(ctx).Model(&models.RuntimeTaskModel{}).
		Where("id = ?", task.ID).
		Updates(map[string]interface{}{
			"title":               task.Title,
			"description":         task.Description,
			"acceptance_criteria": string(criteriaJSON),
			"status":              string(task.Status),
			"priority":            task.Priority,
			"updated_at":          task.UpdatedAt,
			"approved_at":         task.ApprovedAt,
			"completed_at":        task.CompletedAt,
		})
	if result.Error != nil {
		return fmt.Errorf("update task: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("task not found: %s", task.ID)
	}
	slog.DebugContext(ctx, "task updated", "task_id", task.ID, "status", task.Status)
	return nil
}

// GetByID retrieves a task by ID.
func (s *TaskStorage) GetByID(ctx context.Context, id string) (*domain.Task, error) {
	var m models.RuntimeTaskModel
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	return modelToTask(&m)
}

// GetBySessionID retrieves all tasks for a session.
func (s *TaskStorage) GetBySessionID(ctx context.Context, sessionID string) ([]*domain.Task, error) {
	var ms []models.RuntimeTaskModel
	err := s.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("query tasks by session: %w", err)
	}
	return modelsToTasks(ms)
}

// GetByStatus retrieves tasks with a specific status for a session.
func (s *TaskStorage) GetByStatus(ctx context.Context, sessionID string, status domain.TaskStatus) ([]*domain.Task, error) {
	var ms []models.RuntimeTaskModel
	err := s.db.WithContext(ctx).
		Where("session_id = ? AND status = ?", sessionID, string(status)).
		Order("created_at DESC").
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("query tasks by status: %w", err)
	}
	return modelsToTasks(ms)
}

// GetBySessionIDOrdered retrieves tasks for a session, ordered by priority (DESC) then created_at (ASC).
func (s *TaskStorage) GetBySessionIDOrdered(ctx context.Context, sessionID string) ([]*domain.Task, error) {
	var ms []models.RuntimeTaskModel
	err := s.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("priority DESC, created_at ASC").
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("query tasks by session ordered: %w", err)
	}
	return modelsToTasks(ms)
}

// Close is a no-op because the shared DB is owned by the caller.
func (s *TaskStorage) Close() error {
	return nil
}

func taskToModel(task *domain.Task) (models.RuntimeTaskModel, error) {
	criteriaJSON, err := json.Marshal(task.AcceptanceCriteria)
	if err != nil {
		return models.RuntimeTaskModel{}, fmt.Errorf("marshal acceptance_criteria: %w", err)
	}
	return models.RuntimeTaskModel{
		ID:                 task.ID,
		SessionID:          task.SessionID,
		Title:              task.Title,
		Description:        task.Description,
		AcceptanceCriteria: string(criteriaJSON),
		Status:             string(task.Status),
		Priority:           task.Priority,
		CreatedAt:          task.CreatedAt,
		UpdatedAt:          task.UpdatedAt,
		ApprovedAt:         task.ApprovedAt,
		CompletedAt:        task.CompletedAt,
	}, nil
}

func modelToTask(m *models.RuntimeTaskModel) (*domain.Task, error) {
	task := &domain.Task{
		ID:          m.ID,
		SessionID:   m.SessionID,
		Title:       m.Title,
		Description: m.Description,
		Status:      domain.TaskStatus(m.Status),
		Priority:    m.Priority,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		ApprovedAt:  m.ApprovedAt,
		CompletedAt: m.CompletedAt,
	}
	if m.AcceptanceCriteria != "" {
		if err := json.Unmarshal([]byte(m.AcceptanceCriteria), &task.AcceptanceCriteria); err != nil {
			return nil, fmt.Errorf("unmarshal acceptance_criteria: %w", err)
		}
	}
	return task, nil
}

func modelsToTasks(ms []models.RuntimeTaskModel) ([]*domain.Task, error) {
	tasks := make([]*domain.Task, 0, len(ms))
	for i := range ms {
		t, err := modelToTask(&ms[i])
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}
