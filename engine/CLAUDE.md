# ByteBrew Server (Go)

## Stack
- Go 1.25+, HTTP REST + SSE chat
- Cloudwego Eino (ReAct agent framework)
- PostgreSQL + GORM, Viper + YAML config
- OpenAI-compatible API, slog logging

## Structure
```
bytebrew/engine/
├── cmd/ce/                # Production entry point (Community Edition)
├── internal/
│   ├── domain/            # Pure entities (NO external deps, NO tags)
│   ├── usecase/           # Business logic + consumer-side interfaces
│   ├── service/           # Reusable helpers
│   ├── delivery/http/     # HTTP handlers (thin!)
│   └── infrastructure/    # DB, APIs, tools, agents
├── tests/prompt_regression/ # Prompt regression tests
└── logs/                  # Session logs + context snapshots
```

## Commands
```bash
go run ./cmd/ce                  # Start server
go test ./...                    # Unit tests
go test -tags prompt -v -timeout 300s ./tests/prompt_regression/...  # Prompt regression
```

## Go Code Style

### Early Returns (обязательно)
Ошибки сверху, happy path внизу. Flat structure.

### Запрещено
- goto — НИКОГДА
- else после return — убирать
- Глубокая вложенность — инвертировать условия
- Игнорировать ошибки — `_ = err` запрещено

### Error Handling
```go
if err != nil {
    return fmt.Errorf("create user: %w", err)
}
```

### Logging
```go
slog.InfoContext(ctx, "processing request", "user_id", userID)
slog.ErrorContext(ctx, "failed to save", "error", err)
```

## Testing

### Integration Tests (Level 1)
- `tests/integration/` — integration suite hitting the running engine via HTTP REST + SSE.

### Prompt Regression (Level 2)
- `tests/prompt_regression/fixtures/` — JSON fixtures из логов
- Build tag: `//go:build prompt`
- Fixtures из `logs/<session>/supervisor_step_N_context.json`

### Context Logger
- `internal/infrastructure/agents/context_logger.go`
- Логирует контекст LLM в `logs/<session>/step_N_context.json`
