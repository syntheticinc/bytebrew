# Prompt Regression Tests

Integration tests for verifying prompt quality. Send a frozen context (context snapshot) to a **real LLM** and check the response quality.

## Build Tag

All files are tagged with `//go:build prompt` and are **NOT included** in regular tests (`go test ./...`).

Run:
```bash
go test -tags prompt ./tests/prompt_regression/...
```

## Structure

```
prompt_regression/
├── README.md              # This file
├── fixture.go             # LoadFixture(name) — load snapshot
├── harness.go             # Harness — ChatModel + ReconstructMessages
├── tool_schemas.go        # getToolSchemas() — supervisor tools
├── assertions.go          # AssertHasToolCall, AssertSubtaskDescriptionQuality, etc.
└── fixtures/
    └── {name}.json        # Fixture files (snapshot + metadata)
```

## How to Use

### 1. Create a Fixture

Run a headless test, take the context snapshot from `logs/{session}/supervisor_step_N_context.json`:

```json
{
  "name": "create_subtask_with_description",
  "description": "Verifies that Supervisor creates a subtask with a detailed description",
  "snapshot": {
    "timestamp": "2026-02-15T12:00:00Z",
    "step": 3,
    "agent_id": "supervisor",
    "total_messages": 5,
    "messages": [
      {
        "index": 0,
        "role": "system",
        "content": "...",
        ...
      },
      ...
    ],
    ...
  }
}
```

Save as `fixtures/{name}.json`.

### 2. Write a Test

```go
//go:build prompt

package prompt_regression

import (
    "context"
    "testing"
)

func TestSubtaskDescriptionQuality(t *testing.T) {
    // Setup
    harness, err := NewHarness()
    if err != nil {
        t.Fatalf("create harness: %v", err)
    }

    ctx := context.Background()
    if err := harness.BindSupervisorTools(ctx); err != nil {
        t.Fatalf("bind tools: %v", err)
    }

    // Load fixture
    fixture, err := LoadFixture("create_subtask_with_description")
    if err != nil {
        t.Fatalf("load fixture: %v", err)
    }

    // Reconstruct messages
    messages := harness.ReconstructMessages(&fixture.Snapshot, "")

    // Generate response from REAL LLM
    response, err := harness.Generate(ctx, messages)
    if err != nil {
        t.Fatalf("generate: %v", err)
    }

    // Assertions
    AssertHasToolCall(t, response, "manage_subtasks")
    AssertSubtaskDescriptionQuality(t, response)
}
```

### 3. Run the Test

```bash
go test -tags prompt -v ./tests/prompt_regression/... -run TestSubtaskDescriptionQuality
```

## Assertions

| Function | Purpose |
|----------|---------|
| `AssertHasToolCall(t, msg, toolName)` | Verifies the message contains a tool call |
| `AssertToolCallArg(t, msg, toolName, argName)` | Verifies the argument exists and is non-empty, returns value |
| `AssertSubtaskDescriptionQuality(t, msg)` | Checks description quality in manage_subtasks: len>100, description!=title, etc. |

## Configuration

`Harness` uses `config.yaml` from the project root:
- `llm.default_provider` must be `"openrouter"`
- `llm.openrouter.api_key` — API key
- `llm.openrouter.model` — model for tests

## Important

- **Tests do NOT mock the LLM** — they use the real API
- Each run costs tokens ($$)
- Fixtures must be deterministic — same input = expectedly similar output
- If a test flakes — the issue is prompt stability, not the test itself
