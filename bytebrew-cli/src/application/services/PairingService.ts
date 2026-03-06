/**
 * PairingService handles mobile device pairing lifecycle.
 * Port from Go: bytebrew-srv/internal/usecase/pair_device/usecase.go
 *
 * Generates pairing tokens, performs ECDH key exchange, manages paired devices.
 * Consumer-side interfaces defined in this file (ISP).
 */

import { v4 as uuidv4 } from 'uuid';
import { randomBytes } from 'crypto';
import { MobileDevice } from '../../domain/entities/MobileDevice.js';
import { PairingToken } from '../../domain/entities/PairingToken.js';
import { getLogger } from '../../lib/logger.js';

const logger = getLogger();

/** Number of random bytes for device auth token (256-bit) */
const DEVICE_TOKEN_BYTES = 32;

/** Number of random bytes for pairing token (256-bit) */
const PAIRING_TOKEN_BYTES = 32;

/** Short code max value (6-digit numeric code: 000000-999999) */
const SHORT_CODE_MAX = 1_000_000;

/** Default timeout for waiting for pairing completion */
const DEFAULT_PAIRING_TIMEOUT_MS = 5 * 60 * 1000; // 5 minutes

// --- Consumer-side interfaces ---

interface DeviceStore {
  add(device: MobileDevice): void;
  getById(id: string): MobileDevice | undefined;
  getByToken(deviceToken: string): MobileDevice | undefined;
  remove(id: string): boolean;
  list(): MobileDevice[];
}

interface PairingTokenStore {
  add(token: PairingToken): void;
  useToken(tokenOrCode: string): PairingToken | undefined;
  remove(token: string): void;
}

interface CryptoService {
  generateKeyPair(): { publicKey: Uint8Array; privateKey: Uint8Array };
  computeSharedSecret(privateKey: Uint8Array, peerPublicKey: Uint8Array): Uint8Array;
}

interface PairingWaiter {
  wait(token: string, timeoutMs: number): Promise<{ deviceId: string; deviceName: string }>;
  resolve(token: string, deviceId: string, deviceName: string): void;
}

// --- Output types ---

export interface GenerateTokenResult {
  token: string;
  shortCode: string;
  serverPublicKey: Uint8Array;
}

export interface PairResult {
  deviceId: string;
  deviceToken: string;
}

// --- Service ---

export class PairingService {
  private readonly deviceStore: DeviceStore;
  private readonly tokenStore: PairingTokenStore;
  private readonly crypto: CryptoService;
  private readonly pairingWaiter: PairingWaiter;

  constructor(
    deviceStore: DeviceStore,
    tokenStore: PairingTokenStore,
    crypto: CryptoService,
    pairingWaiter: PairingWaiter,
  ) {
    this.deviceStore = deviceStore;
    this.tokenStore = tokenStore;
    this.crypto = crypto;
    this.pairingWaiter = pairingWaiter;
  }

  /**
   * Generates a new pairing token with X25519 keypair.
   * The token and short code are stored for later use during pair().
   */
  generatePairingToken(): GenerateTokenResult {
    logger.info('generating pairing token');

    // Generate random token (hex-encoded)
    const tokenBytes = randomBytes(PAIRING_TOKEN_BYTES);
    const token = tokenBytes.toString('hex');

    // Generate 6-digit short code
    const codeNum = Math.floor(Math.random() * SHORT_CODE_MAX);
    const shortCode = codeNum.toString().padStart(6, '0');

    // Create domain entity
    const pairingToken = PairingToken.create(token, shortCode);

    // Generate X25519 keypair for ECDH key exchange
    const keyPair = this.crypto.generateKeyPair();
    const tokenWithKeys = pairingToken.withKeys(keyPair.publicKey, keyPair.privateKey);

    // Store token for later pair() call
    this.tokenStore.add(tokenWithKeys);

    logger.info('pairing token generated', { shortCode, hasKeys: true });

    return {
      token,
      shortCode,
      serverPublicKey: keyPair.publicKey,
    };
  }

  /**
   * Completes pairing: validates token, performs ECDH key exchange, creates device.
   * Returns device credentials for the mobile device.
   */
  pair(token: string, devicePublicKey: Uint8Array, deviceName: string): PairResult {
    if (!token) {
      throw new Error('token is required');
    }
    if (!deviceName) {
      throw new Error('device name is required');
    }

    logger.info('pairing device', { deviceName });

    // Atomically find, validate, and mark as used
    const pairingToken = this.tokenStore.useToken(token);
    if (!pairingToken) {
      throw new Error('invalid or expired pairing token');
    }

    // Generate device credentials
    const deviceId = uuidv4();
    const deviceTokenBytes = randomBytes(DEVICE_TOKEN_BYTES);
    const deviceToken = deviceTokenBytes.toString('hex');

    // Create device entity
    const device = MobileDevice.create(deviceId, deviceName, deviceToken);

    // Perform ECDH key exchange if mobile sent a public key and server has a private key
    let finalDevice = device;
    if (devicePublicKey.length > 0 && pairingToken.serverPrivateKey.length > 0) {
      const sharedSecret = this.crypto.computeSharedSecret(
        pairingToken.serverPrivateKey,
        devicePublicKey,
      );
      finalDevice = device.withKeys(devicePublicKey, sharedSecret);
      logger.info('ECDH key exchange completed', { deviceId });
    }

    // Save device
    this.deviceStore.add(finalDevice);

    // Clean up used token (includes private key)
    this.tokenStore.remove(pairingToken.token);

    // Notify waiter that pairing is complete
    this.pairingWaiter.resolve(pairingToken.token, deviceId, deviceName);

    logger.info('device paired successfully', { deviceId, deviceName });

    return { deviceId, deviceToken };
  }

  /**
   * Waits for a mobile device to complete pairing.
   * Resolves when pair() is called with a matching token, or rejects on timeout.
   */
  waitForPairing(
    token: string,
    timeoutMs: number = DEFAULT_PAIRING_TIMEOUT_MS,
  ): Promise<{ deviceId: string; deviceName: string }> {
    return this.pairingWaiter.wait(token, timeoutMs);
  }

  /**
   * Returns all paired devices.
   */
  listDevices(): MobileDevice[] {
    return this.deviceStore.list();
  }

  /**
   * Removes a paired device by ID. Returns true if the device was found and removed.
   */
  revokeDevice(deviceId: string): boolean {
    if (!deviceId) {
      throw new Error('device_id is required');
    }

    const device = this.deviceStore.getById(deviceId);
    if (!device) {
      return false;
    }

    const removed = this.deviceStore.remove(deviceId);
    if (removed) {
      logger.info('device revoked', { deviceId, deviceName: device.name });
    }

    return removed;
  }

  /**
   * Authenticates a device by its token. Used on reconnect to verify identity.
   * Returns the device if the token is valid, undefined otherwise.
   */
  authenticateDevice(deviceToken: string): MobileDevice | undefined {
    if (!deviceToken) {
      return undefined;
    }
    return this.deviceStore.getByToken(deviceToken);
  }
}
