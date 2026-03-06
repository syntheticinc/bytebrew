// BridgeMessageRouter — routes messages between BridgeConnector and business logic.
//
// Responsibilities:
// - Incoming data payload -> decrypt (if paired) -> JSON.parse -> MobileMessage -> callback
// - sendMessage -> JSON.stringify -> encrypt (if paired) -> BridgeConnector.sendData
// - For unpaired devices (pairing flow) — no encryption (shared_secret not yet established)
//
// Payload format:
// - Plaintext (pairing): payload is a JSON object (MobileMessage)
// - Encrypted (post-pairing): payload is a base64 string (encrypted bytes)

import type { IBridgeConnector } from './BridgeConnector.js';
import { getLogger, type Logger } from '../../lib/logger.js';

// --- Types ---

/** JSON message format exchanged between CLI and Mobile via Bridge */
export interface MobileMessage {
  type: string;          // "pair_request" | "new_task" | "ask_user_reply" | "cancel" | "subscribe" | "list_sessions" | ...
  request_id: string;    // UUID for request-response correlation
  device_id: string;
  payload: Record<string, unknown>;
}

/** Encryption/decryption interface (consumer-side, will be implemented by CryptoService) */
export interface IMessageCrypto {
  encrypt(deviceId: string, plaintext: Uint8Array): Uint8Array;
  decrypt(deviceId: string, ciphertext: Uint8Array): Uint8Array;
  hasSharedSecret(deviceId: string): boolean;
}

type MessageHandler = (deviceId: string, message: MobileMessage) => void;
type DeviceEventHandler = (deviceId: string) => void;

type Unsubscribe = () => void;

export interface IBridgeMessageRouter {
  start(connector: IBridgeConnector): void;
  stop(): void;
  onMessage(handler: MessageHandler): Unsubscribe;
  onDeviceConnect(handler: DeviceEventHandler): Unsubscribe;
  onDeviceDisconnect(handler: DeviceEventHandler): Unsubscribe;
  sendMessage(deviceId: string, message: MobileMessage): void;
}

// --- Helpers ---

/** Encode bytes to base64 string */
function bytesToBase64(bytes: Uint8Array): string {
  return Buffer.from(bytes).toString('base64');
}

/** Decode base64 string to bytes */
function base64ToBytes(b64: string): Uint8Array {
  return new Uint8Array(Buffer.from(b64, 'base64'));
}

// --- BridgeMessageRouter ---

export class BridgeMessageRouter implements IBridgeMessageRouter {
  private logger: Logger;
  private connector: IBridgeConnector | null = null;
  private crypto: IMessageCrypto | null;
  private messageHandlers: MessageHandler[] = [];
  private deviceConnectHandlers: DeviceEventHandler[] = [];
  private deviceDisconnectHandlers: DeviceEventHandler[] = [];
  private running: boolean = false;

  /**
   * @param crypto Optional crypto service. When null, all messages are sent/received in plaintext.
   */
  constructor(crypto: IMessageCrypto | null = null) {
    this.logger = getLogger().child({ component: 'BridgeMessageRouter' });
    this.crypto = crypto;
  }

  start(connector: IBridgeConnector): void {
    if (this.running) {
      this.logger.warn('Router already started');
      return;
    }

    this.connector = connector;
    this.running = true;

    // Subscribe to data from BridgeConnector
    connector.onData((deviceId, payload) => {
      this.handleData(deviceId, payload);
    });

    connector.onDeviceConnect((deviceId) => {
      this.logger.info('Device connected', { deviceId });
      for (const handler of this.deviceConnectHandlers) {
        handler(deviceId);
      }
    });

    connector.onDeviceDisconnect((deviceId) => {
      this.logger.info('Device disconnected', { deviceId });
      for (const handler of this.deviceDisconnectHandlers) {
        handler(deviceId);
      }
    });

    this.logger.info('BridgeMessageRouter started');
  }

