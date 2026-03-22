// MessageViewMapper - maps domain entities to view models
import { Message, ToolCallInfo, ToolResultInfo, ReasoningInfo } from '../../domain/entities/Message.js';

/**
 * View model for messages - compatible with existing ChatMessage interface
 * for smooth migration.
 */
export interface MessageViewModel {
  id: string;
  role: 'user' | 'assistant' | 'system' | 'tool';
  content: string;
  timestamp: Date;
  isStreaming: boolean;
  isComplete: boolean;
  toolCall?: ToolCallInfo;
  toolResult?: ToolResultInfo;
  reasoning?: ReasoningInfo;
  agentId?: string;
}

/**
 * Maps a Message entity to a MessageViewModel for UI display.
 * This ensures the presentation layer remains decoupled from the domain.
 */
export function toMessageViewModel(message: Message): MessageViewModel {
  const snapshot = message.toSnapshot();
  return {
    id: snapshot.id,
    role: snapshot.role,
    content: snapshot.content,
    timestamp: snapshot.timestamp,
    isStreaming: snapshot.isStreaming,
    isComplete: snapshot.isComplete,
    toolCall: snapshot.toolCall,
    toolResult: snapshot.toolResult,
    reasoning: snapshot.reasoning,
    agentId: snapshot.agentId,
  };
}

/**
 * Maps an array of Message entities to MessageViewModels.
 */
export function toMessageViewModels(messages: Message[]): MessageViewModel[] {
  return messages.map(toMessageViewModel);
}

/**
 * Creates a MessageViewModel from raw data (for testing or direct creation).
 */
export function createMessageViewModel(data: Partial<MessageViewModel> & { id: string; role: MessageViewModel['role'] }): MessageViewModel {
  return {
    id: data.id,
    role: data.role,
    content: data.content ?? '',
    timestamp: data.timestamp ?? new Date(),
    isStreaming: data.isStreaming ?? false,
    isComplete: data.isComplete ?? true,
    toolCall: data.toolCall,
    toolResult: data.toolResult,
    reasoning: data.reasoning,
  };
}
