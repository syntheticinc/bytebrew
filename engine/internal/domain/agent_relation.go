package domain

import (
	"fmt"
	"time"
)

// AgentRelation represents a directed delegation relationship between two agents
// in a schema. There is exactly one relationship type — DELEGATION — expressed
// implicitly by the agent-first runtime: the orchestrator delegates to the
// target agent via a tool call. See docs/architecture/agent-first-runtime.md
// §3.1. Optional Config carries non-typing routing hints (e.g. priority).
type AgentRelation struct {
	ID              string
	SchemaID        string
	SourceAgentName string
	TargetAgentName string
	Config          map[string]interface{}
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewAgentRelation creates a new AgentRelation with validation.
func NewAgentRelation(schemaID, source, target string) (*AgentRelation, error) {
	r := &AgentRelation{
		SchemaID:        schemaID,
		SourceAgentName: source,
		TargetAgentName: target,
		Config:          make(map[string]interface{}),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return r, nil
}

// Validate validates the AgentRelation.
func (r *AgentRelation) Validate() error {
	if r.SchemaID == "" {
		return fmt.Errorf("agent relation schema_id is required")
	}
	if r.SourceAgentName == "" {
		return fmt.Errorf("agent relation source is required")
	}
	if r.TargetAgentName == "" {
		return fmt.Errorf("agent relation target is required")
	}
	if r.SourceAgentName == r.TargetAgentName {
		return fmt.Errorf("agent relation source and target must be different")
	}
	return nil
}
