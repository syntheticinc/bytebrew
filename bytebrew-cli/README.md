# ByteBrew CLI

Terminal client for the ByteBrew AI agent. TUI built on Ink (React) with gRPC connection to the server.

## Stack

- **Bun** (runtime + bundler)
- **TypeScript 5**
- **Ink 6** (React TUI)
- **gRPC** (@grpc/grpc-js)
- **tree-sitter** (code parsing)
- **usearch** (vector search)

> **Runtime: Bun, not Node.** Uses `bun:sqlite` — Node.js is not supported.

## Quick Start

### 1. Install Dependencies

```bash
bun install
```

### 2. Build

```bash
bun run build
```

### 3. Run

```bash
# Interactive mode (TUI)
bun dist/index.js

# With project working directory
bun dist/index.js -C /path/to/your/project

# With server address
bun dist/index.js -s localhost:60401
```

> Requires a running ByteBrew Server (default `localhost:60401`).

## Modes

### Interactive (default)

```bash
bun dist/index.js
```

Full TUI with chat history, syntax highlighting, and tool progress.

### Headless

```bash
# Single question — single answer
bun dist/index.js ask --headless "What does this project do?"

# With file output
bun dist/index.js ask --headless "Architecture analysis" --output result.txt

# With debug output
bun dist/index.js ask --headless --debug "Analyze the code"
```

### Multi-turn Session

```bash
# Interactive input with a single session
bun dist/index.js session

# Automation via pipe
(echo "Question 1"; sleep 30; echo "Question 2") | bun dist/index.js session
```

## CLI Flags

| Flag | Description |
|------|-------------|
| `-C, --directory <path>` | Working directory (similar to `git -C`) |
| `-s, --server <host:port>` | Server address (default `localhost:60401`) |
| `--headless` | Headless mode (plain text, no TUI) |
| `--debug` | Debug output (statuses, reasoning) |
| `--output <file>` | Write output to file (headless only) |
| `--unknown-cmd <mode>` | Unknown command handling mode (`allow-once`, `deny`) |

## Slash Commands

| Command | Description |
|---------|-------------|
| `/provider <name>` | Select LLM provider (ollama, openrouter, anthropic) |
| `/model <name>` | Select model for current provider |
| `/login` | Authenticate with ByteBrew Cloud |
| `/status` | License information |

## Project Structure

```
bytebrew-cli/
├── src/
│   ├── cli.tsx                    # CLI entry point (commander)
│   ├── presentation/
│   │   ├── app/ChatApp.tsx        # Main component (Ink)
│   │   ├── components/            # UI components
│   │   └── hooks/                 # React hooks
│   ├── application/
│   │   └── services/              # Business services
│   ├── infrastructure/
│   │   ├── grpc/                  # gRPC client
│   │   ├── auth/                  # Authentication (Cloud API)
│   │   ├── config/                # Configuration
│   │   ├── license/               # License
│   │   ├── shell/                 # System utilities
│   │   └── api/                   # API client (Cloud)
│   ├── headless/                  # Headless runner
│   └── test-utils/                # Test utilities
├── dist/                          # Built binary
└── patches/                       # Bun dependency patches
```

## Tests

```bash
# All tests
bun test

# Watch mode
bun test --watch

# E2E tests (require built test server)
bun test src/presentation/app/__tests__/ChatApp.e2e.test.tsx
```

## Type Checking

```bash
bun run typecheck
```
