import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import { AuthStorage, type AuthTokens } from '../AuthStorage';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';

describe('AuthStorage', () => {
  let tempDir: string;
  let filePath: string;
  let storage: AuthStorage;

  const sampleTokens: AuthTokens = {
    accessToken: 'access-token-abc123',
    refreshToken: 'refresh-token-xyz789',
    email: 'user@example.com',
    userId: 'user-id-42',
  };

  beforeEach(() => {
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'authstorage-test-'));
    filePath = path.join(tempDir, 'auth.json');
    storage = new AuthStorage(filePath);
  });

  afterEach(() => {
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true });
    }
  });

  test('load() returns null when file does not exist', () => {
    const result = storage.load();
    expect(result).toBeNull();
  });

  test('save() and load() roundtrip', () => {
    storage.save(sampleTokens);
    const loaded = storage.load();

    expect(loaded).not.toBeNull();
    expect(loaded!.accessToken).toBe(sampleTokens.accessToken);
    expect(loaded!.refreshToken).toBe(sampleTokens.refreshToken);
    expect(loaded!.email).toBe(sampleTokens.email);
    expect(loaded!.userId).toBe(sampleTokens.userId);
  });

  test('save() writes snake_case keys in JSON', () => {
    storage.save(sampleTokens);

    const raw = fs.readFileSync(filePath, 'utf-8');
    const parsed = JSON.parse(raw);

    expect(parsed.access_token).toBe(sampleTokens.accessToken);
    expect(parsed.refresh_token).toBe(sampleTokens.refreshToken);
    expect(parsed.email).toBe(sampleTokens.email);
    expect(parsed.user_id).toBe(sampleTokens.userId);

    // camelCase keys should NOT be present
    expect(parsed.accessToken).toBeUndefined();
    expect(parsed.refreshToken).toBeUndefined();
    expect(parsed.userId).toBeUndefined();
  });

  test('save() creates parent directory if missing', () => {
    const nestedPath = path.join(tempDir, 'nested', 'dir', 'auth.json');
    const nestedStorage = new AuthStorage(nestedPath);

    expect(() => nestedStorage.save(sampleTokens)).not.toThrow();
    expect(fs.existsSync(nestedPath)).toBe(true);
  });

  test('save() overwrites previous tokens', () => {
    const newTokens: AuthTokens = {
      accessToken: 'new-access',
      refreshToken: 'new-refresh',
      email: 'new@example.com',
      userId: 'new-user-id',
    };

    storage.save(sampleTokens);
    storage.save(newTokens);
    const loaded = storage.load();

    expect(loaded!.accessToken).toBe(newTokens.accessToken);
    expect(loaded!.email).toBe(newTokens.email);
  });

  test('load() returns null for malformed JSON', () => {
    fs.writeFileSync(filePath, 'not-json{{{', 'utf-8');
    expect(storage.load()).toBeNull();
  });

  test('load() returns null for invalid shape', () => {
    fs.writeFileSync(filePath, JSON.stringify({ foo: 'bar' }), 'utf-8');
    expect(storage.load()).toBeNull();
  });

  test('load() returns null for missing fields', () => {
    fs.writeFileSync(
      filePath,
      JSON.stringify({ access_token: 'tok', refresh_token: 'ref' }),
      'utf-8',
    );
    expect(storage.load()).toBeNull();
  });

  test('clear() removes file', () => {
    storage.save(sampleTokens);
    expect(fs.existsSync(filePath)).toBe(true);

    storage.clear();
    expect(fs.existsSync(filePath)).toBe(false);
  });

  test('clear() does not throw when file does not exist', () => {
    expect(() => storage.clear()).not.toThrow();
  });

  test('load() returns null after clear()', () => {
    storage.save(sampleTokens);
    storage.clear();
    expect(storage.load()).toBeNull();
  });
});
