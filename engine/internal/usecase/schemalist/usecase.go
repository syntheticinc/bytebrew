package schemalist

import (
	"context"
	"fmt"
)

// SchemaRepository defines the repository interface for schema listing.
type SchemaRepository interface {
	List(ctx context.Context) ([]SchemaRecord, error)
}

// SchemaRecord is a simplified record for the usecase boundary.
type SchemaRecord struct {
	ID          uint
	Name        string
	Description string
	AgentNames  []string
}

// Output represents a single schema in the list.
type Output struct {
	ID          uint
	Name        string
	Description string
	AgentNames  []string
}

// Usecase handles schema listing.
type Usecase struct {
	repo SchemaRepository
}

// New creates a new schema listing use case.
func New(repo SchemaRepository) *Usecase {
	return &Usecase{repo: repo}
}

// Execute returns all schemas.
func (u *Usecase) Execute(ctx context.Context) ([]Output, error) {
	records, err := u.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list schemas: %w", err)
	}

	result := make([]Output, 0, len(records))
	for _, r := range records {
		result = append(result, Output{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			AgentNames:  r.AgentNames,
		})
	}
	return result, nil
}
