// Message entity - represents a complete chat message with behavior
import { MessageId } from '../value-objects/MessageId.js';
import { MessageContent } from '../value-objects/MessageContent.js';
import { StreamingState } from '../value-objects/StreamingState.js';
import type { DiffLine } from '../message.js';

export type MessageRole = 'user' | 'assistant' | 'system' | 'tool';

export interface ReasoningInfo {
  thinking: string;
  isComplete: boolean;
}

// SubQuery for grouped tool operations (e.g., smart_search)
export interface SubQuery {
  type: string;    // "vector" | "grep" | "symbol"
  query: string;   // Search query or pattern
  limit: number;   // Max results
}

export interface ToolCallInfo {
  callId: string;
  toolName: string;
  arguments: Record<string, string>;
  subQueries?: SubQuery[];  // Sub-queries for grouped operations
  agentId?: string;  // Agent that triggered this tool call
}

export interface ToolResultInfo {
  callId: string;
  toolName: string;
  result: string;
  error?: string;
  summary?: string;
  diffLines?: DiffLine[];
}

export interface MessageProps {
  id: MessageId;
  role: MessageRole;
  content: MessageContent;
  timestamp: Date;
  streamingState: StreamingState;
  toolCall?: ToolCallInfo;
  toolResult?: ToolResultInfo;
  reasoning?: ReasoningInfo;
  agentId?: string;
}

/**
 * Message entity - immutable, with behavior methods that return new instances.
 * Follows DDD principles for rich domain model.
 */
export class Message {
  private readonly _id: MessageId;
  private readonly _role: MessageRole;
  private readonly _content: MessageContent;
  private readonly _timestamp: Date;
  private readonly _streamingState: StreamingState;
  private readonly _toolCall?: ToolCallInfo;
  private readonly _toolResult?: ToolResultInfo;
  private readonly _reasoning?: ReasoningInfo;
  private readonly _agentId?: string;

  private constructor(props: MessageProps) {
    this._id = props.id;
    this._role = props.role;
    this._content = props.content;
    this._timestamp = props.timestamp;
    this._streamingState = props.streamingState;
    this._toolCall = props.toolCall;
    this._toolResult = props.toolResult;
    this._reasoning = props.reasoning;
    this._agentId = props.agentId;
  }

  // Factory methods for creating messages
  static createUser(content: string): Message {
    return new Message({
      id: MessageId.create(),
      role: 'user',
      content: MessageContent.from(content),
      timestamp: new Date(),
      streamingState: StreamingState.complete(),
    });
  }

  static createAssistant(id?: MessageId): Message {
    return new Message({
      id: id || MessageId.create(),
      role: 'assistant',
      content: MessageContent.empty(),
      timestamp: new Date(),
      streamingState: StreamingState.pending(),
    });
  }

  static createAssistantWithContent(content: string, agentId?: string): Message {
    return new Message({
      id: MessageId.create(),
      role: 'assistant',
      content: MessageContent.from(content),
      timestamp: new Date(),
      streamingState: StreamingState.complete(),
      agentId,
    });
  }

  static createReasoning(thinking: string, isComplete: boolean, agentId?: string): Message {
    return new Message({
      id: MessageId.create(),
      role: 'assistant',
      content: MessageContent.from(thinking),
      timestamp: new Date(),
      streamingState: isComplete ? StreamingState.complete() : StreamingState.streaming(),
      reasoning: { thinking, isComplete },
      agentId,
    });
  }

  static createToolCall(toolCall: ToolCallInfo, agentId?: string): Message {
    return new Message({
      id: MessageId.create(),
      role: 'tool',
      content: MessageContent.from(`Calling ${toolCall.toolName}...`),
      timestamp: new Date(),
      streamingState: StreamingState.streaming(),
      toolCall,
      agentId,
    });
  }

  static fromSnapshot(snapshot: {
    id: string;
    role: MessageRole;
    content: string;
    timestamp: Date;
    isStreaming?: boolean;
    isComplete?: boolean;
    toolCall?: ToolCallInfo;
    toolResult?: ToolResultInfo;
    reasoning?: ReasoningInfo;
    agentId?: string;
  }): Message {
    let streamingState: StreamingState;
    if (snapshot.isComplete) {
      streamingState = StreamingState.complete();
    } else if (snapshot.isStreaming) {
      streamingState = StreamingState.streaming();
    } else {
      streamingState = StreamingState.pending();
    }

    return new Message({
      id: MessageId.from(snapshot.id),
      role: snapshot.role,
      content: MessageContent.from(snapshot.content),
      timestamp: snapshot.timestamp,
      streamingState,
      toolCall: snapshot.toolCall,
      toolResult: snapshot.toolResult,
      reasoning: snapshot.reasoning,
      agentId: snapshot.agentId,
    });
  }

