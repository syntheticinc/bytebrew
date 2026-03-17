// MessageAccumulatorService - accumulates streaming content without UI updates
import { MessageId } from '../../domain/value-objects/MessageId.js';
import { Message, MessageRole, ReasoningInfo } from '../../domain/entities/Message.js';

interface AccumulatingMessage {
  id: MessageId;
  role: MessageRole;
  chunks: string[];
  reasoning?: ReasoningInfo;
  startedAt: Date;
  agentId?: string;
}

/**
 * Service that accumulates streaming message content in memory
 * WITHOUT triggering any UI updates. The UI only sees messages
 * when they are complete.
 *
 * This is the KEY component for static rendering - messages
 * are invisible to React until complete() is called.
 */
export class MessageAccumulatorService {
  private accumulating: Map<string, AccumulatingMessage> = new Map();
  private tokenCounts = { input: 0, output: 0 };

  /**
   * Start accumulating a new message
   */
  startAccumulating(role: 'assistant' | 'reasoning', agentId?: string): MessageId {
    const id = MessageId.create();
    this.accumulating.set(id.value, {
      id,
      role: role === 'reasoning' ? 'assistant' : role,
      chunks: [],
      reasoning: role === 'reasoning' ? { thinking: '', isComplete: false } : undefined,
      startedAt: new Date(),
      agentId,
    });
    return id;
  }

  /**
   * Append a chunk to an accumulating message
   * Returns approximate tokens added
   */
  appendChunk(messageId: MessageId, chunk: string): number {
    const acc = this.accumulating.get(messageId.value);
    if (!acc) {
      return 0;
    }

    acc.chunks.push(chunk);

    // Update reasoning if this is a reasoning message
    if (acc.reasoning) {
      acc.reasoning = {
        thinking: acc.chunks.join(''),
        isComplete: false,
      };
    }

    // Calculate approximate tokens (4 chars per token)
    const tokensAdded = Math.ceil(chunk.length / 4);
    this.tokenCounts.output += tokensAdded;

    return tokensAdded;
  }

  /**
   * Update reasoning content directly (for REASONING response type)
   */
  updateReasoning(messageId: MessageId, thinking: string, isComplete: boolean): void {
    const acc = this.accumulating.get(messageId.value);
    if (!acc || !acc.reasoning) {
      return;
    }

    // Calculate tokens delta
    const prevLength = acc.reasoning.thinking.length;
    const deltaTokens = Math.ceil((thinking.length - prevLength) / 4);
    if (deltaTokens > 0) {
      this.tokenCounts.output += deltaTokens;
    }

    acc.reasoning = { thinking, isComplete };
    acc.chunks = [thinking]; // Replace chunks with full content
  }

  /**
   * Complete an accumulating message and return the final Message entity
   */
  complete(messageId: MessageId): Message | null {
    const acc = this.accumulating.get(messageId.value);
    if (!acc) {
      return null;
    }

    this.accumulating.delete(messageId.value);

    const content = acc.chunks.join('');

    if (acc.reasoning) {
      return Message.createReasoning(content, true, acc.agentId);
    }

    return Message.createAssistantWithContent(content, acc.agentId);
  }

  /**
   * Complete a reasoning message with final thinking content
   */
  completeReasoning(messageId: MessageId, thinking: string): Message | null {
    const acc = this.accumulating.get(messageId.value);
    if (!acc) {
      return null;
    }

    this.accumulating.delete(messageId.value);

    return Message.createReasoning(thinking, true, acc.agentId);
  }

  /**
   * Abort an accumulating message
   */
  abort(messageId: MessageId): void {
    this.accumulating.delete(messageId.value);
  }

  /**
   * Check if a message is being accumulated
   */
  isAccumulating(messageId: MessageId): boolean {
    return this.accumulating.has(messageId.value);
  }

  /**
   * Get current content of an accumulating message (for debugging)
   */
  getCurrentContent(messageId: MessageId): string | null {
    const acc = this.accumulating.get(messageId.value);
    if (!acc) {
      return null;
    }
    return acc.chunks.join('');
  }

  /**
   * Get token counts
   */
  getTokenCounts(): { input: number; output: number } {
    return { ...this.tokenCounts };
  }

  /**
   * Add input tokens (when user sends a message)
   */
  addInputTokens(content: string): void {
    this.tokenCounts.input = Math.ceil(content.length / 4);
    this.tokenCounts.output = 0;
  }

  /**
   * Reset token counts
   */
  resetTokenCounts(): void {
    this.tokenCounts = { input: 0, output: 0 };
  }

  /**
   * Clear all accumulating messages
   */
  clear(): void {
    this.accumulating.clear();
    this.tokenCounts = { input: 0, output: 0 };
  }

  /**
   * Get count of accumulating messages
   */
  getAccumulatingCount(): number {
    return this.accumulating.size;
  }
}
