---
paths:
  - "bytebrew-srv/**/*.go"
  - "bytebrew-cloud-api/**/*.go"
---

# Go Code Style

## Early Returns (обязательно)
```go
// ✅ ПРАВИЛЬНО — flat, ошибки сверху
func Process(ctx context.Context, id string) (*Order, error) {
    if id == "" {
        return nil, fmt.Errorf("id required")
    }
    order, err := repo.Get(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("get order: %w", err)
    }
    if order == nil {
        return nil, fmt.Errorf("not found")
    }
    return order, nil
}
```

## Запрещено
- ❌ **goto** — НИКОГДА
- ❌ **else после return** — убирать
- ❌ **Глубокая вложенность** — инвертировать условия
- ❌ **Игнорировать ошибки** — `_ = err` запрещено

## Error Handling
```go
if err != nil {
    return fmt.Errorf("create user: %w", err)
}
```

## Logging
```go
slog.InfoContext(ctx, "processing request", "user_id", userID)
slog.ErrorContext(ctx, "failed to save", "error", err)
```

## Consumer-Side Interfaces
Интерфейсы определяются **В ФАЙЛЕ USECASE**, не в отдельном contract.go:
```go
// usecase/user_create/usecase.go
package user_create

type UserRepository interface {
    Create(ctx context.Context, user *domain.User) error
}

type Usecase struct {
    userRepo UserRepository
}
```
