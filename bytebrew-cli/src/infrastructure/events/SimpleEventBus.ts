// SimpleEventBus - in-memory event bus implementation
import {
  IEventBus,
  DomainEvent,
  DomainEventType,
  EventHandler,
} from '../../domain/ports/IEventBus.js';

type TypedHandler = {
  eventType: DomainEventType;
  handler: EventHandler<any>;
};

/**
 * Simple in-memory event bus implementation.
 * Synchronous event delivery for simplicity.
 */
export class SimpleEventBus implements IEventBus {
  private handlers: Map<DomainEventType, Set<EventHandler<any>>> = new Map();
  private allHandlers: Set<EventHandler> = new Set();

  /**
   * Publish an event to all subscribers
   */
  publish<T extends DomainEvent>(event: T): void {
    // Notify specific handlers
    const typeHandlers = this.handlers.get(event.type as DomainEventType);
    if (typeHandlers) {
      for (const handler of typeHandlers) {
        try {
          handler(event);
        } catch (error) {
          console.error(`Error in event handler for ${event.type}:`, error);
        }
      }
    }

    // Notify all-event handlers
    for (const handler of this.allHandlers) {
      try {
        handler(event);
      } catch (error) {
        console.error('Error in all-event handler:', error);
      }
    }
  }

  /**
   * Subscribe to events of a specific type
   */
  subscribe<T extends DomainEventType>(
    eventType: T,
    handler: EventHandler<Extract<DomainEvent, { type: T }>>
  ): () => void {
    let typeHandlers = this.handlers.get(eventType);
    if (!typeHandlers) {
      typeHandlers = new Set();
      this.handlers.set(eventType, typeHandlers);
    }
    typeHandlers.add(handler);

    // Return unsubscribe function
    return () => {
      typeHandlers?.delete(handler);
      if (typeHandlers?.size === 0) {
        this.handlers.delete(eventType);
      }
    };
  }

  /**
   * Subscribe to all events
   */
  subscribeAll(handler: EventHandler): () => void {
    this.allHandlers.add(handler);

    return () => {
      this.allHandlers.delete(handler);
    };
  }

  /**
   * Clear all subscribers
   */
  clear(): void {
    this.handlers.clear();
    this.allHandlers.clear();
  }
}

// Singleton instance for convenience
let defaultEventBus: SimpleEventBus | null = null;

export function getEventBus(): SimpleEventBus {
  if (!defaultEventBus) {
    defaultEventBus = new SimpleEventBus();
  }
  return defaultEventBus;
}

export function resetEventBus(): void {
  defaultEventBus?.clear();
  defaultEventBus = null;
}
