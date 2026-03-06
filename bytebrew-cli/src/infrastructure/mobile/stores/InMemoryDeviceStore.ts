/**
 * In-memory device store for paired mobile devices.
 * Port from Go: bytebrew-srv/internal/infrastructure/mobile/device_store.go
 *
 * Uses dual-index pattern: devices by ID + token-to-ID index for fast auth lookups.
 */

import type { MobileDevice } from '../../../domain/entities/MobileDevice.js';

export interface IDeviceStore {
  add(device: MobileDevice): void;
  getById(id: string): MobileDevice | undefined;
  getByToken(deviceToken: string): MobileDevice | undefined;
  remove(id: string): boolean;
  list(): MobileDevice[];
  updateLastSeen(id: string): void;
}

export class InMemoryDeviceStore implements IDeviceStore {
  private readonly devices = new Map<string, MobileDevice>();
  private readonly tokenIndex = new Map<string, string>(); // deviceToken → deviceId

  add(device: MobileDevice): void {
    // If device already exists, clean up old token mapping
    const existing = this.devices.get(device.id);
    if (existing) {
      this.tokenIndex.delete(existing.deviceToken);
    }

    this.devices.set(device.id, device);
    this.tokenIndex.set(device.deviceToken, device.id);
  }

  getById(id: string): MobileDevice | undefined {
    return this.devices.get(id);
  }

  getByToken(deviceToken: string): MobileDevice | undefined {
    const deviceId = this.tokenIndex.get(deviceToken);
    if (!deviceId) {
      return undefined;
    }
    return this.devices.get(deviceId);
  }

  remove(id: string): boolean {
    const device = this.devices.get(id);
    if (!device) {
      return false;
    }

    this.tokenIndex.delete(device.deviceToken);
    this.devices.delete(id);
    return true;
  }

  list(): MobileDevice[] {
    return Array.from(this.devices.values());
  }

  updateLastSeen(id: string): void {
    const device = this.devices.get(id);
    if (!device) {
      return;
    }

    const updated = device.withUpdatedLastSeen();
    this.devices.set(id, updated);
    // Token index stays valid — deviceToken unchanged
  }
}
