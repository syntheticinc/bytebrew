import { describe, it, expect, beforeEach } from 'bun:test';
import { SimpleEventBus } from '../SimpleEventBus.js';
import type {
  DomainEvent,
  MessageCompletedEvent,
  ProcessingStartedEvent,
} from '../../../domain/ports/IEventBus.js';
import { Message } from '../../../domain/entities/Message.js';

describe('SimpleEventBus', () => {
  let eventBus: SimpleEventBus;

  beforeEach(() => {
    eventBus = new SimpleEventBus();
  });

  it('publish delivers to type-specific subscribers', () => {
    const receivedEvents: DomainEvent[] = [];
    const handler = (event: ProcessingStartedEvent) => {
      receivedEvents.push(event);
    };

    eventBus.subscribe('ProcessingStarted', handler);

    const event: ProcessingStartedEvent = { type: 'ProcessingStarted' };
    eventBus.publish(event);

    expect(receivedEvents).toHaveLength(1);
    expect(receivedEvents[0]).toBe(event);
  });

  it('publish delivers to subscribeAll handlers', () => {
    const receivedEvents: DomainEvent[] = [];
    const handler = (event: DomainEvent) => {
      receivedEvents.push(event);
    };

    eventBus.subscribeAll(handler);

    const event1: ProcessingStartedEvent = { type: 'ProcessingStarted' };
    const event2: MessageCompletedEvent = {
      type: 'MessageCompleted',
      message: Message.createUser('test'),
    };

    eventBus.publish(event1);
    eventBus.publish(event2);

    expect(receivedEvents).toHaveLength(2);
    expect(receivedEvents[0]).toBe(event1);
    expect(receivedEvents[1]).toBe(event2);
  });

  it('subscribe returns working unsubscribe function', () => {
    const receivedEvents: DomainEvent[] = [];
    const handler = (event: ProcessingStartedEvent) => {
      receivedEvents.push(event);
    };

    const unsubscribe = eventBus.subscribe('ProcessingStarted', handler);

    eventBus.publish({ type: 'ProcessingStarted' });
    expect(receivedEvents).toHaveLength(1);

    unsubscribe();

    eventBus.publish({ type: 'ProcessingStarted' });
    expect(receivedEvents).toHaveLength(1); // No new event
  });

  it('multiple subscribers for same type all receive event', () => {
    const received1: DomainEvent[] = [];
    const received2: DomainEvent[] = [];
    const received3: DomainEvent[] = [];

    eventBus.subscribe('ProcessingStarted', (e) => received1.push(e));
    eventBus.subscribe('ProcessingStarted', (e) => received2.push(e));
    eventBus.subscribe('ProcessingStarted', (e) => received3.push(e));

    const event: ProcessingStartedEvent = { type: 'ProcessingStarted' };
    eventBus.publish(event);

    expect(received1).toHaveLength(1);
    expect(received2).toHaveLength(1);
    expect(received3).toHaveLength(1);
    expect(received1[0]).toBe(event);
    expect(received2[0]).toBe(event);
    expect(received3[0]).toBe(event);
  });

  it('unsubscribe removes only that handler', () => {
    const received1: DomainEvent[] = [];
    const received2: DomainEvent[] = [];

    const unsub1 = eventBus.subscribe('ProcessingStarted', (e) =>
      received1.push(e)
    );
    eventBus.subscribe('ProcessingStarted', (e) => received2.push(e));

    eventBus.publish({ type: 'ProcessingStarted' });
    expect(received1).toHaveLength(1);
    expect(received2).toHaveLength(1);

    unsub1();

    eventBus.publish({ type: 'ProcessingStarted' });
    expect(received1).toHaveLength(1); // No new event
    expect(received2).toHaveLength(2); // Got the event
  });

  it('subscribeAll receives all event types', () => {
    const receivedEvents: DomainEvent[] = [];

    eventBus.subscribeAll((e) => receivedEvents.push(e));

    eventBus.publish({ type: 'ProcessingStarted' });
    eventBus.publish({ type: 'ProcessingStopped' });
    eventBus.publish({ type: 'ErrorOccurred', error: new Error('test') });

    expect(receivedEvents).toHaveLength(3);
    expect(receivedEvents[0].type).toBe('ProcessingStarted');
    expect(receivedEvents[1].type).toBe('ProcessingStopped');
    expect(receivedEvents[2].type).toBe('ErrorOccurred');
  });

  it('no delivery after unsubscribe', () => {
    const received: DomainEvent[] = [];

    const unsub = eventBus.subscribe('ProcessingStarted', (e) =>
      received.push(e)
    );

    eventBus.publish({ type: 'ProcessingStarted' });
    expect(received).toHaveLength(1);

    unsub();

    eventBus.publish({ type: 'ProcessingStarted' });
    eventBus.publish({ type: 'ProcessingStarted' });
    expect(received).toHaveLength(1); // Still 1
  });

  it('handler error does not break other handlers', () => {
    const received1: DomainEvent[] = [];
    const received2: DomainEvent[] = [];

    eventBus.subscribe('ProcessingStarted', () => {
      throw new Error('Handler 1 failed');
    });
    eventBus.subscribe('ProcessingStarted', (e) => received1.push(e));
    eventBus.subscribeAll((e) => received2.push(e));

    const event: ProcessingStartedEvent = { type: 'ProcessingStarted' };

    // Should not throw
    expect(() => eventBus.publish(event)).not.toThrow();

    // Other handlers still received the event
    expect(received1).toHaveLength(1);
    expect(received2).toHaveLength(1);
  });

  it('clear removes all handlers', () => {
    const received1: DomainEvent[] = [];
    const received2: DomainEvent[] = [];

    eventBus.subscribe('ProcessingStarted', (e) => received1.push(e));
    eventBus.subscribeAll((e) => received2.push(e));

    eventBus.publish({ type: 'ProcessingStarted' });
    expect(received1).toHaveLength(1);
    expect(received2).toHaveLength(1);

    eventBus.clear();

    eventBus.publish({ type: 'ProcessingStarted' });
    expect(received1).toHaveLength(1); // No new events
    expect(received2).toHaveLength(1);
  });

  it('empty handlers do not throw on publish', () => {
    // Should not throw
    expect(() => eventBus.publish({ type: 'ProcessingStarted' })).not.toThrow();
    expect(() => eventBus.publish({ type: 'ProcessingStopped' })).not.toThrow();
  });

  it('nested publish (handler publishes another event)', () => {
    const received1: DomainEvent[] = [];
    const received2: DomainEvent[] = [];

    // Handler for ProcessingStarted publishes ProcessingCompleted
    eventBus.subscribe('ProcessingStarted', () => {
      eventBus.publish({ type: 'ProcessingStopped' });
    });

    eventBus.subscribe('ProcessingStarted', (e) => received1.push(e));
    eventBus.subscribe('ProcessingStopped', (e) => received2.push(e));

    eventBus.publish({ type: 'ProcessingStarted' });

    expect(received1).toHaveLength(1);
    expect(received2).toHaveLength(1); // Nested event delivered
  });

  it('type-safe handler receives correct event type', () => {
    let receivedMessage: Message | undefined;

    const handler = (event: MessageCompletedEvent) => {
      receivedMessage = event.message;
    };

    eventBus.subscribe('MessageCompleted', handler);

    const message = Message.createUser('test content');
    eventBus.publish({ type: 'MessageCompleted', message });

    expect(receivedMessage).toBe(message);
    expect(receivedMessage?.content.value).toBe('test content');
  });
});
