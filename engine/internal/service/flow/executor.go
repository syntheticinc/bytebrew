package flow

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// AgentRunner executes a single agent and returns its output.
type AgentRunner interface {
	RunAgent(ctx context.Context, agentName, input, sessionID string, eventStream domain.AgentEventStream) (string, error)
}

// SchemaEdgeReader reads edges for a schema.
type SchemaEdgeReader interface {
	ListEdges(ctx context.Context, schemaID string) ([]EdgeRecord, error)
}

// EdgeRecord is a simplified edge for the service boundary.
type EdgeRecord struct {
	ID              string
	SchemaID        string
	SourceAgentName string
	TargetAgentName string
	Type            string
	Config          map[string]interface{}
}

// ExecutorConfig holds configuration for the flow executor.
type ExecutorConfig struct {
	SchemaID    string
	SessionID   string
	EventStream domain.AgentEventStream
}

// Executor orchestrates multi-agent pipeline execution based on schema edges.
type Executor struct {
	agentRunner AgentRunner
	edgeReader  SchemaEdgeReader
	edgeRouter  *EdgeRouter
}

// NewExecutor creates a new flow Executor.
func NewExecutor(runner AgentRunner, edgeReader SchemaEdgeReader) *Executor {
	return &Executor{
		agentRunner: runner,
		edgeReader:  edgeReader,
		edgeRouter:  NewEdgeRouter(),
	}
}

// Execute runs the flow pipeline starting from the entry agent.
func (e *Executor) Execute(ctx context.Context, cfg ExecutorConfig, entryAgent, userInput string) (*domain.FlowExecution, error) {
	edges, err := e.edgeReader.ListEdges(ctx, cfg.SchemaID)
	if err != nil {
		return nil, fmt.Errorf("load schema edges: %w", err)
	}

	execution := domain.NewFlowExecution(cfg.SchemaID, cfg.SessionID)
	if err := execution.Start(); err != nil {
		return nil, err
	}

	// Build adjacency: source -> [edges]
	adjacency := buildAdjacency(edges)

	// Execute pipeline starting from entry agent
	_, err = e.executeAgent(ctx, cfg, execution, adjacency, entryAgent, userInput, 0)
	if err != nil {
		execution.Fail()
		return execution, err
	}

	// output is already recorded in the last FlowStep
	execution.Complete()
	return execution, nil
}

// HasOutgoingEdges returns true if the agent has outgoing edges in the schema.
func (e *Executor) HasOutgoingEdges(ctx context.Context, schemaID string, agentName string) (bool, error) {
	edges, err := e.edgeReader.ListEdges(ctx, schemaID)
	if err != nil {
		return false, fmt.Errorf("load schema edges: %w", err)
	}
	for _, edge := range edges {
		if edge.SourceAgentName == agentName {
			return true, nil
		}
	}
	return false, nil
}

