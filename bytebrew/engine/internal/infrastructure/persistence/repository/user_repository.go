package repository

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/adapters"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) *userRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	model := adapters.UserToModel(user)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to create user")
	}

	// Update domain entity with generated ID
	user.ID = model.ID.String()
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidInput, "invalid user id")
	}

	var model models.User
	if err := r.db.WithContext(ctx).First(&model, "id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "user not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get user by id")
	}

	return adapters.UserFromModel(&model)
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	var model models.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "user not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get user by username")
	}

	return adapters.UserFromModel(&model)
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var model models.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "user not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get user by email")
	}

	return adapters.UserFromModel(&model)
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	model := adapters.UserToModel(user)

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to update user")
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	userID, err := uuid.Parse(id)
	if err != nil {
		return errors.Wrap(err, errors.CodeInvalidInput, "invalid user id")
	}

	if err := r.db.WithContext(ctx).Delete(&models.User{}, "id = ?", userID).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to delete user")
	}
	return nil
}
