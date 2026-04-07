# Contributing to ByteBrew

Thank you for your interest in contributing to ByteBrew!

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/bytebrew.git`
3. Create a branch: `git checkout -b feature/my-feature`
4. Make your changes
5. Run tests: `go test ./...`
6. Build admin UI: `cd admin && npm run build`
7. Submit a Pull Request

## Development Setup

```bash
# Backend
go run ./cmd/ce

# Admin Dashboard
cd admin && npm install && npm run dev

# Tests
go test ./...
cd admin && npx vitest
```

## Code Style

- **Go:** Follow standard Go conventions. Early returns, guard clauses, no `goto`.
- **Error handling:** Always wrap errors: `fmt.Errorf("context: %w", err)`
- **Logging:** Use `slog.InfoContext`/`slog.ErrorContext` with context.
- **Architecture:** Clean Architecture — Domain → Usecase → Infrastructure → Delivery.
- **Interfaces:** Consumer-side, defined in the usecase file.
- **React:** TypeScript, functional components, hooks.

## Pull Request Process

1. Ensure `go build ./...` and `go test ./...` pass
2. Ensure `npm run build` passes in `admin/`
3. Update documentation if needed
4. Describe your changes clearly in the PR description

## Reporting Issues

Use GitHub Issues with the provided templates (bug report or feature request).

## License

By contributing, you agree that your contributions will be licensed under the BSL 1.1 license.
