// StreamingGateway - implements IStreamGateway using server-streaming API
// No tool round-trips: server executes tools locally, CLI is a view-only client.
import {
  IStreamGateway,
  StreamResponse,
  StreamConnectionOptions,
  ConnectionStatus,
} from '../../domain/ports/IStreamGateway.js';
import { FlowServiceClient, SessionEvent, SessionEventType, SessionEventStream } from './client.js';
import { ReconnectionManager } from './reconnect.js';
import { getLogger } from '../../lib/logger.js';

type ResponseHandler = (response: StreamResponse) => void;
type ErrorHandler = (error: Error) => void;
type StatusHandler = (status: ConnectionStatus) => void;

/**
 * Gateway implementation using the new server-streaming API.
 * CLI is a thin view client: no tool execution, no tool results sent back.
 *
 * Flow:
 *   1. CreateSession → session_id
 *   2. SubscribeSession → server-streaming SessionEvent
 *   3. SendMessage → unary (user message or ask_user reply)
 *   4. CancelSession → unary
 */
export class StreamingGateway implements IStreamGateway {
  private client: FlowServiceClient | null = null;
  private sessionId: string | null = null;
  private eventStream: SessionEventStream | null = null;
  private reconnect: ReconnectionManager | null = null;
  private lastEventId: string | null = null;

  private status: ConnectionStatus = 'disconnected';
  private reconnectAttempts = 0;
  private connectionOptions: StreamConnectionOptions | null = null;

  /** True when user explicitly cancelled — prevents auto-reconnection */
  private _cancelledByUser = false;

  // Event handlers
  private responseHandlers: Set<ResponseHandler> = new Set();
  private errorHandlers: Set<ErrorHandler> = new Set();
  private statusHandlers: Set<StatusHandler> = new Set();

  /**
   * Connect: create session + subscribe to events
   */
  async connect(options: StreamConnectionOptions): Promise<void> {
    this.connectionOptions = options;
    this.setStatus('connecting');

    try {
      this.client = new FlowServiceClient(options.serverAddress);

      // Wait for gRPC channel ready
      await this.client.waitForReady(10000);

      // Create session
      const context: Record<string, string> = {
        project_root: options.projectRoot,
        platform: process.platform,
      };
      if (options.testingStrategy) {
        context.testing_strategy = options.testingStrategy;
      }
      if (options.clientVersion) {
        context.client_version = options.clientVersion;
      }

      this.sessionId = await this.client.createSession(
        options.projectKey,
        options.userId,
        context,
      );

      // Setup reconnection manager
      this.reconnect = new ReconnectionManager({
        onReconnect: async () => {
          await this.resubscribe();
        },
        onMaxAttemptsReached: () => {
          this.setStatus('disconnected');
        },
        onAttempt: () => {
          this.reconnectAttempts++;
        },
      });

      // Subscribe to session events
      this.startEventStream();
      this.setStatus('connected');

    } catch (error) {
      this.setStatus('disconnected');
      if (this.client) {
        try { this.client.close(); } catch { /* ignore */ }
        this.client = null;
      }
      this.sessionId = null;
      this.reconnect = null;
      throw error;
    }
  }

  /**
   * Disconnect from the server
   */
  disconnect(): void {
    this.reconnect?.stop();
    this.stopEventStream();
    if (this.client) {
      try { this.client.close(); } catch { /* ignore */ }
    }
    this.setStatus('disconnected');
    this.responseHandlers.clear();
    this.errorHandlers.clear();
    this.statusHandlers.clear();
  }

  /**
   * Send a user message
   */
  sendMessage(message: string): void {
    if (!this.client || !this.sessionId) {
      throw new Error('Cannot send message: gateway not connected');
    }
    // Fire-and-forget: errors handled via error handlers
    this.client.sendMessage(this.sessionId, message).catch((err) => {
      this.emitError(err instanceof Error ? err : new Error(String(err)));
    });
  }

  /**
   * Send a tool execution result — NOT USED in streaming API.
   * Tools execute on server side.
   */
  sendToolResult(): void {
    // No-op: server executes tools locally
  }

  /**
   * Cancel the current processing
   */
  cancel(): void {
    this._cancelledByUser = true;
    if (this.client && this.sessionId) {
      this.client.cancelSession(this.sessionId).catch(() => { /* ignore */ });
    }
  }

