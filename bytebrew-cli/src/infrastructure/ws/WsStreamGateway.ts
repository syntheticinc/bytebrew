// WsStreamGateway - implements IStreamGateway using WebSocket
// Uses the same flat event format as mobile clients (EventBroadcaster).
import {
  IStreamGateway,
  StreamResponse,
  StreamConnectionOptions,
  ConnectionStatus,
  SubResult,
} from '../../domain/ports/IStreamGateway.js';
import { ReconnectionManager } from '../connection/reconnect.js';
import { convertEventToStreamResponse, WsSessionEvent } from './eventConverter.js';
import { PortFileReader } from '../server/PortFileReader.js';
import { getLogger } from '../../lib/logger.js';

type ResponseHandler = (response: StreamResponse) => void;
type ErrorHandler = (error: Error) => void;
type StatusHandler = (status: ConnectionStatus) => void;

/** Outbound WS message envelope */
interface WsOutMessage {
  type: string;
  request_id: string;
  payload?: Record<string, unknown>;
}

/** Inbound WS message envelope */
interface WsInMessage {
  type: string;
  request_id?: string;
  payload?: Record<string, unknown>;
}

interface PendingRequest {
  resolve: (payload: Record<string, unknown>) => void;
  reject: (error: Error) => void;
  timeout: ReturnType<typeof setTimeout>;
}

/**
 * Gateway implementation using WebSocket transport.
 * Same logical flow as StreamingGateway (gRPC) but over WS:
 *
 *   1. WS connect
 *   2. create_session -> session_id
 *   3. subscribe -> session events stream
 *   4. send_message / ask_user_reply / cancel_session -> fire-and-forget
 */
export class WsStreamGateway implements IStreamGateway {
  private ws: WebSocket | null = null;
  private sessionId: string | null = null;
  private lastEventId: string | null = null;
  private wsUrl: string = '';
  private seenEventIds = new Set<string>();
  private static readonly MAX_SEEN = 1000;

  private status: ConnectionStatus = 'disconnected';
  private reconnectAttempts = 0;
  private connectionOptions: StreamConnectionOptions | null = null;

  /** True when user explicitly cancelled -- prevents auto-reconnection */
  private _cancelledByUser = false;

  private reconnect: ReconnectionManager | null = null;
  private pendingRequests = new Map<string, PendingRequest>();

  // Event handlers
  private responseHandlers: Set<ResponseHandler> = new Set();
  private errorHandlers: Set<ErrorHandler> = new Set();
  private statusHandlers: Set<StatusHandler> = new Set();

