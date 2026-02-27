package adapters

import (
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
)

// UserToModel converts domain User to persistence model
func UserToModel(user *domain.User) *models.User {
	if user == nil {
		return nil
	}

	var id uuid.UUID
	if user.ID != "" {
		id, _ = uuid.Parse(user.ID)
	}

	return &models.User{
		ID:           id,
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
}

// UserFromModel converts persistence model to domain User
func UserFromModel(model *models.User) (*domain.User, error) {
	if model == nil {
		return nil, nil
	}

	user := &domain.User{
		ID:           model.ID.String(),
		Username:     model.Username,
		Email:        model.Email,
		PasswordHash: model.PasswordHash,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}

	return user, nil
}
