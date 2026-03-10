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
 * E2E tests for license enforcement flows.
 *
 * Tests that license interceptors on the Go test server correctly:
 * - Block connections when license is blocked (PermissionDenied)
 * - Allow connections with grace period (warning header)
 * - Allow connections with active license (normal operation)
 *
 * Uses the same real WS + Go server stack as ChatApp.e2e.test.tsx.
 * Only MockChatModel is different from production.
 */
describe('E2E: License enforcement', () => {
  let server: TestServerHelper;
  let testDir: string;

  beforeAll(() => {
    TestServerHelper.build();
  }, 60000);

  beforeEach(() => {
    server = new TestServerHelper();
    testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'e2e-license-'));
  });

  afterEach(async () => {
    await server.stop();
    await new Promise((r) => setTimeout(r, 200));
    try {
      fs.rmSync(testDir, { recursive: true, force: true });
    } catch {
      // Ignore cleanup errors (Windows file locks)
    }
  });

  // Helper: create container with WsStreamGateway (same as main e2e tests)
  function createTestContainer(port: number, projectRoot?: string): Container {
    const wsPort = server.wsPort;
    const container = new Container({
      projectRoot: projectRoot || testDir,
      serverAddress: `localhost:${port}`,
      wsAddress: wsPort ? `localhost:${wsPort}` : undefined,
      projectKey: 'e2e-license-test',
      headlessMode: true,
      askUserCallback: async () => 'approved',
      disableLspServers: true,
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

  // License-1: Active license works normally (baseline)
  it('works normally with active license', async () => {
    await server.start('echo', { license: 'active' });
    const container = createTestContainer(server.port);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Hello');

      await waitForMessages(container, (msgs) =>
        msgs.some((m) => m.content.value.includes('Hello, world!')),
      );

      await waitForProcessingStopped(container);

      // Rendered output: agent response should be visible
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('Hello, world!');

      // Connection should be established
      expect(frame).toContain('Connected');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // License-2: Grace license works but stream is allowed through
  it('works with grace license (stream allowed)', async () => {
    await server.start('echo', { license: 'grace' });
    const container = createTestContainer(server.port);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Hello');

      await waitForMessages(container, (msgs) =>
        msgs.some((m) => m.content.value.includes('Hello, world!')),
      );

      await waitForProcessingStopped(container);

      // Rendered output: agent response should work despite grace period
      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('Hello, world!');

      // Connection should be established (grace allows the stream)
      expect(frame).toContain('Connected');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // License-3: Blocked license prevents connection
  it('shows disconnected status when license is blocked', async () => {
    await server.start('echo', { license: 'blocked' });
    const container = createTestContainer(server.port);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));

      // Wait for connection attempt to fail.
      // The license interceptor returns PermissionDenied immediately,
      // which causes the stream error handler to fire and status to go to disconnected.
      await new Promise((r) => setTimeout(r, 3000));

      // Rendered output: should NOT show "Connected"
      const frame = instance.lastFrame() || '';

      // The connection should have failed — status should not be "Connected"
      expect(frame).not.toContain('Connected');

      // Should show disconnected or connecting status (the license error prevents connection)
      const hasDisconnected = frame.includes('Disconnected') || frame.includes('Connecting') || frame.includes('Reconnecting');
      expect(hasDisconnected).toBe(true);

      // The input placeholder should indicate connection issue
      expect(frame).toContain('Connecting...');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);

  // License-4: Default (no license flag) = active, works normally
  it('works with default license (no flag = active)', async () => {
    await server.start('echo');
    const container = createTestContainer(server.port);
    let instance: ReturnType<typeof render> | null = null;

    try {
      instance = render(React.createElement(ChatApp, { container }));
      await connectAndSend(container, 'Hello');

      await waitForMessages(container, (msgs) =>
        msgs.some((m) => m.content.value.includes('Hello, world!')),
      );

      await waitForProcessingStopped(container);

      await new Promise((r) => setTimeout(r, 300));
      const frame = instance.lastFrame() || '';
      expect(frame).toContain('Hello, world!');
      expect(frame).toContain('Connected');
    } finally {
      if (instance) instance.unmount();
      await container.dispose();
    }
  }, 30000);
});
