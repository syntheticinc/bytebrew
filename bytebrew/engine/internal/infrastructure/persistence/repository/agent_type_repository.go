package repository

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type agentTypeRepository struct {
	db *gorm.DB
}

// NewAgentTypeRepository creates a new AgentTypeRepository
func NewAgentTypeRepository(db *gorm.DB) *agentTypeRepository {
	return &agentTypeRepository{db: db}
}

func (r *agentTypeRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.AgentType, error) {
	var agentType models.AgentType
	if err := r.db.WithContext(ctx).First(&agentType, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "agent type not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get agent type by id")
	}
	return &agentType, nil
}

func (r *agentTypeRepository) GetByCode(ctx context.Context, code string) (*models.AgentType, error) {
	var agentType models.AgentType
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&agentType).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "agent type not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get agent type by code")
	}
	return &agentType, nil
}

func (r *agentTypeRepository) GetAll(ctx context.Context) ([]*models.AgentType, error) {
	var agentTypes []*models.AgentType
	if err := r.db.WithContext(ctx).Find(&agentTypes).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get all agent types")
	}
	return agentTypes, nil
}
