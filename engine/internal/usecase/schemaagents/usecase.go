package schemaagents

import (
	"context"
	"fmt"
	"log/slog"

	pkgerrors "github.com/syntheticinc/bytebrew/engine/pkg/errors"
)

// SchemaAgentRepository defines the repository interface for schema-agent refs.
type SchemaAgentRepository interface {
	AddAgent(ctx context.Context, schemaID uint, agentName string) error
	RemoveAgent(ctx context.Context, schemaID uint, agentName string) error
	ListAgents(ctx context.Context, schemaID uint) ([]string, error)
	ListSchemasForAgent(ctx context.Context, agentName string) ([]string, error)
}

// Usecase handles adding/removing agent references in schemas.
type Usecase struct {
	repo SchemaAgentRepository
}

// New creates a new schema agents use case.
func New(repo SchemaAgentRepository) *Usecase {
	return &Usecase{repo: repo}
}

// AddAgent adds an agent reference to a schema.
func (u *Usecase) AddAgent(ctx context.Context, schemaID uint, agentName string) error {
	if schemaID == 0 {
		return pkgerrors.InvalidInput("schema id is required")
	}
	if agentName == "" {
		return pkgerrors.InvalidInput("agent name is required")
	}

	if err := u.repo.AddAgent(ctx, schemaID, agentName); err != nil {
		slog.ErrorContext(ctx, "failed to add agent to schema", "error", err, "schema_id", schemaID, "agent", agentName)
		return fmt.Errorf("add agent to schema: %w", err)
	}
	return nil
}

// RemoveAgent removes an agent reference from a schema.
func (u *Usecase) RemoveAgent(ctx context.Context, schemaID uint, agentName string) error {
	if schemaID == 0 {
		return pkgerrors.InvalidInput("schema id is required")
	}
	if agentName == "" {
		return pkgerrors.InvalidInput("agent name is required")
	}

	if err := u.repo.RemoveAgent(ctx, schemaID, agentName); err != nil {
		slog.ErrorContext(ctx, "failed to remove agent from schema", "error", err, "schema_id", schemaID, "agent", agentName)
		return fmt.Errorf("remove agent from schema: %w", err)
	}
	return nil
}

// ListAgents returns agent names for a schema.
func (u *Usecase) ListAgents(ctx context.Context, schemaID uint) ([]string, error) {
	if schemaID == 0 {
		return nil, pkgerrors.InvalidInput("schema id is required")
	}

	names, err := u.repo.ListAgents(ctx, schemaID)
	if err != nil {
		return nil, fmt.Errorf("list schema agents: %w", err)
	}
	return names, nil
}

// ListSchemasForAgent returns schema names that reference a given agent.
func (u *Usecase) ListSchemasForAgent(ctx context.Context, agentName string) ([]string, error) {
	if agentName == "" {
		return nil, pkgerrors.InvalidInput("agent name is required")
	}

	names, err := u.repo.ListSchemasForAgent(ctx, agentName)
	if err != nil {
		return nil, fmt.Errorf("list schemas for agent: %w", err)
	}
	return names, nil
}
