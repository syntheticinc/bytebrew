# Backend Developer Memory

## Phase 7: AgentCallbackHandler SRP Split (completed)
- Old: `agents/agent_callback_handler.go` (559 lines, 1 struct with 5+ responsibilities)
- New: `agents/callbacks/` subpackage (6 files, each < 300 lines)
  - `event_emitter.go` (45 lines) -- AgentID injection + event dispatch
  - `step_counter.go` (63 lines) -- thread-safe step/modelCallCount/pendingContent
  - `model_event_handler.go` (282 lines) -- OnModelEnd, OnModelEndWithStreamOutput, FinalizeAccumulatedText
  - `tool_event_handler.go` (143 lines) -- OnToolStart, OnToolEnd
  - `plan_progress_emitter.go` (78 lines) -- EmitPlanProgress with consumer-side PlanProvider interface
  - `builder.go` (81 lines) -- wires all components, exposes BuildCallbackOption/GetStep/FinalizeAccumulatedText
- Consumer (`react/agent.go`) uses `callbacks.NewBuilder(callbacks.BuilderConfig{...})`
- No circular dependency: callbacks imports agents (parent), agents does NOT import callbacks
- PlanProvider interface redefined in callbacks (consumer-side principle)
- 20 new tests in callbacks package, all passing

## Architecture Insights
- `domain.NewPlan` auto-marks first step as InProgress -- tests must account for this
- `agents.PlanProvider` is defined in `context_rewriter.go` -- also used by ContextRewriter
- `react.PlanManager` is a larger interface (7 methods), only `PlanProvider.GetActivePlan` needed for callbacks
- `StepContentStore` and `ReasoningExtractor` remain in parent `agents` package -- shared across multiple consumers
