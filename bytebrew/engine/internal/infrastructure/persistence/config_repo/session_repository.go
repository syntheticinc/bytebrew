package config_repo

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// GORMSessionRepository implements session CRUD using GORM.
type GORMSessionRepository struct {
	db *gorm.DB
}

// NewGORMSessionRepository creates a new GORMSessionRepository.
func NewGORMSessionRepository(db *gorm.DB) *GORMSessionRepository {
	return &GORMSessionRepository{db: db}
}

// List returns paginated sessions sorted by updated_at desc with optional filters.
func (r *GORMSessionRepository) List(ctx context.Context, agentName, userID, status string, page, perPage int) ([]models.SessionModel, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.SessionModel{})

	if agentName != "" {
		q = q.Where("agent_name = ?", agentName)
	}
	if userID != "" {
		q = q.Where("user_id = ?", userID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count sessions: %w", err)
	}

	var sessions []models.SessionModel
	offset := (page - 1) * perPage
	if err := q.Order("updated_at DESC").Offset(offset).Limit(perPage).Find(&sessions).Error; err != nil {
		return nil, 0, fmt.Errorf("list sessions: %w", err)
	}

	return sessions, total, nil
}

// Get returns a session by ID.
func (r *GORMSessionRepository) Get(ctx context.Context, id string) (*models.SessionModel, error) {
	var session models.SessionModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	return &session, nil
}

// Create inserts a new session record.
func (r *GORMSessionRepository) Create(ctx context.Context, session *models.SessionModel) error {
	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// Update updates session fields by ID.
func (r *GORMSessionRepository) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	result := r.db.WithContext(ctx).Model(&models.SessionModel{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update session: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("session not found: %s", id)
	}
	return nil
}

// Delete removes a session by ID.
func (r *GORMSessionRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.SessionModel{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete session: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("session not found: %s", id)
	}
	return nil
}

// TouchUpdatedAt updates the updated_at timestamp for a session.
func (r *GORMSessionRepository) TouchUpdatedAt(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Model(&models.SessionModel{}).Where("id = ?", id).Update("updated_at", gorm.Expr("NOW()"))
	if result.Error != nil {
		return fmt.Errorf("touch session updated_at: %w", result.Error)
	}
	return nil
}
