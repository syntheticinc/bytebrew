// GrpcStreamGateway - implements IStreamGateway using existing gRPC infrastructure
import {
  IStreamGateway,
  StreamResponse,
  StreamConnectionOptions,
  ConnectionStatus,
  SubResult,
} from '../../domain/ports/IStreamGateway.js';
import { FlowServiceClient, FlowResponse } from './client.js';
import { StreamManager } from './stream.js';
import { ReconnectionManager } from './reconnect.js';
import { ResponseTypeMap } from '../../shared/grpcConstants.js';

type ResponseHandler = (response: StreamResponse) => void;
type ErrorHandler = (error: Error) => void;
type StatusHandler = (status: ConnectionStatus) => void;

/**
 * Gateway implementation that wraps the existing gRPC infrastructure
 * to implement the IStreamGateway port interface.
 */
export class GrpcStreamGateway implements IStreamGateway {
  private client: FlowServiceClient | null = null;
  private stream: StreamManager | null = null;
  private reconnect: ReconnectionManager | null = null;

  private status: ConnectionStatus = 'disconnected';
  private reconnectAttempts = 0;
  private connectionOptions: StreamConnectionOptions | null = null;

  /**
   * True when the user explicitly cancelled the stream via cancel().
   * Prevents auto-reconnection on the resulting stream close.
   */
  private _cancelledByUser = false;

  // Event handlers
  private responseHandlers: Set<ResponseHandler> = new Set();
  private errorHandlers: Set<ErrorHandler> = new Set();
  private statusHandlers: Set<StatusHandler> = new Set();

  /**
   * Connect to the server
   */
  async connect(options: StreamConnectionOptions): Promise<void> {
    this.connectionOptions = options;
    this.setStatus('connecting');

    try {
      // Create client
      this.client = new FlowServiceClient(options.serverAddress);

      // Create stream manager
      this.stream = new StreamManager(this.client, {
        sessionId: options.sessionId,
        userId: options.userId,
        projectKey: options.projectKey,
        projectRoot: options.projectRoot,
        clientVersion: options.clientVersion,
        testingStrategy: options.testingStrategy,
        onResponse: (resp) => this.handleResponse(resp),
        onError: (err) => this.handleError(err),
        onEnd: () => this.handleEnd(),
        onConnect: () => this.handleConnect(),
        onDisconnect: () => this.handleDisconnect(),
      });

      // Create reconnection manager
      this.reconnect = new ReconnectionManager({
        onReconnect: async () => {
          if (this.stream) {
            await this.stream.connect();
          }
        },
        onMaxAttemptsReached: () => {
          this.setStatus('disconnected');
        },
        onAttempt: () => {
          this.reconnectAttempts++;
        },
      });

      // Connect
      await this.stream.connect();
    } catch (error) {
      // On initial connection failure, set status to disconnected (not error)
      // and don't start reconnection - let the app handle retry logic
      this.setStatus('disconnected');
      // Clean up partial state
      if (this.client) {
        try {
          this.client.close();
        } catch {
          // Ignore close errors
        }
        this.client = null;
      }
      this.stream = null;
      this.reconnect = null;
      throw error;
    }
  }

  /**
   * Disconnect from the server
   */
  disconnect(): void {
    this.reconnect?.stop();
    this.stream?.disconnect();
    if (this.client) {
      try {
        this.client.close();
      } catch {
        // Ignore close errors
      }
    }
    this.setStatus('disconnected');

    // Clear all handlers to prevent duplicates on reconnect
    this.responseHandlers.clear();
    this.errorHandlers.clear();
    this.statusHandlers.clear();
  }

  /**
   * Send a user message
   */
  sendMessage(message: string): void {
    this.stream?.sendMessage(message);
  }

  /**
   * Send a tool execution result
   */
  sendToolResult(callId: string, result: string, error?: Error, subResults?: SubResult[]): void {
    this.stream?.sendToolResult(callId, result, error, subResults);
  }

  /**
   * Cancel the current stream.
   * Sets a flag so that the resulting stream close does NOT trigger auto-reconnection.
   */
  cancel(): void {
    this._cancelledByUser = true;
    this.stream?.cancel();
  }

