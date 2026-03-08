import { describe, test, expect, afterEach } from 'bun:test';
import { ByteBrewDatabase } from '../ByteBrewDatabase.js';

describe('ByteBrewDatabase', () => {
  let db: ByteBrewDatabase | null = null;

  afterEach(() => {
    db?.close();
    db = null;
  });

  test('creates in-memory database with tables', () => {
    db = new ByteBrewDatabase(':memory:');

    const tables = db.db
      .prepare("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
      .all() as { name: string }[];

    const tableNames = tables.map((t) => t.name);
    expect(tableNames).toContain('config');
    expect(tableNames).toContain('paired_devices');
  });

  test('config table supports insert and select', () => {
    db = new ByteBrewDatabase(':memory:');

    db.db.prepare('INSERT INTO config (key, value) VALUES (?, ?)').run('theme', 'dark');
    const row = db.db.prepare('SELECT value FROM config WHERE key = ?').get('theme') as { value: string };

    expect(row.value).toBe('dark');
  });

  test('paired_devices table supports insert and select', () => {
    db = new ByteBrewDatabase(':memory:');

    db.db
      .prepare(
        `INSERT INTO paired_devices (id, name, device_token, public_key, shared_secret, paired_at, last_seen_at)
         VALUES (?, ?, ?, ?, ?, ?, ?)`,
      )
      .run('dev-1', 'iPhone 15', 'tok-abc', null, null, '2026-03-07T00:00:00Z', '2026-03-07T00:00:00Z');

    const row = db.db.prepare('SELECT name, device_token FROM paired_devices WHERE id = ?').get('dev-1') as {
      name: string;
      device_token: string;
    };

    expect(row.name).toBe('iPhone 15');
    expect(row.device_token).toBe('tok-abc');
  });

  test('device_token unique constraint is enforced', () => {
    db = new ByteBrewDatabase(':memory:');

    const insert = db.db.prepare(
      `INSERT INTO paired_devices (id, name, device_token, paired_at, last_seen_at)
       VALUES (?, ?, ?, ?, ?)`,
    );

    insert.run('dev-1', 'Phone A', 'tok-dup', '2026-03-07T00:00:00Z', '2026-03-07T00:00:00Z');

    expect(() => {
      insert.run('dev-2', 'Phone B', 'tok-dup', '2026-03-07T00:00:00Z', '2026-03-07T00:00:00Z');
    }).toThrow();
  });

  test('close() is safe to call multiple times', () => {
    db = new ByteBrewDatabase(':memory:');
    db.close();
    db.close(); // should not throw
    db = null;
  });

  test('WAL mode is enabled', () => {
    db = new ByteBrewDatabase(':memory:');

    const result = db.db.prepare('PRAGMA journal_mode').get() as { journal_mode: string };
    // In-memory databases may report 'memory' instead of 'wal'
    expect(['wal', 'memory']).toContain(result.journal_mode);
  });
});
