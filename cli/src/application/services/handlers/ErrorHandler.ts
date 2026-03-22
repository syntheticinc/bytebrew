// ErrorHandler - handles stream errors
import { Message } from '../../../domain/entities/Message.js';
import { StreamResponse } from '../../../domain/ports/IStreamGateway.js';
import { StreamProcessorContext, stopProcessing } from './StreamProcessorContext.js';

/**
 * Handle stream error response
 */
export function handleStreamError(ctx: StreamProcessorContext, response: StreamResponse): void {
  if (response.error) {
    const errorMsg = `Error: ${response.error.message}`;
    const messageId = ctx.getCurrentMessageId();

    if (messageId) {
      ctx.accumulator.appendChunk(messageId, `\n\n${errorMsg}`);
      const message = ctx.accumulator.complete(messageId);
      if (message) {
        ctx.messageRepository.save(message);
        ctx.eventBus.publish({
          type: 'MessageCompleted',
          message,
        });
      }
      ctx.setCurrentMessageId(null);
    } else {
      const message = Message.createAssistantWithContent(errorMsg, ctx.agentId);
      ctx.messageRepository.save(message);
      ctx.eventBus.publish({
        type: 'MessageCompleted',
        message,
      });
    }

    ctx.eventBus.publish({
      type: 'ErrorOccurred',
      error: new Error(response.error.message),
    });
  }

  stopProcessing(ctx);
}

/**
 * Handle error from stream subscription
 */
export function handleError(ctx: StreamProcessorContext, error: Error): void {
  console.error('Stream error:', error.message);
  ctx.eventBus.publish({
    type: 'ErrorOccurred',
    error,
  });
  stopProcessing(ctx);
}
