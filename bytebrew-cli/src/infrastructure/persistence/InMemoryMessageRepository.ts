// InMemoryMessageRepository - in-memory implementation of message repository
import { Message } from '../../domain/entities/Message.js';
import { MessageId } from '../../domain/value-objects/MessageId.js';
import { IMessageRepository } from '../../domain/ports/IMessageRepository.js';

// Maximum number of messages to keep in memory
const MAX_MESSAGES = 500;

type ChangeListener = (messages: Message[]) => void;

/**
 * In-memory implementation of the message repository.
 * Maintains a list of messages and notifies subscribers on changes.
 */
export class InMemoryMessageRepository implements IMessageRepository {
  private messages: Map<string, Message> = new Map();
  private orderedIds: string[] = [];
  private listeners: Set<ChangeListener> = new Set();
  private toolCallIndex: Map<string, string> = new Map(); // callId -> messageId

  /**
   * Save or update a message
   */
  save(message: Message): void {
    const id = message.id.value;
    const isNew = !this.messages.has(id);

    // On update: remove stale tool call index entry before overwriting
    if (!isNew) {
      const existing = this.messages.get(id);
      if (existing?.toolCall) {
        this.toolCallIndex.delete(existing.toolCall.callId);
      }
    }

    this.messages.set(id, message);

    if (isNew) {
      this.orderedIds.push(id);
      this.pruneIfNeeded();
    }

    // Index tool calls for fast lookup
    if (message.toolCall) {
      this.toolCallIndex.set(message.toolCall.callId, id);
    }

    this.notifyListeners();
  }

  /**
   * Find message by ID
   */
  findById(id: MessageId): Message | undefined {
    return this.messages.get(id.value);
  }

  /**
   * Find message by tool call ID
   */
  findByToolCallId(callId: string): Message | undefined {
    const messageId = this.toolCallIndex.get(callId);
    if (!messageId) {
      return undefined;
    }
    return this.messages.get(messageId);
  }

  /**
   * Get all messages in order
   */
  findAll(): Message[] {
    return this.orderedIds
      .map(id => this.messages.get(id))
      .filter((m): m is Message => m !== undefined);
  }

  /**
   * Get only complete messages (for UI display)
   */
  findComplete(): Message[] {
    return this.findAll().filter(m => m.isComplete);
  }

  /**
   * Get messages with limit (most recent)
   */
  findRecent(limit: number): Message[] {
    const all = this.findAll();
    return all.slice(-limit);
  }

  /**
   * Delete a message
   */
  delete(id: MessageId): void {
    const message = this.messages.get(id.value);
    if (message) {
      // Remove from tool call index
      if (message.toolCall) {
        this.toolCallIndex.delete(message.toolCall.callId);
      }

      this.messages.delete(id.value);
      this.orderedIds = this.orderedIds.filter(i => i !== id.value);
      this.notifyListeners();
    }
  }

  /**
   * Clear all messages
   */
  clear(): void {
    this.messages.clear();
    this.orderedIds = [];
    this.toolCallIndex.clear();
    this.notifyListeners();
  }

  /**
   * Get count of messages
   */
  count(): number {
    return this.messages.size;
  }

  /**
   * Subscribe to repository changes
   */
  subscribe(listener: ChangeListener): () => void {
    this.listeners.add(listener);

    // Immediately notify with current state
    listener(this.findAll());

    return () => {
      this.listeners.delete(listener);
    };
  }

  /**
   * Prune old messages if over limit
   */
  private pruneIfNeeded(): void {
    while (this.orderedIds.length > MAX_MESSAGES) {
      const oldestId = this.orderedIds.shift();
      if (oldestId) {
        const message = this.messages.get(oldestId);
        if (message?.toolCall) {
          this.toolCallIndex.delete(message.toolCall.callId);
        }
        this.messages.delete(oldestId);
      }
    }
  }

  /**
   * Notify all listeners of changes
   */
  private notifyListeners(): void {
    const messages = this.findAll();
    for (const listener of this.listeners) {
      try {
        listener(messages);
      } catch (error) {
        console.error('Error in repository listener:', error);
      }
    }
  }
}

// Singleton instance for convenience
let defaultRepository: InMemoryMessageRepository | null = null;

export function getMessageRepository(): InMemoryMessageRepository {
  if (!defaultRepository) {
    defaultRepository = new InMemoryMessageRepository();
  }
  return defaultRepository;
}

export function resetMessageRepository(): void {
  defaultRepository?.clear();
  defaultRepository = null;
}
