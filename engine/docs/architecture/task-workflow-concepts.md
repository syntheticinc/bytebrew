# Task Workflow Concepts (V1 Reference)

This document captures the V1 task workflow patterns that were removed in V2.
Preserved as a reference for future V3 BPMN-based workflow design.

## Overview

V1 introduced a DAG-based multi-agent workflow system where a supervisor agent
decomposed work into tasks and subtasks, then spawned specialized code agents to
execute them. The system was removed in V2 in favor of a simpler session-based
agent model.

## Core Concepts

### Task Hierarchy

```
Session
└── Task (top-level unit of work)
    ├── Subtask 1 (atomic work item)
    ├── Subtask 2
    └── Subtask N
```

- **Task**: A high-level work goal decomposed from the user's request. Had a title,
  description, acceptance criteria, and status (`pending`, `in_progress`, `completed`,
  `failed`).
- **Subtask**: An atomic unit of work assigned to a single code agent. Had an ID,
  title, description, and status (`pending`, `in_progress`, `completed`, `failed`).

### Tools

Three supervisor-level tools drove the workflow:

| Tool | Purpose |
|------|---------|
| `manage_tasks` | Create, list, update, and complete top-level tasks |
| `manage_subtasks` | Create subtasks under a task; list, update statuses |
| `spawn_code_agent` | Spawn a code agent for a specific subtask ID |

### Workflow: Interview → Create → Manage → Spawn

```
1. INTERVIEW
   Supervisor asks user clarifying questions via ask_user.
   Captures scope, constraints, acceptance criteria.

2. CREATE TASKS
   Supervisor calls manage_tasks(action=create, ...) with:
   - title: concise work goal
   - description: detailed spec (>100 chars)
   - acceptance_criteria: list of verifiable conditions

3. DECOMPOSE INTO SUBTASKS
   For each task, supervisor calls manage_subtasks(action=create, ...) with:
   - task_id: parent task
   - title: atomic work item
   - description: implementation details
   Subtasks had quality requirements: description longer than title, >100 chars,
   must include acceptance criteria keywords.

4. SPAWN CODE AGENTS
   For each subtask, supervisor called:
     spawn_code_agent(action=spawn, subtask_id=<id>)
   This created a RunningAgent with blockingSpawn=true/false.

5. WAIT FOR COMPLETION
   Supervisor blocked via WaitForAllSessionAgents() until all agents completed
   or a user interrupt arrived (NotifyUserMessage).

6. HANDLE RESULTS / RETRY
   On completion: supervisor reviewed results and either finished or created
   more subtasks. On interrupt: supervisor handled the user message and decided
   whether to continue, stop, or modify the plan.
```

### Agent Pool: Blocking vs Non-Blocking Spawn

- **Blocking spawn** (`blockingSpawn=true`): Supervisor goroutine blocks on
  `WaitForAllSessionAgents()`. Used when the supervisor needed to act on results
  before continuing. The wait returned a `WaitResult` with `AllDone`, `Interrupted`,
  `IsInterruptResponder`, `UserMessage`, `StillRunning`, and `Results` fields.

- **Non-blocking spawn** (`blockingSpawn=false`): Agent ran in background. Supervisor
  continued without waiting. Results were published to the event bus.

### Interrupt Handling

`NotifyUserMessage(sessionID, message)` broadcast an interrupt signal to all active
`WaitForAllSessionAgents` calls for the session. When multiple parallel waiters
existed (e.g., parallel sub-supervisor agents), exactly one received
`IsInterruptResponder=true` via atomic CAS on a shared counter. The others received
`IsInterruptResponder=false` (paused state) and were expected to yield.

### SubtaskManager Interface

```go
type SubtaskManager interface {
    GetSubtask(ctx context.Context, id string) (*domain.Subtask, error)
    UpdateSubtaskStatus(ctx context.Context, id, status, result string) error
}
```

The pool used this to update subtask status to `in_progress` on spawn and
`completed`/`failed` on terminal state.

### Domain Types (removed in V2)

```go
type Subtask struct {
    ID          string
    TaskID      string
    Title       string
    Description string
    Status      SubtaskStatus   // pending | in_progress | completed | failed
    Result      string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type SubtaskStatus string
const (
    SubtaskStatusPending    SubtaskStatus = "pending"
    SubtaskStatusInProgress SubtaskStatus = "in_progress"
    SubtaskStatusCompleted  SubtaskStatus = "completed"
    SubtaskStatusFailed     SubtaskStatus = "failed"
)
```

### Priority Queue and Scheduling

Subtasks were scheduled via a priority queue inside the `work` package. The scheduler
respected `MaxConcurrent` (max parallel agents per session). When the limit was
reached, new spawns queued. On agent completion the scheduler dequeued the next
pending subtask and spawned it automatically.

### AgentRun Persistence

Each spawned agent produced an `AgentRun` record persisted via `AgentRunStorage`:

```go
type AgentRun struct {
    ID         string
    SubtaskID  string
    SessionID  string
    FlowType   string
    Status     string
    Result     string
    Error      string
    StartedAt  time.Time
    FinishedAt *time.Time
}
```

This provided an audit trail of which agents ran for which subtasks and what they
produced.

## Why It Was Removed in V2

- The task/subtask hierarchy added complexity without proportional user value.
- The interview→decompose→spawn flow was too rigid; real workflows needed more
  dynamic agent collaboration.
- `manage_tasks` and `manage_subtasks` as LLM tools were fragile: prompt quality
  determined plan quality directly.
- The blocking wait mechanism was a source of concurrency bugs.
- V2 replaces this with a session-based model where agents are spawned directly
  via `SpawnWithDescription` and orchestrated by the `Orchestrator` service using
  an event bus.

## Notes for V3 BPMN Design

The core patterns worth preserving in a future BPMN workflow engine:

1. **DAG decomposition**: Tasks as nodes, dependencies as directed edges. The
   scheduler should respect topological order and parallelize independent nodes.

2. **Acceptance criteria as completion gate**: Each work node has a verifiable
   completion condition. The engine should validate it before marking done.

3. **Interrupt / human-in-the-loop**: Any running workflow step should be
   interruptible by a user message. The BPMN intermediate catch event or boundary
   event is the right primitive.

4. **Single-responder on parallel interrupt**: When N parallel branches are
   interrupted simultaneously, exactly one should handle the user message. Others
   pause. This maps to a BPMN event-based gateway with a mutex token.

5. **Agent run audit trail**: Every agent execution is a traceable unit. Persist
   start time, end time, result, and error for replay and debugging.

6. **MaxConcurrent as resource pool**: Model as a BPMN resource with capacity N.
   Tokens consumed on spawn, released on completion.
