// IStreamGateway port - interface for stream communication
import { ToolCallInfo, SubQuery } from '../entities/Message.js';

export type StreamResponseType =
  | 'UNSPECIFIED'
  | 'ANSWER'
  | 'REASONING'
  | 'TOOL_CALL'
  | 'TOOL_RESULT'
  | 'ANSWER_CHUNK'
  | 'ERROR';

// SubResult for grouped tool operations
export interface SubResult {
  type: string;    // "vector" | "grep" | "symbol"
  result: string;  // Result data (text format)
  count: number;   // Number of matches found
  error?: string;  // Error message if failed
}

export interface StreamResponse {
  type: StreamResponseType;
  content: string;
  isFinal: boolean;
  agentId?: string;
  toolCall?: ToolCallInfo;
  toolResult?: {
    callId: string;
    result: string;
    error?: string;
    summary?: string;  // Server-computed display summary
  };
  reasoning?: {
    thinking: string;
    isComplete: boolean;
  };
  error?: {
    message: string;
    code?: string;
  };
}

export interface StreamConnectionOptions {
  serverAddress: string;
  sessionId: string;
  userId: string;
  projectKey: string;
  projectRoot: string;
  clientVersion?: string;
  testingStrategy?: string;
}

export type ConnectionStatus =
  | 'disconnected'
  | 'connecting'
  | 'connected'
  | 'reconnecting'
  | 'error';

/**
 * Gateway interface for stream communication with the server.
 * Abstracts the gRPC details from the application layer.
 */
export interface IStreamGateway {
  /**
   * Connect to the server
   */
  connect(options: StreamConnectionOptions): Promise<void>;

  /**
   * Disconnect from the server
   */
  disconnect(): void;

  /**
   * Send a user message
   */
  sendMessage(message: string): void;

  /**
   * Send a tool execution result
   */
  sendToolResult(callId: string, result: string, error?: Error, subResults?: SubResult[]): void;

  /**
   * Cancel the current stream
   */
  cancel(): void;

  /**
   * Reconnect the stream (e.g. after user cancel).
   * Creates a new gRPC stream on the existing channel.
   */
  reconnectStream(): Promise<void>;

  /**
   * Get current connection status
   */
  getStatus(): ConnectionStatus;

  /**
   * Check if connected
   */
  isConnected(): boolean;

  /**
   * Get reconnection attempt count
   */
  getReconnectAttempts(): number;

  /**
   * Subscribe to responses
   */
  onResponse(handler: (response: StreamResponse) => void): () => void;

  /**
   * Subscribe to errors
   */
  onError(handler: (error: Error) => void): () => void;

  /**
   * Subscribe to connection status changes
   */
  onStatusChange(handler: (status: ConnectionStatus) => void): () => void;
}
