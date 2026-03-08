import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import { CliIdentity } from '../CliIdentity';
import { ByteBrewDatabase } from '../../persistence/ByteBrewDatabase';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';

function createMockCrypto() {
  let callCount = 0;
  return {
    generateKeyPair() {
      callCount++;
      return {
        publicKey: new Uint8Array([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32]),
        privateKey: new Uint8Array([32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1]),
      };
    },
    get callCount() {
      return callCount;
    },
  };
}

async function deleteWithRetry(dir: string, attempts = 5, delayMs = 50): Promise<void> {
  for (let i = 0; i < attempts; i++) {
    try {
      fs.rmSync(dir, { recursive: true, force: true });
      return;
    } catch {
      if (i === attempts - 1) return;
      await new Promise((r) => setTimeout(r, delayMs));
    }
  }
}

describe('CliIdentity', () => {
  let tempDir: string;
  let database: ByteBrewDatabase;

  beforeEach(() => {
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'cli-identity-test-'));
    const dbPath = path.join(tempDir, 'test.db');
    database = new ByteBrewDatabase(dbPath);
  });

  afterEach(async () => {
    database.close();
    await deleteWithRetry(tempDir);
  });

  describe('getServerId', () => {
    test('generates UUID on first call', () => {
      const crypto = createMockCrypto();
      const identity = new CliIdentity(database, crypto);

      const serverId = identity.getServerId();

      expect(serverId).toMatch(
        /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i,
      );
    });

    test('returns same value on subsequent calls', () => {
      const crypto = createMockCrypto();
      const identity = new CliIdentity(database, crypto);

      const first = identity.getServerId();
      const second = identity.getServerId();

      expect(second).toBe(first);
    });

    test('persists across instances', () => {
      const crypto = createMockCrypto();

      const identity1 = new CliIdentity(database, crypto);
      const serverId = identity1.getServerId();

      const identity2 = new CliIdentity(database, crypto);
      const serverId2 = identity2.getServerId();

      expect(serverId2).toBe(serverId);
    });
  });

  describe('getKeyPair', () => {
    test('generates keypair on first call', () => {
      const crypto = createMockCrypto();
      const identity = new CliIdentity(database, crypto);

      const keyPair = identity.getKeyPair();

      expect(keyPair.publicKey).toBeInstanceOf(Uint8Array);
      expect(keyPair.privateKey).toBeInstanceOf(Uint8Array);
      expect(keyPair.publicKey.length).toBe(32);
      expect(keyPair.privateKey.length).toBe(32);
    });

    test('calls crypto.generateKeyPair exactly once', () => {
      const crypto = createMockCrypto();
      const identity = new CliIdentity(database, crypto);

      identity.getKeyPair();
      identity.getKeyPair();

      expect(crypto.callCount).toBe(1);
    });

    test('returns same values on subsequent calls', () => {
      const crypto = createMockCrypto();
      const identity = new CliIdentity(database, crypto);

      const first = identity.getKeyPair();
      const second = identity.getKeyPair();

      expect(second.publicKey).toEqual(first.publicKey);
      expect(second.privateKey).toEqual(first.privateKey);
    });

    test('persists across instances', () => {
      const crypto = createMockCrypto();

      const identity1 = new CliIdentity(database, crypto);
      const keyPair1 = identity1.getKeyPair();

      const identity2 = new CliIdentity(database, crypto);
      const keyPair2 = identity2.getKeyPair();

      expect(keyPair2.publicKey).toEqual(keyPair1.publicKey);
      expect(keyPair2.privateKey).toEqual(keyPair1.privateKey);
      expect(crypto.callCount).toBe(1);
    });

    test('stores keys as base64 in config table', () => {
      const crypto = createMockCrypto();
      const identity = new CliIdentity(database, crypto);
      identity.getKeyPair();

      const pubRow = database.db.prepare('SELECT value FROM config WHERE key = ?').get('server_public_key') as { value: string };
      const privRow = database.db.prepare('SELECT value FROM config WHERE key = ?').get('server_private_key') as { value: string };

      expect(pubRow.value).toBe(Buffer.from(crypto.generateKeyPair().publicKey).toString('base64'));
      expect(privRow.value).toBe(Buffer.from(crypto.generateKeyPair().privateKey).toString('base64'));
    });

    test('correctly round-trips base64 encoding', () => {
      const crypto = createMockCrypto();
      const identity = new CliIdentity(database, crypto);

      const original = identity.getKeyPair();
      const restored = identity.getKeyPair();

      expect(Array.from(restored.publicKey)).toEqual(Array.from(original.publicKey));
      expect(Array.from(restored.privateKey)).toEqual(Array.from(original.privateKey));
    });
  });

  describe('independence', () => {
    test('getServerId and getKeyPair are independent', () => {
      const crypto = createMockCrypto();
      const identity = new CliIdentity(database, crypto);

      const serverId = identity.getServerId();
      const keyPair = identity.getKeyPair();

      expect(serverId).toBeDefined();
      expect(keyPair.publicKey).toBeDefined();
      expect(keyPair.privateKey).toBeDefined();
    });
  });
});
