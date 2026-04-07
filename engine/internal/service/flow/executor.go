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
	ListEdges(ctx context.Context, schemaID uint) ([]EdgeRecord, error)
}

// SchemaGateReader reads gates for a schema.
type SchemaGateReader interface {
	GetGateByID(ctx context.Context, id uint) (*GateRecord, error)
	ListGates(ctx context.Context, schemaID uint) ([]GateRecord, error)
}

// EdgeRecord is a simplified edge for the service boundary.
type EdgeRecord struct {
	ID              uint
	SchemaID        uint
	SourceAgentName string
	TargetAgentName string
	Type            string
	Config          map[string]interface{}
}

// GateRecord is a simplified gate for the service boundary.
type GateRecord struct {
	ID            uint
	SchemaID      uint
	Name          string
	ConditionType string
	Config        map[string]interface{}
	MaxIterations int
	Timeout       int
}

// ExecutorConfig holds configuration for the flow executor.
type ExecutorConfig struct {
	SchemaID    uint
	SessionID   string
	EventStream domain.AgentEventStream
}

// Executor orchestrates multi-agent pipeline execution based on schema edges.
type Executor struct {
	agentRunner   AgentRunner
	edgeReader    SchemaEdgeReader
	gateReader    SchemaGateReader
	edgeRouter    *EdgeRouter
	gateEvaluator *GateEvaluator
}

// NewExecutor creates a new flow Executor.
func NewExecutor(runner AgentRunner, edgeReader SchemaEdgeReader, gateReader SchemaGateReader) *Executor {
	return &Executor{
		agentRunner:   runner,
		edgeReader:    edgeReader,
		gateReader:    gateReader,
		edgeRouter:    NewEdgeRouter(),
		gateEvaluator: NewGateEvaluator(),
	}
}

// Execute runs the flow pipeline starting from the entry agent.
func (e *Executor) Execute(ctx context.Context, cfg ExecutorConfig, entryAgent, userInput string) (*domain.FlowExecution, error) {
	edges, err := e.edgeReader.ListEdges(ctx, cfg.SchemaID)
	if err != nil {
		return nil, fmt.Errorf("load schema edges: %w", err)
	}

	execution := domain.NewFlowExecution(fmt.Sprintf("%d", cfg.SchemaID), cfg.SessionID)
	if err := execution.Start(); err != nil {
		return nil, err
	}

	// Build adjacency: source -> [edges]
	adjacency := buildAdjacency(edges)

	// Execute pipeline starting from entry agent
	output, err := e.executeAgent(ctx, cfg, execution, adjacency, entryAgent, userInput, 0)
	if err != nil {
		execution.Fail()
		return execution, err
	}

	_ = output
	execution.Complete()
	return execution, nil
}

// HasOutgoingEdges returns true if the agent has outgoing edges in the schema.
func (e *Executor) HasOutgoingEdges(ctx context.Context, schemaID uint, agentName string) (bool, error) {
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
	var flowEdges, transferEdges, loopEdges []EdgeRecord
	for _, edge := range outgoing {
		switch domain.EdgeType(edge.Type) {
		case domain.EdgeTypeFlow, domain.EdgeTypeParallel:
			flowEdges = append(flowEdges, edge)
		case domain.EdgeTypeTransfer:
			transferEdges = append(transferEdges, edge)
		case domain.EdgeTypeLoop:
			loopEdges = append(loopEdges, edge)
		case domain.EdgeTypeGate:
			flowEdges = append(flowEdges, edge) // gate edges behave like flow
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

		// Check if target is a gate
		if e.isGateNode(ctx, cfg.SchemaID, edge.TargetAgentName) {
			return e.executeGate(ctx, cfg, execution, adjacency, edge.TargetAgentName, output, depth+1, loopEdges)
		}

		return e.executeAgent(ctx, cfg, execution, adjacency, edge.TargetAgentName, routedInput, depth+1)
	}

	if len(flowEdges) > 1 {
		// Multiple flow edges — parallel fork
		return e.executeFork(ctx, cfg, execution, adjacency, flowEdges, output, depth+1)
	}

	// Handle loop edges
	if len(loopEdges) > 0 {
		return e.executeLoop(ctx, cfg, execution, adjacency, loopEdges, agentName, output, depth+1)
	}

	return output, nil
}

// executeFork runs multiple agents in parallel and collects their outputs.
func (e *Executor) executeFork(ctx context.Context, cfg ExecutorConfig, execution *domain.FlowExecution,
	adjacency map[string][]EdgeRecord, edges []EdgeRecord, input string, depth int) (string, error) {

	type result struct {
		agentName string
		output    string
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
			out, err := e.executeAgent(ctx, cfg, execution, adjacency, e2.TargetAgentName, routedInput, depth)
			results[idx] = result{agentName: e2.TargetAgentName, output: out, err: err}
		}(i, edge)
	}

	wg.Wait()

	// Collect outputs — return concatenated or first error
	var outputs []string
	for _, r := range results {
		if r.err != nil {
			return "", fmt.Errorf("parallel agent %q failed: %w", r.agentName, r.err)
		}
		outputs = append(outputs, r.output)
	}

	// For fork, concatenate all outputs (downstream gate/agent will process)
	combined := ""
	for i, o := range outputs {
		if i > 0 {
			combined += "\n---\n"
		}
		combined += o
	}
	return combined, nil
}

