// BridgeConnector — WebSocket client to Bridge relay.
//
// Maintains an outbound WS connection from CLI to Bridge (/register endpoint),
// which bypasses NAT. The bridge relays JSON messages between mobile devices
// and this CLI instance.

import { getLogger, type Logger } from '../../lib/logger.js';

// --- WS Protocol types ---

/** Outgoing message: CLI registers with Bridge */
interface RegisterMessage {
  type: 'register';
  server_id: string;
  server_name: string;
  auth_token: string;
}

/** Outgoing message: CLI sends data to a specific mobile device */
interface DataOutMessage {
  type: 'data';
  device_id: string;
  payload: unknown;
}

/** Incoming message types from Bridge */
interface RegisteredMessage {
  type: 'registered';
}

interface DeviceConnectedMessage {
  type: 'device_connected';
  device_id: string;
}

interface DeviceDisconnectedMessage {
  type: 'device_disconnected';
  device_id: string;
}

interface DataInMessage {
  type: 'data';
  device_id: string;
  payload: unknown;
}

type BridgeIncomingMessage =
  | RegisteredMessage
  | DeviceConnectedMessage
  | DeviceDisconnectedMessage
  | DataInMessage;

// --- Public types ---

type DataHandler = (deviceId: string, payload: unknown) => void;
type DeviceHandler = (deviceId: string) => void;

export interface IBridgeConnector {
  connect(bridgeUrl: string, serverId: string, serverName: string, authToken: string): Promise<void>;
  disconnect(): void;
  sendData(deviceId: string, payload: unknown): void;
  onData(handler: DataHandler): void;
  onDeviceConnect(handler: DeviceHandler): void;
  onDeviceDisconnect(handler: DeviceHandler): void;
  isConnected(): boolean;
}

// --- Backoff constants ---

const INITIAL_DELAY_MS = 1000;
const MAX_DELAY_MS = 30_000;
const MAX_BACKOFF_EXPONENT = 5; // 1<<5 = 32s, capped to 30s
const CONNECT_TIMEOUT_MS = 10_000;

// --- BridgeConnector ---

export class BridgeConnector implements IBridgeConnector {
  private logger: Logger;

  private bridgeUrl: string = '';
  private serverId: string = '';
  private serverName: string = '';
  private authToken: string = '';

  private ws: WebSocket | null = null;
  private connected: boolean = false;
  private stopped: boolean = false;

  // Disconnect notification (replaces polling)
  private disconnectResolve: (() => void) | null = null;
  private disconnectPromise: Promise<void> | null = null;

  // Handlers
  private dataHandlers: DataHandler[] = [];
  private deviceConnectHandlers: DeviceHandler[] = [];
  private deviceDisconnectHandlers: DeviceHandler[] = [];

  constructor() {
    this.logger = getLogger().child({ component: 'BridgeConnector' });
  }

  /**
   * Connect to Bridge and maintain the connection with auto-reconnect.
   * Resolves after the first successful registration. Reconnects happen in background.
   */
  async connect(bridgeUrl: string, serverId: string, serverName: string, authToken: string): Promise<void> {
    this.bridgeUrl = bridgeUrl;
    this.serverId = serverId;
    this.serverName = serverName;
    this.authToken = authToken;
    this.stopped = false;

    await this.connectOnce();

    // Start reconnect loop in background (does not block)
    this.reconnectLoop();
  }

  disconnect(): void {
    this.stopped = true;
    this.cleanup();
  }

  sendData(deviceId: string, payload: unknown): void {
    if (!this.connected || !this.ws) {
      this.logger.warn('Cannot send data: not connected to bridge');
      return;
    }

    const msg: DataOutMessage = {
      type: 'data',
      device_id: deviceId,
      payload,
    };

    this.ws.send(JSON.stringify(msg));
  }

  onData(handler: DataHandler): void {
    this.dataHandlers.push(handler);
  }

  onDeviceConnect(handler: DeviceHandler): void {
    this.deviceConnectHandlers.push(handler);
  }

  onDeviceDisconnect(handler: DeviceHandler): void {
    this.deviceDisconnectHandlers.push(handler);
  }

  isConnected(): boolean {
    return this.connected;
  }

  // --- Private ---

  /**
   * Build the WS URL for the /register endpoint.
   * Accepts formats: "host:port", "ws://host:port", "wss://host:port".
   */
  private buildWsUrl(): string {
    const addr = this.bridgeUrl;

    // Already has scheme
    if (addr.startsWith('ws://') || addr.startsWith('wss://')) {
      const base = addr.endsWith('/') ? addr.slice(0, -1) : addr;
      return `${base}/register`;
    }

    // Determine scheme: port 443 implies wss, otherwise ws
    const scheme = addr.includes(':443') ? 'wss' : 'ws';
    return `${scheme}://${addr}/register`;
  }

