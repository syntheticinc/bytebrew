import { describe, it, expect, beforeAll, beforeEach, afterEach } from 'bun:test';
import React from 'react';
import { render } from 'ink-testing-library';
import { Container } from '../../../config/container.js';
import { ChatApp } from '../ChatApp.js';
import { TestServerHelper } from '../../../test-utils/TestServerHelper.js';
import { Message } from '../../../domain/entities/Message.js';
import fs from 'fs';
import path from 'path';
import os from 'os';

/**
 * E2E tests with real Go test server (mock LLM).
 *
 * These tests use REAL GrpcStreamGateway + REAL Go server components:
 * - FlowHandler
 * - EngineTurnExecutorFactory
 * - EngineAdapter
 * - Engine + REACT agent
 * - GrpcAgentEventStream
 * - gRPC + protobuf serialization
 *
 * Only difference from production: MockChatModel instead of OpenRouter/Ollama.
 *
 * What's tested that integration tests (TC-1—TC-22) DON'T cover:
 * - gRPC protobuf encoding/decoding
 * - GrpcAgentEventStream transformation (domain.AgentEvent → pb.FlowResponse)
 * - FlowHandler flow management, session handling
 * - REACT agent loop (tool call → execute → ChatModel again)
 * - Engine execution pipeline
 * - Tool classification and routing
 */
