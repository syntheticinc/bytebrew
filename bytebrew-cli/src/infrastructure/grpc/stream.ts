// Stream manager for bidirectional gRPC streaming
import { EventEmitter } from 'events';
import * as grpc from '@grpc/grpc-js';
import { FlowServiceClient, FlowRequest, FlowResponse, FlowStream, SubResult } from './client.js';
import { PING_INTERVAL_MS } from '../../domain/connection.js';
import { getLogger } from '../../lib/logger.js';

export interface StreamManagerOptions {
  sessionId: string;
  userId: string;
  projectKey: string;
  projectRoot: string;
  clientVersion?: string;
  testingStrategy?: string;
  onResponse: (response: FlowResponse) => void;
  onError: (error: Error) => void;
  onEnd: () => void;
  onConnect: () => void;
  onDisconnect: () => void;
}

export class StreamManager extends EventEmitter {
  private client: FlowServiceClient;
  private stream: FlowStream | null = null;
  private options: StreamManagerOptions;
  private isConnected: boolean = false;
  private pingInterval: NodeJS.Timeout | null = null;
  private isClosing: boolean = false;
  private handlersReady: boolean = false;

  constructor(client: FlowServiceClient, options: StreamManagerOptions) {
    super();
    this.client = client;
    this.options = options;
  }

  /**
   * Connect to the server and establish stream
   */
  async connect(): Promise<void> {
    if (this.isClosing) return;

    const logger = getLogger();

    try {
      // Wait for channel to be ready first
      await this.client.waitForReady(10000);
      logger.debug('gRPC channel ready');

      // Create the bidirectional stream with version metadata
      const metadata = new grpc.Metadata();
      if (this.options.clientVersion) {
        metadata.set('x-vector-version', this.options.clientVersion);
      }
      this.stream = this.client.createStream(metadata);

      // Setup handlers synchronously - no race condition since we're in same tick
      this.setupStreamHandlers();
      this.handlersReady = true;

      // Mark as connected only after handlers are attached
      this.isConnected = true;
      this.options.onConnect();
      logger.debug('Stream connected and handlers attached');

      // Start ping loop
      this.startPingLoop();
    } catch (error) {
      this.isConnected = false;
      this.handlersReady = false;
      const errorMessage = (error as Error).message || 'Unknown connection error';
      logger.error('Failed to connect stream', { error: errorMessage });

      // Safely call error handler
      try {
        this.options.onError(error as Error);
      } catch {
        // Ignore handler errors
      }

      throw error;
    }
  }

  /**
   * Setup stream event handlers
   */
  private setupStreamHandlers(): void {
    if (!this.stream) return;

    this.stream.on('data', (response: FlowResponse) => {
      // Handle pong separately (don't forward to UI)
      if (response.pong) {
        this.emit('pong', response.pong);
        return;
      }
      this.options.onResponse(response);
    });

    this.stream.on('error', (error: Error) => {
      if (this.isClosing) return;
      this.isConnected = false;
      this.stopPingLoop();
      this.options.onError(error);
      this.options.onDisconnect();
    });

    this.stream.on('end', () => {
      if (this.isClosing) return;
      this.isConnected = false;
      this.stopPingLoop();
      this.options.onEnd();
      this.options.onDisconnect();
    });
  }

  /**
   * Send a user message
   */
  sendMessage(task: string): void {
    if (!this.stream || !this.isConnected) {
      return;
    }

    const request: FlowRequest = {
      sessionId: this.options.sessionId,
      userId: this.options.userId,
      projectKey: this.options.projectKey,
      task,
      context: {
        project_root: this.options.projectRoot,
        platform: process.platform,
        ...(this.options.testingStrategy && { testing_strategy: this.options.testingStrategy }),
      },
    };

    try {
      this.stream.write(request);
    } catch (err) {
      this.isConnected = false;
      this.options.onError(err as Error);
    }
  }

  /**
   * Send a tool result back to the server
   */
  sendToolResult(callId: string, result: string, error?: Error, subResults?: SubResult[]): void {
    // Tool results must be sent even during shutdown (isClosing=true)
    // to unblock the server's proxy that waits for the result.
    // Only skip if the stream object is truly gone.
    if (!this.stream) {
      console.error('Cannot send tool result: stream not connected');
      return;
    }

    const request: FlowRequest = {
      sessionId: this.options.sessionId,
      userId: this.options.userId,
      projectKey: this.options.projectKey,
      toolResult: {
        callId,
        result,
        error: error ? { code: 'TOOL_ERROR', message: error.message } : undefined,
        subResults,
      },
    };

    try {
      this.stream.write(request);
    } catch {
      // Stream may already be closed — tool result is lost but this is best-effort
    }
  }

  /**
   * Send a ping to keep the connection alive
   */
  private sendPing(): void {
    if (!this.stream || !this.isConnected) return;

    const request: FlowRequest = {
      sessionId: this.options.sessionId,
      userId: this.options.userId,
      projectKey: this.options.projectKey,
      ping: { timestamp: Date.now().toString() },
    };

    try {
      this.stream.write(request);
      this.emit('ping', Date.now());
    } catch (err) {
      // Ignore ping errors
    }
  }

  /**
   * Start the ping loop
   */
  private startPingLoop(): void {
    this.stopPingLoop();
    this.pingInterval = setInterval(() => {
      this.sendPing();
    }, PING_INTERVAL_MS);
  }

  /**
   * Stop the ping loop
   */
  private stopPingLoop(): void {
    if (this.pingInterval) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }
  }

  /**
   * Cancel the current request
   */
  cancel(): void {
    if (!this.stream || !this.isConnected) return;

    const request: FlowRequest = {
      sessionId: this.options.sessionId,
      userId: this.options.userId,
      projectKey: this.options.projectKey,
      cancel: true,
    };

    try {
      this.stream.write(request);
    } catch (err) {
      // Ignore cancel errors
    }
  }

  /**
   * Disconnect from the server
   */
  disconnect(): void {
    this.isClosing = true;
    this.stopPingLoop();
    this.isConnected = false;

    // Defer stream teardown to allow in-flight tool results to be sent.
    // Tool execution is async — a result may arrive just after disconnect() is called.
    // Keep this.stream alive briefly so sendToolResult() can still use it.
    if (this.stream) {
      const stream = this.stream;
      setTimeout(() => {
        try {
          stream.end();
        } catch {
          // Ignore disconnect errors
        }
        if (this.stream === stream) {
          this.stream = null;
        }
      }, 500);
    }
  }

  /**
   * Check if connected
   */
  getIsConnected(): boolean {
    return this.isConnected;
  }
}
