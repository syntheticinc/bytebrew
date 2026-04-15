package domain

import (
	"fmt"
	"time"
)

// Schema represents a named group of agents + agent_relations + triggers.
// Agents are global entities referenced by schemas, not owned by them.
type Schema struct {
	ID          string
	Name        string
	Description string
	IsSystem    bool // system schemas (e.g. builder-schema) are managed by the engine
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewSchema creates a new Schema with validation.
func NewSchema(name, description string) (*Schema, error) {
	s := &Schema{
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return s, nil
}

// NewSystemSchema creates a new system-managed Schema with IsSystem=true.
func NewSystemSchema(name, description string) (*Schema, error) {
	s := &Schema{
		Name:        name,
		Description: description,
		IsSystem:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return s, nil
}

// Validate validates the Schema.
func (s *Schema) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("schema name is required")
	}
	return nil
}
