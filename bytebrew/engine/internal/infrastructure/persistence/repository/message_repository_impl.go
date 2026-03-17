package repository

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/persistence/adapters"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MessageRepositoryImpl implements domain MessageRepository using GORM
type MessageRepositoryImpl struct {
	db *gorm.DB
}

// NewMessageRepositoryImpl creates a new MessageRepositoryImpl
func NewMessageRepositoryImpl(db *gorm.DB) *MessageRepositoryImpl {
	return &MessageRepositoryImpl{db: db}
}

// Create creates a new message
func (r *MessageRepositoryImpl) Create(ctx context.Context, message *domain.Message) error {
	model, err := adapters.MessageToModel(message)
	if err != nil {
		return errors.Wrap(err, errors.CodeInvalidInput, "convert message to model")
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to create message")
	}
	return nil
}

// GetBySessionID retrieves messages by session ID
// CRITICAL: Uses adapters.MessageFromModel to properly deserialize ToolCalls and Metadata
func (r *MessageRepositoryImpl) GetBySessionID(ctx context.Context, sessionID string, limit, offset int) ([]*domain.Message, error) {
	sessID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidInput, "invalid session ID")
	}

	// Load most recent messages first (DESC), then reverse to chronological order.
	// This ensures we always get the LAST N messages, not the first N.
	var messageModels []models.Message
	query := r.db.WithContext(ctx).Where("session_id = ?", sessID).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&messageModels).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get messages")
	}

	// Reverse to chronological order (ASC) and convert to domain entities
	messages := make([]*domain.Message, 0, len(messageModels))
	for i := len(messageModels) - 1; i >= 0; i-- {
		msg, err := adapters.MessageFromModel(&messageModels[i])
		if err != nil {
			return nil, errors.Wrap(err, errors.CodeInternal, "failed to convert message from model")
		}
		if msg != nil {
			messages = append(messages, msg)
		}
	}

	return messages, nil
}

// GetBySessionAndAgent retrieves messages by session ID and agent ID
func (r *MessageRepositoryImpl) GetBySessionAndAgent(ctx context.Context, sessionID, agentID string, limit, offset int) ([]*domain.Message, error) {
	sessID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidInput, "invalid session ID")
	}

	// Load most recent messages first (DESC), then reverse to chronological order
	var messageModels []models.Message
	query := r.db.WithContext(ctx).Where("session_id = ? AND agent_id = ?", sessID, agentID).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&messageModels).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get messages by session and agent")
	}

	// Reverse to chronological order (ASC) and convert to domain entities
	messages := make([]*domain.Message, 0, len(messageModels))
	for i := len(messageModels) - 1; i >= 0; i-- {
		msg, err := adapters.MessageFromModel(&messageModels[i])
		if err != nil {
			return nil, errors.Wrap(err, errors.CodeInternal, "failed to convert message from model")
		}
		if msg != nil {
			messages = append(messages, msg)
		}
	}

	return messages, nil
}
