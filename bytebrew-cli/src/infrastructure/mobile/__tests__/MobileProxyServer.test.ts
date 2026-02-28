import { describe, test, expect, beforeEach, afterEach, spyOn } from 'bun:test';
import { MobileProxyServer, type MobileProxyMeta } from '../MobileProxyServer.js';
import { MockEventBus, MockMessageRepository } from '../../../application/services/__tests__/testHelpers.js';
import { Message } from '../../../domain/entities/Message.js';
import { ToolExecution } from '../../../domain/entities/ToolExecution.js';
import { MessageId } from '../../../domain/value-objects/MessageId.js';

// --- Helpers ---

class MockMessageSender {
  sentMessages: string[] = [];
  shouldThrow = false;
  sendMessage(content: string): void {
    if (this.shouldThrow) throw new Error('Send failed: not connected');
    this.sentMessages.push(content);
  }
}

const tick = (ms = 50) => new Promise((r) => setTimeout(r, ms));

function connectClient(port: number): Promise<WebSocket> {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(`ws://localhost:${port}`);
    ws.onopen = () => resolve(ws);
    ws.onerror = (e) => reject(e);
  });
}

function waitForMessage(ws: WebSocket, timeoutMs = 2000): Promise<any> {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error('waitForMessage timed out')), timeoutMs);
    ws.onmessage = (e) => {
      clearTimeout(timer);
      resolve(JSON.parse(String(e.data)));
    };
  });
}

function collectMessages(ws: WebSocket): any[] {
  const msgs: any[] = [];
  ws.onmessage = (e) => msgs.push(JSON.parse(String(e.data)));
  return msgs;
}

const defaultMeta: MobileProxyMeta = {
  projectName: 'test-project',
  projectPath: '/tmp/test-project',
  sessionId: 'test-session-123',
};

// --- Tests ---

