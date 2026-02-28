import { describe, test, expect, afterEach, spyOn } from 'bun:test';
import { MobileProxyServer, type MobileProxyMeta } from '../MobileProxyServer.js';
import {
  MockStreamGateway,
  MockEventBus,
  MockMessageRepository,
  MockToolExecutor,
} from '../../../application/services/__tests__/testHelpers.js';
import { MessageAccumulatorService } from '../../../application/services/MessageAccumulatorService.js';
import { StreamProcessorService } from '../../../application/services/StreamProcessorService.js';

// --- Helpers ---

const tick = (ms = 50) => new Promise((r) => setTimeout(r, ms));


function connectClient(port: number): Promise<WebSocket> {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(`ws://localhost:${port}`);
    ws.onopen = () => resolve(ws);
    ws.onerror = (e) => reject(e);
  });
}

function waitForMessage(ws: WebSocket, timeoutMs = 3000): Promise<any> {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error('waitForMessage timed out')), timeoutMs);
    ws.onmessage = (e) => {
      clearTimeout(timer);
      resolve(JSON.parse(String(e.data)));
    };
  });
}

/**
 * Collect all WS messages into an array.
 * Returns the array reference; new messages appear as they arrive.
 */
function collectMessages(ws: WebSocket): any[] {
  const msgs: any[] = [];
  ws.onmessage = (e) => msgs.push(JSON.parse(String(e.data)));
  return msgs;
}

/**
 * Wait until the collected messages array contains an item matching predicate,
 * or time out. Returns the matched message.
 */
function waitForCollected(
  collected: any[],
  predicate: (msg: any) => boolean,
  timeoutMs = 3000,
): Promise<any> {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(
      () => reject(new Error(`waitForCollected timed out. Collected: ${JSON.stringify(collected)}`)),
      timeoutMs,
    );

    // Check already-collected messages
    const found = collected.find(predicate);
    if (found) {
      clearTimeout(timer);
      resolve(found);
      return;
    }

    // Poll periodically for new messages
    const interval = setInterval(() => {
      const match = collected.find(predicate);
      if (match) {
        clearInterval(interval);
        clearTimeout(timer);
        resolve(match);
      }
    }, 20);
  });
}

const defaultMeta: MobileProxyMeta = {
  projectName: 'integration-test',
  projectPath: '/tmp/integration-test',
  sessionId: 'int-session-001',
};

// --- Integration setup ---

interface IntegrationHarness {
  gateway: MockStreamGateway;
  eventBus: MockEventBus;
  messageRepo: MockMessageRepository;
  toolExecutor: MockToolExecutor;
  accumulator: MessageAccumulatorService;
  processor: StreamProcessorService;
  proxy: MobileProxyServer;
}

function createHarness(): IntegrationHarness {
  const gateway = new MockStreamGateway();
  const eventBus = new MockEventBus();
  const messageRepo = new MockMessageRepository();
  const toolExecutor = new MockToolExecutor();
  const accumulator = new MessageAccumulatorService();
  const processor = new StreamProcessorService({
    streamGateway: gateway,
    messageRepository: messageRepo,
    toolExecutor,
    accumulator,
    eventBus,
  });
  processor.initialize();

  const messageSender = {
    sendMessage: (content: string) => processor.sendMessage(content),
  };

  const proxy = new MobileProxyServer(messageRepo, eventBus, defaultMeta, messageSender);

  return { gateway, eventBus, messageRepo, toolExecutor, accumulator, processor, proxy };
}

// --- Tests ---