  /**
   * Reconnect after cancel: re-subscribe to event stream
   */
  async reconnectStream(): Promise<void> {
    this._cancelledByUser = false;
    if (this.client && this.sessionId) {
      this.setStatus('connecting');
      this.startEventStream();
      this.setStatus('connected');
    }
  }

  /**
   * Send an ask_user reply (uses replyTo field)
   */
  sendAskUserReply(callId: string, reply: string): void {
    if (!this.client || !this.sessionId) {
      throw new Error('Cannot send reply: gateway not connected');
    }
    this.client.sendMessage(this.sessionId, reply, callId).catch((err) => {
      this.emitError(err instanceof Error ? err : new Error(String(err)));
    });
  }

  getStatus(): ConnectionStatus { return this.status; }

  isConnected(): boolean {
    return this.status === 'connected' && this.eventStream !== null;
  }

  getReconnectAttempts(): number { return this.reconnectAttempts; }

  onResponse(handler: ResponseHandler): () => void {
    this.responseHandlers.add(handler);
    return () => { this.responseHandlers.delete(handler); };
  }

  onError(handler: ErrorHandler): () => void {
    this.errorHandlers.add(handler);
    return () => { this.errorHandlers.delete(handler); };
  }

  onStatusChange(handler: StatusHandler): () => void {
    this.statusHandlers.add(handler);
    return () => { this.statusHandlers.delete(handler); };
  }

  // --- Private ---

  private startEventStream(): void {
    this.stopEventStream();

    if (!this.client || !this.sessionId) return;

    const stream = this.client.subscribeSession(
      this.sessionId,
      this.lastEventId || undefined,
    );

    stream.on('data', (event: SessionEvent) => {
      this.lastEventId = event.eventId || this.lastEventId;
      getLogger().debug(`[StreamingGW] event type=${event.type} content=${(event.content || '').substring(0, 80)} tool=${event.toolName || ''}`);
      this.handleSessionEvent(event);
    });

    stream.on('error', (error: Error) => {
      if (this._cancelledByUser) {
        this._cancelledByUser = false;
        return;
      }
      const errMsg = error.message || '';
      // Session not found = server restarted, need full reconnect
      if (errMsg.includes('session not found') || errMsg.includes('NotFound')) {
        this.sessionId = null;
      }
      this.emitError(error);
      this.handleStreamEnd();
    });

    stream.on('end', () => {
      if (this._cancelledByUser) {
        this._cancelledByUser = false;
        return;
      }
      this.handleStreamEnd();
    });

    this.eventStream = stream;
    this.reconnectAttempts = 0;
  }

  private stopEventStream(): void {
    if (this.eventStream) {
      try { this.eventStream.cancel(); } catch { /* ignore */ }
      this.eventStream = null;
    }
  }

  private handleStreamEnd(): void {
    this.eventStream = null;
    if (this.status !== 'disconnected') {
      this.setStatus('reconnecting');
      this.reconnect?.startReconnection();
    }
  }

  private async resubscribe(): Promise<void> {
    if (!this.connectionOptions) {
      throw new Error('Cannot resubscribe: no connection options');
    }

    try {
      // Try resubscribe with existing session first
      if (this.client && this.sessionId) {
        this.startEventStream();
        this.setStatus('connected');
        return;
      }
    } catch {
      // Session gone — fall through to full reconnect
    }

    // Full reconnect: new client + new session
    await this.fullReconnect();
  }

  /**
   * Full reconnect: create new gRPC client, new session, new event stream.
   * Used when server restarted and old session is gone.
   */
  private async fullReconnect(): Promise<void> {
    const options = this.connectionOptions!;

    // Close old client
    if (this.client) {
      try { this.client.close(); } catch { /* ignore */ }
    }

    this.client = new FlowServiceClient(options.serverAddress);
    await this.client.waitForReady(10000);

    const context: Record<string, string> = {
      project_root: options.projectRoot,
      platform: process.platform,
    };
    if (options.testingStrategy) {
      context.testing_strategy = options.testingStrategy;
    }
    if (options.clientVersion) {
      context.client_version = options.clientVersion;
    }

    this.sessionId = await this.client.createSession(
      options.projectKey,
      options.userId,
      context,
    );
    this.lastEventId = null;

    this.startEventStream();
    this.setStatus('connected');
  }

