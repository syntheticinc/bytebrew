package repository

import (
	"context"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/adapters"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/engine/pkg/errors"
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

// Create creates a new event
func (r *MessageRepositoryImpl) Create(ctx context.Context, message *domain.Message) error {
	model, err := adapters.EventToModel(message)
	if err != nil {
		return errors.Wrap(err, errors.CodeInvalidInput, "convert event to model")
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to create event")
	}
	return nil
}

// GetBySessionID retrieves events by session ID in chronological order
func (r *MessageRepositoryImpl) GetBySessionID(ctx context.Context, sessionID string, limit, offset int) ([]*domain.Message, error) {
	var eventModels []models.RuntimeEventModel
	query := r.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&eventModels).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get events")
	}

	events := make([]*domain.Message, 0, len(eventModels))
	for i := range eventModels {
		ev, err := adapters.EventFromModel(&eventModels[i])
		if err != nil {
			return nil, errors.Wrap(err, errors.CodeInternal, "failed to convert event from model")
		}
		if ev != nil {
			events = append(events, ev)
		}
	}

	return events, nil
}

// GetBySessionAndAgent retrieves events by session ID and agent ID
func (r *MessageRepositoryImpl) GetBySessionAndAgent(ctx context.Context, sessionID, agentID string, limit, offset int) ([]*domain.Message, error) {
	var eventModels []models.RuntimeEventModel
	query := r.db.WithContext(ctx).Where("session_id = ? AND agent_id = ?", sessionID, agentID).Order("created_at ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&eventModels).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get events by session and agent")
	}

	events := make([]*domain.Message, 0, len(eventModels))
	for i := range eventModels {
		ev, err := adapters.EventFromModel(&eventModels[i])
		if err != nil {
			return nil, errors.Wrap(err, errors.CodeInternal, "failed to convert event from model")
		}
		if ev != nil {
			events = append(events, ev)
		}
	}

	return events, nil
}
