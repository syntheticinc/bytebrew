// Message types for the chat

export type MessageRole = 'user' | 'assistant' | 'system' | 'tool';

export interface ChatMessage {
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
}

export interface ToolCallInfo {
  callId: string;
  toolName: string;
  arguments: Record<string, string>;
}

export interface ToolResultInfo {
  callId: string;
  toolName: string;
  result: string;
  error?: string;
  summary?: string;
  diffLines?: DiffLine[];
}

export interface DiffLine {
  type: '+' | '-' | ' ';
  content: string;
}

export interface ReasoningInfo {
  thinking: string;
  isComplete: boolean;
}

export type ResponseType =
  | 'RESPONSE_TYPE_UNSPECIFIED'
  | 'RESPONSE_TYPE_ANSWER'
  | 'RESPONSE_TYPE_REASONING'
  | 'RESPONSE_TYPE_TOOL_CALL'
  | 'RESPONSE_TYPE_TOOL_RESULT'
  | 'RESPONSE_TYPE_ANSWER_CHUNK'
  | 'RESPONSE_TYPE_ERROR';

export const ResponseTypeEnum = {
  UNSPECIFIED: 0,
  ANSWER: 1,
  REASONING: 2,
  TOOL_CALL: 3,
  TOOL_RESULT: 4,
  ANSWER_CHUNK: 5,
  ERROR: 6,
} as const;
