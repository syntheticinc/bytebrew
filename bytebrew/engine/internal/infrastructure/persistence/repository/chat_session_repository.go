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

type chatSessionRepository struct {
	db *gorm.DB
}

// NewChatSessionRepository creates a new chat session repository
func NewChatSessionRepository(db *gorm.DB) *chatSessionRepository {
	return &chatSessionRepository{db: db}
}

func (r *chatSessionRepository) Create(ctx context.Context, session *domain.ChatSession) error {
	model := adapters.ChatSessionToModel(session)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to create chat session")
	}

	// Update domain entity with generated ID
	session.ID = model.ID.String()
	return nil
}

func (r *chatSessionRepository) GetByID(ctx context.Context, id string) (*domain.ChatSession, error) {
	sessionID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidInput, "invalid session id")
	}

	var model models.ChatSession
	if err := r.db.WithContext(ctx).First(&model, "id = ?", sessionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "chat session not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get chat session by id")
	}

	return adapters.ChatSessionFromModel(&model)
}

func (r *chatSessionRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.ChatSession, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidInput, "invalid user id")
	}

	var models []*models.ChatSession
	if err := r.db.WithContext(ctx).Where("user_id = ?", uid).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get chat sessions by user id")
	}

	sessions := make([]*domain.ChatSession, 0, len(models))
	for _, model := range models {
		session, err := adapters.ChatSessionFromModel(model)
		if err != nil {
			return nil, errors.Wrap(err, errors.CodeInternal, "failed to convert chat session")
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

func (r *chatSessionRepository) Update(ctx context.Context, session *domain.ChatSession) error {
	model := adapters.ChatSessionToModel(session)

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to update chat session")
	}
	return nil
}

func (r *chatSessionRepository) Delete(ctx context.Context, id string) error {
	sessionID, err := uuid.Parse(id)
	if err != nil {
		return errors.Wrap(err, errors.CodeInvalidInput, "invalid session id")
	}

	if err := r.db.WithContext(ctx).Delete(&models.ChatSession{}, "id = ?", sessionID).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to delete chat session")
	}
	return nil
}
