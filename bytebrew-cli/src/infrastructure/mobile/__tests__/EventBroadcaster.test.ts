import { describe, test, expect } from 'bun:test';
import { EventBroadcaster, type SerializedEvent } from '../EventBroadcaster';
import type { IEventBus, DomainEvent, EventHandler } from '../../../domain/ports/IEventBus';
import type { IBridgeMessageRouter, MobileMessage } from '../../bridge/BridgeMessageRouter';
import type { IEventBuffer } from '../EventBuffer';
import { Message } from '../../../domain/entities/Message';
import { ToolExecution } from '../../../domain/entities/ToolExecution';
import { MessageId } from '../../../domain/value-objects/MessageId';

// --- Mock EventBus ---

function mockEventBus() {
  let allHandler: EventHandler | null = null;
  let unsubscribed = false;

  const bus: IEventBus = {
    publish: () => {},
    subscribe: () => () => {},
    subscribeAll: (handler) => {
      allHandler = handler;
      return () => { unsubscribed = true; };
    },
    clear: () => {},
  };

  return {
    bus,
    emit: (event: DomainEvent) => allHandler?.(event),
    isUnsubscribed: () => unsubscribed,
  };
}

// --- Mock Router ---

function mockRouter() {
  const sent: Array<{ deviceId: string; message: MobileMessage }> = [];

  const router: IBridgeMessageRouter = {
    start: () => {},
    stop: () => {},
    onMessage: () => () => {},
    onDeviceConnect: () => () => {},
    onDeviceDisconnect: () => () => {},
    sendMessage: (deviceId, message) => sent.push({ deviceId, message }),
  };

  return { router, sent };
}

// --- Mock EventBuffer ---

function mockBuffer() {
  const pushed: Array<{ sessionId: string; event: SerializedEvent }> = [];
  const stored: SerializedEvent[] = [];

  const buffer: IEventBuffer<SerializedEvent> = {
    push: (sessionId, event) => {
      pushed.push({ sessionId, event });
      stored.push(event);
    },
    getAfter: (_sid, afterIndex) => ({
      events: stored.slice(afterIndex + 1),
      lastIndex: stored.length - 1,
    }),
    clear: () => {},
  };

  return { buffer, pushed, stored };
}

// --- Helpers ---

function processingStartedEvent(): DomainEvent {
  return { type: 'ProcessingStarted' };
}

function processingStoppedEvent(): DomainEvent {
  return { type: 'ProcessingStopped' };
}

