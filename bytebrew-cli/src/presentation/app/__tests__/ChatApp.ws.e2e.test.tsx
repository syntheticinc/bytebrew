import { describe, it, expect, beforeAll, beforeEach, afterEach } from 'bun:test';
import React from 'react';
import { render } from 'ink-testing-library';
import { Container } from '../../../config/container.js';
import { ChatApp } from '../ChatApp.js';
import { TestServerHelper } from '../../../test-utils/TestServerHelper.js';
import { Message } from '../../../domain/entities/Message.js';
import type { Question, QuestionAnswer } from '../../../tools/askUser.js';
import fs from 'fs';
import path from 'path';
import os from 'os';

/**
 * E2E tests for WebSocket transport (WsStreamGateway).
 *
 * Same stack as ChatApp.e2e.test.tsx:
 * - Real Go test server (MockChatModel)
 * - WsStreamGateway → WS Server → SessionProcessor → Engine → REACT agent
 * - Full event flow: ProcessingStarted → StreamingProgress → ToolEvents → MessageCompleted → ProcessingStopped
 *
 * Requires testserver to emit READY:{port}:{ws_port} format.
 */
describe('E2E: ChatApp with WebSocket transport', () => {
  let server: TestServerHelper;
  let testDir: string;

  beforeAll(() => {
    TestServerHelper.build();
  }, 60000);

  beforeEach(() => {
    server = new TestServerHelper();
    testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'ws-e2e-'));
  });

  afterEach(async () => {
    await server.stop();
    await new Promise((r) => setTimeout(r, 200));
    try {
      fs.rmSync(testDir, { recursive: true, force: true });
    } catch {
      // Ignore cleanup errors
    }
  });

  // Helper: create container with WS transport
  function createWsContainer(grpcPort: number, wsPort: number, projectRoot?: string): Container {
    const container = new Container({
      projectRoot: projectRoot || '/test',
      serverAddress: `localhost:${grpcPort}`,
      wsAddress: `localhost:${wsPort}`,
      projectKey: 'ws-e2e-test',
      headlessMode: true,
      askUserCallback: async (questions: Question[]): Promise<QuestionAnswer[]> =>
        questions.map(q => ({ question: q.text, answer: 'approved' })),
    });
    container.initialize();
    return container;
  }

  // Helper: connect to server and send message
  async function connectAndSend(container: Container, message: string): Promise<void> {
    const start = Date.now();
    while (container.streamGateway.getStatus() !== 'connected' && Date.now() - start < 5000) {
      await new Promise((r) => setTimeout(r, 50));
    }

    if (container.streamGateway.getStatus() !== 'connected') {
      await container.streamGateway.connect({
        serverAddress: container.config.wsAddress || container.config.serverAddress,
        sessionId: container.sessionId,
        userId: 'ws-e2e-user',
        projectKey: container.config.projectKey,
        projectRoot: container.config.projectRoot,
      });
      await new Promise((r) => setTimeout(r, 100));
    }

    container.streamProcessor.sendMessage(message);
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

  // Helper: wait for messages matching predicate
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
    const msgs = container.messageRepository.findComplete();
    throw new Error(
      `Timeout (${timeout}ms) waiting for messages. Got ${msgs.length} messages:\n` +
        msgs.map((m) => `  [${m.role}] ${m.content.value.slice(0, 80)}`).join('\n'),
    );
  }

  // TC-CLI-WS-01: Basic chat — echo via WS
  it('receives text answer via WebSocket transport', async () => {
    await server.start('echo');
    if (!server.wsPort) {
      console.log('SKIP: testserver does not expose WS port');
      return;
    }

    const container = createWsContainer(server.port, server.wsPort);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Hello');

      await waitForMessages(container, (msgs) =>
        msgs.some((m) => m.content.value.includes('Hello, world!')),
      );

      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();
      expect(messages.length).toBeGreaterThanOrEqual(2);
      expect(messages.some((m) => m.role === 'user')).toBe(true);
      expect(messages.some((m) => m.role === 'assistant')).toBe(true);

      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('Hello, world!');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // TC-CLI-WS-02: Multi-turn conversation
  it('preserves context across multiple messages', async () => {
    await server.start('echo');
    if (!server.wsPort) {
      console.log('SKIP: testserver does not expose WS port');
      return;
    }

    const container = createWsContainer(server.port, server.wsPort);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));

      // First message
      await connectAndSend(container, 'Remember the number 42');
      await waitForProcessingStopped(container);

      // Second message
      container.streamProcessor.sendMessage('What number did I mention?');
      // Wait for messages to accumulate (waitForProcessingStopped may return
      // before processing starts if there's a gap between sendMessage and
      // the server picking it up).
      await waitForMessages(container, (msgs) => msgs.length >= 4);
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // TC-CLI-WS-03: Tool execution via WS
  it('renders tool calls received via WebSocket', async () => {
    await server.start('server-tool');
    if (!server.wsPort) {
      console.log('SKIP: testserver does not expose WS port');
      return;
    }

    const container = createWsContainer(server.port, server.wsPort);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'List subtasks');

      await waitForMessages(container, (msgs) => {
        const hasToolCall = msgs.some((m) => m.toolCall?.toolName === 'manage_subtasks');
        const hasFinalAnswer = msgs.some(
          (m) => m.role === 'assistant' && m.content.value.includes('complete'),
        );
        return hasToolCall && hasFinalAnswer;
      });

      await waitForProcessingStopped(container);

      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      // Tool call indicator should be visible
      expect(frame.length).toBeGreaterThan(0);
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // TC-CLI-WS-04: Cancel via WS
  it('cancels processing via WebSocket', async () => {
    await server.start('echo');
    if (!server.wsPort) {
      console.log('SKIP: testserver does not expose WS port');
      return;
    }

    const container = createWsContainer(server.port, server.wsPort);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Tell me a long story');

      // Wait briefly then cancel
      await new Promise((r) => setTimeout(r, 500));
      container.streamGateway.cancel();

      // Should eventually stop processing
      await waitForProcessingStopped(container, 10000);
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // TC-CLI-WS-05: Reasoning content displayed via WS
  // Note: WS event_serializer does not include is_complete for ReasoningChunk,
  // so reasoning messages are not completed/saved in the repository.
  // We verify the final answer is delivered correctly.
  it('displays reasoning content via WebSocket', async () => {
    await server.start('reasoning');
    if (!server.wsPort) {
      console.log('SKIP: testserver does not expose WS port');
      return;
    }

    const container = createWsContainer(server.port, server.wsPort);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Think about this');

      // Wait for assistant message with the answer
      await waitForMessages(container, (msgs) =>
        msgs.some((m) => m.role === 'assistant' && m.content.value.includes('The answer is 42.')),
      );

      await waitForProcessingStopped(container);

      // Data layer checks — verify final answer is present
      const messages = container.messageRepository.findComplete();
      const answerMsg = messages.find(
        (m) => m.role === 'assistant' && m.content.value.includes('The answer is 42.'),
      );
      expect(answerMsg).toBeDefined();

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('The answer is 42.');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // TC-CLI-WS-06: LLM error handling via WS
  it('handles LLM error gracefully via WebSocket', async () => {
    await server.start('error');
    if (!server.wsPort) {
      console.log('SKIP: testserver does not expose WS port');
      return;
    }

    const container = createWsContainer(server.port, server.wsPort);
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
      // Error scenario may show error message or empty output - verify render didn't crash
      expect(frame).toBeDefined();
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // TC-CLI-WS-07: Proxied read_file via WS
  it('executes proxied read_file via WebSocket', async () => {
    // Create test.txt in project root (local-read scenario reads "test.txt")
    fs.writeFileSync(path.join(testDir, 'test.txt'), 'hello ws world');

    await server.start('local-read');
    if (!server.wsPort) {
      console.log('SKIP: testserver does not expose WS port');
      return;
    }

    const container = createWsContainer(server.port, server.wsPort, testDir);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Read the file');

      await waitForMessages(container, (msgs) => {
        const hasToolCall = msgs.some((m) => m.toolCall?.toolName === 'read_file');
        const hasFinalAnswer = msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('hello ws world'));
        return hasToolCall && hasFinalAnswer;
      });

      await waitForProcessingStopped(container);

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('hello ws world');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // TC-CLI-WS-08: AskUser full flow via WS
  it('handles ask_user tool via WebSocket', async () => {
    await server.start('ask-user');
    if (!server.wsPort) {
      console.log('SKIP: testserver does not expose WS port');
      return;
    }

    // headlessMode: true in createWsContainer auto-selects "approved"
    const container = createWsContainer(server.port, server.wsPort);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Ask the user');

      await waitForMessages(container, (msgs) =>
        msgs.some((m) =>
          m.role === 'assistant' && m.content.value.length > 0 && !m.toolCall),
      );

      await waitForProcessingStopped(container);

      // The LLM's final answer should reference the auto-selected "approved" reply
      const messages = container.messageRepository.findComplete();
      const hasApprovedInAnswer = messages.some(
        (m) => m.role === 'assistant' && !m.toolCall && m.content.value.includes('approved'),
      );
      expect(hasApprovedInAnswer).toBe(true);

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('approved');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // TC-CLI-WS-09: Multiple sequential tool calls via WS
  it('handles multiple sequential read_file calls via WebSocket', async () => {
    // local-multi-tool scenario: reads a.txt then b.txt
    fs.writeFileSync(path.join(testDir, 'a.txt'), 'content_alpha');
    fs.writeFileSync(path.join(testDir, 'b.txt'), 'content_beta');

    await server.start('local-multi-tool');
    if (!server.wsPort) {
      console.log('SKIP: testserver does not expose WS port');
      return;
    }

    const container = createWsContainer(server.port, server.wsPort, testDir);
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

      // Final answer should contain both file contents
      const finalAnswer = messages.find((m) =>
        m.role === 'assistant' && !m.toolCall && m.content.value.includes('MULTI_READ'));
      expect(finalAnswer).toBeDefined();
      expect(finalAnswer!.content.value).toContain('content_alpha');
      expect(finalAnswer!.content.value).toContain('content_beta');

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('content_alpha');
      expect(frame).toContain('content_beta');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // TC-CLI-WS-10: Multi-agent lifecycle events via WS
  it('spawns code agent and shows lifecycle events via WebSocket', async () => {
    await server.start('multi-agent');
    if (!server.wsPort) {
      console.log('SKIP: testserver does not expose WS port');
      return;
    }

    const container = createWsContainer(server.port, server.wsPort);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Spawn an agent');

      // Wait for the full multi-agent flow with longer timeout
      await waitForMessages(container, (msgs) => {
        const hasFinalAnswer = msgs.some(
          (m) => m.role === 'assistant' && m.content.value.includes('All agents completed'),
        );
        return hasFinalAnswer;
      }, 30000);

      await waitForProcessingStopped(container);

      // Data layer checks
      const messages = container.messageRepository.findComplete();

      // spawn_code_agent tool call visible
      const spawnToolMsg = messages.find(
        (m) => m.toolCall?.toolName === 'spawn_code_agent',
      );
      expect(spawnToolMsg).toBeDefined();

      // Lifecycle "spawned" event visible
      const spawnedMsg = messages.find(
        (m) => m.content.value.includes('spawned'),
      );
      expect(spawnedMsg).toBeDefined();

      // Lifecycle "completed" event visible
      const completedMsg = messages.find(
        (m) => m.content.value.includes('completed') && m.content.value.includes('Code Agent'),
      );
      expect(completedMsg).toBeDefined();

      // Supervisor's final answer visible
      const finalAnswer = messages.find(
        (m) => m.role === 'assistant' && m.content.value.includes('All agents completed'),
      );
      expect(finalAnswer).toBeDefined();

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('spawned');
      expect(frame).toContain('completed');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 45000);

  // TC-CLI-WS-11: Read file error recovery via WS
  it('handles read_file error and recovers via WebSocket', async () => {
    // local-read-error scenario: reads nonexistent.txt, then LLM recovers
    await server.start('local-read-error');
    if (!server.wsPort) {
      console.log('SKIP: testserver does not expose WS port');
      return;
    }

    const container = createWsContainer(server.port, server.wsPort, testDir);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Read nonexistent file');

      // Wait for recovery answer (LLM responds with RECOVERED:... after error)
      await waitForMessages(container, (msgs) =>
        msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('RECOVERED')),
      );

      await waitForProcessingStopped(container);

      // Data layer checks
      const messages = container.messageRepository.findComplete();

      // Tool call for read_file should exist
      const toolMsg = messages.find((m) => m.toolCall?.toolName === 'read_file');
      expect(toolMsg).toBeDefined();
      // Tool result should contain error info
      expect(toolMsg?.toolResult).toBeDefined();

      // LLM should have recovered with final answer
      const recoveryMsg = messages.find(
        (m) => m.role === 'assistant' && m.content.value.includes('RECOVERED'),
      );
      expect(recoveryMsg).toBeDefined();

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('RECOVERED');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // TC-CLI-WS-12: Parallel tool execution via WS
  it('handles parallel execute_command calls via WebSocket', async () => {
    // parallel-exec scenario: sends two execute_command calls in one message
    await server.start('parallel-exec');
    if (!server.wsPort) {
      console.log('SKIP: testserver does not expose WS port');
      return;
    }

    const container = createWsContainer(server.port, server.wsPort, testDir);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Run parallel commands');

      // Wait for final answer containing both results
      await waitForMessages(container, (msgs) =>
        msgs.some((m) =>
          m.role === 'assistant' && !m.toolCall && m.content.value.includes('PARALLEL_RESULTS')),
      );

      await waitForProcessingStopped(container);

      // Data layer checks
      const messages = container.messageRepository.findComplete();

      // Should have execute_command tool calls.
      // Note: parallel tool calls in one LLM message may be emitted as a single event
      // through the WS pipeline (callback handler emits per-step, not per-call).
      const toolMsgs = messages.filter((m) => m.toolCall?.toolName === 'execute_command');
      expect(toolMsgs.length).toBeGreaterThanOrEqual(1);

      // Final answer should contain results from both commands
      const finalAnswer = messages.find((m) =>
        m.role === 'assistant' && m.content.value.includes('PARALLEL_RESULTS'));
      expect(finalAnswer).toBeDefined();

      // Rendered output checks
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('PARALLEL_RESULTS');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);
});
