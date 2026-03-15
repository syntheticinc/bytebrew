// IEventBus port - interface for event-driven communication
import { Message } from '../entities/Message.js';
import { ToolExecution } from '../entities/ToolExecution.js';
import type { Question, QuestionAnswer } from '../../tools/askUser.js';

// Agent lifecycle event types matching server-side domain.AgentEventType
export type AgentLifecycleType =
  | 'agent_spawned'
  | 'agent_completed'
  | 'agent_failed'
  | 'agent_restarted';

// Event types for the event bus
export type DomainEventType =
  | 'MessageCompleted'
  | 'MessageStarted'
  | 'StreamingProgress'
  | 'ToolExecutionStarted'
  | 'ToolExecutionCompleted'
  | 'ProcessingStarted'
  | 'ProcessingStopped'
  | 'ErrorOccurred'
  | 'AgentLifecycle'
  | 'AskUserRequested'
  | 'AskUserResolved';

export interface MessageCompletedEvent {
  type: 'MessageCompleted';
  message: Message;
}

export interface MessageStartedEvent {
  type: 'MessageStarted';
  messageId: string;
  role: string;
}

export interface StreamingProgressEvent {
  type: 'StreamingProgress';
  messageId: string;
  tokensAdded: number;
  totalTokens: { input: number; output: number };
}

export interface ToolExecutionStartedEvent {
  type: 'ToolExecutionStarted';
  execution: ToolExecution;
}

export interface ToolExecutionCompletedEvent {
  type: 'ToolExecutionCompleted';
  execution: ToolExecution;
}

export interface ProcessingStartedEvent {
  type: 'ProcessingStarted';
}

export interface ProcessingStoppedEvent {
  type: 'ProcessingStopped';
}

export interface ErrorOccurredEvent {
  type: 'ErrorOccurred';
  error: Error;
  context?: string;
}

export interface AgentLifecycleEvent {
  type: 'AgentLifecycle';
  lifecycleType: AgentLifecycleType;
  agentId: string;
  description: string;
}

export interface AskUserRequestedEvent {
  type: 'AskUserRequested';
  questions: Question[];
  callId?: string;
}

export interface AskUserResolvedEvent {
  type: 'AskUserResolved';
  callId?: string;
  answers?: QuestionAnswer[];
}

export type DomainEvent =
  | MessageCompletedEvent
  | MessageStartedEvent
  | StreamingProgressEvent
  | ToolExecutionStartedEvent
  | ToolExecutionCompletedEvent
  | ProcessingStartedEvent
  | ProcessingStoppedEvent
  | ErrorOccurredEvent
  | AgentLifecycleEvent
  | AskUserRequestedEvent
  | AskUserResolvedEvent;

export type EventHandler<T extends DomainEvent = DomainEvent> = (event: T) => void;

/**
 * Event bus interface for decoupled communication between layers.
 * Follows the Observer pattern.
 */
export interface IEventBus {
  /**
   * Publish an event to all subscribers
   */
  publish<T extends DomainEvent>(event: T): void;

  /**
   * Subscribe to events of a specific type
   * Returns unsubscribe function
   */
  subscribe<T extends DomainEventType>(
    eventType: T,
    handler: EventHandler<Extract<DomainEvent, { type: T }>>
  ): () => void;

  /**
   * Subscribe to all events
   * Returns unsubscribe function
   */
  subscribeAll(handler: EventHandler): () => void;

  /**
   * Clear all subscribers
   */
  clear(): void;
}
