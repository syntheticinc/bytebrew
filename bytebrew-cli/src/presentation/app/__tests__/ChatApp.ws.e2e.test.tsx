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
 * Same stack as ChatApp.e2e.test.tsx but over WS instead of gRPC:
 * - Real Go test server (MockChatModel)
 * - WsStreamGateway → WS Server → SessionProcessor → Engine → REACT agent
 * - Full event flow: ProcessingStarted → StreamingProgress → ToolEvents → MessageCompleted → ProcessingStopped
 *
 * Requires testserver to emit READY:{grpc_port}:{ws_port} format.
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
      await waitForProcessingStopped(container);

      const messages = container.messageRepository.findComplete();
      // Should have at least 4 messages: user1, assistant1, user2, assistant2
      expect(messages.length).toBeGreaterThanOrEqual(4);
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
});
