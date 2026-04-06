package domain

import (
	"fmt"
	"time"
)

// EdgeType represents the type of connection between nodes in a schema.
type EdgeType string

const (
	EdgeTypeFlow     EdgeType = "flow"     // sequential: output of source → input of target
	EdgeTypeTransfer EdgeType = "transfer" // hand-off: source transfers conversation to target
	EdgeTypeLoop     EdgeType = "loop"     // loop back to source after target completes
	EdgeTypeParallel EdgeType = "parallel" // source spawns target in parallel
	EdgeTypeGate     EdgeType = "gate"     // connection to/from a gate node
)

// Edge represents a directed connection between two nodes (agents or gates) in a schema.
type Edge struct {
	ID              string
	SchemaID        string
	SourceAgentName string // agent name or gate ID
	TargetAgentName string // agent name or gate ID
	Type            EdgeType
	Config          map[string]interface{}
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewEdge creates a new Edge with validation.
func NewEdge(schemaID, source, target string, edgeType EdgeType) (*Edge, error) {
	e := &Edge{
		SchemaID:        schemaID,
		SourceAgentName: source,
		TargetAgentName: target,
		Type:            edgeType,
		Config:          make(map[string]interface{}),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return e, nil
}

// Validate validates the Edge.
func (e *Edge) Validate() error {
	if e.SchemaID == "" {
		return fmt.Errorf("edge schema_id is required")
	}
	if e.SourceAgentName == "" {
		return fmt.Errorf("edge source is required")
	}
	if e.TargetAgentName == "" {
		return fmt.Errorf("edge target is required")
	}
	if e.SourceAgentName == e.TargetAgentName && e.Type != EdgeTypeLoop {
		return fmt.Errorf("edge source and target must be different (except for loop edges)")
	}
	switch e.Type {
	case EdgeTypeFlow, EdgeTypeTransfer, EdgeTypeLoop, EdgeTypeParallel, EdgeTypeGate:
		// valid
	default:
		return fmt.Errorf("invalid edge type: %s", e.Type)
	}
	return nil
}