describe('EventBroadcaster', () => {
  test('subscribe + emit event sends to device', () => {
    const { bus, emit } = mockEventBus();
    const { router, sent } = mockRouter();
    const { buffer } = mockBuffer();

    const broadcaster = new EventBroadcaster(bus, router, buffer, 'session-1');
    broadcaster.start();
    broadcaster.subscribe('dev-1');

    emit(processingStartedEvent());

    expect(sent).toHaveLength(1);
    expect(sent[0].deviceId).toBe('dev-1');
    expect(sent[0].message.type).toBe('session_event');
    expect((sent[0].message.payload as Record<string, unknown>).session_id).toBe('session-1');
  });

  test('unsubscribe stops sending to device', () => {
    const { bus, emit } = mockEventBus();
    const { router, sent } = mockRouter();
    const { buffer } = mockBuffer();

    const broadcaster = new EventBroadcaster(bus, router, buffer, 'session-1');
    broadcaster.start();
    broadcaster.subscribe('dev-1');

    emit(processingStartedEvent());
    expect(sent).toHaveLength(1);

    broadcaster.unsubscribe('dev-1');

    emit(processingStoppedEvent());
    expect(sent).toHaveLength(1); // no new messages
  });

  test('multiple devices all receive events', () => {
    const { bus, emit } = mockEventBus();
    const { router, sent } = mockRouter();
    const { buffer } = mockBuffer();

    const broadcaster = new EventBroadcaster(bus, router, buffer, 'session-1');
    broadcaster.start();
    broadcaster.subscribe('dev-1');
    broadcaster.subscribe('dev-2');
    broadcaster.subscribe('dev-3');

    emit(processingStartedEvent());

    expect(sent).toHaveLength(3);
    const deviceIds = sent.map((s) => s.deviceId).sort();
    expect(deviceIds).toEqual(['dev-1', 'dev-2', 'dev-3']);
  });

  test('buffers events in EventBuffer', () => {
    const { bus, emit } = mockEventBus();
    const { router } = mockRouter();
    const { buffer, pushed } = mockBuffer();

    const broadcaster = new EventBroadcaster(bus, router, buffer, 'session-1');
    broadcaster.start();

    emit(processingStartedEvent());
    emit(processingStoppedEvent());

    expect(pushed).toHaveLength(2);
    expect(pushed[0].sessionId).toBe('session-1');
    expect(pushed[0].event.type).toBe('ProcessingStarted');
    expect(pushed[1].event.type).toBe('ProcessingStopped');
  });

  test('getBufferedEvents returns buffered events', () => {
    const { bus, emit } = mockEventBus();
    const { router } = mockRouter();
    const { buffer } = mockBuffer();

    const broadcaster = new EventBroadcaster(bus, router, buffer, 'session-1');
    broadcaster.start();

    emit(processingStartedEvent());
    emit(processingStoppedEvent());

    const result = broadcaster.getBufferedEvents('session-1', -1);
    expect(result.events).toHaveLength(2);
    expect(result.events[0].type).toBe('ProcessingStarted');
  });

  test('stop unsubscribes from EventBus', () => {
    const { bus, emit, isUnsubscribed } = mockEventBus();
    const { router, sent } = mockRouter();
    const { buffer } = mockBuffer();

    const broadcaster = new EventBroadcaster(bus, router, buffer, 'session-1');
    broadcaster.start();
    broadcaster.subscribe('dev-1');

    broadcaster.stop();
    expect(isUnsubscribed()).toBe(true);

    // Events after stop should not be sent (handler is removed in EventBus)
    emit(processingStartedEvent());
    expect(sent).toHaveLength(0);
  });

  test('events not sent when no subscriptions', () => {
    const { bus, emit } = mockEventBus();
    const { router, sent } = mockRouter();
    const { buffer, pushed } = mockBuffer();

    const broadcaster = new EventBroadcaster(bus, router, buffer, 'session-1');
    broadcaster.start();

    emit(processingStartedEvent());

    // Buffered but not sent (no subscriptions)
    expect(pushed).toHaveLength(1);
    expect(sent).toHaveLength(0);
  });
});

// --- Serialization format tests ---

/** Helper: emit event through broadcaster, return the serialized event from buffer */
function emitAndCapture(event: DomainEvent): SerializedEvent {
  const { bus, emit } = mockEventBus();
  const { router } = mockRouter();
  const { buffer, pushed } = mockBuffer();

  const broadcaster = new EventBroadcaster(bus, router, buffer, 'session-1');
  broadcaster.start();
  emit(event);

  return pushed[0].event;
}

