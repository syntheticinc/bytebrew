import { Database } from 'bun:sqlite';
import * as fs from 'fs';
import * as path from 'path';
import { ByteBrewHome } from '../config/ByteBrewHome.js';

const DEFAULT_DB_NAME = 'bytebrew.db';

export class ByteBrewDatabase {
  private _db: Database;
  private closed = false;

  constructor(dbPath?: string) {
    const resolvedPath = dbPath ?? path.join(ByteBrewHome.dir(), DEFAULT_DB_NAME);

    if (resolvedPath !== ':memory:') {
      const dir = path.dirname(resolvedPath);
      fs.mkdirSync(dir, { recursive: true });
    }

    this._db = new Database(resolvedPath);
    this._db.exec('PRAGMA journal_mode = WAL');
    this._db.exec('PRAGMA busy_timeout = 5000');

    this.migrate();
  }

  get db(): Database {
    return this._db;
  }

  close(): void {
    if (this.closed) return;
    this.closed = true;
    this._db.close();
  }

  private migrate(): void {
    this._db.exec(`
      CREATE TABLE IF NOT EXISTS config (
        key TEXT PRIMARY KEY,
        value TEXT NOT NULL
      );

      CREATE TABLE IF NOT EXISTS paired_devices (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL,
        device_token TEXT NOT NULL UNIQUE,
        public_key BLOB,
        shared_secret BLOB,
        paired_at TEXT NOT NULL,
        last_seen_at TEXT NOT NULL
      );

      CREATE UNIQUE INDEX IF NOT EXISTS idx_device_token ON paired_devices(device_token);
    `);
  }
}
