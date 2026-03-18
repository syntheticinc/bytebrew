package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// SubtaskStorage implements subtask persistence using GORM (PostgreSQL).
type SubtaskStorage struct {
	db *gorm.DB
}

// NewSubtaskStorage creates a new subtask storage.
func NewSubtaskStorage(db *gorm.DB) *SubtaskStorage {
	slog.Info("subtask storage initialized (PostgreSQL)")
	return &SubtaskStorage{db: db}
}

// Save persists a new subtask.
func (s *SubtaskStorage) Save(ctx context.Context, subtask *domain.Subtask) error {
	m, err := subtaskToModel(subtask)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Create(&m).Error; err != nil {
		if strings.Contains(err.Error(), "violates foreign key") || strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
			return fmt.Errorf("task %q does not exist. Create a task first using manage_tasks(action=create), then create subtasks for it", subtask.TaskID)
		}
		return fmt.Errorf("insert subtask: %w", err)
	}
	slog.DebugContext(ctx, "subtask saved", "subtask_id", subtask.ID, "task_id", subtask.TaskID)
	return nil
}

// Update updates an existing subtask.
func (s *SubtaskStorage) Update(ctx context.Context, subtask *domain.Subtask) error {
	blockedByJSON, err := json.Marshal(subtask.BlockedBy)
	if err != nil {
		return fmt.Errorf("marshal blocked_by: %w", err)
	}
	filesJSON, err := json.Marshal(subtask.FilesInvolved)
	if err != nil {
		return fmt.Errorf("marshal files_involved: %w", err)
	}
	contextJSON, err := json.Marshal(subtask.Context)
	if err != nil {
		return fmt.Errorf("marshal context: %w", err)
	}

	result := s.db.WithContext(ctx).Model(&models.RuntimeSubtaskModel{}).
		Where("id = ?", subtask.ID).
		Updates(map[string]interface{}{
			"title":             subtask.Title,
			"description":       subtask.Description,
			"status":            string(subtask.Status),
			"assigned_agent_id": subtask.AssignedAgentID,
			"blocked_by":        string(blockedByJSON),
			"files_involved":    string(filesJSON),
			"result":            subtask.Result,
			"context":           string(contextJSON),
			"updated_at":        subtask.UpdatedAt,
			"completed_at":      subtask.CompletedAt,
		})
	if result.Error != nil {
		return fmt.Errorf("update subtask: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("subtask not found: %s", subtask.ID)
	}
	slog.DebugContext(ctx, "subtask updated", "subtask_id", subtask.ID, "status", subtask.Status)
	return nil
}

// GetByID retrieves a subtask by ID.
func (s *SubtaskStorage) GetByID(ctx context.Context, id string) (*domain.Subtask, error) {
	var m models.RuntimeSubtaskModel
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get subtask: %w", err)
	}
	return modelToSubtask(&m)
}

// GetByTaskID retrieves all subtasks for a task.
func (s *SubtaskStorage) GetByTaskID(ctx context.Context, taskID string) ([]*domain.Subtask, error) {
	var ms []models.RuntimeSubtaskModel
	err := s.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("created_at ASC").
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("query subtasks by task: %w", err)
	}
	return modelsToSubtasks(ms)
}

// GetBySessionID retrieves all subtasks for a session.
func (s *SubtaskStorage) GetBySessionID(ctx context.Context, sessionID string) ([]*domain.Subtask, error) {
	var ms []models.RuntimeSubtaskModel
	err := s.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("query subtasks by session: %w", err)
	}
	return modelsToSubtasks(ms)
}

// GetReadySubtasks returns subtasks that are pending and have no unfinished blockers.
func (s *SubtaskStorage) GetReadySubtasks(ctx context.Context, taskID string) ([]*domain.Subtask, error) {
	// Get all pending subtasks for the task
	var ms []models.RuntimeSubtaskModel
	err := s.db.WithContext(ctx).
		Where("task_id = ? AND status = ?", taskID, "pending").
		Order("created_at ASC").
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("query ready subtasks: %w", err)
	}

	// Filter out subtasks with unfinished blockers
	var ready []*domain.Subtask
	for i := range ms {
		subtask, err := modelToSubtask(&ms[i])
		if err != nil {
			return nil, err
		}

		if len(subtask.BlockedBy) == 0 {
			ready = append(ready, subtask)
			continue
		}

		// Check if all blockers are completed
		allCompleted := true
		for _, blockerID := range subtask.BlockedBy {
			var blocker models.RuntimeSubtaskModel
			bErr := s.db.WithContext(ctx).Where("id = ?", blockerID).First(&blocker).Error
			if bErr != nil {
				allCompleted = false
				break
			}
			if blocker.Status != "completed" {
				allCompleted = false
				break
			}
		}
		if allCompleted {
			ready = append(ready, subtask)
		}
	}

	return ready, nil
}