func (e *Executor) executeAgent(ctx context.Context, cfg ExecutorConfig, execution *domain.FlowExecution,
	adjacency map[string][]EdgeRecord, agentName, input string, depth int) (string, error) {

	if depth > 50 {
		return "", fmt.Errorf("flow execution depth exceeded (>50), possible infinite loop")
	}

	stepIdx := len(execution.Steps)
	step := execution.AddStep(agentName)
	step.Status = domain.StepStatusRunning
	step.StartedAt = time.Now()

	// Emit flow.step_started event
	if cfg.EventStream != nil {
		cfg.EventStream.Send(domain.NewFlowStepStartedEvent(agentName, cfg.SessionID, stepIdx))
	}

	slog.Info("flow: executing agent", "agent", agentName, "step", stepIdx, "session", cfg.SessionID)

	// Run the agent
	output, err := e.agentRunner.RunAgent(ctx, agentName, input, cfg.SessionID, cfg.EventStream)
	if err != nil {
		step.Status = domain.StepStatusFailed
		step.Error = err.Error()
		step.FinishedAt = time.Now()
		return "", fmt.Errorf("agent %q failed: %w", agentName, err)
	}

	step.Status = domain.StepStatusCompleted
	step.Output = output
	step.FinishedAt = time.Now()

	// Emit flow.step_completed event
	if cfg.EventStream != nil {
		cfg.EventStream.Send(domain.NewFlowStepCompletedEvent(agentName, cfg.SessionID, stepIdx))
	}

	// Get outgoing edges
	outgoing := adjacency[agentName]
	if len(outgoing) == 0 {
		return output, nil // Terminal node
	}

	// Separate edges by type
	var flowEdges, transferEdges []EdgeRecord
	for _, edge := range outgoing {
		switch domain.EdgeType(edge.Type) {
		case domain.EdgeTypeFlow, domain.EdgeTypeParallel:
			flowEdges = append(flowEdges, edge)
		case domain.EdgeTypeTransfer:
			transferEdges = append(transferEdges, edge)
		}
	}

	// Handle transfer edges — hand off and stop source agent
	if len(transferEdges) > 0 {
		edge := transferEdges[0] // only one transfer target
		routedInput, err := e.edgeRouter.RouteOutput(output, edge.Config)
		if err != nil {
			return "", fmt.Errorf("route output for transfer to %q: %w", edge.TargetAgentName, err)
		}
		return e.executeAgent(ctx, cfg, execution, adjacency, edge.TargetAgentName, routedInput, depth+1)
	}

	// Handle flow edges — may be parallel (fork)
	if len(flowEdges) == 1 {
		// Single flow edge — sequential execution
		edge := flowEdges[0]
		routedInput, err := e.edgeRouter.RouteOutput(output, edge.Config)
		if err != nil {
			return "", fmt.Errorf("route output for flow to %q: %w", edge.TargetAgentName, err)
		}

		return e.executeAgent(ctx, cfg, execution, adjacency, edge.TargetAgentName, routedInput, depth+1)
	}

	if len(flowEdges) > 1 {
		// Multiple flow edges — parallel fork
		return e.executeFork(ctx, cfg, execution, adjacency, flowEdges, output, depth+1)
	}

	return output, nil
}

// executeFork runs multiple agents in parallel using isolated sub-executions,
// then merges their steps into the main execution after all branches complete.
func (e *Executor) executeFork(ctx context.Context, cfg ExecutorConfig, execution *domain.FlowExecution,
	adjacency map[string][]EdgeRecord, edges []EdgeRecord, input string, depth int) (string, error) {

	type result struct {
		agentName string
		output    string
		steps     []domain.FlowStep
		err       error
	}

	var wg sync.WaitGroup
	results := make([]result, len(edges))

	for i, edge := range edges {
		wg.Add(1)
		go func(idx int, e2 EdgeRecord) {
			defer wg.Done()
			routedInput, err := e.edgeRouter.RouteOutput(input, e2.Config)
			if err != nil {
				results[idx] = result{agentName: e2.TargetAgentName, err: err}
				return
			}
			// Each branch gets its own isolated sub-execution to avoid concurrent
			// writes to the shared execution's Steps slice.
			sub := domain.NewFlowExecution(cfg.SchemaID, cfg.SessionID)
			if startErr := sub.Start(); startErr != nil {
				results[idx] = result{agentName: e2.TargetAgentName, err: startErr}
				return
			}
			out, err := e.executeAgent(ctx, cfg, sub, adjacency, e2.TargetAgentName, routedInput, depth)
			results[idx] = result{agentName: e2.TargetAgentName, output: out, steps: sub.Steps, err: err}
		}(i, edge)
	}

	wg.Wait()

	// Merge branch steps into main execution sequentially (no concurrent access).
	var outputs []string
	for _, r := range results {
		if r.err != nil {
			return "", fmt.Errorf("parallel agent %q failed: %w", r.agentName, r.err)
		}
		execution.MergeSteps(r.steps)
		outputs = append(outputs, r.output)
	}

	// Concatenate all branch outputs for downstream processing.
	combined := ""
	for i, o := range outputs {
		if i > 0 {
			combined += "\n---\n"
		}
		combined += o
	}
	return combined, nil
}

func buildAdjacency(edges []EdgeRecord) map[string][]EdgeRecord {
	adj := make(map[string][]EdgeRecord)
	for _, edge := range edges {
		adj[edge.SourceAgentName] = append(adj[edge.SourceAgentName], edge)
	}
	return adj
}
