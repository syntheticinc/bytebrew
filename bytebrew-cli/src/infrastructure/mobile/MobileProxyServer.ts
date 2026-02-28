// MobileProxyServer - WebSocket proxy that broadcasts domain events to mobile clients
import type { ServerWebSocket } from 'bun';
import path from 'path';
import type { IEventBus, DomainEvent } from '../../domain/ports/IEventBus.js';
import type { IMessageRepository } from '../../domain/ports/IMessageRepository.js';
import { resolveAskUser, type QuestionAnswer } from '../../tools/askUser.js';

/** Port for sending user messages through the CLI pipeline. */
interface IMessageSender {
  sendMessage(content: string): void;
}

interface IncomingMessage {
  type: string;
  text?: string;
  answers?: QuestionAnswer[];
}

/** Metadata about the CLI session, sent to mobile clients on connect. */
export interface MobileProxyMeta {
  projectName: string;
  projectPath: string;
  sessionId: string;
}

/**
 * WebSocket server that proxies domain events to mobile clients.
 * On connect: sends full current state (init).
 * On event: broadcasts serialized domain events.
 * Accepts ask_user answers from mobile clients.
 */
export class MobileProxyServer {
  private server: ReturnType<typeof Bun.serve> | null = null;
  private clients: Set<ServerWebSocket<unknown>> = new Set();
  private unsubscribe: (() => void) | null = null;
  private heartbeatInterval: ReturnType<typeof setInterval> | null = null;

  /** Heartbeat interval in milliseconds (30 seconds). */
  private static readonly HEARTBEAT_MS = 30_000;

  constructor(
    private readonly messageRepository: IMessageRepository,
    private readonly eventBus: IEventBus,
    private readonly meta: MobileProxyMeta,
    private readonly messageSender?: IMessageSender,
  ) {}

  /** The port the server is actually listening on (useful when started with port 0). */
  get port(): number {
    return this.server?.port ?? 0;
  }

  start(port: number): void {
    this.unsubscribe = this.eventBus.subscribeAll((event) => {
      this.broadcast(this.serializeEvent(event));
    });

    this.server = Bun.serve<undefined>({
      port,
      fetch: (req, server) => {
        const upgraded = server.upgrade(req);
        if (upgraded) return undefined;
        return new Response('WebSocket endpoint', { status: 426 });
      },
      websocket: {
        // Disable idle timeout — mobile clients may be idle while CLI
        // waits for long server responses. Without this, Bun closes the
        // connection after 120s of inactivity with code 1000.
        idleTimeout: 0,
        sendPings: true,
        open: (ws) => this.handleConnect(ws),
        message: (ws, data) => this.handleMessage(ws, String(data)),
        close: (ws) => {
          this.clients.delete(ws);
          console.log(`[MobileProxy] Client disconnected (remaining: ${this.clients.size})`);
        },
      },
    });

    // Start heartbeat to keep mobile connections alive and allow clients
    // to detect stale connections even when no domain events are flowing.
    this.startHeartbeat();

    console.log(`[MobileProxy] Started on port ${this.server.port}`);
  }

  stop(): void {
    this.stopHeartbeat();

    if (this.unsubscribe) {
      this.unsubscribe();
      this.unsubscribe = null;
    }

    for (const ws of this.clients) {
      try { ws.close(); } catch { /* ignore */ }
    }
    this.clients.clear();

    if (this.server) {
      this.server.stop();
      this.server = null;
    }

    console.log('[MobileProxy] Stopped');
  }

  private handleConnect(ws: ServerWebSocket<unknown>): void {
    this.clients.add(ws);

    const messages = this.messageRepository.findAll();
    const snapshots = messages.map((m) => m.toSnapshot());

    const initPayload = {
      type: 'init',
      messages: snapshots,
      meta: this.meta,
    };

    try {
      ws.send(JSON.stringify(initPayload));
    } catch {
      this.clients.delete(ws);
    }

    console.log(`[MobileProxy] Client connected (total: ${this.clients.size})`);
  }

  private handleMessage(ws: ServerWebSocket<unknown>, data: string): void {
    try {
      const msg: IncomingMessage = JSON.parse(data);

      if (msg.type === 'ask_user_answer' && Array.isArray(msg.answers)) {
        resolveAskUser(msg.answers);
        return;
      }

      if (msg.type === 'user_message' && typeof msg.text === 'string' && msg.text.trim()) {
        if (!this.messageSender) {
          console.error('[MobileProxy] user_message received but no messageSender configured');
          return;
        }
        try {
          this.messageSender.sendMessage(msg.text);
        } catch (err) {
          const errorMessage = err instanceof Error ? err.message : String(err);
          try {
            ws.send(JSON.stringify({ type: 'error', message: errorMessage }));
          } catch {
            // WebSocket send failed — client likely disconnected
          }
        }
        return;
      }
    } catch (error) {
      console.error('[MobileProxy] Failed to parse message:', error);
    }
  }

  private broadcast(payload: object): void {
    const data = JSON.stringify({ type: 'event', event: payload });
    const deadClients: ServerWebSocket<unknown>[] = [];

    for (const ws of this.clients) {
      try {
        ws.send(data);
      } catch {
        deadClients.push(ws);
      }
    }

    for (const ws of deadClients) {
      this.clients.delete(ws);
    }
  }

  private startHeartbeat(): void {
    this.stopHeartbeat();
    this.heartbeatInterval = setInterval(() => {
      if (this.clients.size === 0) return;
      this.broadcast({ type: 'heartbeat', timestamp: Date.now() });
    }, MobileProxyServer.HEARTBEAT_MS);
  }

  private stopHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }

  private serializeEvent(event: DomainEvent): object {
    switch (event.type) {
      case 'MessageCompleted':
        return {
          type: event.type,
          message: event.message.toSnapshot(),
        };

      case 'MessageStarted':
        return {
          type: event.type,
          messageId: event.messageId,
          role: event.role,
        };

      case 'ToolExecutionStarted':
      case 'ToolExecutionCompleted':
        return {
          type: event.type,
          execution: event.execution.toSnapshot(),
        };

      case 'AskUserRequested':
        return {
          type: event.type,
          questions: event.questions,
        };

      case 'StreamingProgress':
        return {
          type: event.type,
          messageId: event.messageId,
          tokensAdded: event.tokensAdded,
          totalTokens: event.totalTokens,
        };

      case 'ProcessingStarted':
      case 'ProcessingStopped':
        return { type: event.type };

      case 'ErrorOccurred':
        return {
          type: event.type,
          message: event.error.message,
          context: event.context,
        };

      case 'AgentLifecycle':
        return {
          type: event.type,
          lifecycleType: event.lifecycleType,
          agentId: event.agentId,
          description: event.description,
        };

      case 'AskUserResolved':
        return { type: event.type };
    }
  }
}
