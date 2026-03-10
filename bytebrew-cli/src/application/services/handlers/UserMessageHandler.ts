// UserMessageHandler - handles user messages received from other clients (e.g. mobile)
import { Message } from '../../../domain/entities/Message.js';
import { StreamResponse } from '../../../domain/ports/IStreamGateway.js';
import { StreamProcessorContext } from './StreamProcessorContext.js';

/**
 * Handle USER_MESSAGE response from server.
 * These are messages sent by the user from another client (e.g. mobile app)
 * or replayed from backfill history.
 *
 * Deduplicates against existing messages: if a user message with the same
 * content already exists (added optimistically by this CLI), skip it.
 */
export function handleUserMessage(ctx: StreamProcessorContext, response: StreamResponse): void {
  const content = response.content?.trim();
  if (!content) return;

  // Check if this user message already exists (optimistic local add or previous backfill).
  const existing = ctx.messageRepository.findAll();
  const duplicate = existing.some(
    (m) => m.isUser && m.content.toString() === content,
  );
  if (duplicate) return;

  const message = Message.createUser(content);
  ctx.messageRepository.save(message);
  ctx.eventBus.publish({
    type: 'MessageCompleted',
    message,
  });
}
