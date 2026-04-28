package memoryclear

import (
	"context"
	"fmt"
	"log/slog"
)

// MemoryRepository deletes memory entries.
type MemoryRepository interface {
	DeleteBySchema(ctx context.Context, schemaID string) (int64, error)
	DeleteByID(ctx context.Context, id string) error
}

// Usecase clears memories for a schema (AC-MEM-03).
type Usecase struct {
	repo MemoryRepository
}

// New creates a new memory_clear usecase.
func New(repo MemoryRepository) *Usecase {
	return &Usecase{repo: repo}
}

// ClearAll deletes all memories for a schema.
func (u *Usecase) ClearAll(ctx context.Context, schemaID string) (int64, error) {
	if schemaID == "" {
		return 0, fmt.Errorf("schema_id is required")
	}

	deleted, err := u.repo.DeleteBySchema(ctx, schemaID)
	if err != nil {
		return 0, fmt.Errorf("clear memories: %w", err)
	}

	slog.InfoContext(ctx, "memories cleared", "schema_id", schemaID, "deleted", deleted)
	return deleted, nil
}

// DeleteOne deletes a single memory entry by ID.
func (u *Usecase) DeleteOne(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("memory id is required")
	}

	if err := u.repo.DeleteByID(ctx, id); err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}

	return nil
}
