package http

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ErrRefNotFound is returned by Resolve* helpers when the referenced entity
// does not exist within the caller's tenant. Handlers map this to 404.
var ErrRefNotFound = errors.New("ref not found")

// AgentRefRepo is the consumer-side interface used by ResolveAgentRef.
// Only the two lookup methods needed by the resolver are required.
type AgentRefRepo interface {
	GetAgentByID(ctx context.Context, id string) (string, error)   // returns canonical name (unused here) or error
	GetAgentIDByName(ctx context.Context, name string) (string, error) // returns UUID or ErrRefNotFound
}

// ModelRefRepo is the consumer-side interface used by ResolveModelRef.
type ModelRefRepo interface {
	GetModelByID(ctx context.Context, id string) (string, error)
	GetModelIDByName(ctx context.Context, name string) (string, error)
}

// ErrModelKindMismatch is returned by ResolveModelRefWithKind when the resolved
// model exists but its kind does not match the expected kind.
var ErrModelKindMismatch = errors.New("model kind mismatch")

// ModelKindRepo extends ModelRefRepo with a kind lookup.
// Consumer-side interface defined here per project conventions.
type ModelKindRepo interface {
	ModelRefRepo
	GetModelKindByID(ctx context.Context, id string) (string, error)
}

// ResolveModelRefWithKind resolves a model reference exactly like ResolveModelRef
// and additionally verifies that the resolved model has the expected kind.
// Returns ErrRefNotFound when no model matches, ErrModelKindMismatch when the
// model exists but kind != wantKind.
func ResolveModelRefWithKind(ctx context.Context, repo ModelKindRepo, ref, wantKind string) (string, error) {
	id, err := ResolveModelRef(ctx, repo, ref)
	if err != nil {
		return "", err
	}
	kind, err := repo.GetModelKindByID(ctx, id)
	if err != nil {
		return "", ErrRefNotFound
	}
	if kind != wantKind {
		return "", fmt.Errorf("%w: want %s, got %s", ErrModelKindMismatch, wantKind, kind)
	}
	return id, nil
}

// SchemaRefRepo is the consumer-side interface used by ResolveSchemaRef.
type SchemaRefRepo interface {
	GetSchemaByID(ctx context.Context, id string) (string, error)
	GetSchemaIDByName(ctx context.Context, name string) (string, error)
}

// KBRefRepo is the consumer-side interface used by ResolveKBRef.
type KBRefRepo interface {
	GetKBByID(ctx context.Context, id string) (string, error)
	GetKBIDByName(ctx context.Context, name string) (string, error)
}

// MCPRefRepo is the consumer-side interface used by ResolveMCPRef.
type MCPRefRepo interface {
	GetMCPByID(ctx context.Context, id string) (string, error)
	GetMCPIDByName(ctx context.Context, name string) (string, error)
}

// ResolveAgentRef returns the canonical UUID for an agent reference.
// ref may be a UUID string or an agent name; both are looked up within
// the caller's tenant (ctx-scoped).
// Returns ErrRefNotFound when no matching entity exists.
// NEVER returns ref verbatim without a DB round-trip — tenant isolation guaranteed.
func ResolveAgentRef(ctx context.Context, repo AgentRefRepo, ref string) (string, error) {
	if _, err := uuid.Parse(ref); err == nil {
		// Looks like a UUID — verify it exists and belongs to this tenant.
		id, err := repo.GetAgentByID(ctx, ref)
		if err != nil {
			return "", ErrRefNotFound
		}
		return id, nil
	}
	// Treat as name.
	id, err := repo.GetAgentIDByName(ctx, ref)
	if err != nil {
		return "", ErrRefNotFound
	}
	return id, nil
}

// ResolveModelRef returns the canonical UUID for a model reference.
func ResolveModelRef(ctx context.Context, repo ModelRefRepo, ref string) (string, error) {
	if _, err := uuid.Parse(ref); err == nil {
		id, err := repo.GetModelByID(ctx, ref)
		if err != nil {
			return "", ErrRefNotFound
		}
		return id, nil
	}
	id, err := repo.GetModelIDByName(ctx, ref)
	if err != nil {
		return "", ErrRefNotFound
	}
	return id, nil
}

// ResolveSchemaRef returns the canonical UUID for a schema reference.
func ResolveSchemaRef(ctx context.Context, repo SchemaRefRepo, ref string) (string, error) {
	if _, err := uuid.Parse(ref); err == nil {
		id, err := repo.GetSchemaByID(ctx, ref)
		if err != nil {
			return "", ErrRefNotFound
		}
		return id, nil
	}
	id, err := repo.GetSchemaIDByName(ctx, ref)
	if err != nil {
		return "", ErrRefNotFound
	}
	return id, nil
}

// ResolveKBRef returns the canonical UUID for a knowledge base reference.
func ResolveKBRef(ctx context.Context, repo KBRefRepo, ref string) (string, error) {
	if _, err := uuid.Parse(ref); err == nil {
		id, err := repo.GetKBByID(ctx, ref)
		if err != nil {
			return "", ErrRefNotFound
		}
		return id, nil
	}
	id, err := repo.GetKBIDByName(ctx, ref)
	if err != nil {
		return "", ErrRefNotFound
	}
	return id, nil
}

// ResolveMCPRef returns the canonical UUID for an MCP server reference.
func ResolveMCPRef(ctx context.Context, repo MCPRefRepo, ref string) (string, error) {
	if _, err := uuid.Parse(ref); err == nil {
		id, err := repo.GetMCPByID(ctx, ref)
		if err != nil {
			return "", ErrRefNotFound
		}
		return id, nil
	}
	id, err := repo.GetMCPIDByName(ctx, ref)
	if err != nil {
		return "", ErrRefNotFound
	}
	return id, nil
}
