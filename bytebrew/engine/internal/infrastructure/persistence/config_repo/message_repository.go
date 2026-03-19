package config_repo

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// GORMMessageRepository implements message persistence using GORM.
type GORMMessageRepository struct {
	db *gorm.DB
}

// NewGORMMessageRepository creates a new GORMMessageRepository.
func NewGORMMessageRepository(db *gorm.DB) *GORMMessageRepository {
	return &GORMMessageRepository{db: db}
}

// SaveMessage saves a single message to the database.
func (r *GORMMessageRepository) SaveMessage(ctx context.Context, msg *models.RuntimeMessageModel) error {
	if err := r.db.WithContext(ctx).Create(msg).Error; err != nil {
		return fmt.Errorf("save message: %w", err)
	}
	return nil
}

// ListBySession returns messages for a session, sorted by created_at ASC.
func (r *GORMMessageRepository) ListBySession(ctx context.Context, sessionID string) ([]models.RuntimeMessageModel, error) {
	var messages []models.RuntimeMessageModel
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("list messages by session: %w", err)
	}
	return messages, nil
}

// DeleteBySession deletes all messages for a session.
func (r *GORMMessageRepository) DeleteBySession(ctx context.Context, sessionID string) error {
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Delete(&models.RuntimeMessageModel{}).Error; err != nil {
		return fmt.Errorf("delete messages by session: %w", err)
	}
	return nil
}
