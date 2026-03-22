// UserMessageHandler - handles user messages received from other clients (e.g. mobile)
import { Message } from '../../../domain/entities/Message.js';
import { StreamResponse } from '../../../domain/ports/IStreamGateway.js';
import { StreamProcessorContext } from './StreamProcessorContext.js';

/**
 * Handle USER_MESSAGE response from server.
 *
 * The server echoes every user message back as a USER_MESSAGE event
 * (needed for backfill and multi-client sync). If this CLI already
 * saved the message locally (via executeSend), consumeSentMessage()
 * returns true and we skip the echo to avoid duplicates.
 */
export function handleUserMessage(ctx: StreamProcessorContext, response: StreamResponse): void {
  const content = response.content?.trim();
  if (!content) return;

  // This client sent this message — already saved in executeSend(), skip echo
  if (ctx.consumeSentMessage(content)) return;

  // Message from another client (mobile) or backfill — save it
  const message = Message.createUser(content);
  ctx.messageRepository.save(message);
  ctx.eventBus.publish({
    type: 'MessageCompleted',
    message,
  });
}
