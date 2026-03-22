package persistence

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// SessionStorage implements session persistence using GORM (PostgreSQL).
type SessionStorage struct {
	db *gorm.DB
}

// NewSessionStorage creates a new session storage.
func NewSessionStorage(db *gorm.DB) *SessionStorage {
	slog.Info("session storage initialized (PostgreSQL)")
	return &SessionStorage{db: db}
}

// Save persists a new session.
func (s *SessionStorage) Save(ctx context.Context, session *domain.Session) error {
	m := sessionToModel(session)
	if err := s.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	slog.DebugContext(ctx, "session saved", "session_id", session.ID, "project_key", session.ProjectKey)
	return nil
}

// Update updates an existing session.
func (s *SessionStorage) Update(ctx context.Context, session *domain.Session) error {
	result := s.db.WithContext(ctx).Model(&models.RuntimeSessionModel{}).
		Where("id = ?", session.ID).
		Updates(map[string]interface{}{
			"project_key":      session.ProjectKey,
			"status":           string(session.Status),
			"updated_at":       session.UpdatedAt,
			"last_activity_at": session.LastActivityAt,
		})
	if result.Error != nil {
		return fmt.Errorf("update session: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("session not found: %s", session.ID)
	}
	slog.DebugContext(ctx, "session updated", "session_id", session.ID, "status", session.Status)
	return nil
}

// GetByID retrieves a session by ID (returns nil if not found).
func (s *SessionStorage) GetByID(ctx context.Context, id string) (*domain.Session, error) {
	var m models.RuntimeSessionModel
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	return modelToSession(&m), nil
}

// GetLatestByProjectKey retrieves the most recent session for a project.
func (s *SessionStorage) GetLatestByProjectKey(ctx context.Context, projectKey string) (*domain.Session, error) {
	var m models.RuntimeSessionModel
	err := s.db.WithContext(ctx).
		Where("project_key = ?", projectKey).
		Order("last_activity_at DESC").
		First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest session by project: %w", err)
	}
	return modelToSession(&m), nil
}

// ListByProjectKey retrieves all sessions for a project, ordered by recent activity.
func (s *SessionStorage) ListByProjectKey(ctx context.Context, projectKey string) ([]*domain.Session, error) {
	var ms []models.RuntimeSessionModel
	err := s.db.WithContext(ctx).
		Where("project_key = ?", projectKey).
		Order("last_activity_at DESC").
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("query sessions by project: %w", err)
	}
	return modelsToSessions(ms), nil
}

// SuspendActiveSessions marks all 'active' sessions as 'suspended'.
// Called at server startup to handle crash recovery.
func (s *SessionStorage) SuspendActiveSessions(ctx context.Context) (int64, error) {
	result := s.db.WithContext(ctx).
		Model(&models.RuntimeSessionModel{}).
		Where("status = ?", "active").
		Updates(map[string]interface{}{
			"status":     "suspended",
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return 0, fmt.Errorf("suspend active sessions: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// Close is a no-op because the shared DB is owned by the caller.
func (s *SessionStorage) Close() error {
	return nil
}

func sessionToModel(session *domain.Session) models.RuntimeSessionModel {
	return models.RuntimeSessionModel{
		ID:             session.ID,
		ProjectKey:     session.ProjectKey,
		Status:         string(session.Status),
		CreatedAt:      session.CreatedAt,
		UpdatedAt:      session.UpdatedAt,
		LastActivityAt: session.LastActivityAt,
	}
}

func modelToSession(m *models.RuntimeSessionModel) *domain.Session {
	return &domain.Session{
		ID:             m.ID,
		ProjectKey:     m.ProjectKey,
		Status:         domain.SessionStatus(m.Status),
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
		LastActivityAt: m.LastActivityAt,
	}
}

func modelsToSessions(ms []models.RuntimeSessionModel) []*domain.Session {
	sessions := make([]*domain.Session, 0, len(ms))
	for i := range ms {
		sessions = append(sessions, modelToSession(&ms[i]))
	}
	return sessions
}
