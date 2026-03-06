import { describe, test, expect } from 'bun:test';
import { PairingService } from '../PairingService';
import { InMemoryDeviceStore } from '../../../infrastructure/mobile/stores/InMemoryDeviceStore';
import { InMemoryPairingTokenStore } from '../../../infrastructure/mobile/stores/InMemoryPairingTokenStore';

// --- Mock CryptoService (deterministic keys) ---

const FAKE_PUBLIC = new Uint8Array([1, 2, 3]);
const FAKE_PRIVATE = new Uint8Array([4, 5, 6]);
const FAKE_SHARED = new Uint8Array([7, 8, 9]);

function mockCrypto() {
  return {
    generateKeyPair: () => ({ publicKey: FAKE_PUBLIC, privateKey: FAKE_PRIVATE }),
    computeSharedSecret: () => FAKE_SHARED,
  };
}

// --- Mock PairingWaiter ---

function mockWaiter() {
  const resolved: Array<{ token: string; deviceId: string; deviceName: string }> = [];
  return {
    waiter: {
      wait: (_token: string, _timeout: number) =>
        Promise.resolve({ deviceId: '', deviceName: '' }),
      resolve: (token: string, deviceId: string, deviceName: string) => {
        resolved.push({ token, deviceId, deviceName });
      },
    },
    resolved,
  };
}

function createService() {
  const deviceStore = new InMemoryDeviceStore();
  const tokenStore = new InMemoryPairingTokenStore();
  const crypto = mockCrypto();
  const { waiter, resolved } = mockWaiter();

  const service = new PairingService(deviceStore, tokenStore, crypto, waiter);
  return { service, deviceStore, tokenStore, resolved };
}

describe('PairingService', () => {
  test('generatePairingToken creates token and returns publicKey', () => {
    const { service } = createService();

    const result = service.generatePairingToken();

    expect(result.token).toBeTruthy();
    expect(result.shortCode).toMatch(/^\d{1,6}$/);
    expect(result.serverPublicKey).toEqual(FAKE_PUBLIC);
  });

  test('pair with valid token creates device and resolves waiter', () => {
    const { service, deviceStore, resolved } = createService();

    const { token } = service.generatePairingToken();
    const devicePub = new Uint8Array([10, 11, 12]);
    const result = service.pair(token, devicePub, 'iPhone');

    expect(result.deviceId).toBeTruthy();
    expect(result.deviceToken).toBeTruthy();
    expect(result.serverPublicKey).toEqual(FAKE_PUBLIC);

    // Device saved in store
    const device = deviceStore.getById(result.deviceId);
    expect(device).toBeDefined();
    expect(device!.name).toBe('iPhone');
    expect(device!.sharedSecret).toEqual(FAKE_SHARED);

    // Waiter resolved
    expect(resolved).toHaveLength(1);
    expect(resolved[0].deviceName).toBe('iPhone');
  });

  test('pair with invalid token throws', () => {
    const { service } = createService();

    expect(() => service.pair('nonexistent', new Uint8Array(0), 'Phone')).toThrow(
      'invalid or expired pairing token',
    );
  });

  test('pair with already used token throws', () => {
    const { service } = createService();

    const { token } = service.generatePairingToken();
    service.pair(token, new Uint8Array(0), 'Phone1');

    // Second use of same token
    expect(() => service.pair(token, new Uint8Array(0), 'Phone2')).toThrow(
      'invalid or expired pairing token',
    );
  });

  test('authenticateDevice with valid token returns device', () => {
    const { service } = createService();

    const { token } = service.generatePairingToken();
    const { deviceToken } = service.pair(token, new Uint8Array(0), 'Pixel');

    const device = service.authenticateDevice(deviceToken);
    expect(device).toBeDefined();
    expect(device!.name).toBe('Pixel');
  });

  test('authenticateDevice with unknown token returns undefined', () => {
    const { service } = createService();
    expect(service.authenticateDevice('unknown-token')).toBeUndefined();
  });

  test('authenticateDevice with empty token returns undefined', () => {
    const { service } = createService();
    expect(service.authenticateDevice('')).toBeUndefined();
  });

  test('listDevices returns all paired devices', () => {
    const { service } = createService();

    const t1 = service.generatePairingToken();
    const t2 = service.generatePairingToken();
    service.pair(t1.token, new Uint8Array(0), 'Dev1');
    service.pair(t2.token, new Uint8Array(0), 'Dev2');

    const devices = service.listDevices();
    expect(devices).toHaveLength(2);
    const names = devices.map((d) => d.name).sort();
    expect(names).toEqual(['Dev1', 'Dev2']);
  });

  test('revokeDevice removes device and returns true', () => {
    const { service } = createService();

    const { token } = service.generatePairingToken();
    const { deviceId } = service.pair(token, new Uint8Array(0), 'ToRevoke');

    expect(service.revokeDevice(deviceId)).toBe(true);
    expect(service.listDevices()).toHaveLength(0);
  });

  test('revokeDevice for unknown ID returns false', () => {
    const { service } = createService();
    expect(service.revokeDevice('nonexistent')).toBe(false);
  });
});
