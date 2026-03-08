/**
 * CliIdentity manages persistent CLI identity (server_id + X25519 keypair).
 * Stores values in SQLite config table (key-value). Generates once, returns same values thereafter.
 *
 * Consumer-side interfaces defined in this file (ISP).
 */

import { v4 as uuidv4 } from 'uuid';
import type { Database } from 'bun:sqlite';

// --- Consumer-side interfaces ---

interface ConfigDatabase {
  get db(): Database;
}

interface CryptoKeyPairGenerator {
  generateKeyPair(): { publicKey: Uint8Array; privateKey: Uint8Array };
}

// --- Public interface ---

export interface ICliIdentity {
  getServerId(): string;
  getKeyPair(): { publicKey: Uint8Array; privateKey: Uint8Array };
}

// --- Implementation ---

export class CliIdentity implements ICliIdentity {
  private readonly database: ConfigDatabase;
  private readonly crypto: CryptoKeyPairGenerator;

  constructor(database: ConfigDatabase, crypto: CryptoKeyPairGenerator) {
    this.database = database;
    this.crypto = crypto;
  }

  getServerId(): string {
    const existing = this.getConfig('server_id');
    if (existing) {
      return existing;
    }

    const serverId = uuidv4();
    this.setConfig('server_id', serverId);
    return serverId;
  }

  getKeyPair(): { publicKey: Uint8Array; privateKey: Uint8Array } {
    const publicKeyB64 = this.getConfig('server_public_key');
    const privateKeyB64 = this.getConfig('server_private_key');

    if (publicKeyB64 && privateKeyB64) {
      return {
        publicKey: new Uint8Array(Buffer.from(publicKeyB64, 'base64')),
        privateKey: new Uint8Array(Buffer.from(privateKeyB64, 'base64')),
      };
    }

    const keyPair = this.crypto.generateKeyPair();

    const pubB64 = Buffer.from(keyPair.publicKey).toString('base64');
    const privB64 = Buffer.from(keyPair.privateKey).toString('base64');

    this.setConfig('server_public_key', pubB64);
    this.setConfig('server_private_key', privB64);

    return keyPair;
  }

  private getConfig(key: string): string | null {
    const stmt = this.database.db.prepare('SELECT value FROM config WHERE key = ?');
    const row = stmt.get(key) as { value: string } | null;
    return row?.value ?? null;
  }

  private setConfig(key: string, value: string): void {
    const stmt = this.database.db.prepare(
      'INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)',
    );
    stmt.run(key, value);
  }
}
