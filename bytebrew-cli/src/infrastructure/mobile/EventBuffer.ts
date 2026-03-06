const DEFAULT_MAX_EVENTS = 1000;

export interface IEventBuffer<T = unknown> {
  push(sessionId: string, event: T): void;
  getAfter(
    sessionId: string,
    afterIndex: number,
  ): { events: T[]; lastIndex: number };
  clear(sessionId: string): void;
}

interface SessionBuffer<T> {
  events: T[];
  startIndex: number;
}

/**
 * Ring buffer for storing the last N events per session.
 * Used for backfill when a mobile client reconnects mid-session.
 *
 * Each event gets a global index within its session. When the buffer
 * exceeds maxEvents, oldest events are dropped and startIndex advances.
 * getAfter(afterIndex) returns all events with index > afterIndex.
 */
export class EventBuffer<T = unknown> implements IEventBuffer<T> {
  private readonly buffers = new Map<string, SessionBuffer<T>>();
  private readonly maxEvents: number;

  constructor(maxEvents: number = DEFAULT_MAX_EVENTS) {
    this.maxEvents = maxEvents;
  }

  push(sessionId: string, event: T): void {
    let buffer = this.buffers.get(sessionId);
    if (!buffer) {
      buffer = { events: [], startIndex: 0 };
      this.buffers.set(sessionId, buffer);
    }

    buffer.events.push(event);

    // Trim oldest events when exceeding capacity
    if (buffer.events.length > this.maxEvents) {
      const overflow = buffer.events.length - this.maxEvents;
      buffer.events.splice(0, overflow);
      buffer.startIndex += overflow;
    }
  }

  getAfter(
    sessionId: string,
    afterIndex: number,
  ): { events: T[]; lastIndex: number } {
    const buffer = this.buffers.get(sessionId);
    if (!buffer || buffer.events.length === 0) {
      return { events: [], lastIndex: afterIndex };
    }

    const currentLastIndex = buffer.startIndex + buffer.events.length - 1;

    if (afterIndex >= currentLastIndex) {
      return { events: [], lastIndex: currentLastIndex };
    }

    // Calculate the offset into the events array
    const startOffset = Math.max(0, afterIndex - buffer.startIndex + 1);
    const events = buffer.events.slice(startOffset);

    return { events, lastIndex: currentLastIndex };
  }

  clear(sessionId: string): void {
    this.buffers.delete(sessionId);
  }
}
