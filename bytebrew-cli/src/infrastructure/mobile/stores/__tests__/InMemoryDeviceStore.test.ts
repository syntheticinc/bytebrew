import { describe, it, expect } from 'bun:test';
import { InMemoryDeviceStore } from '../InMemoryDeviceStore.js';
import { MobileDevice } from '../../../../domain/entities/MobileDevice.js';

function createDevice(id: string, name: string, token: string): MobileDevice {
  return MobileDevice.create(id, name, token);
}

describe('InMemoryDeviceStore', () => {
  describe('add + getById', () => {
    it('stores and retrieves device by id', () => {
      const store = new InMemoryDeviceStore();
      const device = createDevice('dev-1', 'iPhone', 'tok-1');

      store.add(device);

      const found = store.getById('dev-1');
      expect(found).toBeDefined();
      expect(found!.name).toBe('iPhone');
    });

    it('returns undefined for unknown id', () => {
      const store = new InMemoryDeviceStore();
      expect(store.getById('unknown')).toBeUndefined();
    });
  });

  describe('getByToken', () => {
    it('retrieves device by deviceToken via index', () => {
      const store = new InMemoryDeviceStore();
      store.add(createDevice('dev-1', 'Phone', 'tok-abc'));

      const found = store.getByToken('tok-abc');
      expect(found).toBeDefined();
      expect(found!.id).toBe('dev-1');
    });

    it('returns undefined for unknown token', () => {
      const store = new InMemoryDeviceStore();
      expect(store.getByToken('unknown')).toBeUndefined();
    });
  });

  describe('remove', () => {
    it('removes device from both indices', () => {
      const store = new InMemoryDeviceStore();
      store.add(createDevice('dev-1', 'Phone', 'tok-1'));

      const removed = store.remove('dev-1');
      expect(removed).toBe(true);
      expect(store.getById('dev-1')).toBeUndefined();
      expect(store.getByToken('tok-1')).toBeUndefined();
    });

    it('returns false for unknown id', () => {
      const store = new InMemoryDeviceStore();
      expect(store.remove('unknown')).toBe(false);
    });
  });

  describe('updateLastSeen', () => {
    it('updates lastSeenAt for existing device', () => {
      const store = new InMemoryDeviceStore();
      const device = MobileDevice.fromProps({
        id: 'dev-1',
        name: 'Phone',
        deviceToken: 'tok-1',
        publicKey: new Uint8Array(0),
        sharedSecret: new Uint8Array(0),
        pairedAt: new Date('2026-01-01'),
        lastSeenAt: new Date('2026-01-01'),
      });
      store.add(device);

      const before = Date.now();
      store.updateLastSeen('dev-1');

      const updated = store.getById('dev-1');
      expect(updated!.lastSeenAt.getTime()).toBeGreaterThanOrEqual(before);
    });

    it('does nothing for unknown id', () => {
      const store = new InMemoryDeviceStore();
      expect(() => store.updateLastSeen('unknown')).not.toThrow();
    });
  });

  describe('list', () => {
    it('returns all devices', () => {
      const store = new InMemoryDeviceStore();
      store.add(createDevice('dev-1', 'Phone A', 'tok-1'));
      store.add(createDevice('dev-2', 'Phone B', 'tok-2'));

      const all = store.list();
      expect(all).toHaveLength(2);
      expect(all.map((d) => d.id).sort()).toEqual(['dev-1', 'dev-2']);
    });

    it('returns empty array when no devices', () => {
      const store = new InMemoryDeviceStore();
      expect(store.list()).toEqual([]);
    });
  });

  describe('add with duplicate token', () => {
    it('re-adding same device updates token index correctly', () => {
      const store = new InMemoryDeviceStore();
      const original = createDevice('dev-1', 'Phone', 'tok-old');
      store.add(original);

      // Re-add same id with new token
      const updated = MobileDevice.create('dev-1', 'Phone', 'tok-new');
      store.add(updated);

      expect(store.getByToken('tok-old')).toBeUndefined();
      expect(store.getByToken('tok-new')).toBeDefined();
      expect(store.getByToken('tok-new')!.id).toBe('dev-1');
    });
  });
});
