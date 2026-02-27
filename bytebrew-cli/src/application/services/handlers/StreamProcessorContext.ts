// StreamProcessorContext - shared context for stream handlers
import { MessageId } from '../../../domain/value-objects/MessageId.js';
import { IMessageRepository } from '../../../domain/ports/IMessageRepository.js';
import { IStreamGateway } from '../../../domain/ports/IStreamGateway.js';
import { IToolExecutor } from '../../../domain/ports/IToolExecutor.js';
import { IEventBus } from '../../../domain/ports/IEventBus.js';
import { MessageAccumulatorService } from '../MessageAccumulatorService.js';

// Re-export from shared for backwards compatibility
export { ResponseTypeMap } from '../../../shared/grpcConstants.js';

/**
 * Shared context for all stream handlers.
 * Contains state and dependencies needed across handlers.
 */
export interface StreamProcessorContext {
  // Dependencies
  streamGateway: IStreamGateway;
  messageRepository: IMessageRepository;
  toolExecutor: IToolExecutor;
  accumulator: MessageAccumulatorService;
  eventBus: IEventBus;

  // Mutable state (accessed via getters/setters to ensure fresh values)
  getCurrentMessageId(): MessageId | null;
  getCurrentReasoningId(): MessageId | null;
  getIsProcessing(): boolean;

  // State setters
  setCurrentMessageId(id: MessageId | null): void;
  setCurrentReasoningId(id: MessageId | null): void;
  setIsProcessing(value: boolean): void;

  // Agent identification (for multi-agent routing)
  agentId?: string;
}

/**
 * Helper to complete current message and save to repository.
 * Skips saving messages with empty/whitespace-only content to prevent
 * ghost messages (e.g., when TOOL_CALL arrives before any text was accumulated).
 */
export function completeCurrentMessage(ctx: StreamProcessorContext): void {
  const messageId = ctx.getCurrentMessageId();
  if (messageId) {
    const message = ctx.accumulator.complete(messageId);
    if (message && message.content.value.trim()) {
      ctx.messageRepository.save(message);
      ctx.eventBus.publish({
        type: 'MessageCompleted',
        message,
      });
    }
    ctx.setCurrentMessageId(null);
  }
}

/**
 * Helper to stop processing
 */
export function stopProcessing(ctx: StreamProcessorContext): void {
  if (ctx.getIsProcessing()) {
    ctx.setIsProcessing(false);
    ctx.eventBus.publish({ type: 'ProcessingStopped' });
  }
}