  /**
   * Connect: open WS, create session, subscribe to events
   */
  async connect(options: StreamConnectionOptions): Promise<void> {
    this.connectionOptions = options;
    this._cancelledByUser = false;
    this.setStatus('connecting');

    try {
      // Build WS URL from serverAddress (host:port)
      this.wsUrl = `ws://${options.serverAddress}/ws`;

      await this.openWebSocket();

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

      const sessionPayload = await this.sendRequest('create_session', {
        project_key: options.projectKey,
        user_id: options.userId,
        project_root: options.projectRoot,
        platform: process.platform,
        context,
      });

      this.sessionId = sessionPayload.session_id as string;
      if (!this.sessionId) {
        throw new Error('Server did not return session_id');
      }

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
      await this.sendRequest('subscribe', {
        session_id: this.sessionId,
        last_event_id: this.lastEventId || '',
      });

      this.setStatus('connected');
    } catch (error) {
      this.setStatus('disconnected');
      this.closeWebSocket();
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
    this.closeWebSocket();
    this.setStatus('disconnected');
    this.responseHandlers.clear();
    this.errorHandlers.clear();
    this.statusHandlers.clear();
  }

  /**
   * Send a user message
   */
  sendMessage(message: string): void {
    if (!this.sessionId) {
      throw new Error('Cannot send message: gateway not connected');
    }
    this.wsSend({
      type: 'send_message',
      request_id: this.nextRequestId(),
      payload: { session_id: this.sessionId, content: message },
    });
  }

  /**
   * Send a tool execution result -- NOT USED in streaming API.
   * Tools execute on server side.
   */
  sendToolResult(_callId: string, _result: string, _error?: Error, _subResults?: SubResult[]): void {
    // No-op: server executes tools locally
  }

  /**
   * Cancel the current processing
   */
  cancel(): void {
    this._cancelledByUser = true;
    if (this.sessionId) {
      this.wsSend({
        type: 'cancel_session',
        request_id: this.nextRequestId(),
        payload: { session_id: this.sessionId },
      });
    }
  }

  /**
   * Reconnect after cancel: re-subscribe to event stream
   */
  async reconnectStream(): Promise<void> {
    this._cancelledByUser = false;
    if (this.ws && this.sessionId) {
      this.setStatus('connecting');
      await this.sendRequest('subscribe', {
        session_id: this.sessionId,
        last_event_id: this.lastEventId || '',
      });
      this.setStatus('connected');
    }
  }

  /**
   * Send an ask_user reply
   */
  sendAskUserReply(callId: string, reply: string): void {
    if (!this.sessionId) {
      throw new Error('Cannot send reply: gateway not connected');
    }
    this.wsSend({
      type: 'ask_user_reply',
      request_id: this.nextRequestId(),
      payload: { session_id: this.sessionId, call_id: callId, reply },
    });
  }

  getStatus(): ConnectionStatus { return this.status; }

  isConnected(): boolean {
    return this.status === 'connected' && this.ws !== null && this.ws.readyState === WebSocket.OPEN;
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

  // --- Private: WebSocket lifecycle ---

  /** Open a new WebSocket connection and wire up event handlers */
  private async openWebSocket(): Promise<void> {
    this.closeWebSocket();

    const ws = new WebSocket(this.wsUrl);

    await new Promise<void>((resolve, reject) => {
      const onOpen = () => {
        ws.removeEventListener('open', onOpen);
        ws.removeEventListener('error', onError);
        clearTimeout(timer);
        resolve();
      };
      const onError = () => {
        ws.removeEventListener('open', onOpen);
        ws.removeEventListener('error', onError);
        clearTimeout(timer);
        reject(new Error(`WebSocket connection failed: ${this.wsUrl}`));
      };
      const timer = setTimeout(() => {
        ws.removeEventListener('open', onOpen);
        ws.removeEventListener('error', onError);
        try { ws.close(); } catch { /* ignore */ }
        reject(new Error('WebSocket connection timeout'));
      }, 10000);

      ws.addEventListener('open', onOpen);
      ws.addEventListener('error', onError);
    });

    // Wire persistent handlers
    ws.addEventListener('message', (event) => {
      const data = typeof event.data === 'string' ? event.data : '';
      this.handleWsMessage(data);
    });
    ws.addEventListener('close', () => this.handleWsClose());
    ws.addEventListener('error', () => this.handleWsError());

    this.ws = ws;
  }

  /** Close the WebSocket, cleanup pending requests */
  private closeWebSocket(): void {
    if (!this.ws) return;

    try { this.ws.close(); } catch { /* ignore */ }
    this.ws = null;

    // Reject all pending requests
    for (const [id, pending] of this.pendingRequests) {
      clearTimeout(pending.timeout);
      pending.reject(new Error('WebSocket closed'));
    }
    this.pendingRequests.clear();
  }

  // --- Private: Message handling ---

  private handleWsMessage(data: string): void {
    let msg: WsInMessage;
    try {
      msg = JSON.parse(data);
    } catch {
      return;
    }

    // Handle ack responses (resolve pending requests)
    if (msg.request_id && this.pendingRequests.has(msg.request_id)) {
      const pending = this.pendingRequests.get(msg.request_id)!;
      this.pendingRequests.delete(msg.request_id);
      clearTimeout(pending.timeout);

      if (msg.type === 'error') {
        const errorMsg = (msg.payload?.error as string) || 'Unknown server error';
        pending.reject(new Error(errorMsg));
      } else {
        pending.resolve(msg.payload || {});
      }
      return;
    }

    // Handle session events
    if (msg.type === 'session_event') {
      const payload = msg.payload || {};
      const eventData = payload.event as WsSessionEvent | undefined;
      const eventId = payload.event_id as string | undefined;

      // ID-based dedup: skip events we've already processed
      if (eventId && this.seenEventIds.has(eventId)) {
        return;
      }
      if (eventId) {
        this.seenEventIds.add(eventId);
        if (this.seenEventIds.size > WsStreamGateway.MAX_SEEN) {
          const first = this.seenEventIds.values().next().value;
          if (first) this.seenEventIds.delete(first);
        }
      }

      if (eventData) {
        getLogger().debug(
          `[WsStreamGW] event type=${eventData.type} content=${(eventData.content || '').substring(0, 80)} tool=${eventData.tool_name || ''}`
        );
        const response = convertEventToStreamResponse(eventData);
        if (response) {
          response.eventId = eventId;
          this.emitResponse(response);
        }
      }

      // Update cursor AFTER processing
      if (eventId) {
        this.lastEventId = eventId;
      }
      return;
    }

    // Handle backfill_complete
    if (msg.type === 'backfill_complete') {
      return;
    }
  }

  private handleWsClose(): void {
    if (this._cancelledByUser) return;
    if (this.status === 'disconnected') return;

    this.setStatus('reconnecting');
    this.reconnect?.startReconnection();
  }

  private handleWsError(): void {
    // Error details are limited in browser-style WebSocket API.
    // The close handler will fire after this and trigger reconnection.
    this.emitError(new Error('WebSocket error'));
  }

  // --- Private: Reconnection ---

  /**
   * Re-read port file and update wsUrl if server restarted on a different port.
   */
  private rediscoverServer(): void {
    const reader = new PortFileReader();
    const info = reader.read();
    if (!info) return;

    const host = (!info.host || info.host === '0.0.0.0') ? '127.0.0.1' : info.host;
    const port = info.ws_port || info.port;
    const newUrl = `ws://${host}:${port}/ws`;

    if (newUrl !== this.wsUrl) {
      getLogger().debug('[WsStreamGW] server port changed, updating URL', { old: this.wsUrl, new: newUrl });
      this.wsUrl = newUrl;
    }
  }

  private async resubscribe(): Promise<void> {
    if (!this.connectionOptions) {
      throw new Error('Cannot resubscribe: no connection options');
    }

    try {
      // Re-read port file in case server restarted on a different port
      this.rediscoverServer();

      // Re-open WS connection
      await this.openWebSocket();

      if (this.sessionId) {
        // Try resubscribe with existing session
        await this.sendRequest('subscribe', {
          session_id: this.sessionId,
          last_event_id: this.lastEventId || '',
        });
        this.setStatus('connected');
        this.reconnectAttempts = 0;
        return;
      }
    } catch {
      // Session gone or WS failed -- fall through to full reconnect
    }

    // Full reconnect: new session
    await this.fullReconnect();
  }

  /**
   * Full reconnect: new WS, new session, new subscription.
   * Used when server restarted and old session is gone.
   */
  private async fullReconnect(): Promise<void> {
    const options = this.connectionOptions!;

    this.rediscoverServer();
    await this.openWebSocket();

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

    const sessionPayload = await this.sendRequest('create_session', {
      project_key: options.projectKey,
      user_id: options.userId,
      project_root: options.projectRoot,
      platform: process.platform,
      context,
    });

    this.sessionId = sessionPayload.session_id as string;
    this.lastEventId = null;

    await this.sendRequest('subscribe', {
      session_id: this.sessionId,
      last_event_id: '',
    });

    this.setStatus('connected');
    this.reconnectAttempts = 0;
  }

  // --- Public: Pairing ---

  /**
   * Request pairing data from the server (QR code + short code).
   * Only works over localhost WS connection.
   */
  async generatePairing(): Promise<{ qrData: string; shortCode: string; expiresInSeconds: number }> {
    const payload = await this.sendRequest('generate_pairing', {});
    return {
      qrData: payload.qr_data as string,
      shortCode: payload.short_code as string,
      expiresInSeconds: payload.expires_in_seconds as number,
    };
  }

  // --- Private: Transport helpers ---

  /**
   * Send a request and wait for the corresponding ack.
   * Rejects on timeout or error response.
   */
  private sendRequest(type: string, payload: Record<string, unknown> = {}): Promise<Record<string, unknown>> {
    return new Promise((resolve, reject) => {
      const requestId = this.nextRequestId();

      const timeout = setTimeout(() => {
        this.pendingRequests.delete(requestId);
        reject(new Error(`Request '${type}' timed out after 30s`));
      }, 30000);

      this.pendingRequests.set(requestId, { resolve, reject, timeout });

      this.wsSend({ type, request_id: requestId, payload });
    });
  }

  private wsSend(msg: WsOutMessage): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
    this.ws.send(JSON.stringify(msg));
  }

  private requestCounter = 0;

  private nextRequestId(): string {
    this.requestCounter++;
    return `req-${Date.now()}-${this.requestCounter}`;
  }

  // --- Private: Event emitters ---

  private setStatus(status: ConnectionStatus): void {
    this.status = status;
    for (const handler of this.statusHandlers) {
      try { handler(status); } catch (err) {
        console.error('Error in status handler:', err);
      }
    }
  }

  private emitResponse(response: StreamResponse): void {
    for (const handler of this.responseHandlers) {
      try { handler(response); } catch (err) {
        console.error('Error in response handler:', err);
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
