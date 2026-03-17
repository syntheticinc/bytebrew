package repository

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type memoryRepository struct {
	db *gorm.DB
}

// NewMemoryRepository creates a new MemoryRepository
func NewMemoryRepository(db *gorm.DB) *memoryRepository {
	return &memoryRepository{db: db}
}

func (r *memoryRepository) Create(ctx context.Context, memory *models.Memory) error {
	if err := r.db.WithContext(ctx).Create(memory).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to create memory")
	}
	return nil
}

func (r *memoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Memory, error) {
	var memory models.Memory
	if err := r.db.WithContext(ctx).First(&memory, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "memory not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get memory by id")
	}
	return &memory, nil
}

func (r *memoryRepository) GetByLevel(ctx context.Context, level string, limit, offset int) ([]*models.Memory, error) {
	var memories []*models.Memory
	query := r.db.WithContext(ctx).Where("level = ?", level).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&memories).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get memories by level")
	}
	return memories, nil
}

func (r *memoryRepository) Update(ctx context.Context, memory *models.Memory) error {
	if err := r.db.WithContext(ctx).Save(memory).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to update memory")
	}
	return nil
}

func (r *memoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.Memory{}, "id = ?", id).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to delete memory")
	}
	return nil
}