  /**
   * Reconnect the stream (creates a new gRPC stream on the existing channel).
   * Used after cancel to re-establish the connection for the next message.
   */
  async reconnectStream(): Promise<void> {
    if (this.stream) {
      this.setStatus('connecting');
      await this.stream.connect();
    }
  }

  /**
   * Get current connection status
   */
  getStatus(): ConnectionStatus {
    return this.status;
  }

  /**
   * Check if connected
   */
  isConnected(): boolean {
    return this.status === 'connected' && (this.stream?.getIsConnected() ?? false);
  }

  /**
   * Get reconnection attempt count
   */
  getReconnectAttempts(): number {
    return this.reconnectAttempts;
  }

  /**
   * Subscribe to responses
   */
  onResponse(handler: ResponseHandler): () => void {
    this.responseHandlers.add(handler);
    return () => {
      this.responseHandlers.delete(handler);
    };
  }

  /**
   * Subscribe to errors
   */
  onError(handler: ErrorHandler): () => void {
    this.errorHandlers.add(handler);
    return () => {
      this.errorHandlers.delete(handler);
    };
  }

  /**
   * Subscribe to connection status changes
   */
  onStatusChange(handler: StatusHandler): () => void {
    this.statusHandlers.add(handler);
    return () => {
      this.statusHandlers.delete(handler);
    };
  }

  // Private handlers

  private handleResponse(response: FlowResponse): void {
    const streamResponse = this.convertResponse(response);
    for (const handler of this.responseHandlers) {
      try {
        handler(streamResponse);
      } catch (error) {
        console.error('Error in response handler:', error);
      }
    }
  }

  private handleError(error: Error): void {
    for (const handler of this.errorHandlers) {
      try {
        handler(error);
      } catch (err) {
        console.error('Error in error handler:', err);
      }
    }
  }

  private handleConnect(): void {
    this.setStatus('connected');
    this.reconnectAttempts = 0;
  }

  private handleDisconnect(): void {
    if (this._cancelledByUser) {
      // Cancel is intentional — skip auto-reconnection.
      // Reset flag here (last handler in the chain after handleEnd).
      this._cancelledByUser = false;
      return;
    }
    if (this.status !== 'disconnected') {
      this.setStatus('reconnecting');
      this.reconnect?.startReconnection();
    }
  }

  private handleEnd(): void {
    if (this._cancelledByUser) {
      // Cancel is intentional — don't auto-reconnect.
      // Don't reset flag here — handleDisconnect() is called next and needs it.
      return;
    }
    if (this.status !== 'disconnected') {
      this.reconnect?.startReconnection();
    }
  }

  private setStatus(status: ConnectionStatus): void {
    this.status = status;
    for (const handler of this.statusHandlers) {
      try {
        handler(status);
      } catch (error) {
        console.error('Error in status handler:', error);
      }
    }
  }

  private convertResponse(response: FlowResponse): StreamResponse {
    // Debug: log agentId for TOOL_CALL responses
    if (process.env.BYTEBREW_DEBUG_FILTER === '1' && response.toolCall) {
      process.stderr.write(`[GRPC] TOOL_CALL received: tool=${response.toolCall.toolName} callId=${response.toolCall.callId} agentId="${response.agentId}" raw_type=${response.type}\n`);
    }
    return {
      type: ResponseTypeMap[response.type] || 'UNSPECIFIED',
      content: response.content,
      isFinal: response.isFinal,
      agentId: response.agentId || undefined,
      toolCall: response.toolCall
        ? {
            callId: response.toolCall.callId,
            toolName: response.toolCall.toolName,
            arguments: response.toolCall.arguments,
            subQueries: response.toolCall.subQueries,
          }
        : undefined,
      toolResult: response.toolResult
        ? {
            callId: response.toolResult.callId,
            result: response.toolResult.result,
            error: response.toolResult.error?.message,
            summary: response.toolResult.summary || undefined,
          }
        : undefined,
      reasoning: response.reasoning
        ? {
            thinking: response.reasoning.thinking,
            isComplete: response.reasoning.isComplete,
          }
        : undefined,
      error: response.error
        ? {
            message: response.error.message,
            code: response.error.code,
          }
        : undefined,
    };
  }
}
