package configrepo

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// GORMEventRepository implements event persistence using GORM.
type GORMEventRepository struct {
	db *gorm.DB
}

// NewGORMEventRepository creates a new GORMEventRepository.
func NewGORMEventRepository(db *gorm.DB) *GORMEventRepository {
	return &GORMEventRepository{db: db}
}

// ListBySession returns events for a session, sorted by created_at ASC.
func (r *GORMEventRepository) ListBySession(ctx context.Context, sessionID string) ([]models.MessageModel, error) {
	var events []models.MessageModel
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&events).Error; err != nil {
		return nil, fmt.Errorf("list events by session: %w", err)
	}
	return events, nil
}

// DeleteBySession deletes all events for a session.
func (r *GORMEventRepository) DeleteBySession(ctx context.Context, sessionID string) error {
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Delete(&models.MessageModel{}).Error; err != nil {
		return fmt.Errorf("delete events by session: %w", err)
	}
	return nil
}
