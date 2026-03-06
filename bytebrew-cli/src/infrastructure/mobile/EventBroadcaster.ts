// EventBroadcaster - subscribes to IEventBus, converts DomainEvents to MobileMessages,
// and broadcasts to subscribed mobile devices via BridgeMessageRouter.

import { v4 as uuidv4 } from 'uuid';
import { getLogger, type Logger } from '../../lib/logger.js';
import type {
  IEventBus,
  DomainEvent,
} from '../../domain/ports/IEventBus.js';
import type { IBridgeMessageRouter, MobileMessage } from '../bridge/BridgeMessageRouter.js';
import type { IEventBuffer } from './EventBuffer.js';

/** Serialized event payload sent to mobile clients */
export interface SerializedEvent {
  type: string;
  [key: string]: unknown;
}

/** Subscription entry: a device subscribed to a specific session's events */
interface DeviceSubscription {
  deviceId: string;
  sessionId: string | undefined; // undefined = all sessions
}

/**
 * Subscribes to the CLI EventBus, converts DomainEvents into MobileMessage
 * format, and sends them to all subscribed mobile devices through the
 * BridgeMessageRouter. Buffers events via IEventBuffer for reconnect backfill.
 */
export class EventBroadcaster {
  private readonly logger: Logger;
  private readonly subscriptions = new Map<string, DeviceSubscription>(); // deviceId -> subscription
  private unsubscribeEventBus: (() => void) | null = null;

  constructor(
    private readonly eventBus: IEventBus,
    private readonly router: IBridgeMessageRouter,
    private readonly eventBuffer: IEventBuffer<SerializedEvent>,
    private readonly sessionId: string,
  ) {
    this.logger = getLogger().child({ component: 'EventBroadcaster' });
  }

  /** Subscribe to all domain events and start broadcasting */
  start(): void {
    if (this.unsubscribeEventBus) {
      this.logger.warn('EventBroadcaster already started');
      return;
    }

    this.unsubscribeEventBus = this.eventBus.subscribeAll((event) => {
      this.handleDomainEvent(event);
    });

    this.logger.info('EventBroadcaster started');
  }

  /** Unsubscribe from all domain events */
  stop(): void {
    if (this.unsubscribeEventBus) {
      this.unsubscribeEventBus();
      this.unsubscribeEventBus = null;
    }

    this.subscriptions.clear();
    this.logger.info('EventBroadcaster stopped');
  }

  /** Subscribe a mobile device to receive events */
  subscribe(deviceId: string, sessionId?: string): void {
    this.subscriptions.set(deviceId, { deviceId, sessionId });
    this.logger.info('Device subscribed', { deviceId, sessionId });
  }

  /** Unsubscribe a mobile device */
  unsubscribe(deviceId: string): void {
    if (this.subscriptions.delete(deviceId)) {
      this.logger.info('Device unsubscribed', { deviceId });
    }
  }

  /** Get buffered events for backfill on reconnect */
  getBufferedEvents(
    sessionId: string,
    afterIndex: number,
  ): { events: SerializedEvent[]; lastIndex: number } {
    return this.eventBuffer.getAfter(sessionId, afterIndex);
  }

  // --- Private ---

  private handleDomainEvent(event: DomainEvent): void {
    const serialized = serializeEvent(event);

    // Buffer the event for potential backfill
    this.eventBuffer.push(this.sessionId, serialized);

    // Broadcast to all subscribed devices
    if (this.subscriptions.size === 0) return;

    for (const [deviceId, sub] of this.subscriptions) {
      // If device subscribed to a specific session, filter by it
      if (sub.sessionId && sub.sessionId !== this.sessionId) {
        continue;
      }

      const message: MobileMessage = {
        type: 'session_event',
        request_id: uuidv4(),
        device_id: deviceId,
        payload: {
          session_id: this.sessionId,
          event: serialized,
        },
      };

      try {
        this.router.sendMessage(deviceId, message);
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : String(err);
        this.logger.error('Failed to broadcast event to device', {
          deviceId,
          eventType: event.type,
          error: errorMessage,
        });
      }
    }
  }
}

/**
 * Converts a DomainEvent into a serializable plain object for mobile clients.
 * Mirrors the mapping from MobileProxyServer.serializeEvent.
 */
function serializeEvent(event: DomainEvent): SerializedEvent {
  switch (event.type) {
    case 'MessageCompleted':
      return {
        type: event.type,
        message: event.message.toSnapshot(),
      };

    case 'MessageStarted':
      return {
        type: event.type,
        messageId: event.messageId,
        role: event.role,
      };

    case 'ToolExecutionStarted':
    case 'ToolExecutionCompleted':
      return {
        type: event.type,
        execution: event.execution.toSnapshot(),
      };

    case 'AskUserRequested':
      return {
        type: event.type,
        questions: event.questions,
      };

    case 'StreamingProgress':
      return {
        type: event.type,
        messageId: event.messageId,
        tokensAdded: event.tokensAdded,
        totalTokens: event.totalTokens,
      };

    case 'ProcessingStarted':
    case 'ProcessingStopped':
      return { type: event.type };

    case 'ErrorOccurred':
      return {
        type: event.type,
        message: event.error.message,
        context: event.context,
      };

    case 'AgentLifecycle':
      return {
        type: event.type,
        lifecycleType: event.lifecycleType,
        agentId: event.agentId,
        description: event.description,
      };

    case 'AskUserResolved':
      return { type: event.type };
  }
}