  stop(): void {
    this.running = false;
    this.connector = null;
    this.messageHandlers = [];
    this.deviceConnectHandlers = [];
    this.deviceDisconnectHandlers = [];
    this.logger.info('BridgeMessageRouter stopped');
  }

  onMessage(handler: MessageHandler): Unsubscribe {
    this.messageHandlers.push(handler);
    return () => {
      this.messageHandlers = this.messageHandlers.filter((h) => h !== handler);
    };
  }

  onDeviceConnect(handler: DeviceEventHandler): Unsubscribe {
    this.deviceConnectHandlers.push(handler);
    return () => {
      this.deviceConnectHandlers = this.deviceConnectHandlers.filter((h) => h !== handler);
    };
  }

  onDeviceDisconnect(handler: DeviceEventHandler): Unsubscribe {
    this.deviceDisconnectHandlers.push(handler);
    return () => {
      this.deviceDisconnectHandlers = this.deviceDisconnectHandlers.filter((h) => h !== handler);
    };
  }

  sendMessage(deviceId: string, message: MobileMessage): void {
    if (!this.running || !this.connector) {
      this.logger.warn('Cannot send message: router not started');
      return;
    }

    const encrypted = this.shouldEncrypt(deviceId);

    if (encrypted) {
      // Encrypt: JSON -> bytes -> encrypt -> base64 string payload
      const jsonBytes = new TextEncoder().encode(JSON.stringify(message));
      const ciphertext = this.crypto!.encrypt(deviceId, jsonBytes);
      const b64 = bytesToBase64(ciphertext);
      this.connector.sendData(deviceId, b64);
    } else {
      // Plaintext: send message object directly as JSON payload
      this.connector.sendData(deviceId, message);
    }

    this.logger.debug('Message sent', {
      deviceId,
      type: message.type,
      requestId: message.request_id,
      encrypted,
    });
  }

  // --- Private ---

  /**
   * Handle incoming data from BridgeConnector.
   *
   * Payload is `unknown`:
   * - string => base64 encrypted bytes (post-pairing)
   * - object => plaintext MobileMessage (pairing flow)
   */
  private handleData(deviceId: string, payload: unknown): void {
    if (!this.running) return;

    try {
      let message: MobileMessage;

      if (typeof payload === 'string' && this.shouldEncrypt(deviceId)) {
        // Encrypted: base64 -> bytes -> decrypt -> JSON.parse
        const ciphertext = base64ToBytes(payload);
        const plaintext = this.crypto!.decrypt(deviceId, ciphertext);
        const jsonStr = new TextDecoder().decode(plaintext);
        message = JSON.parse(jsonStr) as MobileMessage;
      } else if (typeof payload === 'object' && payload !== null) {
        // Plaintext JSON object (pairing flow)
        message = payload as MobileMessage;
      } else if (typeof payload === 'string') {
        // String but no encryption — try JSON.parse (could be JSON string payload)
        message = JSON.parse(payload) as MobileMessage;
      } else {
        this.logger.warn('Unexpected payload type from bridge', {
          deviceId,
          payloadType: typeof payload,
        });
        return;
      }

      this.logger.debug('Message received', {
        deviceId,
        type: message.type,
        requestId: message.request_id,
        encrypted: typeof payload === 'string' && this.shouldEncrypt(deviceId),
      });

      for (const handler of this.messageHandlers) {
        handler(deviceId, message);
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : String(err);
      this.logger.error('Failed to process incoming data', {
        deviceId,
        error: errorMessage,
      });
    }
  }

  /**
   * Determines whether encryption should be used for a given device.
   * Returns true only when a CryptoService is available AND the device
   * has an established shared secret (post-pairing).
   */
  private shouldEncrypt(deviceId: string): boolean {
    if (!this.crypto) return false;
    return this.crypto.hasSharedSecret(deviceId);
  }
}