// executeGate evaluates a gate condition and proceeds accordingly.
func (e *Executor) executeGate(ctx context.Context, cfg ExecutorConfig, execution *domain.FlowExecution,
	adjacency map[string][]EdgeRecord, gateName, output string, depth int, loopEdges []EdgeRecord) (string, error) {

	gates, err := e.gateReader.ListGates(ctx, cfg.SchemaID)
	if err != nil {
		return "", fmt.Errorf("load gates: %w", err)
	}

	var gate *GateRecord
	for _, g := range gates {
		if g.Name == gateName {
			gate = &g
			break
		}
	}
	if gate == nil {
		return "", fmt.Errorf("gate %q not found", gateName)
	}

	domainGate := &domain.Gate{
		ID:            fmt.Sprintf("%d", gate.ID),
		Name:          gate.Name,
		ConditionType: domain.GateConditionType(gate.ConditionType),
		Config:        gate.Config,
		MaxIterations: gate.MaxIterations,
		Timeout:       gate.Timeout,
	}

	result, err := e.gateEvaluator.Evaluate(domainGate, output)
	if err != nil {
		return "", fmt.Errorf("evaluate gate %q: %w", gateName, err)
	}

	// Emit gate evaluated event
	if cfg.EventStream != nil {
		cfg.EventStream.Send(domain.NewFlowGateEvaluatedEvent(gateName, result.Passed, result.Reason))
	}

	slog.Info("flow: gate evaluated", "gate", gateName, "passed", result.Passed, "reason", result.Reason)

	if result.Passed {
		// Proceed to next node after gate
		outgoing := adjacency[gateName]
		if len(outgoing) == 0 {
			return output, nil
		}
		nextEdge := outgoing[0]
		routedInput, err := e.edgeRouter.RouteOutput(output, nextEdge.Config)
		if err != nil {
			return "", fmt.Errorf("route gate output: %w", err)
		}
		return e.executeAgent(ctx, cfg, execution, adjacency, nextEdge.TargetAgentName, routedInput, depth+1)
	}

	// Gate failed — handle on_failure action
	action := e.resolveGateAction(domainGate)
	switch action {
	case domain.GateActionBlock:
		return "", fmt.Errorf("gate %q condition failed: %s", gateName, result.Reason)
	case domain.GateActionSkip:
		// Skip gate, proceed with next node
		outgoing := adjacency[gateName]
		if len(outgoing) == 0 {
			return output, nil
		}
		return e.executeAgent(ctx, cfg, execution, adjacency, outgoing[0].TargetAgentName, output, depth+1)
	case domain.GateActionEscalate:
		return "", fmt.Errorf("gate %q escalated: %s", gateName, result.Reason)
	default:
		return "", fmt.Errorf("gate %q failed: %s", gateName, result.Reason)
	}
}

// executeLoop runs a loop: re-execute the source agent, check gate, repeat up to max_iterations.
func (e *Executor) executeLoop(ctx context.Context, cfg ExecutorConfig, execution *domain.FlowExecution,
	adjacency map[string][]EdgeRecord, loopEdges []EdgeRecord, sourceAgent, lastOutput string, depth int) (string, error) {

	maxIter := 3 // default
	if len(loopEdges) > 0 {
		// Check if there's a gate with max_iterations
		gates, _ := e.gateReader.ListGates(ctx, cfg.SchemaID)
		for _, g := range gates {
			if g.MaxIterations > 0 {
				maxIter = g.MaxIterations
				break
			}
		}
	}

	for i := 0; i < maxIter; i++ {
		slog.Info("flow: loop iteration", "agent", sourceAgent, "iteration", i+1, "max", maxIter)

		output, err := e.executeAgent(ctx, cfg, execution, adjacency, sourceAgent, lastOutput, depth+i)
		if err != nil {
			return "", err
		}
		lastOutput = output

		// Check if loop should continue (re-evaluate outgoing edges)
		outgoing := adjacency[sourceAgent]
		hasMoreLoop := false
		for _, edge := range outgoing {
			if domain.EdgeType(edge.Type) == domain.EdgeTypeLoop {
				hasMoreLoop = true
				break
			}
		}
		if !hasMoreLoop {
			return output, nil
		}
	}

	return lastOutput, fmt.Errorf("loop max iterations (%d) exceeded for agent %q", maxIter, sourceAgent)
}

func (e *Executor) isGateNode(ctx context.Context, schemaID uint, nodeName string) bool {
	gates, err := e.gateReader.ListGates(ctx, schemaID)
	if err != nil {
		return false
	}
	for _, g := range gates {
		if g.Name == nodeName {
			return true
		}
	}
	return false
}

func (e *Executor) resolveGateAction(gate *domain.Gate) domain.GateAction {
	if gate.Config == nil {
		return domain.GateActionBlock
	}
	actionStr, _ := gate.Config["on_failure"].(string)
	switch domain.GateAction(actionStr) {
	case domain.GateActionSkip:
		return domain.GateActionSkip
	case domain.GateActionEscalate:
		return domain.GateActionEscalate
	default:
		return domain.GateActionBlock
	}
}

func buildAdjacency(edges []EdgeRecord) map[string][]EdgeRecord {
	adj := make(map[string][]EdgeRecord)
	for _, edge := range edges {
		adj[edge.SourceAgentName] = append(adj[edge.SourceAgentName], edge)
	}
	return adj
}
