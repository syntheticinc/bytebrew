package assistant

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// AdminOperations defines the operations the assembler can perform on the admin workspace.
type AdminOperations interface {
	CreateSchema(ctx context.Context, name, description string) (uint, error)
	CreateAgent(ctx context.Context, name, systemPrompt, model string) error
	AddAgentToSchema(ctx context.Context, schemaID uint, agentName string) error
	CreateEdge(ctx context.Context, schemaID uint, source, target, edgeType string) error
	CreateTrigger(ctx context.Context, agentName, triggerType string) error
}

// AssemblyPlan represents a plan for creating a schema with agents and connections.
type AssemblyPlan struct {
	SchemaName  string
	Description string
	Agents      []PlannedAgent
	Edges       []PlannedEdge
	Trigger     *PlannedTrigger
}

// PlannedAgent represents an agent to be created.
type PlannedAgent struct {
	Name         string
	SystemPrompt string
	Role         string // classifier, handler, escalation, etc.
}

// PlannedEdge represents an edge to be created.
type PlannedEdge struct {
	Source string
	Target string
	Type   string
}

// PlannedTrigger represents a trigger to be created.
type PlannedTrigger struct {
	AgentName string
	Type      string // webhook, cron
}

// Assembler creates schemas, agents, edges, and triggers based on interview results.
type Assembler struct {
	ops AdminOperations
}

// NewAssembler creates a new Assembler.
func NewAssembler(ops AdminOperations) *Assembler {
	return &Assembler{ops: ops}
}

// PlanFromInterview creates an assembly plan from interview results.
func (a *Assembler) PlanFromInterview(interview *InterviewState) *AssemblyPlan {
	schemaName := interview.SchemaName
	if schemaName == "" {
		schemaName = "my-workflow"
	}

	plan := &AssemblyPlan{
		SchemaName:  slugify(schemaName),
		Description: fmt.Sprintf("Workflow for: %s", strings.Join(interview.Queries, ", ")),
	}

	// Entry agent (classifier or direct handler)
	if len(interview.Queries) > 2 {
		// Multiple query types → classifier + specialized agents
		plan.Agents = append(plan.Agents, PlannedAgent{
			Name:         plan.SchemaName + "-classifier",
			SystemPrompt: fmt.Sprintf("You are a classifier agent. Route incoming messages to the appropriate specialist. Categories: %s", strings.Join(interview.Queries, ", ")),
			Role:         "classifier",
		})

		for i, query := range interview.Queries {
			agentName := fmt.Sprintf("%s-handler-%d", plan.SchemaName, i+1)
			plan.Agents = append(plan.Agents, PlannedAgent{
				Name:         agentName,
				SystemPrompt: fmt.Sprintf("You are a specialist agent handling: %s. Be helpful and professional.", query),
				Role:         "handler",
			})
			plan.Edges = append(plan.Edges, PlannedEdge{
				Source: plan.SchemaName + "-classifier",
				Target: agentName,
				Type:   "flow",
			})
		}
	} else {
		// Simple → single agent
		plan.Agents = append(plan.Agents, PlannedAgent{
			Name:         plan.SchemaName + "-agent",
			SystemPrompt: fmt.Sprintf("You are a helpful assistant. You handle: %s. Be professional and helpful.", strings.Join(interview.Queries, ", ")),
			Role:         "handler",
		})
	}

	// Add escalation agent if integrations suggest complex workflows
	if len(interview.Integrations) > 0 {
		plan.Agents = append(plan.Agents, PlannedAgent{
			Name:         plan.SchemaName + "-escalation",
			SystemPrompt: "You handle escalated cases that other agents cannot resolve. Transfer to a human operator if needed.",
			Role:         "escalation",
		})
	}

	// Add webhook trigger for entry agent
	entryAgent := plan.Agents[0].Name
	plan.Trigger = &PlannedTrigger{
		AgentName: entryAgent,
		Type:      "webhook",
	}

	return plan
}

// Execute executes an assembly plan, creating all resources and emitting SSE events.
func (a *Assembler) Execute(ctx context.Context, plan *AssemblyPlan, eventStream domain.AgentEventStream) error {
	slog.InfoContext(ctx, "assembler: starting assembly", "schema", plan.SchemaName)

	// 1. Create schema
	schemaID, err := a.ops.CreateSchema(ctx, plan.SchemaName, plan.Description)
	if err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	// 2. Create agents with animation events
	for i, agent := range plan.Agents {
		if err := a.ops.CreateAgent(ctx, agent.Name, agent.SystemPrompt, ""); err != nil {
			return fmt.Errorf("create agent %q: %w", agent.Name, err)
		}

		// Emit node_create event for live animation
		if eventStream != nil {
			position := map[string]float64{
				"x": float64(200 + i*300),
				"y": float64(200),
			}
			eventStream.Send(NewNodeCreateEvent(agent.Name, position))
		}

		// Add agent to schema
		if err := a.ops.AddAgentToSchema(ctx, schemaID, agent.Name); err != nil {
			return fmt.Errorf("add agent %q to schema: %w", agent.Name, err)
		}

		slog.InfoContext(ctx, "assembler: created agent", "name", agent.Name, "role", agent.Role)
	}

	// 3. Create edges with animation events
	for _, edge := range plan.Edges {
		if err := a.ops.CreateEdge(ctx, schemaID, edge.Source, edge.Target, edge.Type); err != nil {
			return fmt.Errorf("create edge %s→%s: %w", edge.Source, edge.Target, err)
		}

		if eventStream != nil {
			eventStream.Send(NewEdgeCreateEvent(edge.Source, edge.Target, edge.Type))
		}

		slog.InfoContext(ctx, "assembler: created edge", "source", edge.Source, "target", edge.Target)
	}

	// 4. Create trigger
	if plan.Trigger != nil {
		if err := a.ops.CreateTrigger(ctx, plan.Trigger.AgentName, plan.Trigger.Type); err != nil {
			slog.ErrorContext(ctx, "assembler: trigger creation failed (non-fatal)", "error", err)
		}
	}

	slog.InfoContext(ctx, "assembler: assembly complete", "schema", plan.SchemaName,
		"agents", len(plan.Agents), "edges", len(plan.Edges))

	return nil
}

func slugify(name string) string {
	lower := strings.ToLower(name)
	lower = strings.ReplaceAll(lower, " ", "-")
	// Remove non-alphanumeric chars (except hyphens)
	var result strings.Builder
	for _, ch := range lower {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
			result.WriteRune(ch)
		}
	}
	return result.String()
}