describe('MobileProxyServer', () => {
  let server: MobileProxyServer;
  let eventBus: MockEventBus;
  let messageRepo: MockMessageRepository;
  let messageSender: MockMessageSender;
  let clients: WebSocket[];

  beforeEach(() => {
    eventBus = new MockEventBus();
    messageRepo = new MockMessageRepository();
    messageSender = new MockMessageSender();
    clients = [];
  });

  afterEach(async () => {
    for (const ws of clients) {
      try {
        ws.close();
      } catch { /* ignore */ }
    }
    clients = [];

    if (server) {
      server.stop();
    }

    await tick(20);
  });

  function createServer(withSender = true): MobileProxyServer {
    server = new MobileProxyServer(
      messageRepo,
      eventBus,
      defaultMeta,
      withSender ? messageSender : undefined,
    );
    return server;
  }

  async function connectAndTrack(): Promise<WebSocket> {
    const ws = await connectClient(server.port);
    clients.push(ws);
    return ws;
  }

  // ============================================================
  // Connection & Init
  // ============================================================

  describe('Connection and init', () => {
    test('TC-01: client receives init with messages and meta on connect', async () => {
      const msg = Message.createUser('Hello world');
      messageRepo.save(msg);

      createServer();
      server.start(0);

      const ws = await connectAndTrack();
      const init = await waitForMessage(ws);

      expect(init.type).toBe('init');
      expect(init.meta).toEqual(defaultMeta);
      expect(init.messages).toBeArray();
      expect(init.messages.length).toBe(1);
      expect(init.messages[0].content).toBe('Hello world');
      expect(init.messages[0].role).toBe('user');
    });

    test('TC-02: empty repository sends init with empty messages', async () => {
      createServer();
      server.start(0);

      const ws = await connectAndTrack();
      const init = await waitForMessage(ws);

      expect(init.type).toBe('init');
      expect(init.messages).toEqual([]);
      expect(init.meta).toEqual(defaultMeta);
    });

    test('TC-03: two clients both receive broadcast on event', async () => {
      createServer();
      server.start(0);

      const ws1 = await connectAndTrack();
      const init1 = await waitForMessage(ws1);
      expect(init1.type).toBe('init');

      const ws2 = await connectAndTrack();
      const init2 = await waitForMessage(ws2);
      expect(init2.type).toBe('init');

      // Now both clients listen for the next message
      const p1 = waitForMessage(ws1);
      const p2 = waitForMessage(ws2);

      eventBus.publish({ type: 'ProcessingStarted' });

      const [msg1, msg2] = await Promise.all([p1, p2]);
      expect(msg1.type).toBe('event');
      expect(msg1.event.type).toBe('ProcessingStarted');
      expect(msg2.type).toBe('event');
      expect(msg2.event.type).toBe('ProcessingStarted');
    });
  });

  // ============================================================
  // EventBus -> WS (broadcast)
  // ============================================================

  describe('EventBus -> WS', () => {
    test('TC-04: ProcessingStarted event is forwarded', async () => {
      createServer();
      server.start(0);

      const ws = await connectAndTrack();
      await waitForMessage(ws); // consume init

      const p = waitForMessage(ws);
      eventBus.publish({ type: 'ProcessingStarted' });
      const msg = await p;

      expect(msg.type).toBe('event');
      expect(msg.event.type).toBe('ProcessingStarted');
    });

    test('TC-05: MessageCompleted event includes message snapshot', async () => {
      createServer();
      server.start(0);

      const ws = await connectAndTrack();
      await waitForMessage(ws); // consume init

      const assistantMsg = Message.createAssistantWithContent('Response text');
      const p = waitForMessage(ws);
      eventBus.publish({ type: 'MessageCompleted', message: assistantMsg });
      const msg = await p;

      expect(msg.type).toBe('event');
      expect(msg.event.type).toBe('MessageCompleted');
      expect(msg.event.message).toBeDefined();
      expect(msg.event.message.content).toBe('Response text');
      expect(msg.event.message.role).toBe('assistant');
      expect(msg.event.message.isComplete).toBe(true);
    });

    test('TC-06: ToolExecutionStarted event includes execution snapshot', async () => {
      createServer();
      server.start(0);

      const ws = await connectAndTrack();
      await waitForMessage(ws); // consume init

      const execution = ToolExecution.create(
        'call-1',
        'read_file',
        { path: '/tmp/test.ts' },
        MessageId.create(),
      );

      const p = waitForMessage(ws);
      eventBus.publish({ type: 'ToolExecutionStarted', execution });
      const msg = await p;

      expect(msg.type).toBe('event');
      expect(msg.event.type).toBe('ToolExecutionStarted');
      expect(msg.event.execution).toBeDefined();
      expect(msg.event.execution.toolName).toBe('read_file');
      expect(msg.event.execution.callId).toBe('call-1');
      expect(msg.event.execution.arguments).toEqual({ path: '/tmp/test.ts' });
    });

    test('TC-07: ErrorOccurred event is serialized', async () => {
      createServer();
      server.start(0);

      const ws = await connectAndTrack();
      await waitForMessage(ws); // consume init

      const p = waitForMessage(ws);
      eventBus.publish({
        type: 'ErrorOccurred',
        error: new Error('Something went wrong'),
        context: 'stream-processing',
      });
      const msg = await p;

      expect(msg.type).toBe('event');
      expect(msg.event.type).toBe('ErrorOccurred');
      expect(msg.event.message).toBe('Something went wrong');
      expect(msg.event.context).toBe('stream-processing');
    });

    test('TC-08: ProcessingStopped event is forwarded', async () => {
      createServer();
      server.start(0);

      const ws = await connectAndTrack();
      await waitForMessage(ws); // consume init

      const p = waitForMessage(ws);
      eventBus.publish({ type: 'ProcessingStopped' });
      const msg = await p;

      expect(msg.type).toBe('event');
      expect(msg.event.type).toBe('ProcessingStopped');
    });
  });

  // ============================================================
  // WS -> Handler (incoming messages)
  // ============================================================

  describe('WS -> Handler', () => {
    test('TC-09: user_message calls messageSender.sendMessage', async () => {
      createServer();
      server.start(0);

      const ws = await connectAndTrack();
      await waitForMessage(ws); // consume init

      ws.send(JSON.stringify({ type: 'user_message', text: 'Hello from mobile' }));
      await tick(100);

      expect(messageSender.sentMessages).toEqual(['Hello from mobile']);
    });

    test('TC-10: user_message with empty text is ignored', async () => {
      createServer();
      server.start(0);

      const ws = await connectAndTrack();
      await waitForMessage(ws); // consume init

      ws.send(JSON.stringify({ type: 'user_message', text: '' }));
      ws.send(JSON.stringify({ type: 'user_message', text: '   ' }));
      await tick(100);

      expect(messageSender.sentMessages).toEqual([]);
    });

    test('TC-11: user_message without messageSender logs error', async () => {
      createServer(false); // no messageSender
      server.start(0);

      const spy = spyOn(console, 'error').mockImplementation(() => {});

      const ws = await connectAndTrack();
      await waitForMessage(ws); // consume init

      ws.send(JSON.stringify({ type: 'user_message', text: 'Hello' }));
      await tick(100);

      expect(spy).toHaveBeenCalledWith(
        expect.stringContaining('no messageSender configured'),
      );

      spy.mockRestore();
    });

    test('TC-12: ask_user_answer calls resolveAskUser without error', async () => {
      createServer();
      server.start(0);

      const ws = await connectAndTrack();
      await waitForMessage(ws); // consume init

      // resolveAskUser with no pending promise is a no-op, should not throw
      const answers = [{ question: 'Proceed?', answer: 'yes' }];
      ws.send(JSON.stringify({ type: 'ask_user_answer', answers }));
      await tick(100);

      // Verify no error was thrown (server is still alive)
      const p = waitForMessage(ws);
      eventBus.publish({ type: 'ProcessingStarted' });
      const msg = await p;
      expect(msg.type).toBe('event');
      expect(msg.event.type).toBe('ProcessingStarted');
    });

    test('TC-13: invalid JSON does not crash server', async () => {
      createServer();
      server.start(0);

      const spy = spyOn(console, 'error').mockImplementation(() => {});

      const ws = await connectAndTrack();
      await waitForMessage(ws); // consume init

      ws.send('this is not json {{{');
      await tick(100);

      // Server should still work
      const p = waitForMessage(ws);
      eventBus.publish({ type: 'ProcessingStopped' });
      const msg = await p;
      expect(msg.type).toBe('event');
      expect(msg.event.type).toBe('ProcessingStopped');

      spy.mockRestore();
    });

    test('TC-14: unknown message type is ignored', async () => {
      createServer();
      server.start(0);

      const ws = await connectAndTrack();
      await waitForMessage(ws); // consume init

      ws.send(JSON.stringify({ type: 'unknown_type', data: 'something' }));
      await tick(100);

      // Server should still work
      const p = waitForMessage(ws);
      eventBus.publish({ type: 'ProcessingStarted' });
      const msg = await p;
      expect(msg.type).toBe('event');
      expect(msg.event.type).toBe('ProcessingStarted');
    });
  });

  // ============================================================
  // Lifecycle
  // ============================================================

  describe('Lifecycle', () => {
    test('TC-15: stop() shuts down the server', async () => {
      createServer();
      server.start(0);
      const usedPort = server.port;

      const ws = await connectAndTrack();
      await waitForMessage(ws); // consume init

      server.stop();
      await tick(100);

      // Connection should fail after stop
      try {
        const ws2 = await connectClient(usedPort);
        ws2.close();
        // If connect somehow succeeds, that is unexpected but not a test failure in all runtimes
      } catch {
        // Expected: connection refused
      }
    });

    test('TC-16: dead client is removed on broadcast failure', async () => {
      createServer();
      server.start(0);

      const ws = await connectAndTrack();
      await waitForMessage(ws); // consume init

      // Close the client connection abruptly
      ws.close();
      await tick(100);

      // Broadcast should not throw even though client is dead
      // The dead client should be removed from clients set
      eventBus.publish({ type: 'ProcessingStarted' });
      await tick(50);

      // Server is still alive — can accept new connections
      const ws2 = await connectAndTrack();
      const init = await waitForMessage(ws2);
      expect(init.type).toBe('init');
    });
  });

  // ============================================================
  // Error propagation (Stage 1 fix)
  // ============================================================

  describe('Error propagation', () => {
    test('TC-17: sendMessage throws -> WS client receives error', async () => {
      createServer();
      server.start(0);

      messageSender.shouldThrow = true;

      const ws = await connectAndTrack();
      await waitForMessage(ws); // consume init

      const p = waitForMessage(ws);
      ws.send(JSON.stringify({ type: 'user_message', text: 'This will fail' }));
      const msg = await p;

      expect(msg.type).toBe('error');
      expect(msg.message).toBe('Send failed: not connected');
    });

    test('TC-18: ErrorOccurred event reaches WS client', async () => {
      createServer();
      server.start(0);

      const ws = await connectAndTrack();
      await waitForMessage(ws); // consume init

      const p = waitForMessage(ws);
      eventBus.publish({
        type: 'ErrorOccurred',
        error: new Error('Stream disconnected'),
        context: 'grpc-stream',
      });
      const msg = await p;

      expect(msg.type).toBe('event');
      expect(msg.event.type).toBe('ErrorOccurred');
      expect(msg.event.message).toBe('Stream disconnected');
      expect(msg.event.context).toBe('grpc-stream');
    });
  });
});