// GetByAgentID retrieves the subtask assigned to a specific agent.
func (s *SubtaskStorage) GetByAgentID(ctx context.Context, agentID string) (*domain.Subtask, error) {
	var m models.RuntimeSubtaskModel
	err := s.db.WithContext(ctx).
		Where("assigned_agent_id = ? AND status = ?", agentID, "in_progress").
		First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get subtask by agent: %w", err)
	}
	return modelToSubtask(&m)
}

// Close is a no-op because the shared DB is owned by the caller.
func (s *SubtaskStorage) Close() error {
	return nil
}

func subtaskToModel(subtask *domain.Subtask) (models.RuntimeSubtaskModel, error) {
	blockedByJSON, err := json.Marshal(subtask.BlockedBy)
	if err != nil {
		return models.RuntimeSubtaskModel{}, fmt.Errorf("marshal blocked_by: %w", err)
	}
	filesJSON, err := json.Marshal(subtask.FilesInvolved)
	if err != nil {
		return models.RuntimeSubtaskModel{}, fmt.Errorf("marshal files_involved: %w", err)
	}
	contextJSON, err := json.Marshal(subtask.Context)
	if err != nil {
		return models.RuntimeSubtaskModel{}, fmt.Errorf("marshal context: %w", err)
	}
	return models.RuntimeSubtaskModel{
		ID:              subtask.ID,
		SessionID:       subtask.SessionID,
		TaskID:          subtask.TaskID,
		Title:           subtask.Title,
		Description:     subtask.Description,
		Status:          string(subtask.Status),
		AssignedAgentID: subtask.AssignedAgentID,
		BlockedBy:       string(blockedByJSON),
		FilesInvolved:   string(filesJSON),
		Result:          subtask.Result,
		Context:         string(contextJSON),
		CreatedAt:       subtask.CreatedAt,
		UpdatedAt:       subtask.UpdatedAt,
		CompletedAt:     subtask.CompletedAt,
	}, nil
}

func modelToSubtask(m *models.RuntimeSubtaskModel) (*domain.Subtask, error) {
	subtask := &domain.Subtask{
		ID:              m.ID,
		SessionID:       m.SessionID,
		TaskID:          m.TaskID,
		Title:           m.Title,
		Description:     m.Description,
		Status:          domain.SubtaskStatus(m.Status),
		AssignedAgentID: m.AssignedAgentID,
		Result:          m.Result,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
		CompletedAt:     m.CompletedAt,
	}

	if m.BlockedBy != "" && m.BlockedBy != "null" {
		if err := json.Unmarshal([]byte(m.BlockedBy), &subtask.BlockedBy); err != nil {
			return nil, fmt.Errorf("unmarshal blocked_by: %w", err)
		}
	}
	if m.FilesInvolved != "" && m.FilesInvolved != "null" {
		if err := json.Unmarshal([]byte(m.FilesInvolved), &subtask.FilesInvolved); err != nil {
			return nil, fmt.Errorf("unmarshal files_involved: %w", err)
		}
	}
	if m.Context != "" && m.Context != "null" {
		subtask.Context = make(map[string]string)
		if err := json.Unmarshal([]byte(m.Context), &subtask.Context); err != nil {
			return nil, fmt.Errorf("unmarshal context: %w", err)
		}
	}

	return subtask, nil
}

func modelsToSubtasks(ms []models.RuntimeSubtaskModel) ([]*domain.Subtask, error) {
	subtasks := make([]*domain.Subtask, 0, len(ms))
	for i := range ms {
		st, err := modelToSubtask(&ms[i])
		if err != nil {
			return nil, err
		}
		subtasks = append(subtasks, st)
	}
	return subtasks, nil
}
