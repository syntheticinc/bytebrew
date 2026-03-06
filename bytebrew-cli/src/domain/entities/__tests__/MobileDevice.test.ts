import { describe, it, expect } from 'bun:test';
import { MobileDevice } from '../MobileDevice.js';

describe('MobileDevice', () => {
  describe('create()', () => {
    it('creates device with given id, name, deviceToken', () => {
      const device = MobileDevice.create('dev-1', 'iPhone 15', 'tok-abc');
      expect(device.id).toBe('dev-1');
      expect(device.name).toBe('iPhone 15');
      expect(device.deviceToken).toBe('tok-abc');
    });

    it('sets pairedAt and lastSeenAt to current time', () => {
      const before = Date.now();
      const device = MobileDevice.create('dev-1', 'Phone', 'tok-1');
      const after = Date.now();

      expect(device.pairedAt.getTime()).toBeGreaterThanOrEqual(before);
      expect(device.pairedAt.getTime()).toBeLessThanOrEqual(after);
      expect(device.lastSeenAt.getTime()).toBe(device.pairedAt.getTime());
    });

    it('sets empty keys by default', () => {
      const device = MobileDevice.create('dev-1', 'Phone', 'tok-1');
      expect(device.publicKey.length).toBe(0);
      expect(device.sharedSecret.length).toBe(0);
    });

    it('throws if id is empty', () => {
      expect(() => MobileDevice.create('', 'Phone', 'tok')).toThrow('id is required');
    });

    it('throws if name is empty', () => {
      expect(() => MobileDevice.create('id', '', 'tok')).toThrow('name is required');
    });

    it('throws if deviceToken is empty', () => {
      expect(() => MobileDevice.create('id', 'Phone', '')).toThrow('device_token is required');
    });
  });

  describe('fromProps()', () => {
    it('restores device from raw props', () => {
      const props = {
        id: 'dev-42',
        name: 'Pixel 8',
        deviceToken: 'tok-xyz',
        publicKey: new Uint8Array([1, 2, 3]),
        sharedSecret: new Uint8Array([4, 5, 6]),
        pairedAt: new Date('2026-01-01'),
        lastSeenAt: new Date('2026-02-01'),
      };

      const device = MobileDevice.fromProps(props);
      expect(device.id).toBe('dev-42');
      expect(device.name).toBe('Pixel 8');
      expect(device.deviceToken).toBe('tok-xyz');
      expect(device.publicKey).toEqual(new Uint8Array([1, 2, 3]));
      expect(device.sharedSecret).toEqual(new Uint8Array([4, 5, 6]));
      expect(device.pairedAt.toISOString()).toBe('2026-01-01T00:00:00.000Z');
      expect(device.lastSeenAt.toISOString()).toBe('2026-02-01T00:00:00.000Z');
    });
  });

  describe('withKeys()', () => {
    it('returns new instance with keys set', () => {
      const original = MobileDevice.create('dev-1', 'Phone', 'tok-1');
      const pubKey = new Uint8Array([10, 20, 30]);
      const secret = new Uint8Array([40, 50, 60]);

      const updated = original.withKeys(pubKey, secret);

      expect(updated.publicKey).toEqual(pubKey);
      expect(updated.sharedSecret).toEqual(secret);
      expect(updated.id).toBe(original.id);
      expect(updated.name).toBe(original.name);
    });

    it('does not mutate original (immutable)', () => {
      const original = MobileDevice.create('dev-1', 'Phone', 'tok-1');
      original.withKeys(new Uint8Array([1]), new Uint8Array([2]));

      expect(original.publicKey.length).toBe(0);
      expect(original.sharedSecret.length).toBe(0);
    });
  });

  describe('withUpdatedLastSeen()', () => {
    it('returns new instance with updated lastSeenAt', () => {
      const original = MobileDevice.fromProps({
        id: 'dev-1',
        name: 'Phone',
        deviceToken: 'tok-1',
        publicKey: new Uint8Array(0),
        sharedSecret: new Uint8Array(0),
        pairedAt: new Date('2026-01-01'),
        lastSeenAt: new Date('2026-01-01'),
      });

      const before = Date.now();
      const updated = original.withUpdatedLastSeen();
      const after = Date.now();

      expect(updated.lastSeenAt.getTime()).toBeGreaterThanOrEqual(before);
      expect(updated.lastSeenAt.getTime()).toBeLessThanOrEqual(after);
      expect(updated.pairedAt.toISOString()).toBe('2026-01-01T00:00:00.000Z');
    });

    it('does not mutate original (immutable)', () => {
      const original = MobileDevice.fromProps({
        id: 'dev-1',
        name: 'Phone',
        deviceToken: 'tok-1',
        publicKey: new Uint8Array(0),
        sharedSecret: new Uint8Array(0),
        pairedAt: new Date('2026-01-01'),
        lastSeenAt: new Date('2026-01-01'),
      });

      original.withUpdatedLastSeen();
      expect(original.lastSeenAt.toISOString()).toBe('2026-01-01T00:00:00.000Z');
    });
  });

  describe('validate()', () => {
    it('passes for valid device', () => {
      const device = MobileDevice.create('dev-1', 'Phone', 'tok-1');
      expect(() => device.validate()).not.toThrow();
    });

    it('throws if pairedAt is epoch zero', () => {
      const device = MobileDevice.fromProps({
        id: 'dev-1',
        name: 'Phone',
        deviceToken: 'tok-1',
        publicKey: new Uint8Array(0),
        sharedSecret: new Uint8Array(0),
        pairedAt: new Date(0),
        lastSeenAt: new Date(),
      });
      expect(() => device.validate()).toThrow('pairedAt is required');
    });
  });
});
