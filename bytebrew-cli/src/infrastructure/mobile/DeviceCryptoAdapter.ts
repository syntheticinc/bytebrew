/**
 * DeviceCryptoAdapter adapts CryptoService + DeviceStore into the IMessageCrypto
 * interface expected by BridgeMessageRouter.
 *
 * The CryptoService works with raw (sharedSecret, counter) parameters.
 * BridgeMessageRouter needs encrypt/decrypt by deviceId.
 * This adapter bridges the two by looking up the device's sharedSecret
 * from the DeviceStore and managing per-device counters.
 */

import type { ICryptoService } from './CryptoService.js';
import type { IMessageCrypto } from '../bridge/BridgeMessageRouter.js';

/** Consumer-side interface — only needs sharedSecret lookup */
interface DeviceSecretProvider {
  getById(id: string): { sharedSecret: Uint8Array } | undefined;
}

export class DeviceCryptoAdapter implements IMessageCrypto {
  private readonly crypto: ICryptoService;
  private readonly deviceStore: DeviceSecretProvider;
  private readonly counters = new Map<string, number>();
  /** Maps bridge-level device IDs to authenticated device IDs */
  private readonly aliases = new Map<string, string>();

  constructor(crypto: ICryptoService, deviceStore: DeviceSecretProvider) {
    this.crypto = crypto;
    this.deviceStore = deviceStore;
  }

  /**
   * Register an alias so that bridge-level deviceId resolves to the
   * authenticated deviceId stored in DeviceStore.
   */
  registerAlias(bridgeDeviceId: string, authenticatedDeviceId: string): void {
    if (bridgeDeviceId !== authenticatedDeviceId) {
      this.aliases.set(bridgeDeviceId, authenticatedDeviceId);
    }
  }

  hasSharedSecret(deviceId: string): boolean {
    const device = this.deviceStore.getById(this.resolveId(deviceId));
    if (!device) return false;
    return device.sharedSecret.length > 0;
  }

  encrypt(deviceId: string, plaintext: Uint8Array): Uint8Array {
    const resolved = this.resolveId(deviceId);
    const device = this.deviceStore.getById(resolved);
    if (!device || device.sharedSecret.length === 0) {
      throw new Error(`No shared secret for device: ${deviceId}`);
    }

    const counter = this.nextCounter(resolved);
    return this.crypto.encrypt(plaintext, device.sharedSecret, counter);
  }

  decrypt(deviceId: string, ciphertext: Uint8Array): Uint8Array {
    const resolved = this.resolveId(deviceId);
    const device = this.deviceStore.getById(resolved);
    if (!device || device.sharedSecret.length === 0) {
      throw new Error(`No shared secret for device: ${deviceId}`);
    }

    return this.crypto.decrypt(ciphertext, device.sharedSecret);
  }

  /** Resolve a potentially aliased device ID to the authenticated one. */
  private resolveId(deviceId: string): string {
    return this.aliases.get(deviceId) ?? deviceId;
  }

  private nextCounter(deviceId: string): number {
    const current = this.counters.get(deviceId) ?? 0;
    const next = current + 1;
    this.counters.set(deviceId, next);
    return next;
  }
}
