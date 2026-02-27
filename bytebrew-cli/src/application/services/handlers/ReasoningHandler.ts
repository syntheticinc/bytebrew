// ReasoningHandler - handles reasoning/thinking responses
import { StreamResponse } from '../../../domain/ports/IStreamGateway.js';
import { StreamProcessorContext, completeCurrentMessage } from './StreamProcessorContext.js';

/**
 * Handle reasoning/thinking content
 */
export function handleReasoning(ctx: StreamProcessorContext, response: StreamResponse): void {
  if (!response.reasoning) {
    return;
  }

  // Complete current answer message before reasoning
  completeCurrentMessage(ctx);

  let reasoningId = ctx.getCurrentReasoningId();

  if (!reasoningId) {
    reasoningId = ctx.accumulator.startAccumulating('reasoning', ctx.agentId);
    ctx.setCurrentReasoningId(reasoningId);
    ctx.eventBus.publish({
      type: 'MessageStarted',
      messageId: reasoningId.value,
      role: 'reasoning',
    });
  }

  ctx.accumulator.updateReasoning(
    reasoningId,
    response.reasoning.thinking,
    response.reasoning.isComplete
  );

  // Publish progress
  ctx.eventBus.publish({
    type: 'StreamingProgress',
    messageId: reasoningId.value,
    tokensAdded: 0,
    totalTokens: ctx.accumulator.getTokenCounts(),
  });

  if (response.reasoning.isComplete) {
    const message = ctx.accumulator.completeReasoning(
      reasoningId,
      response.reasoning.thinking
    );
    if (message) {
      ctx.messageRepository.save(message);
      ctx.eventBus.publish({
        type: 'MessageCompleted',
        message,
      });
    }
    ctx.setCurrentReasoningId(null);
  }
}