describe('MobileProxy Integration (WS -> Proxy -> StreamProcessor -> MockGateway)', () => {
  let harness: IntegrationHarness;
  let clients: WebSocket[];
  let consoleSpy: ReturnType<typeof spyOn> | null = null;

  afterEach(async () => {
    for (const ws of clients) {
      try { ws.close(); } catch { /* ignore */ }
    }
    clients = [];

    if (harness) {
      harness.processor.dispose();
      harness.proxy.stop();
    }

    if (consoleSpy) {
      consoleSpy.mockRestore();
      consoleSpy = null;
    }

    await tick(30);
  });

  function startHarness(): IntegrationHarness {
    harness = createHarness();
    clients = [];
    // Suppress MobileProxy console.log noise
    consoleSpy = spyOn(console, 'log').mockImplementation(() => {});
    harness.proxy.start(0);
    return harness;
  }

  async function connectAndConsume(port: number): Promise<WebSocket> {
    const ws = await connectClient(port);
    clients.push(ws);
    // Consume the init message
    await waitForMessage(ws);
    return ws;
  }

  // ============================================================
  // TC-INT-01: user_message -> gateway.sentMessages
  // ============================================================

  test('TC-INT-01: user_message through WS reaches gateway.sentMessages', async () => {
    const h = startHarness();
    const ws = await connectAndConsume(h.proxy.port);

    // Set up collector BEFORE sending message (to catch events)
    const collected = collectMessages(ws);

    ws.send(JSON.stringify({ type: 'user_message', text: 'Hello' }));
    await tick(200);

    // Message reached the gateway
    expect(h.gateway.sentMessages).toContain('Hello');

    // User message saved in repository
    const allMessages = h.messageRepo.findAll();
    const userMsg = allMessages.find(
      (m) => m.role === 'user' && m.content.value === 'Hello',
    );
    expect(userMsg).toBeDefined();
  });

  // ============================================================
  // TC-INT-02: Gateway disconnected + reconnect fail -> ErrorOccurred
  // ============================================================

  test('TC-INT-02: gateway disconnected + reconnect fail -> WS client gets ErrorOccurred', async () => {
    const h = startHarness();
    const ws = await connectAndConsume(h.proxy.port);

    // Disconnect gateway and make reconnect fail
    h.gateway.disconnect();
    h.gateway.reconnectStream = () => Promise.reject(new Error('Reconnect failed'));

    const collected = collectMessages(ws);

    ws.send(JSON.stringify({ type: 'user_message', text: 'Should fail' }));

    // Wait for ErrorOccurred event to be broadcast via EventBus -> WS
    const errorMsg = await waitForCollected(
      collected,
      (msg) => msg.type === 'event' && msg.event?.type === 'ErrorOccurred',
    );

    expect(errorMsg.type).toBe('event');
    expect(errorMsg.event.type).toBe('ErrorOccurred');
    expect(errorMsg.event.message).toContain('Reconnect failed');
  });

  // ============================================================
  // TC-INT-03: Server ANSWER response -> WS client gets MessageCompleted
  // ============================================================

  test('TC-INT-03: server ANSWER response -> WS client receives MessageCompleted', async () => {
    const h = startHarness();
    const ws = await connectAndConsume(h.proxy.port);
    const collected = collectMessages(ws);

    // Send a user message first to start processing
    ws.send(JSON.stringify({ type: 'user_message', text: 'Tell me something' }));
    await tick(100);

    // Simulate server response: ANSWER with content and isFinal=true
    h.gateway.simulateResponse({
      type: 'ANSWER',
      content: 'Hello back',
      isFinal: true,
    });

    // Wait for MessageCompleted with assistant content
    const completed = await waitForCollected(
      collected,
      (msg) =>
        msg.type === 'event' &&
        msg.event?.type === 'MessageCompleted' &&
        msg.event.message &&
        msg.event.message.role === 'assistant' &&
        msg.event.message.content === 'Hello back',
    );

    expect(completed.event.message.content).toBe('Hello back');
    expect(completed.event.message.role).toBe('assistant');
    expect(completed.event.message.isComplete).toBe(true);
  });

  // ============================================================
  // TC-INT-04: TOOL_CALL -> WS client gets ToolExecutionStarted and ToolExecutionCompleted
  // ============================================================

  test('TC-INT-04: TOOL_CALL -> WS client receives ToolExecutionStarted', async () => {
    const h = startHarness();
    const ws = await connectAndConsume(h.proxy.port);
    const collected = collectMessages(ws);

    // Send user message to start processing
    ws.send(JSON.stringify({ type: 'user_message', text: 'Read a file' }));
    await tick(100);

    // Simulate TOOL_CALL from server (client-side tool, no "server-" prefix)
    h.gateway.simulateResponse({
      type: 'TOOL_CALL',
      content: '',
      isFinal: false,
      toolCall: {
        callId: 'call-1',
        toolName: 'read_file',
        arguments: { path: '/tmp/test.ts' },
      },
    });

    // Wait for ToolExecutionStarted
    const startedMsg = await waitForCollected(
      collected,
      (msg) => msg.type === 'event' && msg.event?.type === 'ToolExecutionStarted',
    );

    expect(startedMsg.event.execution.toolName).toBe('read_file');
    expect(startedMsg.event.execution.callId).toBe('call-1');

    // Wait a bit for async tool execution to complete
    await tick(200);

    // Should also have ToolExecutionCompleted
    const completedMsg = collected.find((msg) => msg.type === 'event' && msg.event?.type === 'ToolExecutionCompleted');
    expect(completedMsg).toBeDefined();
    expect(completedMsg.event.execution.callId).toBe('call-1');
  });

  // ============================================================
  // TC-INT-05: Two sequential messages -> both reach gateway
  // ============================================================

  test('TC-INT-05: two sequential messages both reach gateway', async () => {
    const h = startHarness();
    const ws = await connectAndConsume(h.proxy.port);

    // First message
    ws.send(JSON.stringify({ type: 'user_message', text: 'first' }));
    await tick(200);

    expect(h.gateway.sentMessages).toContain('first');

    // At this point _isProcessing = true (no final response sent).
    // Second message goes through interrupt path.
    ws.send(JSON.stringify({ type: 'user_message', text: 'second' }));
    await tick(200);

    expect(h.gateway.sentMessages).toContain('second');
    expect(h.gateway.sentMessages.length).toBe(2);
  });

  // ============================================================
  // TC-INT-06: ask_user_answer -> resolveAskUser called without error
  // ============================================================

  test('TC-INT-06: ask_user_answer does not crash (no-op when no pending)', async () => {
    const h = startHarness();
    const ws = await connectAndConsume(h.proxy.port);
    const collected = collectMessages(ws);

    // Send ask_user_answer with no pending promise -- should be a no-op
    ws.send(
      JSON.stringify({
        type: 'ask_user_answer',
        answers: [{ question: 'Proceed?', answer: 'yes' }],
      }),
    );
    await tick(100);

    // Server (proxy) is still alive -- verify by publishing an event
    h.eventBus.publish({ type: 'ProcessingStarted' });

    const startedMsg = await waitForCollected(
      collected,
      (msg) => msg.type === 'event' && msg.event?.type === 'ProcessingStarted',
    );
    expect(startedMsg.type).toBe('event');
    expect(startedMsg.event.type).toBe('ProcessingStarted');
  });
});
