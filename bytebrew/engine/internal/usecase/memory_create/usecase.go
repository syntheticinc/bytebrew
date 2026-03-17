package memory_create

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/errors"
)

// MemoryRepository defines repository interface for Memory
type MemoryRepository interface {
	Create(ctx context.Context, memory *domain.Memory) error
	GetByID(ctx context.Context, id string) (*domain.Memory, error)
	GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Memory, error)
	Search(ctx context.Context, userID, query string, limit int) ([]*domain.Memory, error)
}

// Input represents input for create memory use case
type Input struct {
	UserID  string
	Content string
	Level   domain.MemoryLevel
}

// Output represents output from create memory use case
type Output struct {
	Memory *domain.Memory
}

// Usecase handles memory creation
type Usecase struct {
	memoryRepo MemoryRepository
}

// New creates a new create memory use case
func New(memoryRepo MemoryRepository) *Usecase {
	return &Usecase{
		memoryRepo: memoryRepo,
	}
}

// Execute creates a new memory
func (u *Usecase) Execute(ctx context.Context, input Input) (*Output, error) {
	memory, err := domain.NewMemory(input.UserID, input.Content, input.Level)
	if err != nil {
		return nil, err
	}

	if err := u.memoryRepo.Create(ctx, memory); err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to create memory")
	}

	return &Output{Memory: memory}, nil
}