  /**
   * Convert SessionEvent → StreamResponse and emit
   */
  private handleSessionEvent(event: SessionEvent): void {
    const response = this.convertEvent(event);
    if (!response) return;

    for (const handler of this.responseHandlers) {
      try { handler(response); } catch (err) {
        console.error('Error in response handler:', err);
      }
    }
  }

  /**
   * Map SessionEvent type to StreamResponse
   */
  private convertEvent(event: SessionEvent): StreamResponse | null {
    const agentId = event.agentId || undefined;

    switch (event.type) {
      case SessionEventType.PROCESSING_STARTED:
        return {
          type: 'ANSWER_CHUNK',
          content: '',
          isFinal: false,
          agentId,
        };

      case SessionEventType.ANSWER_CHUNK:
        return {
          type: 'ANSWER_CHUNK',
          content: event.content,
          isFinal: false,
          agentId,
        };

      case SessionEventType.ANSWER:
        return {
          type: 'ANSWER',
          content: event.content,
          isFinal: true,
          agentId,
        };

      case SessionEventType.TOOL_EXECUTION_START:
        return {
          type: 'TOOL_CALL',
          content: '',
          isFinal: false,
          agentId,
          toolCall: {
            callId: event.callId,
            toolName: event.toolName,
            arguments: event.toolArguments || {},
          },
        };

      case SessionEventType.TOOL_EXECUTION_END:
        return {
          type: 'TOOL_RESULT',
          content: '',
          isFinal: false,
          agentId,
          toolResult: {
            callId: event.callId,
            result: event.toolResultSummary || '',
            error: event.toolHasError ? (event.toolResultSummary || 'Tool error') : undefined,
            summary: event.toolResultSummary || undefined,
          },
        };

      case SessionEventType.REASONING:
        return {
          type: 'REASONING',
          content: event.content,
          isFinal: false,
          agentId,
          reasoning: {
            thinking: event.content,
            isComplete: false,
          },
        };

      case SessionEventType.PLAN_UPDATE:
        // Encode plan as a TOOL_CALL for manage_plan (reuse existing UI)
        return {
          type: 'TOOL_CALL',
          content: '',
          isFinal: false,
          agentId,
          toolCall: {
            callId: `plan-${Date.now()}`,
            toolName: 'manage_plan',
            arguments: {
              goal: event.planName || '',
              steps: JSON.stringify(
                (event.planSteps || []).map((s, i) => ({
                  index: i,
                  description: s.title,
                  status: s.status || 'pending',
                }))
              ),
            },
          },
        };

      case SessionEventType.ASK_USER:
        // Encode ask_user as a TOOL_CALL for ask_user (reuse existing flow)
        return {
          type: 'TOOL_CALL',
          content: '',
          isFinal: false,
          agentId,
          toolCall: {
            callId: event.callId || `ask-${Date.now()}`,
            toolName: 'ask_user',
            arguments: {
              questions: JSON.stringify([{
                text: event.question || 'Please respond',
                options: (event.options || []).map(o => ({ label: o })),
              }]),
            },
          },
        };

      case SessionEventType.PROCESSING_STOPPED:
        return {
          type: 'ANSWER_CHUNK',
          content: '',
          isFinal: true,
          agentId,
        };

      case SessionEventType.ERROR:
        return {
          type: 'ERROR',
          content: event.errorDetail?.message || event.content || 'Unknown error',
          isFinal: false,
          agentId,
          error: {
            message: event.errorDetail?.message || event.content || 'Unknown error',
            code: event.errorDetail?.code,
          },
        };

      default:
        return null;
    }
  }

  private setStatus(status: ConnectionStatus): void {
    this.status = status;
    for (const handler of this.statusHandlers) {
      try { handler(status); } catch (err) {
        console.error('Error in status handler:', err);
      }
    }
  }

  private emitError(error: Error): void {
    for (const handler of this.errorHandlers) {
      try { handler(error); } catch (err) {
        console.error('Error in error handler:', err);
      }
    }
  }
}
