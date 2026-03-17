# Plan: Vector CLI Node (React/Ink/TypeScript)

## Overview

Create a CLI client in `vector-cli-node/` that replicates Claude CLI functionality using React/Ink for terminal UI, connecting to the existing `vector-srv` gRPC server.

## Tech Stack

| Layer | Technology |
|-------|------------|
| UI | React + Ink 5.x |
| Language | TypeScript |
| Build | Bun |
| Runtime | Node.js 18+ |
| gRPC | @grpc/grpc-js |
| State | Zustand |
| Markdown | marked + marked-terminal |

## Project Structure

```
vector-cli-node/
├── src/
│   ├── index.tsx              # Entry point
│   ├── cli.tsx                # Commander CLI setup
│   ├── app/
│   │   ├── App.tsx            # Root component
│   │   └── ChatApp.tsx        # Main chat application
│   ├── components/
│   │   ├── chat/
│   │   │   ├── ChatView.tsx       # Message list viewport
│   │   │   ├── Message.tsx        # Single message
│   │   │   ├── AssistantMessage.tsx
│   │   │   ├── UserMessage.tsx
│   │   │   ├── ReasoningBlock.tsx # Thinking display
│   │   │   └── StreamingText.tsx  # Char-by-char streaming
│   │   ├── tools/
│   │   │   ├── ToolCallView.tsx
│   │   │   └── ToolResultView.tsx
│   │   ├── input/
│   │   │   └── InputField.tsx     # Text input with history
│   │   ├── status/
│   │   │   ├── StatusBar.tsx
│   │   │   └── ConnectionIndicator.tsx
│   │   └── common/
│   │       ├── Spinner.tsx
│   │       └── Markdown.tsx
│   ├── hooks/
│   │   ├── useGrpcStream.ts   # Bidirectional stream
│   │   ├── useConnection.ts   # Connection state
│   │   ├── usePingPong.ts     # Heartbeat
│   │   ├── useToolExecution.ts
│   │   └── useInputHistory.ts
│   ├── store/
│   │   ├── chatStore.ts       # Messages state
│   │   └── connectionStore.ts # Connection state
│   ├── grpc/
│   │   ├── client.ts          # FlowServiceClient
│   │   ├── stream.ts          # StreamManager
│   │   └── reconnect.ts       # ReconnectionManager
│   ├── tools/
│   │   ├── executor.ts        # Tool dispatcher
│   │   ├── registry.ts        # Tool registry
│   │   ├── readFile.ts        # read_file
│   │   ├── searchCode.ts      # search_code
│   │   └── projectTree.ts     # get_project_tree
│   ├── domain/
│   │   ├── message.ts         # ChatMessage type
│   │   └── connection.ts      # ConnectionState type
│   └── config/
│       └── index.ts           # Config loader
├── proto/
│   ├── flow_service.proto     # Copy from vector-srv
│   └── common.proto
├── package.json
├── tsconfig.json
└── buf.gen.yaml
```

## gRPC Protocol (from vector-srv)

### FlowRequest
```protobuf
message FlowRequest {
  string session_id = 1;      // Required
  string user_id = 2;         // Required
  string project_key = 3;     // Required
  string task = 4;            // User message
  bool is_first_message = 6;
  bool cancel = 7;
  PingRequest ping = 8;
  ToolResult tool_result = 9;
}
```

### FlowResponse Types
- `RESPONSE_TYPE_ANSWER` - Final answer
- `RESPONSE_TYPE_ANSWER_CHUNK` - Streaming chunk
- `RESPONSE_TYPE_REASONING` - Thinking/reasoning
- `RESPONSE_TYPE_TOOL_CALL` - Tool invocation request
- `RESPONSE_TYPE_TOOL_RESULT` - Tool result
- `RESPONSE_TYPE_ERROR` - Error

### Tools (executed on client)
1. **read_file** - `{file_path, start_line?, end_line?}`
2. **search_code** - `{query, limit?}`
3. **get_project_tree** - `{max_depth?}`

## Implementation Steps

### Phase 1: Project Setup
1. Create `vector-cli-node/` directory
2. Initialize package.json with dependencies
3. Configure tsconfig.json for ESM + JSX
4. Copy proto files from vector-srv
5. Setup buf.gen.yaml for proto generation

### Phase 2: gRPC Layer
1. Implement `FlowServiceClient` with @grpc/grpc-js
2. Implement `StreamManager` for bidirectional streaming
3. Implement `ReconnectionManager` with exponential backoff
4. Add ping/pong heartbeat (20s interval)

### Phase 3: State Management
1. Create `chatStore` with Zustand (messages, isProcessing)
2. Create `connectionStore` (status, sessionId, reconnectAttempts)
3. Implement streaming message updates

### Phase 4: Tools Layer
1. Implement `ToolRegistry` and `ToolExecutor`
2. Implement `read_file` tool with security checks
3. Implement `search_code` tool with glob search
4. Implement `get_project_tree` tool

### Phase 5: React/Ink Components
1. Create `App.tsx` root component
2. Create `ChatView` with scrollable message list
3. Create `Message` components (User, Assistant, Reasoning)
4. Create `StreamingText` for character animation
5. Create `InputField` with history (up/down arrows)
6. Create `StatusBar` with connection indicator
7. Create `Spinner` component

### Phase 6: Hooks Integration
1. Implement `useGrpcStream` hook
2. Implement `useToolExecution` hook
3. Implement `useInputHistory` hook
4. Wire everything in `ChatApp.tsx`

### Phase 7: CLI Interface
1. Setup Commander.js with commands:
   - `vector chat` - Interactive mode
   - `vector ask <question>` - Single question
2. Add options: `--server`, `--project`, `--model`
3. Handle graceful shutdown (Ctrl+C)

### Phase 8: Polish & Testing
1. Add Markdown rendering with marked-terminal
2. Test all tool executions
3. Test reconnection scenarios
4. Test streaming display

## Key Dependencies

```json
{
  "dependencies": {
    "ink": "^5.0.1",
    "ink-spinner": "^5.0.0",
    "ink-text-input": "^6.0.0",
    "react": "^18.3.1",
    "@grpc/grpc-js": "^1.10.0",
    "@grpc/proto-loader": "^0.7.10",
    "zustand": "^4.5.0",
    "commander": "^12.0.0",
    "marked": "^12.0.0",
    "marked-terminal": "^7.0.0",
    "glob": "^10.3.0",
    "uuid": "^9.0.0"
  }
}
```

## Critical Files to Reference

| File | Purpose |
|------|---------|
| `vector-srv/api/proto/flow_service.proto` | gRPC contract |
| `vector-srv/api/proto/common.proto` | Shared types |
| `vector-cli/internal/infrastructure/grpc_client/streaming_client.go` | Reference streaming impl |
| `vector-cli/internal/infrastructure/tools/` | Reference tool impl |

## Verification Plan

1. **Build test**: `bun build` completes without errors
2. **Connection test**: Connect to `localhost:60401`, see "Connected" status
3. **Chat test**: Send message, receive streaming response
4. **Tool test**: AI calls `read_file`, client executes and returns result
5. **Reconnect test**: Kill server, see reconnection attempts
6. **Ctrl+C test**: Graceful shutdown without errors

## Notes

- Server runs on `localhost:60401` (insecure connection)
- Ping interval: 20 seconds
- Max reconnect attempts: 10
- Reconnect backoff: 1s to 30s (exponential)
- File size limit for read_file: 1MB
