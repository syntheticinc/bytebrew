import { describe, test, expect } from 'bun:test';
import { EventBroadcaster, type SerializedEvent } from '../EventBroadcaster';
import type { IEventBus, DomainEvent, EventHandler } from '../../../domain/ports/IEventBus';
import type { IBridgeMessageRouter, MobileMessage } from '../../bridge/BridgeMessageRouter';
import type { IEventBuffer } from '../EventBuffer';

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
