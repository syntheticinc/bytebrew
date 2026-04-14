package configrepo

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// GORMMessageRepository implements event persistence using GORM.
// Named GORMMessageRepository for backward compatibility with wiring code.
type GORMMessageRepository struct {
	db *gorm.DB
}

// NewGORMMessageRepository creates a new GORMMessageRepository.
func NewGORMMessageRepository(db *gorm.DB) *GORMMessageRepository {
	return &GORMMessageRepository{db: db}
}

// ListBySession returns events for a session, sorted by created_at ASC.
func (r *GORMMessageRepository) ListBySession(ctx context.Context, sessionID string) ([]models.RuntimeEventModel, error) {
	var events []models.RuntimeEventModel
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&events).Error; err != nil {
		return nil, fmt.Errorf("list events by session: %w", err)
	}
	return events, nil
}

// DeleteBySession deletes all events for a session.
func (r *GORMMessageRepository) DeleteBySession(ctx context.Context, sessionID string) error {
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Delete(&models.RuntimeEventModel{}).Error; err != nil {
		return fmt.Errorf("delete events by session: %w", err)
	}
	return nil
}
