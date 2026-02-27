// IMessageRepository port - repository interface for message persistence
import { Message } from '../entities/Message.js';
import { MessageId } from '../value-objects/MessageId.js';

/**
 * Repository interface for message storage.
 * Follows Repository pattern from DDD.
 */
export interface IMessageRepository {
  /**
   * Save or update a message
   */
  save(message: Message): void;

  /**
   * Find message by ID
   */
  findById(id: MessageId): Message | undefined;

  /**
   * Find message by tool call ID
   */
  findByToolCallId(callId: string): Message | undefined;

  /**
   * Get all messages
   */
  findAll(): Message[];

  /**
   * Get only complete messages (for UI display)
   */
  findComplete(): Message[];

  /**
   * Get messages with limit (most recent)
   */
  findRecent(limit: number): Message[];

  /**
   * Delete a message
   */
  delete(id: MessageId): void;

  /**
   * Clear all messages
   */
  clear(): void;

  /**
   * Get count of messages
   */
  count(): number;

  /**
   * Subscribe to repository changes
   */
  subscribe(listener: (messages: Message[]) => void): () => void;
}
