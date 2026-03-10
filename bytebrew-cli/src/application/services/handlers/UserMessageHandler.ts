// UserMessageHandler - handles user messages received from other clients (e.g. mobile)
import { Message } from '../../../domain/entities/Message.js';
import { StreamResponse } from '../../../domain/ports/IStreamGateway.js';
import { StreamProcessorContext } from './StreamProcessorContext.js';

/**
 * Handle USER_MESSAGE response from server.
 * These are messages sent by the user from another client (e.g. mobile app).
 * Always complete — no streaming needed.
 */
export function handleUserMessage(ctx: StreamProcessorContext, response: StreamResponse): void {
  const content = response.content?.trim();
  if (!content) return;

  const message = Message.createUser(content);
  ctx.messageRepository.save(message);
  ctx.eventBus.publish({
    type: 'MessageCompleted',
    message,
  });
}
