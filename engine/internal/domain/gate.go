package domain

import (
	"fmt"
	"time"
)

// GateConditionType represents the type of condition a gate evaluates.
type GateConditionType string

const (
	GateConditionAll     GateConditionType = "all"      // wait for all incoming agents
	GateConditionAny     GateConditionType = "any"      // proceed when any incoming agent completes
	GateConditionCustom  GateConditionType = "custom"   // custom condition expression
)

// Gate represents a control-flow node in a schema that joins or branches agent execution.
type Gate struct {
	ID            string
	SchemaID      string
	Name          string
	ConditionType GateConditionType
	Config        map[string]interface{}
	MaxIterations int
	Timeout       int // seconds, 0 = no timeout
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewGate creates a new Gate with validation.
func NewGate(schemaID, name string, conditionType GateConditionType) (*Gate, error) {
	g := &Gate{
		SchemaID:      schemaID,
		Name:          name,
		ConditionType: conditionType,
		Config:        make(map[string]interface{}),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return g, nil
}

// Validate validates the Gate.
func (g *Gate) Validate() error {
	if g.SchemaID == "" {
		return fmt.Errorf("gate schema_id is required")
	}
	if g.Name == "" {
		return fmt.Errorf("gate name is required")
	}
	switch g.ConditionType {
	case GateConditionAll, GateConditionAny, GateConditionCustom:
		// valid
	default:
		return fmt.Errorf("invalid gate condition type: %s", g.ConditionType)
	}
	if g.MaxIterations < 0 {
		return fmt.Errorf("max_iterations must be non-negative")
	}
	if g.Timeout < 0 {
		return fmt.Errorf("timeout must be non-negative")
	}
	return nil
}
