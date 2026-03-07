/**
 * WsMobileSimulator -- WebSocket client that behaves like a real mobile app.
 *
 * Connects to Bridge /connect?server_id=xxx&device_id=xxx, sends and receives
 * MobileMessage-format JSON through the Bridge relay.
 *
 * Message flow:
 *   Simulator -> Bridge: {"type":"data","payload": <MobileMessage | base64>}
 *   Bridge -> Simulator: {"type":"data","payload": <MobileMessage | base64>}
 *
 * For pairing flow (no shared secret yet), payload is a plaintext MobileMessage object.
 * After pairing with E2E encryption, payload is a base64 string containing encrypted bytes.
 *
 * Encryption: X25519 ECDH key exchange + XChaCha20-Poly1305 (same as CryptoService in CLI).
 */

import { v4 as uuidv4 } from 'uuid';
import { CryptoService } from '../../src/infrastructure/mobile/CryptoService.js';

// --- Types ---

export interface PairResult {
  deviceId: string;
  deviceToken: string;
}

export interface SessionEvent {
  type: string;
  [key: string]: unknown;
}

interface BridgeMessage {
  type: string;
  payload?: unknown;
  device_id?: string;
}

interface MobileMessage {
  type: string;
  request_id: string;
  device_id: string;
  payload: Record<string, unknown>;
}

interface PendingRequest {
  resolve: (value: MobileMessage) => void;
  reject: (reason: Error) => void;
  timeout: ReturnType<typeof setTimeout>;
}

interface EventWaiter {
  predicate: (event: SessionEvent) => boolean;
  resolve: (event: SessionEvent) => void;
  reject: (reason: Error) => void;
  timeout: ReturnType<typeof setTimeout>;
}

// --- Simulator ---

export class WsMobileSimulator {
  private ws: WebSocket | null = null;
  private _deviceId: string;
  private _deviceToken: string | null = null;
  private _bridgeUrl: string = '';
  private _serverId: string = '';

  // E2E encryption
  private crypto = new CryptoService();
  private myKeyPair: { publicKey: Uint8Array; privateKey: Uint8Array } | null = null;
  private sharedSecret: Uint8Array | null = null;
  private encryptCounter = 0;

  // Request-response correlation
  private pendingRequests = new Map<string, PendingRequest>();

  // Event collection
  private eventQueue: SessionEvent[] = [];
  private eventWaiters: EventWaiter[] = [];

  constructor() {
    this._deviceId = uuidv4();
  }

  get deviceId(): string {
    return this._deviceId;
  }

  get isPaired(): boolean {
    return this._deviceToken !== null;
  }

  get deviceToken(): string | null {
    return this._deviceToken;
  }

  get isEncrypted(): boolean {
    return this.sharedSecret !== null;
  }

  /**
   * Connect to Bridge /connect endpoint as a mobile device.
   */
  async connect(bridgeUrl: string, serverId: string): Promise<void> {
    this._bridgeUrl = bridgeUrl;
    this._serverId = serverId;
    await this.connectWs();
  }

