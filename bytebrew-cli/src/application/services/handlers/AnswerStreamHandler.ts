// AnswerStreamHandler - handles answer and answer_chunk responses
import { Message } from '../../../domain/entities/Message.js';
import { StreamResponse } from '../../../domain/ports/IStreamGateway.js';
import { StreamProcessorContext, stopProcessing } from './StreamProcessorContext.js';

/**
 * Handle answer chunk (streaming content)
 */
export function handleAnswerChunk(ctx: StreamProcessorContext, response: StreamResponse): void {
  let messageId = ctx.getCurrentMessageId();

  if (!messageId) {
    messageId = ctx.accumulator.startAccumulating('assistant', ctx.agentId);
    ctx.setCurrentMessageId(messageId);
    ctx.eventBus.publish({
      type: 'MessageStarted',
      messageId: messageId.value,
      role: 'assistant',
    });
  }

  const tokensAdded = ctx.accumulator.appendChunk(
    messageId,
    response.content
  );

  // Publish progress for StatusBar
  ctx.eventBus.publish({
    type: 'StreamingProgress',
    messageId: messageId.value,
    tokensAdded,
    totalTokens: ctx.accumulator.getTokenCounts(),
  });
}

/**
 * Handle ANSWER response from server.
 *
 * Two scenarios:
 *
 * 1. Streaming mode (normal): Server sends ANSWER_CHUNKs, then FinalizeAccumulatedText
 *    emits ANSWER with the same accumulated content (IsComplete=false, isFinal=false).
 *    - If messageId exists → chunks still accumulating → complete them
 *    - If messageId is null → chunks already completed by completeCurrentMessage
 *      (TOOL_CALL arrived first) → SKIP to prevent duplicate
 *
 * 2. Non-streaming mode (code agents): Server sends ANSWER with content.
 *    No preceding ANSWER_CHUNKs. messageId is null → create message directly.
 *    Handles both intermediate text (FinalizeAccumulatedText, isFinal=false)
 *    and final answers (isFinal=true).
 */
export function handleAnswer(ctx: StreamProcessorContext, response: StreamResponse): void {
  const messageId = ctx.getCurrentMessageId();

  if (messageId) {
    // Normal path: ANSWER follows ANSWER_CHUNKs → complete accumulated message
    const message = ctx.accumulator.complete(messageId);
    if (message && message.content.value.trim()) {
      ctx.messageRepository.save(message);
      ctx.eventBus.publish({
        type: 'MessageCompleted',
        message,
      });
    }
    ctx.setCurrentMessageId(null);
  } else if (response.content?.trim() && (response.isFinal || isNonSupervisorAgent(ctx.agentId))) {
    // Non-streaming path: create message directly (no preceding ANSWER_CHUNKs).
    //
    // Two cases reach here:
    // - isFinal=true: genuine standalone answer (any agent)
    // - Code agent (non-supervisor): intermediate text from non-streaming mode
    //   (FinalizeAccumulatedText sends ANSWER with isFinal=false)
    //
    // Supervisor streaming mode with messageId=null means chunks were already
    // completed by completeCurrentMessage (TOOL_CALL) → skip to prevent duplicate.
    const message = Message.createAssistantWithContent(response.content, ctx.agentId);
    ctx.messageRepository.save(message);
    ctx.eventBus.publish({
      type: 'MessageCompleted',
      message,
    });
  }

  // Only stop processing for supervisor's final answer (not code agents)
  if (response.isFinal && (!ctx.agentId || ctx.agentId === 'supervisor')) {
    stopProcessing(ctx);
  }
}

/** Code agents have explicit agentId that is not 'supervisor' */
function isNonSupervisorAgent(agentId: string | undefined): boolean {
  return !!agentId && agentId !== 'supervisor';
}