describe('serializeEvent — flat format for mobile', () => {
  test('MessageCompleted: flat content + agent_id', () => {
    const message = Message.createAssistantWithContent('Hello world', 'agent-1');
    const serialized = emitAndCapture({ type: 'MessageCompleted', message });

    expect(serialized.type).toBe('MessageCompleted');
    expect(serialized.content).toBe('Hello world');
    expect(serialized.agent_id).toBe('agent-1');
    // Must NOT have nested message object
    expect(serialized).not.toHaveProperty('message');
  });

  test('MessageCompleted: no agent_id when undefined', () => {
    const message = Message.createAssistantWithContent('No agent');
    const serialized = emitAndCapture({ type: 'MessageCompleted', message });

    expect(serialized.content).toBe('No agent');
    expect(serialized.agent_id).toBeUndefined();
  });

  test('ToolExecutionStarted: flat call_id, tool_name, arguments, agent_id', () => {
    const execution = ToolExecution.create(
      'call-42',
      'readFile',
      { path: '/tmp/test.ts' },
      MessageId.create(),
      'agent-2',
    );
    const serialized = emitAndCapture({ type: 'ToolExecutionStarted', execution });

    expect(serialized.type).toBe('ToolExecutionStarted');
    expect(serialized.call_id).toBe('call-42');
    expect(serialized.tool_name).toBe('readFile');
    expect(serialized.arguments).toEqual({ path: '/tmp/test.ts' });
    expect(serialized.agent_id).toBe('agent-2');
    // Must NOT have nested execution object
    expect(serialized).not.toHaveProperty('execution');
  });

  test('ToolExecutionCompleted: flat call_id, tool_name, result_summary, has_error', () => {
    const execution = ToolExecution.create(
      'call-43',
      'writeFile',
      { path: '/tmp/out.ts' },
      MessageId.create(),
      'agent-3',
    ).complete('Written 50 lines', '50 lines written');

    const serialized = emitAndCapture({ type: 'ToolExecutionCompleted', execution });

    expect(serialized.type).toBe('ToolExecutionCompleted');
    expect(serialized.call_id).toBe('call-43');
    expect(serialized.tool_name).toBe('writeFile');
    expect(serialized.result_summary).toBe('50 lines written');
    expect(serialized.has_error).toBe(false);
    expect(serialized.agent_id).toBe('agent-3');
    expect(serialized).not.toHaveProperty('execution');
  });

  test('ToolExecutionCompleted: has_error true when execution failed', () => {
    const execution = ToolExecution.create(
      'call-44',
      'readFile',
      { path: '/nope' },
      MessageId.create(),
    ).fail('File not found');

    const serialized = emitAndCapture({ type: 'ToolExecutionCompleted', execution });

    expect(serialized.has_error).toBe(true);
    expect(serialized.result_summary).toBe('');  // no summary on failure
  });

  test('AskUserRequested: flat question + options', () => {
    const serialized = emitAndCapture({
      type: 'AskUserRequested',
      questions: [
        {
          text: 'Continue?',
          options: [
            { label: 'Yes' },
            { label: 'No', description: 'Cancel operation' },
          ],
        },
      ],
    });

    expect(serialized.type).toBe('AskUserRequested');
    expect(serialized.question).toBe('Continue?');
    expect(serialized.options).toEqual(['Yes', 'No']);
    // Must NOT have questions array
    expect(serialized).not.toHaveProperty('questions');
  });

  test('AskUserRequested: empty questions array', () => {
    const serialized = emitAndCapture({
      type: 'AskUserRequested',
      questions: [],
    });

    expect(serialized.question).toBe('');
    expect(serialized.options).toEqual([]);
  });

  test('ErrorOccurred: type becomes Error, has code field', () => {
    const serialized = emitAndCapture({
      type: 'ErrorOccurred',
      error: new Error('Connection lost'),
      context: 'grpc',
    });

    expect(serialized.type).toBe('Error');
    expect(serialized.message).toBe('Connection lost');
    expect(serialized.code).toBe('error');
    // Must NOT have context (not expected by mobile)
    expect(serialized).not.toHaveProperty('context');
  });

  test('ProcessingStarted: includes state=processing', () => {
    const serialized = emitAndCapture({ type: 'ProcessingStarted' });

    expect(serialized.type).toBe('ProcessingStarted');
    expect(serialized.state).toBe('processing');
  });

  test('ProcessingStopped: includes state=idle', () => {
    const serialized = emitAndCapture({ type: 'ProcessingStopped' });

    expect(serialized.type).toBe('ProcessingStopped');
    expect(serialized.state).toBe('idle');
  });

  test('AgentLifecycle: unchanged format', () => {
    const serialized = emitAndCapture({
      type: 'AgentLifecycle',
      lifecycleType: 'agent_spawned',
      agentId: 'agent-5',
      description: 'New agent spawned',
    });

    expect(serialized.type).toBe('AgentLifecycle');
    expect(serialized.lifecycleType).toBe('agent_spawned');
    expect(serialized.agentId).toBe('agent-5');
    expect(serialized.description).toBe('New agent spawned');
  });

  test('AskUserResolved: unchanged format', () => {
    const serialized = emitAndCapture({ type: 'AskUserResolved' });
    expect(serialized.type).toBe('AskUserResolved');
  });

  test('MessageStarted: unchanged format', () => {
    const serialized = emitAndCapture({
      type: 'MessageStarted',
      messageId: 'msg-1',
      role: 'assistant',
    });

    expect(serialized.type).toBe('MessageStarted');
    expect(serialized.messageId).toBe('msg-1');
    expect(serialized.role).toBe('assistant');
  });
});