  /**
   * Single connection attempt: open WS, send register, wait for 'registered'.
   */
  private connectOnce(): Promise<void> {
    return new Promise<void>((resolve, reject) => {
      const url = this.buildWsUrl();

      this.logger.info('Connecting to bridge', {
        url,
        serverId: this.serverId,
      });

      const ws = new WebSocket(url);
      let registered = false;

      const timeout = setTimeout(() => {
        if (!registered) {
          ws.close();
          reject(new Error(`Bridge connect timeout after ${CONNECT_TIMEOUT_MS}ms`));
        }
      }, CONNECT_TIMEOUT_MS);

      ws.addEventListener('open', () => {
        // Send register message
        const registerMsg: RegisterMessage = {
          type: 'register',
          server_id: this.serverId,
          server_name: this.serverName,
          auth_token: this.authToken,
        };
        ws.send(JSON.stringify(registerMsg));
      });

      ws.addEventListener('message', (event: MessageEvent) => {
        const data = typeof event.data === 'string' ? event.data : '';
        let parsed: BridgeIncomingMessage;

        try {
          parsed = JSON.parse(data) as BridgeIncomingMessage;
        } catch {
          this.logger.warn('Failed to parse bridge message', { data });
          return;
        }

        if (!registered && parsed.type === 'registered') {
          registered = true;
          clearTimeout(timeout);

          this.ws = ws;
          this.connected = true;

          // Create a promise that resolves when connection drops
          this.disconnectPromise = new Promise<void>((res) => {
            this.disconnectResolve = res;
          });

          this.logger.info('Connected to bridge', {
            url,
            serverId: this.serverId,
          });

          resolve();
          return;
        }

        // After registration, route messages
        if (registered) {
          this.handleIncomingMessage(parsed);
        }
      });

      ws.addEventListener('error', (event: Event) => {
        const errorEvent = event as ErrorEvent;
        const message = errorEvent.message ?? 'WebSocket error';

        if (!registered) {
          clearTimeout(timeout);
          reject(new Error(`Bridge connection error: ${message}`));
          return;
        }

        this.logger.error('Bridge WS error', { error: message });
        this.setConnected(false);
      });

      ws.addEventListener('close', () => {
        if (!registered) {
          clearTimeout(timeout);
          reject(new Error('Bridge connection closed before registration'));
          return;
        }

        this.logger.info('Bridge WS closed');
        this.setConnected(false);
      });
    });
  }

  /**
   * Route incoming message to the appropriate handler set.
   */
  private handleIncomingMessage(msg: BridgeIncomingMessage): void {
    switch (msg.type) {
      case 'device_connected': {
        const deviceId = msg.device_id;
        this.logger.info('Device connected via bridge', { deviceId });
        for (const handler of this.deviceConnectHandlers) {
          handler(deviceId);
        }
        break;
      }

      case 'device_disconnected': {
        const deviceId = msg.device_id;
        this.logger.info('Device disconnected via bridge', { deviceId });
        for (const handler of this.deviceDisconnectHandlers) {
          handler(deviceId);
        }
        break;
      }

      case 'data': {
        const deviceId = msg.device_id;
        for (const handler of this.dataHandlers) {
          handler(deviceId, msg.payload);
        }
        break;
      }

      default:
        this.logger.warn('Unknown message type from bridge', {
          type: (msg as { type: string }).type,
        });
    }
  }

  /**
   * Background reconnect loop with exponential backoff.
   * 1s, 2s, 4s, 8s, 16s, cap 30s.
   */
  private async reconnectLoop(): Promise<void> {
    let attempt = 0;

    while (!this.stopped) {
      // Wait until disconnected
      await this.waitUntilDisconnected();

      if (this.stopped) break;

      const delay = this.backoffDelay(attempt);
      attempt++;

      this.logger.info('Bridge reconnecting', { delay, attempt });

      await this.sleep(delay);

      if (this.stopped) break;

      try {
        this.cleanup();
        await this.connectOnce();
        // Reset attempt counter on success
        attempt = 0;
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        this.logger.error('Bridge reconnect failed', { error: message, attempt });
      }
    }
  }

  /**
   * Exponential backoff: 1s, 2s, 4s, 8s, 16s, capped at 30s.
   */
  private backoffDelay(attempt: number): number {
    const exp = Math.min(attempt, MAX_BACKOFF_EXPONENT);
    const delay = INITIAL_DELAY_MS * (1 << exp);
    return Math.min(delay, MAX_DELAY_MS);
  }

  /**
   * Wait until the connection drops.
   * Returns immediately if already disconnected.
   */
  private async waitUntilDisconnected(): Promise<void> {
    if (!this.connected || !this.disconnectPromise) return;
    await this.disconnectPromise;
  }

  private setConnected(value: boolean): void {
    this.connected = value;
    if (!value && this.disconnectResolve) {
      this.disconnectResolve();
      this.disconnectResolve = null;
      this.disconnectPromise = null;
    }
  }

  private cleanup(): void {
    if (this.ws) {
      try {
        this.ws.close();
      } catch {
        // Ignore close errors
      }
      this.ws = null;
    }

    this.connected = false;
  }

  private sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}