  // Getters
  get id(): MessageId {
    return this._id;
  }

  get role(): MessageRole {
    return this._role;
  }

  get content(): MessageContent {
    return this._content;
  }

  get timestamp(): Date {
    return this._timestamp;
  }

  get streamingState(): StreamingState {
    return this._streamingState;
  }

  get toolCall(): ToolCallInfo | undefined {
    return this._toolCall;
  }

  get toolResult(): ToolResultInfo | undefined {
    return this._toolResult;
  }

  get reasoning(): ReasoningInfo | undefined {
    return this._reasoning;
  }

  get isStreaming(): boolean {
    return this._streamingState.isStreaming;
  }

  get isComplete(): boolean {
    return this._streamingState.isComplete;
  }

  get isToolMessage(): boolean {
    return this._role === 'tool';
  }

  get isAssistantMessage(): boolean {
    return this._role === 'assistant';
  }

  get isUserMessage(): boolean {
    return this._role === 'user';
  }

  get hasReasoning(): boolean {
    return !!this._reasoning;
  }

  get agentId(): string | undefined {
    return this._agentId;
  }

  // Behavior methods - return new instances (immutability)
  appendContent(chunk: string): Message {
    return new Message({
      ...this.toProps(),
      content: this._content.append(chunk),
      streamingState: this._streamingState.isComplete
        ? this._streamingState
        : StreamingState.streaming(),
    });
  }

  withContent(content: string): Message {
    return new Message({
      ...this.toProps(),
      content: MessageContent.from(content),
    });
  }

  markComplete(): Message {
    return new Message({
      ...this.toProps(),
      streamingState: this._streamingState.markComplete(),
    });
  }

  markAborted(): Message {
    return new Message({
      ...this.toProps(),
      streamingState: this._streamingState.abort(),
    });
  }

  withToolResult(result: string, error?: string, diffLines?: DiffLine[]): Message {
    if (!this._toolCall) {
      throw new Error('Cannot add tool result to non-tool message');
    }

    const displayContent = error
      ? `Error: ${error}`
      : result.length > 500
      ? result.slice(0, 500) + '...'
      : result;

    return new Message({
      ...this.toProps(),
      content: MessageContent.from(displayContent),
      streamingState: StreamingState.complete(),
      toolResult: {
        callId: this._toolCall.callId,
        toolName: this._toolCall.toolName,
        result,
        error,
        diffLines,
      },
    });
  }

  updateReasoning(thinking: string, isComplete: boolean): Message {
    return new Message({
      ...this.toProps(),
      content: MessageContent.from(thinking),
      reasoning: { thinking, isComplete },
      streamingState: isComplete
        ? StreamingState.complete()
        : StreamingState.streaming(),
    });
  }

  // Serialization
  private toProps(): MessageProps {
    return {
      id: this._id,
      role: this._role,
      content: this._content,
      timestamp: this._timestamp,
      streamingState: this._streamingState,
      toolCall: this._toolCall,
      toolResult: this._toolResult,
      reasoning: this._reasoning,
      agentId: this._agentId,
    };
  }

  toSnapshot(): {
    id: string;
    role: MessageRole;
    content: string;
    timestamp: Date;
    isStreaming: boolean;
    isComplete: boolean;
    toolCall?: ToolCallInfo;
    toolResult?: ToolResultInfo;
    reasoning?: ReasoningInfo;
    agentId?: string;
  } {
    return {
      id: this._id.value,
      role: this._role,
      content: this._content.value,
      timestamp: this._timestamp,
      isStreaming: this._streamingState.isStreaming,
      isComplete: this._streamingState.isComplete,
      toolCall: this._toolCall,
      toolResult: this._toolResult,
      reasoning: this._reasoning,
      agentId: this._agentId,
    };
  }

  equals(other: Message): boolean {
    return this._id.equals(other._id);
  }
}
