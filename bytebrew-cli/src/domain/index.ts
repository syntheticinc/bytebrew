// Domain layer exports

// Entities
export { Message, type MessageRole, type ReasoningInfo, type ToolCallInfo, type ToolResultInfo } from './entities/Message.js';
export { ToolExecution, type ToolExecutionStatus } from './entities/ToolExecution.js';

// Value Objects
export { MessageId } from './value-objects/MessageId.js';
export { MessageContent } from './value-objects/MessageContent.js';
export { StreamingState, type StreamingStatus } from './value-objects/StreamingState.js';

// Ports (Interfaces)
export { type IMessageRepository } from './ports/IMessageRepository.js';
export { type IStreamGateway, type StreamResponse, type StreamResponseType, type StreamConnectionOptions, type ConnectionStatus } from './ports/IStreamGateway.js';
export { type IToolExecutor, type ToolExecutionResult } from './ports/IToolExecutor.js';
export {
  type IEventBus,
  type DomainEvent,
  type DomainEventType,
  type EventHandler,
  type MessageCompletedEvent,
  type MessageStartedEvent,
  type StreamingProgressEvent,
  type ToolExecutionStartedEvent,
  type ToolExecutionCompletedEvent,
  type ProcessingStartedEvent,
  type ProcessingStoppedEvent,
  type ErrorOccurredEvent,
} from './ports/IEventBus.js';

// Legacy types (for backward compatibility during migration)
export { ResponseTypeEnum, type ResponseType } from './message.js';
export type { ChatMessage } from './message.js';
