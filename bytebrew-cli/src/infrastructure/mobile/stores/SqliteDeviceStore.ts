/**
 * SQLite-backed device store for paired mobile devices.
 *
 * Persists devices across CLI restarts using bun:sqlite.
 * Uses prepared statements for all operations.
 */

import type { Statement } from 'bun:sqlite';
import type { Database } from 'bun:sqlite';
import type { IDeviceStore } from './InMemoryDeviceStore.js';
import { MobileDevice } from '../../../domain/entities/MobileDevice.js';

interface DeviceDatabase {
  get db(): Database;
}

interface DeviceRow {
  id: string;
  name: string;
  device_token: string;
  public_key: Uint8Array | null;
  shared_secret: Uint8Array | null;
  paired_at: string;
  last_seen_at: string;
}

export class SqliteDeviceStore implements IDeviceStore {
  private readonly stmtInsert: Statement;
  private readonly stmtGetById: Statement;
  private readonly stmtGetByToken: Statement;
  private readonly stmtDelete: Statement;
  private readonly stmtList: Statement;
  private readonly stmtUpdateLastSeen: Statement;
  private readonly stmtChanges: Statement;

  constructor(database: DeviceDatabase) {
    const db = database.db;

    this.stmtInsert = db.prepare(`
      INSERT OR REPLACE INTO paired_devices (id, name, device_token, public_key, shared_secret, paired_at, last_seen_at)
      VALUES (?, ?, ?, ?, ?, ?, ?)
    `);

    this.stmtGetById = db.prepare(`
      SELECT id, name, device_token, public_key, shared_secret, paired_at, last_seen_at
      FROM paired_devices WHERE id = ?
    `);

    this.stmtGetByToken = db.prepare(`
      SELECT id, name, device_token, public_key, shared_secret, paired_at, last_seen_at
      FROM paired_devices WHERE device_token = ?
    `);

    this.stmtDelete = db.prepare(`
      DELETE FROM paired_devices WHERE id = ?
    `);

    this.stmtList = db.prepare(`
      SELECT id, name, device_token, public_key, shared_secret, paired_at, last_seen_at
      FROM paired_devices
    `);

    this.stmtUpdateLastSeen = db.prepare(`
      UPDATE paired_devices SET last_seen_at = ? WHERE id = ?
    `);

    this.stmtChanges = db.prepare('SELECT changes() as c');
  }

  add(device: MobileDevice): void {
    this.stmtInsert.run(
      device.id,
      device.name,
      device.deviceToken,
      device.publicKey.length > 0 ? device.publicKey : null,
      device.sharedSecret.length > 0 ? device.sharedSecret : null,
      device.pairedAt.toISOString(),
      device.lastSeenAt.toISOString(),
    );
  }

  getById(id: string): MobileDevice | undefined {
    const row = this.stmtGetById.get(id) as DeviceRow | null;
    if (!row) {
      return undefined;
    }
    return this.rowToDevice(row);
  }

  getByToken(deviceToken: string): MobileDevice | undefined {
    const row = this.stmtGetByToken.get(deviceToken) as DeviceRow | null;
    if (!row) {
      return undefined;
    }
    return this.rowToDevice(row);
  }

  remove(id: string): boolean {
    this.stmtDelete.run(id);
    const result = this.stmtChanges.get() as { c: number };
    return result.c > 0;
  }

  list(): MobileDevice[] {
    const rows = this.stmtList.all() as DeviceRow[];
    return rows.map((row) => this.rowToDevice(row));
  }

  updateLastSeen(id: string): void {
    this.stmtUpdateLastSeen.run(new Date().toISOString(), id);
  }

  private rowToDevice(row: DeviceRow): MobileDevice {
    return MobileDevice.fromProps({
      id: row.id,
      name: row.name,
      deviceToken: row.device_token,
      publicKey: row.public_key ? new Uint8Array(row.public_key) : new Uint8Array(0),
      sharedSecret: row.shared_secret ? new Uint8Array(row.shared_secret) : new Uint8Array(0),
      pairedAt: new Date(row.paired_at),
      lastSeenAt: new Date(row.last_seen_at),
    });
  }
}