  /**
   * Internal: establish WS connection with current device_id.
   */
  private async connectWs(): Promise<void> {
    const wsUrl = `${this._bridgeUrl}/connect?server_id=${this._serverId}&device_id=${this._deviceId}`;
    this.ws = new WebSocket(wsUrl);

    await new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('WS connect timeout')), 5000);

      this.ws!.addEventListener('open', () => {
        clearTimeout(timeout);
        resolve();
      });

      this.ws!.addEventListener('error', (event: Event) => {
        clearTimeout(timeout);
        const errorEvent = event as ErrorEvent;
        reject(new Error(`WS connect failed: ${errorEvent.message ?? 'unknown'}`));
      });
    });

    this.ws.addEventListener('message', (event: MessageEvent) => {
      const data = typeof event.data === 'string' ? event.data : '';
      try {
        const bridgeMsg = JSON.parse(data) as BridgeMessage;
        this.handleBridgeMessage(bridgeMsg);
      } catch {
        // Ignore unparseable messages
      }
    });
  }

  // --- Flows ---

  /**
   * Send pair_request and wait for pair_response.
   * Generates X25519 keys and includes device_public_key in request.
   * If server returns server_public_key, computes sharedSecret for E2E encryption.
   * Pairing messages themselves are always plaintext.
   */
  async pair(pairingToken: string, deviceName = 'Test Device'): Promise<PairResult> {
    // Generate X25519 keypair for ECDH key exchange
    this.myKeyPair = this.crypto.generateKeyPair();

    const response = await this.sendRequest('pair_request', {
      token: pairingToken,
      device_name: deviceName,
      device_public_key: Buffer.from(this.myKeyPair.publicKey).toString('base64'),
    });

    // Check for error response
    if (response.type === 'error') {
      const errorMsg = (response.payload?.message as string) ?? 'unknown pairing error';
      throw new Error(`Pairing failed: ${errorMsg}`);
    }

    const payload = response.payload ?? {};
    const deviceToken = payload.device_token as string | undefined;

    if (!deviceToken) {
      throw new Error('Pairing response missing device_token');
    }

    this._deviceToken = deviceToken;

    // The server may assign a different device_id
    const authenticatedDeviceId = (payload.device_id as string) ?? this._deviceId;

    // If server returned server_public_key, compute sharedSecret for E2E encryption
    const serverPublicKeyB64 = payload.server_public_key as string | undefined;
    if (serverPublicKeyB64) {
      const serverPublicKey = new Uint8Array(Buffer.from(serverPublicKeyB64, 'base64'));
      this.sharedSecret = this.crypto.computeSharedSecret(
        this.myKeyPair.privateKey,
        serverPublicKey,
      );
    }

    // Reconnect with authenticated device_id (like real mobile app).
    if (authenticatedDeviceId !== this._deviceId) {
      this._deviceId = authenticatedDeviceId;
      this.ws?.close();
      this.ws = null;
      await this.connectWs();
      await new Promise((r) => setTimeout(r, 100));
    }

    return {
      deviceId: this._deviceId,
      deviceToken: this._deviceToken,
    };
  }

  /**
   * Ping the CLI and wait for pong response.
   */
  async ping(): Promise<Record<string, unknown>> {
    const response = await this.sendRequest('ping', {});
    return response.payload ?? {};
  }

  /**
   * List sessions (requires authentication via device_token).
   */
  async listSessions(): Promise<Array<Record<string, unknown>>> {
    const response = await this.sendRequest('list_sessions', {
      device_token: this._deviceToken!,
    });

    const payload = response.payload ?? {};
    return (payload.sessions ?? []) as Array<Record<string, unknown>>;
  }

  /**
   * Subscribe to session events.
   */
  async subscribe(sessionId: string): Promise<void> {
    await this.sendRequest('subscribe', {
      device_token: this._deviceToken!,
      session_id: sessionId,
    });
  }

  /**
   * Send a new task (user message) to the CLI agent.
   */
  async sendNewTask(text: string): Promise<MobileMessage> {
    return await this.sendRequest('new_task', {
      device_token: this._deviceToken!,
      text,
    });
  }

  /**
   * Send ask_user reply.
   */
  async sendAskUserReply(sessionId: string, reply: string): Promise<void> {
    await this.sendRequest('ask_user_reply', {
      device_token: this._deviceToken!,
      session_id: sessionId,
      reply,
    });
  }

  /**
   * Cancel current session task.
   */
  async cancelSession(sessionId: string): Promise<void> {
    await this.sendRequest('cancel', {
      device_token: this._deviceToken!,
      session_id: sessionId,
    });
  }

  // --- Event collection ---

  /**
   * Wait for a session event matching the predicate.
   * Checks already-collected events first, then waits for new ones.
   */
  waitForEvent(predicate: (e: SessionEvent) => boolean, timeoutMs = 10000): Promise<SessionEvent> {
    // Check already-collected events
    const idx = this.eventQueue.findIndex(predicate);
    if (idx !== -1) {
      const [event] = this.eventQueue.splice(idx, 1);
      return Promise.resolve(event);
    }

    return new Promise<SessionEvent>((resolve, reject) => {
      const timeout = setTimeout(() => {
        this.eventWaiters = this.eventWaiters.filter((w) => w.resolve !== resolve);
        reject(
          new Error(
            `Event wait timeout (${timeoutMs}ms). Collected events: ${JSON.stringify(this.eventQueue.map((e) => e.type))}`,
          ),
        );
      }, timeoutMs);

      this.eventWaiters.push({ predicate, resolve, reject, timeout });
    });
  }

  /**
   * Wait for N events in sequence.
   */
  async waitForEvents(count: number, timeoutMs = 15000): Promise<SessionEvent[]> {
    const events: SessionEvent[] = [];
    for (let i = 0; i < count; i++) {
      events.push(await this.waitForEvent(() => true, timeoutMs));
    }
    return events;
  }

  /**
   * Collect all events that have arrived so far without waiting.
   */
  drainEvents(): SessionEvent[] {
    const events = [...this.eventQueue];
    this.eventQueue = [];
    return events;
  }

  /**
   * Disconnect from bridge.
   */
  disconnect(): void {
    if (this.ws) {
      try {
        this.ws.close();
      } catch {
        // Ignore close errors
      }
      this.ws = null;
    }

    // Clean up pending requests
    for (const [, pending] of this.pendingRequests) {
      clearTimeout(pending.timeout);
      pending.reject(new Error('Disconnected'));
    }
    this.pendingRequests.clear();

    // Clean up event waiters
    for (const waiter of this.eventWaiters) {
      clearTimeout(waiter.timeout);
      waiter.reject(new Error('Disconnected'));
    }
    this.eventWaiters = [];
  }

  // --- Private ---

  /**
   * Handle incoming bridge-level message.
   *
   * Bridge wraps all CLI responses in: {"type": "data", "payload": ...}
   * Payload is either:
   * - A plaintext MobileMessage object (pairing flow, no encryption)
   * - A base64 string containing encrypted bytes (post-pairing with E2E)
   */
  private handleBridgeMessage(bridgeMsg: BridgeMessage): void {
    if (bridgeMsg.type !== 'data') return;

    const payload = bridgeMsg.payload;
    if (payload === undefined || payload === null) return;

    let innerMessage: MobileMessage;

    if (typeof payload === 'string' && this.sharedSecret !== null) {
      // Encrypted: base64 -> bytes -> decrypt -> JSON.parse
      const ciphertext = new Uint8Array(Buffer.from(payload, 'base64'));
      const plaintext = this.crypto.decrypt(ciphertext, this.sharedSecret);
      const jsonStr = new TextDecoder().decode(plaintext);
      innerMessage = JSON.parse(jsonStr) as MobileMessage;
    } else if (typeof payload === 'object') {
      // Plaintext JSON object (pairing flow or no encryption)
      innerMessage = payload as MobileMessage;
    } else if (typeof payload === 'string') {
      // String but no encryption — try JSON.parse
      innerMessage = JSON.parse(payload) as MobileMessage;
    } else {
      return;
    }

    const requestId = innerMessage.request_id;
    const type = innerMessage.type;

    // Check if this is a response to a pending request
    if (requestId && this.pendingRequests.has(requestId)) {
      const pending = this.pendingRequests.get(requestId)!;
      this.pendingRequests.delete(requestId);
      clearTimeout(pending.timeout);
      pending.resolve(innerMessage);
      return;
    }

    // Otherwise it's a pushed event (session_event)
    if (type === 'session_event') {
      const eventPayload = (innerMessage.payload ?? {}) as Record<string, unknown>;
      const event = (eventPayload.event ?? {}) as SessionEvent;

      // Attach sessionId from the event envelope
      if (eventPayload.session_id) {
        event.sessionId = eventPayload.session_id as string;
      }

      // Check if any waiter matches
      const waiterIdx = this.eventWaiters.findIndex((w) => w.predicate(event));
      if (waiterIdx !== -1) {
        const waiter = this.eventWaiters[waiterIdx];
        this.eventWaiters.splice(waiterIdx, 1);
        clearTimeout(waiter.timeout);
        waiter.resolve(event);
      } else {
        this.eventQueue.push(event);
      }
    }
  }

  /**
   * Send a request and wait for the response with matching request_id.
   */
  private sendRequest(
    type: string,
    payload: Record<string, unknown>,
    requestTimeoutMs = 30000,
  ): Promise<MobileMessage> {
    const requestId = uuidv4();

    return new Promise<MobileMessage>((resolve, reject) => {
      const timeout = setTimeout(() => {
        this.pendingRequests.delete(requestId);
        reject(new Error(`Request ${type} timeout (${requestTimeoutMs}ms)`));
      }, requestTimeoutMs);

      this.pendingRequests.set(requestId, {
        resolve,
        reject,
        timeout,
      });

      this.sendRawMessage(type, requestId, payload);
    });
  }

  /**
   * Send a raw message through the bridge.
   *
   * If E2E encryption is established (sharedSecret !== null):
   *   - Serialize MobileMessage to JSON bytes
   *   - Encrypt with XChaCha20-Poly1305
   *   - Base64 encode -> payload is a string
   *
   * Otherwise (pairing flow, no encryption):
   *   - payload is the MobileMessage object directly
   *
   * Bridge envelope: {"type": "data", "payload": <string | object>}
   */
  private sendRawMessage(
    type: string,
    requestId: string,
    payload: Record<string, unknown>,
  ): void {
    const innerMessage: MobileMessage = {
      type,
      request_id: requestId,
      device_id: this._deviceId,
      payload,
    };

    let bridgePayload: unknown;

    if (this.sharedSecret !== null) {
      // Encrypt: JSON -> bytes -> encrypt -> base64 string
      const jsonBytes = new TextEncoder().encode(JSON.stringify(innerMessage));
      const ciphertext = this.crypto.encrypt(jsonBytes, this.sharedSecret, this.encryptCounter++);
      bridgePayload = Buffer.from(ciphertext).toString('base64');
    } else {
      // Plaintext: send message object directly
      bridgePayload = innerMessage;
    }

    // Wrap in bridge envelope
    const bridgeMessage: BridgeMessage = {
      type: 'data',
      payload: bridgePayload,
    };

    this.ws!.send(JSON.stringify(bridgeMessage));
  }
}
