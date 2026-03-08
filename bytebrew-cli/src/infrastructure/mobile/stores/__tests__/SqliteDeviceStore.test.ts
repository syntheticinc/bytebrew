import { describe, it, expect, beforeEach, afterEach } from 'bun:test';
import { tmpdir } from 'os';
import path from 'path';
import { randomUUID } from 'crypto';
import { ByteBrewDatabase } from '../../../persistence/ByteBrewDatabase.js';
import { SqliteDeviceStore } from '../SqliteDeviceStore.js';
import { MobileDevice } from '../../../../domain/entities/MobileDevice.js';

function createDevice(id: string, name: string, token: string): MobileDevice {
  return MobileDevice.create(id, name, token);
}

describe('SqliteDeviceStore', () => {
  let database: ByteBrewDatabase;
  let store: SqliteDeviceStore;

  beforeEach(() => {
    database = new ByteBrewDatabase(':memory:');
    store = new SqliteDeviceStore(database);
  });

  afterEach(() => {
    database.close();
  });

  describe('add + getById', () => {
    it('stores and retrieves device by id', () => {
      const device = createDevice('dev-1', 'iPhone 14', 'token-abc123');

      store.add(device);

      const found = store.getById('dev-1');
      expect(found).toBeDefined();
      expect(found!.id).toBe('dev-1');
      expect(found!.name).toBe('iPhone 14');
      expect(found!.deviceToken).toBe('token-abc123');
    });

    it('returns undefined for unknown id', () => {
      expect(store.getById('unknown')).toBeUndefined();
    });

    it('preserves pairedAt and lastSeenAt timestamps', () => {
      const device = createDevice('dev-1', 'Phone', 'tok-1');
      store.add(device);

      const found = store.getById('dev-1')!;
      // Timestamps should round-trip through ISO string serialization
      expect(Math.abs(found.pairedAt.getTime() - device.pairedAt.getTime())).toBeLessThan(1000);
      expect(Math.abs(found.lastSeenAt.getTime() - device.lastSeenAt.getTime())).toBeLessThan(1000);
    });
  });

  describe('getByToken', () => {
    it('retrieves device by deviceToken', () => {
      store.add(createDevice('dev-1', 'Phone', 'tok-abc'));

      const found = store.getByToken('tok-abc');
      expect(found).toBeDefined();
      expect(found!.id).toBe('dev-1');
    });

    it('returns undefined for unknown token', () => {
      expect(store.getByToken('unknown')).toBeUndefined();
    });

    it('distinguishes between different tokens', () => {
      store.add(createDevice('dev-1', 'Phone A', 'tok-aaa'));
      store.add(createDevice('dev-2', 'Phone B', 'tok-bbb'));

      expect(store.getByToken('tok-aaa')!.id).toBe('dev-1');
      expect(store.getByToken('tok-bbb')!.id).toBe('dev-2');
    });
  });

  describe('remove', () => {
    it('removes device and returns true', () => {
      store.add(createDevice('dev-1', 'Phone', 'tok-1'));

      const removed = store.remove('dev-1');
      expect(removed).toBe(true);
      expect(store.getById('dev-1')).toBeUndefined();
      expect(store.getByToken('tok-1')).toBeUndefined();
    });

    it('returns false for non-existent id', () => {
      expect(store.remove('non-existent')).toBe(false);
    });

    it('returns false on second removal of same id', () => {
      store.add(createDevice('dev-1', 'Phone', 'tok-1'));
      store.remove('dev-1');

      expect(store.remove('dev-1')).toBe(false);
    });
  });

  describe('list', () => {
    it('returns all devices', () => {
      store.add(createDevice('dev-1', 'Phone A', 'tok-1'));
      store.add(createDevice('dev-2', 'Phone B', 'tok-2'));
      store.add(createDevice('dev-3', 'Phone C', 'tok-3'));

      const all = store.list();
      expect(all).toHaveLength(3);
      expect(all.map((d) => d.id).sort()).toEqual(['dev-1', 'dev-2', 'dev-3']);
    });

    it('returns empty array when no devices', () => {
      expect(store.list()).toEqual([]);
    });

    it('reflects removals', () => {
      store.add(createDevice('dev-1', 'Phone A', 'tok-1'));
      store.add(createDevice('dev-2', 'Phone B', 'tok-2'));
      store.remove('dev-1');

      const all = store.list();
      expect(all).toHaveLength(1);
      expect(all[0].id).toBe('dev-2');
    });
  });

  describe('updateLastSeen', () => {
    it('updates lastSeenAt timestamp', () => {
      const device = MobileDevice.fromProps({
        id: 'dev-1',
        name: 'Phone',
        deviceToken: 'tok-1',
        publicKey: new Uint8Array(0),
        sharedSecret: new Uint8Array(0),
        pairedAt: new Date('2026-01-01T00:00:00Z'),
        lastSeenAt: new Date('2026-01-01T00:00:00Z'),
      });
      store.add(device);

      const before = Date.now();
      store.updateLastSeen('dev-1');

      const updated = store.getById('dev-1')!;
      expect(updated.lastSeenAt.getTime()).toBeGreaterThanOrEqual(before - 1000);
    });

    it('does not throw for unknown id', () => {
      expect(() => store.updateLastSeen('unknown')).not.toThrow();
    });

    it('preserves other fields when updating lastSeen', () => {
      const publicKey = new Uint8Array(32).fill(0xaa);
      const sharedSecret = new Uint8Array(32).fill(0xbb);
      const device = createDevice('dev-1', 'My Phone', 'tok-1').withKeys(publicKey, sharedSecret);
      store.add(device);

      store.updateLastSeen('dev-1');

      const updated = store.getById('dev-1')!;
      expect(updated.name).toBe('My Phone');
      expect(updated.deviceToken).toBe('tok-1');
      expect(new Uint8Array(updated.publicKey)).toEqual(publicKey);
      expect(new Uint8Array(updated.sharedSecret)).toEqual(sharedSecret);
    });
  });

  describe('re-add with new token (INSERT OR REPLACE)', () => {
    it('replaces device and old token lookup fails', () => {
      store.add(createDevice('dev-1', 'Phone', 'tok-old'));

      const replacement = MobileDevice.create('dev-1', 'Phone Updated', 'tok-new');
      store.add(replacement);

      expect(store.getByToken('tok-old')).toBeUndefined();
      expect(store.getByToken('tok-new')).toBeDefined();
      expect(store.getByToken('tok-new')!.name).toBe('Phone Updated');
      expect(store.list()).toHaveLength(1);
    });
  });

  describe('persistence across close/reopen', () => {
    it('retains devices after database is closed and reopened', () => {
      const tmpPath = path.join(tmpdir(), `bytebrew-test-${randomUUID()}.db`);

      try {
        const db1 = new ByteBrewDatabase(tmpPath);
        const store1 = new SqliteDeviceStore(db1);
        const publicKey = new Uint8Array(32).fill(0x11);
        const sharedSecret = new Uint8Array(32).fill(0x22);
        const device = createDevice('dev-persist', 'Persistent Phone', 'tok-persist').withKeys(
          publicKey,
          sharedSecret,
        );
        store1.add(device);
        db1.close();

        // Reopen
        const db2 = new ByteBrewDatabase(tmpPath);
        const store2 = new SqliteDeviceStore(db2);

        const found = store2.getById('dev-persist');
        expect(found).toBeDefined();
        expect(found!.name).toBe('Persistent Phone');
        expect(found!.deviceToken).toBe('tok-persist');
        expect(new Uint8Array(found!.publicKey)).toEqual(publicKey);
        expect(new Uint8Array(found!.sharedSecret)).toEqual(sharedSecret);

        db2.close();
      } finally {
        // Clean up temp files
        const { unlinkSync, existsSync } = require('fs');
        for (const suffix of ['', '-wal', '-shm']) {
          const f = tmpPath + suffix;
          if (existsSync(f)) {
            try {
              unlinkSync(f);
            } catch {
              // Ignore cleanup errors on Windows
            }
          }
        }
      }
    });
  });

  describe('binary fields round-trip', () => {
    it('stores and retrieves 32-byte publicKey and sharedSecret', () => {
      const publicKey = new Uint8Array(32);
      const sharedSecret = new Uint8Array(32);
      // Fill with distinct patterns
      for (let i = 0; i < 32; i++) {
        publicKey[i] = i;
        sharedSecret[i] = 255 - i;
      }

      const device = createDevice('dev-bin', 'Crypto Phone', 'tok-bin').withKeys(
        publicKey,
        sharedSecret,
      );
      store.add(device);

      const found = store.getById('dev-bin')!;
      expect(new Uint8Array(found.publicKey)).toEqual(publicKey);
      expect(new Uint8Array(found.sharedSecret)).toEqual(sharedSecret);
      expect(found.publicKey.length).toBe(32);
      expect(found.sharedSecret.length).toBe(32);
    });

    it('handles all-zero keys correctly', () => {
      const zeroKey = new Uint8Array(32); // all zeros
      const device = createDevice('dev-zero', 'Zero Phone', 'tok-zero').withKeys(zeroKey, zeroKey);
      store.add(device);

      const found = store.getById('dev-zero')!;
      expect(new Uint8Array(found.publicKey)).toEqual(zeroKey);
      expect(new Uint8Array(found.sharedSecret)).toEqual(zeroKey);
    });

    it('handles all-0xFF keys correctly', () => {
      const maxKey = new Uint8Array(32).fill(0xff);
      const device = createDevice('dev-ff', 'FF Phone', 'tok-ff').withKeys(maxKey, maxKey);
      store.add(device);

      const found = store.getById('dev-ff')!;
      expect(new Uint8Array(found.publicKey)).toEqual(maxKey);
      expect(new Uint8Array(found.sharedSecret)).toEqual(maxKey);
    });
  });

  describe('empty binary fields', () => {
    it('returns Uint8Array(0) when no keys are set', () => {
      const device = createDevice('dev-nokeys', 'Plain Phone', 'tok-nokeys');
      // device.publicKey and sharedSecret are Uint8Array(0) by default
      store.add(device);

      const found = store.getById('dev-nokeys')!;
      expect(found.publicKey).toBeInstanceOf(Uint8Array);
      expect(found.publicKey.length).toBe(0);
      expect(found.sharedSecret).toBeInstanceOf(Uint8Array);
      expect(found.sharedSecret.length).toBe(0);
    });
  });
});