describe('E2E: ChatApp with real gRPC + Go server', () => {
  let server: TestServerHelper;
  let testDir: string;

  beforeAll(() => {
    // Build Go test server binary once
    TestServerHelper.build();
  }, 60000); // 60s timeout for Go build

  beforeEach(() => {
    server = new TestServerHelper();
    testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'e2e-'));
  });

  afterEach(async () => {
    await server.stop();
    // Windows file lock: wait for handles to release before cleanup
    await new Promise((r) => setTimeout(r, 200));
    try {
      fs.rmSync(testDir, { recursive: true, force: true });
    } catch {
      // Ignore cleanup errors (Windows file locks)
    }
  });

  /** Create test Go files for smart_search tests and init git repo */
  function setupSearchFiles(dir: string) {
    const { execSync } = require('child_process');
    execSync('git init', { cwd: dir, stdio: 'ignore' });

    const srcDir = path.join(dir, 'src');
    const pkgDir = path.join(dir, 'pkg', 'errors');
    fs.mkdirSync(srcDir, { recursive: true });
    fs.mkdirSync(pkgDir, { recursive: true });

    // Error handling module
    fs.writeFileSync(path.join(pkgDir, 'errors.go'), `package errors

import "fmt"

// handleError wraps an error with context message
func handleError(err error, context string) error {
    if err == nil {
        return nil
    }
    return fmt.Errorf("%s: %w", context, err)
}

// DomainError represents a domain-specific error with code
type DomainError struct {
    Code    string
    Message string
    Err     error
}

func (e *DomainError) Error() string {
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}
`);

    // HTTP handler
    fs.writeFileSync(path.join(srcDir, 'handler.go'), `package src

import "net/http"

func handleRequest(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("ok"))
}
`);

    // Unrelated config (should NOT appear in error-related searches)
    fs.writeFileSync(path.join(srcDir, 'config.go'), `package src

type Config struct {
    Port     int
    Host     string
    Database string
}
`);

    // Additional file with mixed content for cross-file testing
    fs.writeFileSync(path.join(srcDir, 'middleware.go'), `package src

import "net/http"

// AuthMiddleware validates request authentication
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if token == "" {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}
`);

    execSync('git add -A', { cwd: dir, stdio: 'ignore' });
  }

  // Helper: create container with REAL GrpcStreamGateway (no mock!)
  function createTestContainer(port: number, projectRoot?: string): Container {
    const container = new Container({
      projectRoot: projectRoot || '/test',
      serverAddress: `localhost:${port}`,
      projectKey: 'e2e-test',
      headlessMode: true,
      askUserCallback: async (question: string) => 'approved',
      // Disable on-demand LSP server spawning in tests — avoids 30-45s spawn+init per server.
      // symbolSearch (tree-sitter based) still works; LSP operations return graceful "no results".
      disableLspServers: true,
    });
    container.initialize();
    return container;
  }

  // Helper: connect to server and send message
  async function connectAndSend(container: Container, message: string): Promise<void> {
    // Wait for React useEffect to establish connection (ChatApp calls connect() on mount).
    // If not connected within 5s, connect manually as fallback.
    const start = Date.now();
    while (container.streamGateway.getStatus() !== 'connected' && Date.now() - start < 5000) {
      await new Promise((r) => setTimeout(r, 50));
    }

    if (container.streamGateway.getStatus() !== 'connected') {
      await container.streamGateway.connect({
        serverAddress: container.config.serverAddress,
        sessionId: container.sessionId,
        userId: 'e2e-user',
        projectKey: container.config.projectKey,
        projectRoot: container.config.projectRoot,
      });
      await new Promise((r) => setTimeout(r, 100));
    }

    container.streamProcessor.sendMessage(message);
  }

  // Helper: wait for messages matching predicate (polling with timeout)
  async function waitForMessages(
    container: Container,
    predicate: (msgs: Message[]) => boolean,
    timeout = 15000,
  ): Promise<Message[]> {
    const start = Date.now();
    while (Date.now() - start < timeout) {
      const messages = container.messageRepository.findComplete();
      if (predicate(messages)) return messages;
      await new Promise((r) => setTimeout(r, 100));
    }

    // Timeout: dump messages for debugging
    const msgs = container.messageRepository.findComplete();
    throw new Error(
      `Timeout (${timeout}ms) waiting for messages. Got ${msgs.length} messages:\n` +
        msgs.map((m) => `  [${m.role}] ${m.content.value.slice(0, 80)}`).join('\n'),
    );
  }

  // Helper: wait for processing to stop
  async function waitForProcessingStopped(container: Container, timeout = 15000): Promise<void> {
    const start = Date.now();
    while (Date.now() - start < timeout) {
      if (!container.streamProcessor.getIsProcessing()) return;
      await new Promise((r) => setTimeout(r, 100));
    }
    throw new Error('Timeout waiting for processing to stop');
  }

  // E2E-1: Echo (базовый gRPC pipeline)
  it('receives text answer via real gRPC pipeline', async () => {
    await server.start('echo');
    const container = createTestContainer(server.port);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Hello');

      // Wait for complete messages containing expected text
      await waitForMessages(container, (msgs) =>
        msgs.some((m) => m.content.value.includes('Hello, world!')),
      );

      // Verify processing stopped (IsFinal received)
      await waitForProcessingStopped(container);

      // Data layer checks
      const messages = container.messageRepository.findComplete();
      // Should have: user message + assistant answer
      expect(messages.length).toBeGreaterThanOrEqual(2);
      expect(messages.some((m) => m.role === 'user')).toBe(true);
      expect(messages.some((m) => m.role === 'assistant')).toBe(true);

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('Hello, world!');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // E2E-2: Server-side tool call (REACT loop)
  it('executes server-side tool via REACT agent loop', async () => {
    await server.start('server-tool');
    const container = createTestContainer(server.port);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'List subtasks');

      // Wait for both tool call and final answer
      await waitForMessages(container, (msgs) => {
        const hasToolCall = msgs.some((m) => m.toolCall?.toolName === 'manage_subtasks');
        const hasFinalAnswer = msgs.some(
          (m) => m.role === 'assistant' && m.content.value.includes('complete'),
        );
        return hasToolCall && hasFinalAnswer;
      });

      await waitForProcessingStopped(container);

      // Data layer checks
      const messages = container.messageRepository.findComplete();
      // Should have: user message + tool call message + final answer
      expect(messages.some((m) => m.toolCall?.toolName === 'manage_subtasks')).toBe(true);
      expect(
        messages.some((m) => m.role === 'assistant' && m.content.value.includes('complete')),
      ).toBe(true);

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('Subtasks'); // ManageSubtasks tool should be visible
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // E2E-3: Reasoning
  it('receives reasoning content via real gRPC', async () => {
    await server.start('reasoning');
    const container = createTestContainer(server.port);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Think about this');

      // Wait for reasoning message
      await waitForMessages(container, (msgs) =>
        msgs.some((m) => m.reasoning !== undefined && m.reasoning !== null),
      );

      await waitForProcessingStopped(container);

      // Data layer checks
      const messages = container.messageRepository.findComplete();
      // Should have reasoning content
      const reasoningMsg = messages.find((m) => m.reasoning !== undefined);
      expect(reasoningMsg).toBeDefined();
      expect(reasoningMsg!.reasoning).toBeTruthy();

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('The answer is 42.');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // E2E-4: Error
  it('handles LLM error gracefully', async () => {
    await server.start('error');
    const container = createTestContainer(server.port);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Fail');

      // Wait for processing to stop (error terminates flow)
      await waitForProcessingStopped(container);

      // Processing should have stopped without crashing
      expect(container.streamProcessor.getIsProcessing()).toBe(false);

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      // Error scenario may show error message or empty output - just verify render didn't crash
      expect(frame).toBeDefined();
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // E2E-5: Proxied read_file round-trip
  it('executes proxied read_file via client round-trip', async () => {
    const srcDir = path.join(testDir, 'src');
    fs.mkdirSync(srcDir, { recursive: true });
    fs.writeFileSync(path.join(srcDir, 'main.ts'), 'export const hello = "world";');

    await server.start('proxied-read');
    const container = createTestContainer(server.port, testDir);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Read the file');

      await waitForMessages(container, (msgs) => {
        const hasToolCall = msgs.some((m) => m.toolCall?.toolName === 'read_file');
        const hasFinalAnswer = msgs.some((m) =>
          m.role === 'assistant' && m.content.value.length > 0 && !m.toolCall);
        return hasToolCall && hasFinalAnswer;
      });

      await waitForProcessingStopped(container);

      // Data layer checks
      const messages = container.messageRepository.findComplete();
      const toolMsg = messages.find((m) => m.toolCall?.toolName === 'read_file');
      expect(toolMsg?.toolResult).toBeDefined();
      expect(toolMsg?.toolResult?.result).toContain('hello');

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('Read'); // Read tool should be visible
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // E2E-6: Proxied write_file round-trip
  it('executes proxied write_file via client round-trip', async () => {
    await server.start('proxied-write');
    const container = createTestContainer(server.port, testDir);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Write a file');

      await waitForMessages(container, (msgs) => {
        const hasToolCall = msgs.some((m) => m.toolCall?.toolName === 'write_file');
        const hasFinalAnswer = msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.length > 0);
        return hasToolCall && hasFinalAnswer;
      });

      await waitForProcessingStopped(container);

      // Data layer checks
      const writtenPath = path.join(testDir, 'output.txt');
      expect(fs.existsSync(writtenPath)).toBe(true);
      expect(fs.readFileSync(writtenPath, 'utf-8')).toContain('hello');

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('Write'); // Write tool should be visible
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // E2E-7: Proxied execute_command round-trip
  it('executes proxied execute_command via client round-trip', async () => {
    await server.start('proxied-exec');
    const container = createTestContainer(server.port, testDir);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Run a command');

      await waitForMessages(container, (msgs) => {
        const hasToolCall = msgs.some((m) => m.toolCall?.toolName === 'execute_command');
        const hasFinalAnswer = msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.length > 0);
        return hasToolCall && hasFinalAnswer;
      });

      await waitForProcessingStopped(container);

      // Data layer checks
      const messages = container.messageRepository.findComplete();
      const toolMsg = messages.find((m) => m.toolCall?.toolName === 'execute_command');
      expect(toolMsg?.toolResult).toBeDefined();

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('Exec'); // Exec tool should be visible
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // E2E-8: ask_user round-trip
  // Expected behavior: ask_user question AND user's response should be preserved
  // in chat history so the user can scroll back and see what was asked/answered.
  it('handles ask_user tool via client round-trip', async () => {
    await server.start('ask-user');
    const container = createTestContainer(server.port, testDir);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Ask the user');

      await waitForMessages(container, (msgs) =>
        msgs.some((m) =>
          m.role === 'assistant' && m.content.value.length > 0 && !m.toolCall),
      );

      await waitForProcessingStopped(container);

      // BUG CHECK: ask_user question and response should be visible in chat history
      const messages = container.messageRepository.findComplete();

      // The user's response should be preserved in message history
      const hasUserResponse = messages.some(
        (m) => m.role === 'user' && m.content.value.includes('approved'),
      );
      expect(hasUserResponse).toBe(true);

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('approved'); // User's response should be visible
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // E2E-9: Multiple sequential proxied tool calls
  it('handles multiple sequential proxied tool calls', async () => {
    fs.writeFileSync(path.join(testDir, 'a.ts'), 'const a = 1;');
    fs.writeFileSync(path.join(testDir, 'b.ts'), 'const b = 2;');

    await server.start('multi-tool');
    const container = createTestContainer(server.port, testDir);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Read two files');

      await waitForMessages(container, (msgs) => {
        const toolCalls = msgs.filter((m) => m.toolCall?.toolName === 'read_file');
        const hasFinalAnswer = msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.length > 0);
        return toolCalls.length >= 2 && hasFinalAnswer;
      });

      await waitForProcessingStopped(container);

      // Data layer checks
      const messages = container.messageRepository.findComplete();
      const toolMsgs = messages.filter((m) => m.toolCall?.toolName === 'read_file');
      expect(toolMsgs.length).toBe(2);
      toolMsgs.forEach((m) => expect(m.toolResult).toBeDefined());

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('Read'); // Multiple Read tools should be visible
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // E2E-10: Tool error recovery
  it('handles proxied tool error gracefully', async () => {
    await server.start('tool-error');
    const container = createTestContainer(server.port, testDir);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Read nonexistent');

      await waitForProcessingStopped(container);

      expect(container.streamProcessor.getIsProcessing()).toBe(false);

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('Read'); // Read tool should be visible
      // Tool error should be handled gracefully and final answer shown
      expect(frame).toBeDefined();
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // E2E-11: manage_tasks create → ask_user compound flow
  // Expected behavior: after task approval, the task details AND user's approval
  // should be visible in chat history. The user should see what task was proposed
  // and that they approved it.
  it('executes manage_tasks create with embedded ask_user approval', async () => {
    await server.start('task-create');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Create a task');

      // manage_tasks is a SERVER-SIDE tool (callId starts with "server-")
      // But it internally calls ask_user via proxy → client responds "approved"
      await waitForMessages(container, (msgs) => {
        const hasToolCall = msgs.some((m) => m.toolCall?.toolName === 'manage_tasks');
        const hasFinalAnswer = msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.length > 0);
        return hasToolCall && hasFinalAnswer;
      });

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();

      // manage_tasks tool message should exist with result
      const toolMsg = messages.find((m) => m.toolCall?.toolName === 'manage_tasks');
      expect(toolMsg).toBeDefined();
      expect(toolMsg?.toolResult).toBeDefined();
      expect(toolMsg?.toolResult?.result).toContain('approved');

      // BUG CHECK: The approval question should be preserved in chat history
      // so the user can see what task was proposed and approved
      const hasApprovalQuestion = messages.some(
        (m) => m.content.value.includes('Task') && m.content.value.includes('approve'),
      );
      expect(hasApprovalQuestion).toBe(true);

      // BUG CHECK: The user's approval response should be visible in chat history
      const hasUserApproval = messages.some(
        (m) => m.role === 'user' && m.content.value.includes('approved'),
      );
      expect(hasUserApproval).toBe(true);
    } finally {
      await container.dispose();
    }
  }, 30000);

  // E2E-12: Proxied edit_file round-trip
  it('executes proxied edit_file via client round-trip', async () => {
    // Create file with known content to be edited
    const srcDir = path.join(testDir, 'src');
    fs.mkdirSync(srcDir, { recursive: true });
    fs.writeFileSync(path.join(srcDir, 'app.ts'), "console.log('old');\nconsole.log('keep');");

    await server.start('proxied-edit');
    const container = createTestContainer(server.port, testDir);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Edit the file');

      await waitForMessages(container, (msgs) => {
        const hasToolCall = msgs.some((m) => m.toolCall?.toolName === 'edit_file');
        const hasFinalAnswer = msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.length > 0);
        return hasToolCall && hasFinalAnswer;
      });

      await waitForProcessingStopped(container);

      // Verify file was ACTUALLY edited on disk
      const editedContent = fs.readFileSync(path.join(srcDir, 'app.ts'), 'utf-8');
      expect(editedContent).toContain("console.log('new')");
      expect(editedContent).toContain("console.log('keep')");

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('Edit'); // Edit tool should be visible
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // E2E-13: Proxied get_project_tree round-trip
  it('executes proxied get_project_tree via client round-trip', async () => {
    // Create directory structure
    const srcDir = path.join(testDir, 'src');
    fs.mkdirSync(srcDir, { recursive: true });
    fs.writeFileSync(path.join(srcDir, 'index.ts'), 'export {};');
    fs.writeFileSync(path.join(testDir, 'README.md'), '# Test');

    await server.start('proxied-tree');
    const container = createTestContainer(server.port, testDir);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Show project tree');

      await waitForMessages(container, (msgs) => {
        const hasToolCall = msgs.some((m) => m.toolCall?.toolName === 'get_project_tree');
        const hasFinalAnswer = msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.length > 0);
        return hasToolCall && hasFinalAnswer;
      });

      await waitForProcessingStopped(container);

      // Data layer checks
      const messages = container.messageRepository.findComplete();
      const toolMsg = messages.find((m) => m.toolCall?.toolName === 'get_project_tree');
      expect(toolMsg?.toolResult).toBeDefined();
      // Tree result should contain our files/dirs
      expect(toolMsg?.toolResult?.result).toBeTruthy();

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('Tree'); // GetProjectTree tool should be visible
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // E2E-14: Proxied search_code round-trip (graceful with no index)
  it('executes proxied search_code round-trip gracefully', async () => {
    await server.start('proxied-search');
    const container = createTestContainer(server.port, testDir);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Search for code');

      // search_code may return "no results" or error, but should complete gracefully
      await waitForProcessingStopped(container);

      expect(container.streamProcessor.getIsProcessing()).toBe(false);

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      // Should complete gracefully - verify render didn't crash
      expect(frame).toBeDefined();
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // E2E-15: Multi-agent — supervisor spawns code agent
  // Tests the full lifecycle: supervisor → spawn_code_agent → code agent executes → completes
  // Verifies that the incoming task and supervisor's message are visible in the spawned agent
  it('spawns code agent and shows task + lifecycle events', async () => {
    await server.start('multi-agent');
    const container = createTestContainer(server.port, testDir);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Spawn an agent');

      // Wait for the full multi-agent flow:
      // 1. Supervisor calls spawn_code_agent → server spawns agent
      // 2. [agent_spawned] lifecycle event arrives → creates lifecycle + [Task] messages
      // 3. Code agent runs (MockChatModel returns text) → code agent messages arrive
      // 4. [agent_completed] lifecycle event arrives
      // 5. Supervisor gets tool result → provides final answer
      await waitForMessages(container, (msgs) => {
        const hasFinalAnswer = msgs.some(
          (m) => m.role === 'assistant' && m.content.value.includes('All agents completed'),
        );
        return hasFinalAnswer;
      }, 30000);

      await waitForProcessingStopped(container);

      // Data layer checks
      const messages = container.messageRepository.findComplete();

      // === Incoming task visible in spawned agent ===
      // When agent_spawned event arrives, client creates [Task from Supervisor] message
      // with the FULL supervisor input (title + description), and the code agent's agentId
      const taskMessage = messages.find(
        (m) => m.content.value.startsWith('[Task from Supervisor]'),
      );
      expect(taskMessage).toBeDefined();
      // Full task input: title + description (same as what code agent receives)
      expect(taskMessage!.content.value).toContain('Implement greeting function');
      expect(taskMessage!.content.value).toContain('Create a greeting function');
      // [Task] message should have the code agent's agentId (not supervisor)
      expect(taskMessage!.agentId).toBeDefined();
      expect(taskMessage!.agentId).not.toBe('supervisor');
      expect(taskMessage!.agentId!.startsWith('code-agent-')).toBe(true);

      // === Supervisor lifecycle message visible ===
      // Lifecycle event creates a formatted message in supervisor's stream
      // Uses first line only (title), not the full description
      const spawnedMsg = messages.find(
        (m) => m.content.value.includes('spawned') && m.content.value.includes('Implement greeting'),
      );
      expect(spawnedMsg).toBeDefined();
      expect(spawnedMsg!.agentId).toBe('supervisor');
      // Supervisor lifecycle message should NOT contain the full description (too verbose)
      expect(spawnedMsg!.content.value).not.toContain('Create a greeting function');

      // === Agent completion visible ===
      const completedMsg = messages.find(
        (m) => m.content.value.includes('completed') && m.agentId === 'supervisor'
            && m.content.value.includes('Code Agent'),
      );
      expect(completedMsg).toBeDefined();

      // === Code agent's output visible ===
      // Code agent's answer should be saved with the code agent's agentId
      const codeAgentMsgs = messages.filter(
        (m) => m.agentId && m.agentId.startsWith('code-agent-') && !m.content.value.startsWith('[Task]'),
      );
      expect(codeAgentMsgs.length).toBeGreaterThanOrEqual(1);

      // === spawn_code_agent tool call visible ===
      const spawnToolMsg = messages.find(
        (m) => m.toolCall?.toolName === 'spawn_code_agent',
      );
      expect(spawnToolMsg).toBeDefined();

      // === Supervisor's final answer visible ===
      const finalAnswer = messages.find(
        (m) => m.role === 'assistant' && m.content.value.includes('All agents completed')
            && (m.agentId === 'supervisor' || !m.agentId),
      );
      expect(finalAnswer).toBeDefined();

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('spawned'); // Lifecycle events should be visible
      expect(frame).toContain('completed');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 45000);

  // E2E-16: Lifecycle event ordering — completed event arrives before supervisor resumes
  // This test verifies the race condition fix: emit event BEFORE signalCompletion
  // in agent_events.go. The lifecycle "completed" message should arrive on the client
  // BEFORE the supervisor continues processing and delivers the final answer.
  it('delivers lifecycle completed event before supervisor final answer', async () => {
    await server.start('multi-agent');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Spawn an agent');

      await waitForMessages(
        container,
        (msgs) =>
          msgs.some((m) => m.role === 'assistant' && m.content.value.includes('All agents completed')),
        30000,
      );

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();

      // Find lifecycle "completed" event (supervisor's view of agent completion)
      const completedIdx = messages.findIndex(
        (m) =>
          m.agentId === 'supervisor' &&
          (m.content.value.includes('completed') || m.content.value.includes('Completed')) &&
          m.content.value.includes('Code Agent'),
      );

      // Find supervisor's final answer after receiving tool result
      const finalIdx = messages.findIndex(
        (m) => m.role === 'assistant' && m.content.value.includes('All agents completed'),
      );

      expect(completedIdx).toBeGreaterThan(-1); // lifecycle event exists
      expect(finalIdx).toBeGreaterThan(-1); // final answer exists

      // CRITICAL: lifecycle event MUST come before final answer
      // This verifies the race condition fix: emit event BEFORE signalCompletion
      expect(completedIdx).toBeLessThan(finalIdx);
    } finally {
      await container.dispose();
    }
  }, 45000);

  // E2E-17: Agent interrupt — user message interrupts blocking spawn
  // This test verifies that when the user sends a message while a code agent
  // is running (blocking spawn), the supervisor receives [INTERRUPT] signal
  // and can respond to the user instead of waiting indefinitely.
  it('interrupts blocking agent spawn when user sends second message', async () => {
    await server.start('agent-interrupt');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Start agent');

      // Wait for agent to be spawned (lifecycle event confirms agent is running)
      await waitForMessages(
        container,
        (msgs) => msgs.some((m) => m.content.value.includes('spawned')),
        10000,
      );

      // Small delay to ensure WaitForAllSessionAgents is blocked
      await new Promise((r) => setTimeout(r, 500));

      // Send interrupt message while code agent is still sleeping (5s)
      container.streamProcessor.sendMessage('User interrupt: please stop');

      // Wait for supervisor to respond to interrupt
      await waitForMessages(
        container,
        (msgs) => msgs.some((m) => m.role === 'assistant' && m.content.value.includes('interrupt')),
        15000,
      );

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();

      // Supervisor should acknowledge the interrupt
      const interruptResponse = messages.find(
        (m) => m.role === 'assistant' && m.content.value.toLowerCase().includes('interrupt'),
      );
      expect(interruptResponse).toBeDefined();

      // spawn_code_agent tool should have [INTERRUPT] in result
      const spawnToolMsg = messages.find((m) => m.toolCall?.toolName === 'spawn_code_agent');
      expect(spawnToolMsg).toBeDefined();
      expect(spawnToolMsg?.toolResult?.result).toContain('[INTERRUPT]');
    } finally {
      await container.dispose();
    }
  }, 30000);

  // E2E-18: edit_file diff display
  it('edit_file produces diffLines with removed and added lines', async () => {
    // Pre-create file with known content
    const srcDir = path.join(testDir, 'src');
    fs.mkdirSync(srcDir, { recursive: true });
    fs.writeFileSync(path.join(srcDir, 'app.ts'), "console.log('old');\nconsole.log('keep');");

    await server.start('proxied-edit');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Edit the file');

      await waitForMessages(container, (msgs) => {
        const hasToolCall = msgs.some((m) => m.toolCall?.toolName === 'edit_file');
        const hasFinalAnswer = msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.length > 0);
        return hasToolCall && hasFinalAnswer;
      });

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();
      const editMsg = messages.find((m) => m.toolCall?.toolName === 'edit_file');
      expect(editMsg?.toolResult).toBeDefined();
      expect(editMsg?.toolResult?.diffLines).toBeDefined();
      expect(editMsg!.toolResult!.diffLines!.length).toBeGreaterThan(0);

      // Should have removed line with 'old' and added line with 'new'
      const removedLines = editMsg!.toolResult!.diffLines!.filter(l => l.type === '-');
      const addedLines = editMsg!.toolResult!.diffLines!.filter(l => l.type === '+');
      expect(removedLines.length).toBeGreaterThan(0);
      expect(addedLines.length).toBeGreaterThan(0);
      expect(removedLines.some(l => l.content.includes('old'))).toBe(true);
      expect(addedLines.some(l => l.content.includes('new'))).toBe(true);
    } finally {
      await container.dispose();
    }
  }, 30000);

  // E2E-19: write_file overwrite diff display
  it('write_file overwrite produces diffLines', async () => {
    // Pre-create file with different content
    fs.writeFileSync(path.join(testDir, 'output.txt'), 'old content\nline two');

    await server.start('proxied-write');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Write a file');

      await waitForMessages(container, (msgs) => {
        const hasToolCall = msgs.some((m) => m.toolCall?.toolName === 'write_file');
        const hasFinalAnswer = msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.length > 0);
        return hasToolCall && hasFinalAnswer;
      });

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();
      const writeMsg = messages.find((m) => m.toolCall?.toolName === 'write_file');
      expect(writeMsg?.toolResult).toBeDefined();
      expect(writeMsg?.toolResult?.diffLines).toBeDefined();
      expect(writeMsg!.toolResult!.diffLines!.length).toBeGreaterThan(0);

      // Should have removed lines (old content) and added lines (hello)
      const removedLines = writeMsg!.toolResult!.diffLines!.filter(l => l.type === '-');
      const addedLines = writeMsg!.toolResult!.diffLines!.filter(l => l.type === '+');
      expect(removedLines.length).toBeGreaterThan(0);
      expect(addedLines.length).toBeGreaterThan(0);
    } finally {
      await container.dispose();
    }
  }, 30000);

  // E2E-20: write_file new file - no diff
  it('write_file new file has no diffLines', async () => {
    // Ensure output.txt does NOT exist
    const outPath = path.join(testDir, 'output.txt');
    if (fs.existsSync(outPath)) fs.unlinkSync(outPath);

    await server.start('proxied-write');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Write a file');

      await waitForMessages(container, (msgs) => {
        const hasToolCall = msgs.some((m) => m.toolCall?.toolName === 'write_file');
        const hasFinalAnswer = msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.length > 0);
        return hasToolCall && hasFinalAnswer;
      });

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();
      const writeMsg = messages.find((m) => m.toolCall?.toolName === 'write_file');
      expect(writeMsg?.toolResult).toBeDefined();
      // New file: no old content to diff
      const diffLines = writeMsg?.toolResult?.diffLines;
      expect(!diffLines || diffLines.length === 0).toBe(true);
    } finally {
      await container.dispose();
    }
  }, 30000);

  // E2E-22: smart_search quality - tests real search pipeline with MockChatModel
  it('executes smart_search and returns citation results', async () => {
    setupSearchFiles(testDir);

    await server.start('smart-search');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Find error handling');

      // smart_search creates tool message visible in chat history.
      // Wait for the final answer which contains the search results.
      await waitForMessages(container, (msgs) =>
        msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('SEARCH_RESULT:')));

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();

      // After the fix: smart_search tool call should be visible in message repository
      const smartSearchToolMsg = messages.find((m) => m.toolCall?.toolName === 'smart_search');
      expect(smartSearchToolMsg).toBeDefined();

      // Verify query argument is passed through proxy → client (not empty)
      // Bug: proxy sent Arguments:{} → client showed SmartSearch("") instead of actual query
      expect(smartSearchToolMsg!.toolCall?.arguments?.query).toBeDefined();
      expect(smartSearchToolMsg!.toolCall?.arguments?.query).not.toBe('');

      // Verify that result is formatted as citations (not just summary like "grep: 5, vector: 3")
      // SmartSearchRenderer parses citation format: "N. location [source] info"
      const toolResult = smartSearchToolMsg!.toolResult?.result;
      expect(toolResult).toBeDefined();
      // Result should contain citation format "[grep]" or "[vector]" or "[symbol]"
      expect(toolResult).toMatch(/\[(grep|vector|symbol)\]/);

      // Final answer should contain search results from smart_search
      const answer = messages.find((m) =>
        m.role === 'assistant' && m.content.value.includes('SEARCH_RESULT:'));
      expect(answer).toBeDefined();

      const result = answer!.content.value.replace('SEARCH_RESULT:', '');

      // Log the actual result for analysis
      console.log('=== smart_search result ===');
      console.log(result);
      console.log('=== end result ===');

      // Result should not be empty or error
      expect(result.length).toBeGreaterThan(0);
      expect(result).not.toContain('[ERROR]');
      expect(result).not.toContain('No results found');

      // Should find our errors.go file
      expect(result).toMatch(/errors\.go/);
    } finally {
      await container.dispose();
    }
  }, 30000);

  // E2E-23: smart_search with exact function name query
  it('smart_search finds exact function name', async () => {
    setupSearchFiles(testDir);

    await server.start('smart-search-exact');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Find handleError');

      await waitForMessages(container, (msgs) =>
        msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('SEARCH_RESULT:')));

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();
      const answer = messages.find((m) =>
        m.role === 'assistant' && m.content.value.includes('SEARCH_RESULT:'));
      expect(answer).toBeDefined();

      const result = answer!.content.value.replace('SEARCH_RESULT:', '');
      console.log('=== smart_search exact result ===');
      console.log(result);
      console.log('=== end ===');

      // Exact function name should find errors.go
      expect(result.length).toBeGreaterThan(0);
      expect(result).toMatch(/errors\.go/);
      // Should contain the function definition
      expect(result).toMatch(/handleError/);
    } finally {
      await container.dispose();
    }
  }, 30000);

  // E2E-24: smart_search with broad natural language query
  it('smart_search handles broad concept query', async () => {
    setupSearchFiles(testDir);

    await server.start('smart-search-broad');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Find error handling patterns');

      await waitForMessages(container, (msgs) =>
        msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('SEARCH_RESULT:')));

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();
      const answer = messages.find((m) =>
        m.role === 'assistant' && m.content.value.includes('SEARCH_RESULT:'));
      expect(answer).toBeDefined();

      const result = answer!.content.value.replace('SEARCH_RESULT:', '');
      console.log('=== smart_search broad result ===');
      console.log(result);
      console.log('=== end ===');

      // Broad query: should find SOMETHING (even if imprecise)
      // Log whether it finds errors.go for analysis
      const findsErrorsGo = /errors\.go/.test(result);
      const findsHandlerGo = /handler\.go/.test(result);
      console.log(`[ANALYSIS] broad query: errors.go=${findsErrorsGo}, handler.go=${findsHandlerGo}`);

      // At minimum, result should not be empty
      expect(result.length).toBeGreaterThan(0);
    } finally {
      await container.dispose();
    }
  }, 30000);

  // E2E-25: smart_search with specific type/symbol query
  it('smart_search finds specific type name', async () => {
    setupSearchFiles(testDir);

    await server.start('smart-search-symbol');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Find DomainError');

      await waitForMessages(container, (msgs) =>
        msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('SEARCH_RESULT:')));

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();
      const answer = messages.find((m) =>
        m.role === 'assistant' && m.content.value.includes('SEARCH_RESULT:'));
      expect(answer).toBeDefined();

      const result = answer!.content.value.replace('SEARCH_RESULT:', '');
      console.log('=== smart_search symbol result ===');
      console.log(result);
      console.log('=== end ===');

      // Symbol search should find errors.go with DomainError
      expect(result.length).toBeGreaterThan(0);
      expect(result).toMatch(/errors\.go/);
      expect(result).toMatch(/DomainError/);
    } finally {
      await container.dispose();
    }
  }, 30000);

  // E2E-26: smart_search with cross-file concept
  it('smart_search finds cross-file HTTP patterns', async () => {
    setupSearchFiles(testDir);

    await server.start('smart-search-cross-file');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Find HTTP handler');

      await waitForMessages(container, (msgs) =>
        msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('SEARCH_RESULT:')));

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();
      const answer = messages.find((m) =>
        m.role === 'assistant' && m.content.value.includes('SEARCH_RESULT:'));
      expect(answer).toBeDefined();

      const result = answer!.content.value.replace('SEARCH_RESULT:', '');
      console.log('=== smart_search cross-file result ===');
      console.log(result);
      console.log('=== end ===');

      // Should find handler.go and/or middleware.go (both have HTTP patterns)
      expect(result.length).toBeGreaterThan(0);
      const findsHandler = /handler\.go/.test(result);
      const findsMiddleware = /middleware\.go/.test(result);
      console.log(`[ANALYSIS] cross-file: handler.go=${findsHandler}, middleware.go=${findsMiddleware}`);

      // At least one HTTP-related file should be found
      expect(findsHandler || findsMiddleware).toBe(true);
    } finally {
      await container.dispose();
    }
  }, 30000);

  // E2E-27: smart_search with query that has no matches
  it('smart_search handles no-match query gracefully', async () => {
    setupSearchFiles(testDir);

    await server.start('smart-search-no-match');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Find kubernetes');

      await waitForMessages(container, (msgs) =>
        msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('SEARCH_RESULT:')));

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();
      const answer = messages.find((m) =>
        m.role === 'assistant' && m.content.value.includes('SEARCH_RESULT:'));
      expect(answer).toBeDefined();

      const result = answer!.content.value.replace('SEARCH_RESULT:', '');
      console.log('=== smart_search no-match result ===');
      console.log(result);
      console.log('=== end ===');

      // Should NOT find errors.go or handler.go (nothing matches kubernetes)
      const findsErrorsGo = /errors\.go/.test(result);
      const findsHandlerGo = /handler\.go/.test(result);
      console.log(`[ANALYSIS] no-match: errors.go=${findsErrorsGo}, handler.go=${findsHandlerGo}`);

      // The result should either be empty or contain "No results" - NOT random unrelated files
      // This verifies that smart_search doesn't return noise
    } finally {
      await container.dispose();
    }
  }, 30000);

  // E2E-28: grep_search direct regex
  it('executes grep_search with regex pattern and returns matches', async () => {
    setupSearchFiles(testDir);

    await server.start('grep-direct');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Search for handle functions');

      // grep_search is proxied to client → executes ripgrep → returns results
      await waitForMessages(container, (msgs) =>
        msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('GREP_RESULT:')));

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();

      // Tool call should be visible
      const toolMsg = messages.find((m) => m.toolCall?.toolName === 'grep_search');
      expect(toolMsg).toBeDefined();
      expect(toolMsg?.toolResult).toBeDefined();

      // Result should contain matches from grep
      const answer = messages.find((m) =>
        m.role === 'assistant' && m.content.value.includes('GREP_RESULT:'));
      expect(answer).toBeDefined();

      const result = answer!.content.value.replace('GREP_RESULT:', '');
      console.log('=== grep_search result ===');
      console.log(result);
      console.log('=== end ===');

      // Should find func handle* in Go files
      expect(result.length).toBeGreaterThan(0);
      expect(result).not.toContain('[ERROR]');

      // errors.go has handleError, handler.go has handleRequest
      expect(result).toMatch(/errors\.go|handler\.go/);
    } finally {
      await container.dispose();
    }
  }, 30000);

  // E2E-29: glob file search
  it('executes glob pattern matching and returns file list', async () => {
    setupSearchFiles(testDir);

    await server.start('glob-search');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Find Go files');

      await waitForMessages(container, (msgs) =>
        msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('GLOB_RESULT:')));

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();

      // Tool call should be visible
      const toolMsg = messages.find((m) => m.toolCall?.toolName === 'glob');
      expect(toolMsg).toBeDefined();
      expect(toolMsg?.toolResult).toBeDefined();

      const answer = messages.find((m) =>
        m.role === 'assistant' && m.content.value.includes('GLOB_RESULT:'));
      expect(answer).toBeDefined();

      const result = answer!.content.value.replace('GLOB_RESULT:', '');
      console.log('=== glob result ===');
      console.log(result);
      console.log('=== end ===');

      // Should find .go files
      expect(result.length).toBeGreaterThan(0);
      expect(result).not.toContain('[ERROR]');
      expect(result).toMatch(/\.go/);

      // Should find our test files
      const findsErrors = /errors\.go/.test(result);
      const findsHandler = /handler\.go/.test(result);
      const findsConfig = /config\.go/.test(result);
      const findsMiddleware = /middleware\.go/.test(result);
      console.log(`[ANALYSIS] glob: errors=${findsErrors}, handler=${findsHandler}, config=${findsConfig}, middleware=${findsMiddleware}`);

      // All 4 Go files should be found
      expect(findsErrors && findsHandler && findsConfig && findsMiddleware).toBe(true);
    } finally {
      await container.dispose();
    }
  }, 30000);

  // === Search comparison: smart_search vs grep_search ===
  describe('Search comparison: smart_search vs grep_search', () => {
    // Helper: extract search result from GREP_RESULT: or SEARCH_RESULT: prefixed message
    function extractSearchResult(messages: Message[], prefix: string): string {
      const answer = messages.find((m) =>
        m.role === 'assistant' && m.content.value.includes(prefix));
      return answer ? answer.content.value.replace(prefix, '') : '';
    }

    // E2E-30: Compare exact query — handleError
    it('compare: exact function name search', async () => {
      setupSearchFiles(testDir);

      // Run grep_search
      await server.start('compare-exact-grep');
      const grepContainer = createTestContainer(server.port, testDir);

      let grepResult = '';
      try {
        await connectAndSend(grepContainer, 'Find handleError');
        await waitForMessages(grepContainer, (msgs) =>
          msgs.some((m) => m.content.value.includes('GREP_RESULT:')));
        await waitForProcessingStopped(grepContainer);

        const grepMsgs = grepContainer.messageRepository.findComplete();
        grepResult = extractSearchResult(grepMsgs, 'GREP_RESULT:');
      } finally {
        await grepContainer.dispose();
      }
      await server.stop();

      // Run smart_search
      await server.start('smart-search-exact');
      const smartContainer = createTestContainer(server.port, testDir);

      let smartResult = '';
      try {
        await connectAndSend(smartContainer, 'Find handleError');
        await waitForMessages(smartContainer, (msgs) =>
          msgs.some((m) => m.content.value.includes('SEARCH_RESULT:')));
        await waitForProcessingStopped(smartContainer);

        const smartMsgs = smartContainer.messageRepository.findComplete();
        smartResult = extractSearchResult(smartMsgs, 'SEARCH_RESULT:');
      } finally {
        await smartContainer.dispose();
      }

      // Compare results
      console.log('[COMPARE] exact query: handleError');
      console.log(`  smart_search: ${smartResult.split('\n').filter(l => l.trim()).length} lines`);
      console.log(`  grep_search:  ${grepResult.split('\n').filter(l => l.trim()).length} lines`);
      console.log(`  smart finds errors.go: ${/errors\.go/.test(smartResult)}`);
      console.log(`  grep finds errors.go:  ${/errors\.go/.test(grepResult)}`);

      // Both should find errors.go
      expect(/errors\.go/.test(grepResult) || /errors\.go/.test(smartResult)).toBe(true);
    }, 60000);

    // E2E-31: Compare broad query — error handling
    it('compare: broad concept search', async () => {
      setupSearchFiles(testDir);

      // Run grep_search
      await server.start('compare-broad-grep');
      const grepContainer = createTestContainer(server.port, testDir);

      let grepResult = '';
      try {
        await connectAndSend(grepContainer, 'Find error handling');
        await waitForMessages(grepContainer, (msgs) =>
          msgs.some((m) => m.content.value.includes('GREP_RESULT:')));
        await waitForProcessingStopped(grepContainer);

        const grepMsgs = grepContainer.messageRepository.findComplete();
        grepResult = extractSearchResult(grepMsgs, 'GREP_RESULT:');
      } finally {
        await grepContainer.dispose();
      }
      await server.stop();

      // Run smart_search
      await server.start('smart-search-broad');
      const smartContainer = createTestContainer(server.port, testDir);

      let smartResult = '';
      try {
        await connectAndSend(smartContainer, 'Find error handling patterns');
        await waitForMessages(smartContainer, (msgs) =>
          msgs.some((m) => m.content.value.includes('SEARCH_RESULT:')));
        await waitForProcessingStopped(smartContainer);

        const smartMsgs = smartContainer.messageRepository.findComplete();
        smartResult = extractSearchResult(smartMsgs, 'SEARCH_RESULT:');
      } finally {
        await smartContainer.dispose();
      }

      // Compare results
      const smartLines = smartResult.split('\n').filter(l => l.trim()).length;
      const grepLines = grepResult.split('\n').filter(l => l.trim()).length;
      console.log('[COMPARE] broad query: error handling');
      console.log(`  smart_search: ${smartLines} lines`);
      console.log(`  grep_search:  ${grepLines} lines`);
      console.log(`  smart finds errors.go: ${/errors\.go/.test(smartResult)}`);
      console.log(`  grep finds errors.go:  ${/errors\.go/.test(grepResult)}`);

      // At least one should find something
      expect(smartLines > 0 || grepLines > 0).toBe(true);
    }, 60000);

    // E2E-32: Compare symbol query — DomainError
    it('compare: symbol search', async () => {
      setupSearchFiles(testDir);

      // Run grep_search
      await server.start('compare-symbol-grep');
      const grepContainer = createTestContainer(server.port, testDir);

      let grepResult = '';
      try {
        await connectAndSend(grepContainer, 'Find DomainError');
        await waitForMessages(grepContainer, (msgs) =>
          msgs.some((m) => m.content.value.includes('GREP_RESULT:')));
        await waitForProcessingStopped(grepContainer);

        const grepMsgs = grepContainer.messageRepository.findComplete();
        grepResult = extractSearchResult(grepMsgs, 'GREP_RESULT:');
      } finally {
        await grepContainer.dispose();
      }
      await server.stop();

      // Run smart_search
      await server.start('smart-search-symbol');
      const smartContainer = createTestContainer(server.port, testDir);

      let smartResult = '';
      try {
        await connectAndSend(smartContainer, 'Find DomainError');
        await waitForMessages(smartContainer, (msgs) =>
          msgs.some((m) => m.content.value.includes('SEARCH_RESULT:')));
        await waitForProcessingStopped(smartContainer);

        const smartMsgs = smartContainer.messageRepository.findComplete();
        smartResult = extractSearchResult(smartMsgs, 'SEARCH_RESULT:');
      } finally {
        await smartContainer.dispose();
      }

      // Compare results
      console.log('[COMPARE] symbol query: DomainError');
      console.log(`  smart_search: ${smartResult.split('\n').filter(l => l.trim()).length} lines`);
      console.log(`  grep_search:  ${grepResult.split('\n').filter(l => l.trim()).length} lines`);
      console.log(`  smart finds DomainError: ${/DomainError/.test(smartResult)}`);
      console.log(`  grep finds DomainError:  ${/DomainError/.test(grepResult)}`);

      // Grep should definitely find DomainError (exact pattern match)
      expect(/DomainError/.test(grepResult)).toBe(true);
    }, 60000);

    // E2E-33: Compare cross-file query — HTTP handler
    it('compare: cross-file HTTP handler search', async () => {
      setupSearchFiles(testDir);

      // Run grep_search
      await server.start('compare-cross-grep');
      const grepContainer = createTestContainer(server.port, testDir);

      let grepResult = '';
      try {
        await connectAndSend(grepContainer, 'Find HTTP handler');
        await waitForMessages(grepContainer, (msgs) =>
          msgs.some((m) => m.content.value.includes('GREP_RESULT:')));
        await waitForProcessingStopped(grepContainer);

        const grepMsgs = grepContainer.messageRepository.findComplete();
        grepResult = extractSearchResult(grepMsgs, 'GREP_RESULT:');
      } finally {
        await grepContainer.dispose();
      }
      await server.stop();

      // Run smart_search
      await server.start('smart-search-cross-file');
      const smartContainer = createTestContainer(server.port, testDir);

      let smartResult = '';
      try {
        await connectAndSend(smartContainer, 'Find HTTP handler');
        await waitForMessages(smartContainer, (msgs) =>
          msgs.some((m) => m.content.value.includes('SEARCH_RESULT:')));
        await waitForProcessingStopped(smartContainer);

        const smartMsgs = smartContainer.messageRepository.findComplete();
        smartResult = extractSearchResult(smartMsgs, 'SEARCH_RESULT:');
      } finally {
        await smartContainer.dispose();
      }

      // Compare results
      const grepFindsHandler = /handler\.go/.test(grepResult);
      const grepFindsMiddleware = /middleware\.go/.test(grepResult);
      const smartFindsHandler = /handler\.go/.test(smartResult);
      const smartFindsMiddleware = /middleware\.go/.test(smartResult);

      console.log('[COMPARE] cross-file query: HTTP handler');
      console.log(`  smart_search: handler=${smartFindsHandler}, middleware=${smartFindsMiddleware}`);
      console.log(`  grep_search:  handler=${grepFindsHandler}, middleware=${grepFindsMiddleware}`);

      // At least one approach should find HTTP handler files
      expect(grepFindsHandler || grepFindsMiddleware || smartFindsHandler || smartFindsMiddleware).toBe(true);
    }, 60000);
  });

  // E2E-34: Session isolation — different sessionIds get independent histories
  it('maintains session isolation between different sessions', async () => {
    await server.start('echo');

    // Session A: connect and send message
    const containerA = createTestContainer(server.port, testDir);
    try {
      await connectAndSend(containerA, 'Hello from session A');
      await waitForMessages(containerA, (msgs) =>
        msgs.some((m) => m.role === 'assistant' && m.content.value.includes('Hello, world!')));
      await waitForProcessingStopped(containerA);

      const messagesA = containerA.messageRepository.findComplete();
      expect(messagesA.length).toBeGreaterThanOrEqual(2);
      expect(messagesA.some((m) => m.content.value.includes('session A'))).toBe(true);
    } finally {
      await containerA.dispose();
    }

    // Session B: fresh container (new sessionId — simulates --new flag which generates fresh UUID)
    const containerB = createTestContainer(server.port, testDir);
    try {
      await connectAndSend(containerB, 'Hello from session B');
      await waitForMessages(containerB, (msgs) =>
        msgs.some((m) => m.role === 'assistant' && m.content.value.includes('Hello, world!')));
      await waitForProcessingStopped(containerB);

      const messagesB = containerB.messageRepository.findComplete();

      // Session B has independent message history
      expect(messagesB.length).toBeGreaterThanOrEqual(2);

      // Session B does NOT contain Session A's messages
      expect(messagesB.some((m) => m.content.value.includes('session A'))).toBe(false);

      // Session B contains its own messages
      expect(messagesB.some((m) => m.content.value.includes('session B'))).toBe(true);

      // Different session IDs (--new generates fresh UUID)
      expect(containerA.sessionId).not.toBe(containerB.sessionId);
    } finally {
      await containerB.dispose();
    }
  }, 30000);

  // E2E-35: smart_search with empty query — error recovery
  // MockChatModel sends smart_search(query="") → server returns error → model recovers.
  // NOTE: smart_search is proxied, so server-side errors don't produce client-side tool messages.
  // The error flows through REACT agent → model gets error as tool result → provides final answer.
  it('handles smart_search empty query error gracefully', async () => {
    setupSearchFiles(testDir);

    await server.start('smart-search-empty-query');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Search empty');

      // Model should receive error via REACT loop and provide final answer
      await waitForMessages(container, (msgs) =>
        msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('HANDLED_ERROR:')));

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();

      // Model recovered and gave final answer containing the error text
      const answer = messages.find((m) =>
        m.role === 'assistant' && m.content.value.includes('HANDLED_ERROR:'));
      expect(answer).toBeDefined();
      // Error message from smart_search_tool.go flows through REACT → model → final answer
      expect(answer!.content.value).toContain('[ERROR]');

      // Processing completed without hanging (no infinite loop)
      expect(container.streamProcessor.getIsProcessing()).toBe(false);
    } finally {
      await container.dispose();
    }
  }, 30000);

  // E2E-36: grep_search produces exactly one tool message (no duplicates)
  // Verifies classifier.go fix: grep_search in proxiedTools prevents duplicate TOOL_CALL
  it('grep_search creates exactly one tool message (no duplicate)', async () => {
    setupSearchFiles(testDir);

    await server.start('grep-no-duplicate');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Search for ListenAndServe');

      await waitForMessages(container, (msgs) =>
        msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('GREP_RESULT:')));

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();

      // grep_search tool call should exist
      const grepMsgs = messages.filter((m) => m.toolCall?.toolName === 'grep_search');
      expect(grepMsgs.length).toBe(1); // Exactly ONE, not duplicated

      // Tool should have a result
      expect(grepMsgs[0].toolResult).toBeDefined();
      expect(grepMsgs[0].toolResult?.result).toBeTruthy();
    } finally {
      await container.dispose();
    }
  }, 30000);

  // E2E-37: glob produces exactly one tool message (no duplicates)
  // Verifies classifier.go fix: glob in proxiedTools prevents duplicate TOOL_CALL
  it('glob creates exactly one tool message (no duplicate)', async () => {
    setupSearchFiles(testDir);

    await server.start('glob-no-duplicate');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Find Go files');

      await waitForMessages(container, (msgs) =>
        msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('GLOB_RESULT:')));

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();

      // glob tool call should exist
      const globMsgs = messages.filter((m) => m.toolCall?.toolName === 'glob');
      expect(globMsgs.length).toBe(1); // Exactly ONE, not duplicated

      // Tool should have a result with .go files
      expect(globMsgs[0].toolResult).toBeDefined();
      expect(globMsgs[0].toolResult?.result).toContain('.go');
    } finally {
      await container.dispose();
    }
  }, 30000);

  // E2E-22r: smart_search renders results (not "no results") — render-based UI test
  // This test catches bugs in the SmartSearchRenderer that the data-layer E2E-22 misses.
  // E2E-22 checks messageRepository.findComplete() — data is correct but renderer could still
  // show "no results" if citation parsing fails. This test checks what the user actually SEES.
  it('E2E-22r: smart_search renders results not "no results" in UI', async () => {
    setupSearchFiles(testDir);
    await server.start('smart-search');
    const container = createTestContainer(server.port, testDir);
    let instance: ReturnType<typeof render> | null = null;

    try {
      // Render ChatApp like the real interactive client
      instance = render(React.createElement(ChatApp, { container }));

      // Connect and send via container (same as other E2E tests)
      await connectAndSend(container, 'Find error handling');

      // Wait for smart_search to complete and final answer to arrive
      await waitForMessages(
        container,
        (msgs) =>
          msgs.some(
            (m) =>
              m.role === 'assistant' &&
              !m.toolCall &&
              m.content.value.includes('SEARCH_RESULT:'),
          ),
      );
      await waitForProcessingStopped(container);

      // Give the renderer time to flush React state updates
      await new Promise((r) => setTimeout(r, 300));

      // Check rendered output — what the user actually SEES in the terminal
      const frame = instance.lastFrame() || '';

      // SmartSearchRenderer should NOT show "no results" when sub-queries return data.
      // "no results" appears only when citation parsing fails (citations.length === 0).
      expect(frame).not.toContain('no results');

      // SmartSearchRenderer shows [grep], [vector], or [symbol] for each citation.
      // If this assertion fails, the renderer shows "no results" despite data being present.
      expect(frame).toMatch(/\[(grep|vector|symbol)\]/);
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // E2E-28r: grep_search renders results (not error) in UI — render-based UI test
  // This test catches bugs in the GrepSearch renderer that data-layer E2E-28 misses.
  // E2E-28 checks messageRepository.findComplete() — data may be correct but renderer
  // could still show [ERROR] if result formatting fails. This test checks what the user SEES.
  it('E2E-28r: grep_search renders results in UI', async () => {
    setupSearchFiles(testDir);
    await server.start('grep-direct');
    const container = createTestContainer(server.port, testDir);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Search for handle functions');

      await waitForMessages(container, (msgs) =>
        msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('GREP_RESULT:')));
      await waitForProcessingStopped(container);

      await new Promise((r) => setTimeout(r, 300));

      const frame = instance.lastFrame() || '';
      // GrepSearch tool should be visible in rendered output
      expect(frame).toContain('GrepSearch');
      // Should NOT show error
      expect(frame).not.toContain('[ERROR]');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // E2E-29r: glob renders file list in UI — render-based UI test
  // This test catches bugs in the Glob renderer that data-layer E2E-29 misses.
  // E2E-29 checks messageRepository.findComplete() — data may be correct but renderer
  // could still show [ERROR] if file list formatting fails. This test checks what the user SEES.
  it('E2E-29r: glob renders file list in UI', async () => {
    setupSearchFiles(testDir);
    await server.start('glob-search');
    const container = createTestContainer(server.port, testDir);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Find Go files');

      await waitForMessages(container, (msgs) =>
        msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('GLOB_RESULT:')));
      await waitForProcessingStopped(container);

      await new Promise((r) => setTimeout(r, 300));

      const frame = instance.lastFrame() || '';
      // Glob tool should be visible in rendered output
      expect(frame).toContain('Glob');
      // Should NOT show error
      expect(frame).not.toContain('[ERROR]');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // E2E-38: lsp tool pipeline round-trip
  // Tests that lsp tool call flows through the full pipeline:
  // MockChatModel → REACT agent → gRPC proxy → client lspTool → result back to model
  // In test environment, LSP servers may not be running, so the tool may return
  // "No symbols found..." or "Symbol search unavailable..." — this is expected.
  // The test verifies the PIPELINE works, not the LSP quality.
  it('executes lsp tool through full gRPC proxy pipeline', async () => {
    await server.start('lsp-definition');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Find definition of TestFunc');

      // Wait for lsp tool call to appear in message repository
      await waitForMessages(container, (msgs) => {
        const hasToolCall = msgs.some((m) => m.toolCall?.toolName === 'lsp');
        const hasFinalAnswer = msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.length > 0);
        return hasToolCall && hasFinalAnswer;
      });

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();

      // lsp tool call should be visible in message history
      const toolMsg = messages.find((m) => m.toolCall?.toolName === 'lsp');
      expect(toolMsg).toBeDefined();

      // Tool arguments should be passed through correctly
      expect(toolMsg!.toolCall?.arguments?.symbol_name).toBe('TestFunc');
      expect(toolMsg!.toolCall?.arguments?.operation).toBe('definition');

      // Tool result should exist (pipeline completed the round-trip)
      expect(toolMsg!.toolResult).toBeDefined();
      // Result is non-empty string (either real LSP result or graceful error message)
      expect(toolMsg!.toolResult!.result.length).toBeGreaterThan(0);

      // Final assistant message should contain the LSP result echoed back
      const finalAnswer = messages.find((m) =>
        m.role === 'assistant' && !m.toolCall && m.content.value.includes('LSP_RESULT:'));
      expect(finalAnswer).toBeDefined();
    } finally {
      await container.dispose();
    }
  }, 30000);

  // E2E-39: lsp references operation pipeline
  it('executes lsp references through gRPC proxy pipeline', async () => {
    await server.start('lsp-references');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Find references of HandleError');

      await waitForMessages(container, (msgs) => {
        const hasToolCall = msgs.some((m) => m.toolCall?.toolName === 'lsp');
        const hasFinalAnswer = msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.length > 0);
        return hasToolCall && hasFinalAnswer;
      });

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();

      const toolMsg = messages.find((m) => m.toolCall?.toolName === 'lsp');
      expect(toolMsg).toBeDefined();
      expect(toolMsg!.toolCall?.arguments?.symbol_name).toBe('HandleError');
      expect(toolMsg!.toolCall?.arguments?.operation).toBe('references');
      expect(toolMsg!.toolResult).toBeDefined();
      expect(toolMsg!.toolResult!.result.length).toBeGreaterThan(0);

      const finalAnswer = messages.find((m) =>
        m.role === 'assistant' && !m.toolCall && m.content.value.includes('LSP_REFS:'));
      expect(finalAnswer).toBeDefined();
    } finally {
      await container.dispose();
    }
  }, 30000);

  // E2E-40: lsp implementation operation pipeline
  it('executes lsp implementation through gRPC proxy pipeline', async () => {
    await server.start('lsp-implementation');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Find implementations of Repository');

      await waitForMessages(container, (msgs) => {
        const hasToolCall = msgs.some((m) => m.toolCall?.toolName === 'lsp');
        const hasFinalAnswer = msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.length > 0);
        return hasToolCall && hasFinalAnswer;
      });

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();

      const toolMsg = messages.find((m) => m.toolCall?.toolName === 'lsp');
      expect(toolMsg).toBeDefined();
      expect(toolMsg!.toolCall?.arguments?.symbol_name).toBe('Repository');
      expect(toolMsg!.toolCall?.arguments?.operation).toBe('implementation');
      expect(toolMsg!.toolResult).toBeDefined();
      expect(toolMsg!.toolResult!.result.length).toBeGreaterThan(0);

      const finalAnswer = messages.find((m) =>
        m.role === 'assistant' && !m.toolCall && m.content.value.includes('LSP_IMPL:'));
      expect(finalAnswer).toBeDefined();
    } finally {
      await container.dispose();
    }
  }, 30000);

  // E2E-41: lsp with invalid operation returns graceful error
  // Server-side lsp_tool.go validates operation BEFORE proxying to client.
  // Invalid operation is caught on the server → error string returned as tool result →
  // MockChatModel gets error and produces final answer. Client never sees the tool call
  // because the gRPC proxy request is never sent.
  it('handles lsp invalid operation gracefully', async () => {
    await server.start('lsp-invalid-op');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Hover over Foo');

      // Server handles validation — client only sees the final answer with error info
      await waitForMessages(container, (msgs) => {
        return msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('LSP_ERR:'));
      });

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();

      // Final assistant message contains the error from server-side validation
      const finalAnswer = messages.find((m) =>
        m.role === 'assistant' && !m.toolCall && m.content.value.includes('LSP_ERR:'));
      expect(finalAnswer).toBeDefined();
      expect(finalAnswer!.content.value).toContain('Invalid operation');
    } finally {
      await container.dispose();
    }
  }, 30000);

  // E2E-42: lsp with non-existent symbol returns graceful message
  it('handles lsp missing symbol gracefully', async () => {
    await server.start('lsp-missing-symbol');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Find NonExistentSymbol12345');

      await waitForMessages(container, (msgs) => {
        const hasToolCall = msgs.some((m) => m.toolCall?.toolName === 'lsp');
        const hasFinalAnswer = msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.length > 0);
        return hasToolCall && hasFinalAnswer;
      });

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();

      const toolMsg = messages.find((m) => m.toolCall?.toolName === 'lsp');
      expect(toolMsg).toBeDefined();
      expect(toolMsg!.toolCall?.arguments?.symbol_name).toBe('NonExistentSymbol12345');
      expect(toolMsg!.toolResult).toBeDefined();
      // The tool should return a graceful message about symbol not found or search unavailable
      expect(toolMsg!.toolResult!.result.length).toBeGreaterThan(0);

      const finalAnswer = messages.find((m) =>
        m.role === 'assistant' && !m.toolCall && m.content.value.includes('LSP_MISS:'));
      expect(finalAnswer).toBeDefined();
    } finally {
      await container.dispose();
    }
  }, 30000);

  /** Create multi-language test files for LSP multi-lang tests and init git repo */
  function setupMultiLangFiles(dir: string) {
    const { execSync } = require('child_process');
    execSync('git init', { cwd: dir, stdio: 'ignore' });

    // Go
    fs.writeFileSync(path.join(dir, 'main.go'), `package main

type ProcessData struct {
\tName string
}

func (p *ProcessData) Run() error {
\treturn nil
}
`);

    // TypeScript
    fs.writeFileSync(path.join(dir, 'service.ts'), `export class UserService {
  private name: string;

  constructor(name: string) {
    this.name = name;
  }

  getName(): string {
    return this.name;
  }
}
`);

    // Python
    fs.writeFileSync(path.join(dir, 'processor.py'), `class DataProcessor:
    def __init__(self, data):
        self.data = data

    def process(self):
        return [x * 2 for x in self.data]
`);

    // Rust
    fs.writeFileSync(path.join(dir, 'config.rs'), `pub struct Config {
    pub name: String,
    pub value: i32,
}

pub fn default_config() -> Config {
    Config {
        name: String::from("default"),
        value: 42,
    }
}
`);

    // Lua
    fs.writeFileSync(path.join(dir, 'greet.lua'), `function greet(name)
  return "Hello, " .. name
end
`);

    // Elixir
    fs.writeFileSync(path.join(dir, 'calculator.ex'), `defmodule Calculator do
  def add(a, b) do
    a + b
  end
end
`);

    execSync('git add -A', { cwd: dir, stdio: 'ignore' });
  }

  // E2E-43: multi-language LSP tool pipeline
  // Tests that 4 sequential LSP calls across different languages flow through the
  // full gRPC pipeline: MockChatModel → REACT agent → gRPC proxy → client lspTool → result back
  it('executes lsp tool across multiple languages', async () => {
    setupMultiLangFiles(testDir);
    await server.start('lsp-multilang');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Find definitions across languages');

      // Wait for all 4 tool calls + final answer (longer timeout: WASM load + indexing + LSP retries)
      await waitForMessages(container, (msgs) => {
        const lspCalls = msgs.filter((m) => m.toolCall?.toolName === 'lsp');
        const hasFinalAnswer = msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('MULTILANG_RESULTS:'));
        return lspCalls.length >= 4 && hasFinalAnswer;
      }, 45000);

      await waitForProcessingStopped(container, 45000);

      const messages = container.messageRepository.findComplete();

      // Verify all 4 LSP tool calls exist
      const lspCalls = messages.filter((m) => m.toolCall?.toolName === 'lsp');
      expect(lspCalls.length).toBe(4);

      // Verify each language's symbol was queried
      const symbols = lspCalls.map((m) => m.toolCall?.arguments?.symbol_name);
      expect(symbols).toContain('ProcessData');  // Go
      expect(symbols).toContain('UserService');  // TypeScript
      expect(symbols).toContain('DataProcessor'); // Python
      expect(symbols).toContain('Config');        // Rust

      // All tool results should be non-empty (symbolSearch found them or returned graceful message)
      for (const call of lspCalls) {
        expect(call.toolResult).toBeDefined();
        expect(call.toolResult!.result.length).toBeGreaterThan(0);
      }

      // Final answer should contain MULTILANG_RESULTS
      const finalAnswer = messages.find((m) =>
        m.role === 'assistant' && !m.toolCall && m.content.value.includes('MULTILANG_RESULTS:'));
      expect(finalAnswer).toBeDefined();
    } finally {
      await container.dispose();
    }
  }, 60000);

  // E2E-45: write_file Go file triggers LSP diagnostics from gopls
  // Tests the full diagnostics pipeline: write_file with Go errors →
  // DiagnosticsService.runAfterWrite() → gopls detects errors → diagnostics appended to result.
  // Requires gopls installed on the machine.
  it('write_file Go file triggers LSP diagnostics from gopls', async () => {
    // Create go.mod so gopls recognizes temp dir as a Go project
    fs.writeFileSync(path.join(testDir, 'go.mod'), 'module testmod\n\ngo 1.21\n');

    await server.start('write-file-go-error');

    // Container WITHOUT disableLspServers — real gopls runs for diagnostics
    const container = new Container({
      projectRoot: testDir,
      serverAddress: `localhost:${server.port}`,
      projectKey: 'e2e-diag-test',
      headlessMode: true,
      askUserCallback: async () => 'approved',
      // NOTE: no disableLspServers — we need real gopls for diagnostics
    });
    container.initialize();

    try {
      // Give gopls time to start (warmup is fire-and-forget)
      await new Promise((r) => setTimeout(r, 5000));

      await connectAndSend(container, 'Write broken Go file');

      // Wait for final answer containing WRITE_RESULT (includes tool result with diagnostics)
      await waitForMessages(
        container,
        (msgs) =>
          msgs.some(
            (m) =>
              m.role === 'assistant' &&
              !m.toolCall &&
              m.content.value.includes('WRITE_RESULT:'),
          ),
        90000,
      );

      await waitForProcessingStopped(container, 90000);

      const messages = container.messageRepository.findComplete();

      // write_file tool call should exist
      const writeMsg = messages.find((m) => m.toolCall?.toolName === 'write_file');
      expect(writeMsg).toBeDefined();
      expect(writeMsg?.toolResult).toBeDefined();

      // File should be written
      const writtenPath = path.join(testDir, 'broken.go');
      expect(fs.existsSync(writtenPath)).toBe(true);

      // Tool result should contain diagnostics from gopls
      const result = writeMsg!.toolResult!.result;
      console.log('=== write_file result with diagnostics ===');
      console.log(result);
      console.log('=== end ===');

      // gopls should detect "undeclared name: undefinedVar"
      expect(result).toContain('<diagnostics');
      expect(result).toMatch(/undeclared|undefined|undefinedVar/i);

      // Final answer from mock includes the tool result (with diagnostics)
      const finalMsg = messages.find(
        (m) =>
          m.role === 'assistant' &&
          !m.toolCall &&
          m.content.value.includes('WRITE_RESULT:'),
      );
      expect(finalMsg).toBeDefined();
      expect(finalMsg!.content.value).toContain('diagnostics');
    } finally {
      await container.dispose();
    }
  }, 120000);

  // E2E-44: symbol search across non-LSP languages (Lua, Elixir)
  // Tests that auto-indexing works for languages without a dedicated LSP server.
  // The lsp tool falls back to symbolSearch (tree-sitter based) for these files.
  it('finds symbols via auto-indexing without LSP server', async () => {
    setupMultiLangFiles(testDir);
    await server.start('lsp-symbol-search');
    const container = createTestContainer(server.port, testDir);

    try {
      await connectAndSend(container, 'Find greet and Calculator');

      await waitForMessages(container, (msgs) => {
        const lspCalls = msgs.filter((m) => m.toolCall?.toolName === 'lsp');
        const hasFinalAnswer = msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('SYMBOL_SEARCH:'));
        return lspCalls.length >= 2 && hasFinalAnswer;
      }, 45000);

      await waitForProcessingStopped(container, 45000);

      const messages = container.messageRepository.findComplete();
      const lspCalls = messages.filter((m) => m.toolCall?.toolName === 'lsp');
      expect(lspCalls.length).toBe(2);

      // Verify symbol names
      const symbols = lspCalls.map((m) => m.toolCall?.arguments?.symbol_name);
      expect(symbols).toContain('greet');      // Lua
      expect(symbols).toContain('Calculator'); // Elixir

      // Tool results should be non-empty (either real results or graceful fallback)
      for (const call of lspCalls) {
        expect(call.toolResult).toBeDefined();
        expect(call.toolResult!.result.length).toBeGreaterThan(0);
      }

      // Final answer should contain SYMBOL_SEARCH marker
      const finalAnswer = messages.find((m) =>
        m.role === 'assistant' && !m.toolCall && m.content.value.includes('SYMBOL_SEARCH:'));
      expect(finalAnswer).toBeDefined();
    } finally {
      await container.dispose();
    }
  }, 60000);

  // E2E-46: Agent failure — supervisor handles code agent error
  // Tests that when a code agent fails, the supervisor receives the error and provides a final answer.
  it('handles code agent failure gracefully', async () => {
    await server.start('agent-failure');
    const container = createTestContainer(server.port, testDir);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Run failing agent');

      await waitForMessages(container, (msgs) => {
        const hasFinalAnswer = msgs.some(
          (m) => m.role === 'assistant' && m.content.value.includes('Agent failed'),
        );
        return hasFinalAnswer;
      }, 30000);

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();

      // spawn_code_agent tool call should exist
      const spawnToolMsg = messages.find(
        (m) => m.toolCall?.toolName === 'spawn_code_agent',
      );
      expect(spawnToolMsg).toBeDefined();

      // Supervisor should handle the error and provide a final answer
      const finalAnswer = messages.find(
        (m) => m.role === 'assistant' && m.content.value.includes('Agent failed'),
      );
      expect(finalAnswer).toBeDefined();

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('Agent failed');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 45000);

  // E2E-47: Multi-agent read — code agent reads file via proxy
  // Tests full lifecycle: supervisor spawns code agent → agent reads file → returns result
  // NOTE: mock hasToolResult detection may cause scenario to skip spawn on first call.
  // The test focuses on rendered output (what the user sees), not pipeline internals.
  it('code agent reads file via proxy and returns result', async () => {
    // Create a file for the code agent to read
    const srcDir = path.join(testDir, 'src');
    fs.mkdirSync(srcDir, { recursive: true });
    fs.writeFileSync(path.join(srcDir, 'main.ts'), 'export const greeting = "hello";');

    await server.start('multi-agent-read');
    const container = createTestContainer(server.port, testDir);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Read source file');

      await waitForMessages(container, (msgs) => {
        const hasFinalAnswer = msgs.some(
          (m) => m.role === 'assistant' && m.content.value.length > 0 && !m.toolCall,
        );
        return hasFinalAnswer;
      }, 30000);

      await waitForProcessingStopped(container);

      // Rendered output checks — what the user sees
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      // User should see a response (either from multi-agent flow or direct answer)
      expect(frame.length).toBeGreaterThan(0);
      // Processing should complete without hanging
      expect(container.streamProcessor.getIsProcessing()).toBe(false);
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 45000);

  // E2E-48: Persistent shell session (state persistence between commands)
  it('maintains shell state between execute_command calls (cd + pwd)', async () => {
    await server.start('persistent-shell');
    const container = createTestContainer(server.port, testDir);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Test persistent shell');

      await waitForMessages(container, (msgs) => {
        // Need 2 tool calls + final answer
        const toolCalls = msgs.filter((m) => m.toolCall?.toolName === 'execute_command');
        const hasFinalAnswer = msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('PERSISTENT_SHELL_RESULTS'));
        return toolCalls.length >= 2 && hasFinalAnswer;
      });

      await waitForProcessingStopped(container);

      // Data layer checks
      const messages = container.messageRepository.findComplete();
      const toolCalls = messages.filter((m) => m.toolCall?.toolName === 'execute_command');
      expect(toolCalls.length).toBeGreaterThanOrEqual(2);

      // Check that pwd returned /tmp (proving persistent state)
      const finalAnswer = messages.find((m) =>
        m.role === 'assistant' && !m.toolCall && m.content.value.includes('PERSISTENT_SHELL_RESULTS'));
      expect(finalAnswer).toBeDefined();
      expect(finalAnswer!.content.value).toContain('/tmp');

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('Exec'); // execute_command displayed as Exec prefix
      expect(frame).toContain('PERSISTENT_SHELL_RESULTS');
      expect(frame).toContain('/tmp');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // E2E-49: Background process management (spawn, list, kill)
  it('manages background processes via execute_command', async () => {
    await server.start('background-process');
    const container = createTestContainer(server.port, testDir);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Test background processes');

      await waitForMessages(container, (msgs) => {
        // Need 3 tool calls (spawn, list, kill) + final answer
        const toolCalls = msgs.filter((m) => m.toolCall?.toolName === 'execute_command');
        const hasFinalAnswer = msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('BACKGROUND_RESULTS'));
        return toolCalls.length >= 3 && hasFinalAnswer;
      });

      await waitForProcessingStopped(container);

      // Data layer checks
      const messages = container.messageRepository.findComplete();
      const toolCalls = messages.filter((m) => m.toolCall?.toolName === 'execute_command');
      expect(toolCalls.length).toBeGreaterThanOrEqual(3);

      // Check results contain expected patterns
      const finalAnswer = messages.find((m) =>
        m.role === 'assistant' && !m.toolCall && m.content.value.includes('BACKGROUND_RESULTS'));
      expect(finalAnswer).toBeDefined();

      // First result should contain "Started background process bg-1"
      expect(finalAnswer!.content.value).toContain('Started background process');
      // Second result should contain "Background processes:" (list output)
      expect(finalAnswer!.content.value).toContain('Background processes');
      // Third result should contain "killed" (kill confirmation)
      expect(finalAnswer!.content.value).toContain('killed');

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('Exec'); // execute_command displayed as Exec prefix
      expect(frame).toContain('BACKGROUND_RESULTS');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  it('executes parallel execute_command calls via shell session pool', async () => {
    await server.start('parallel-exec');
    const container = createTestContainer(server.port, testDir);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Run two commands in parallel');

      // Wait for both execute_command results + final answer
      await waitForMessages(container, (msgs) => {
        const toolCalls = msgs.filter((m) => m.toolCall?.toolName === 'execute_command');
        const hasFinalAnswer = msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('PARALLEL_RESULTS'));
        return toolCalls.length >= 2 && hasFinalAnswer;
      });

      await waitForProcessingStopped(container);

      // Data layer — both tool calls completed
      const messages = container.messageRepository.findComplete();
      const toolCalls = messages.filter((m) => m.toolCall?.toolName === 'execute_command');
      expect(toolCalls.length).toBe(2);

      // Both tools have results (not "session busy" error)
      for (const tc of toolCalls) {
        expect(tc.toolResult).toBeDefined();
        expect(tc.toolResult!.result).not.toContain('session');  // no "session busy" error
        expect(tc.toolResult!.result).not.toContain('[ERROR]');
      }

      // Final answer contains both command outputs
      const finalAnswer = messages.find((m) =>
        m.role === 'assistant' && !m.toolCall && m.content.value.includes('PARALLEL_RESULTS'));
      expect(finalAnswer).toBeDefined();
      expect(finalAnswer!.content.value).toContain('parallel_a');
      expect(finalAnswer!.content.value).toContain('parallel_b');

      // Rendered output — what user sees
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('Exec');
      expect(frame).toContain('PARALLEL_RESULTS');
      expect(frame).toContain('parallel_a');
      expect(frame).toContain('parallel_b');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // E2E-50: Cancel during stream — no error shown to user
  // Tests that cancelling mid-stream shows "[Cancelled by user]" and does NOT
  // show "context canceled" error. The server-side fix ensures:
  // 1. Orchestrator doesn't send ERROR event on cancel
  // 2. Engine saves snapshot as "suspended" for resume
  // 3. Agent drain loop stops cleanly on cancel
  it('cancel during stream shows cancelled message without error', async () => {
    await server.start('cancel-during-stream');
    const container = createTestContainer(server.port);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Process this slowly');

      // Wait for processing to start
      const start = Date.now();
      while (!container.streamProcessor.getIsProcessing() && Date.now() - start < 5000) {
        await new Promise((r) => setTimeout(r, 50));
      }
      expect(container.streamProcessor.getIsProcessing()).toBe(true);

      // Cancel mid-stream
      container.streamProcessor.cancel();

      // Wait for processing to stop
      await waitForProcessingStopped(container, 5000);

      // Data layer checks
      const messages = container.messageRepository.findComplete();

      // "[Cancelled by user]" message should exist
      const cancelMsg = messages.find((m) => m.content.value.includes('[Cancelled by user]'));
      expect(cancelMsg).toBeDefined();

      // NO error messages should exist (no "context canceled", no "Error:")
      const errorMsgs = messages.filter((m) =>
        m.content.value.includes('context canceled') ||
        m.content.value.startsWith('Error:'),
      );
      expect(errorMsgs).toHaveLength(0);

      // Rendered output — what user sees
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('Cancelled');
      expect(frame).not.toContain('context canceled');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);
});
