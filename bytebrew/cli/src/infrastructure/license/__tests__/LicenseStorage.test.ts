import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import { LicenseStorage } from '../LicenseStorage';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';

describe('LicenseStorage', () => {
  let tempDir: string;
  let filePath: string;
  let storage: LicenseStorage;

  const sampleJwt =
    'eyJhbGciOiJFZERTQSJ9.eyJ0aWVyIjoicGVyc29uYWwiLCJleHAiOjk5OTk5OTk5OTl9.signature';

  beforeEach(() => {
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'licensestorage-test-'));
    filePath = path.join(tempDir, 'license.jwt');
    storage = new LicenseStorage(filePath);
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
    storage.save(sampleJwt);
    const loaded = storage.load();

    expect(loaded).toBe(sampleJwt);
  });

  test('save() writes raw string (not JSON)', () => {
    storage.save(sampleJwt);

    const raw = fs.readFileSync(filePath, 'utf-8');
    expect(raw).toBe(sampleJwt);
  });

  test('save() creates parent directory if missing', () => {
    const nestedPath = path.join(tempDir, 'nested', 'dir', 'license.jwt');
    const nestedStorage = new LicenseStorage(nestedPath);

    expect(() => nestedStorage.save(sampleJwt)).not.toThrow();
    expect(fs.existsSync(nestedPath)).toBe(true);
  });

  test('save() overwrites previous JWT', () => {
    const newJwt = 'new.jwt.token';
    storage.save(sampleJwt);
    storage.save(newJwt);

    expect(storage.load()).toBe(newJwt);
  });

  test('load() trims whitespace', () => {
    fs.writeFileSync(filePath, `  ${sampleJwt}  \n`, 'utf-8');
    expect(storage.load()).toBe(sampleJwt);
  });

  test('load() returns null for empty file', () => {
    fs.writeFileSync(filePath, '', 'utf-8');
    expect(storage.load()).toBeNull();
  });

  test('load() returns null for whitespace-only file', () => {
    fs.writeFileSync(filePath, '   \n  ', 'utf-8');
    expect(storage.load()).toBeNull();
  });

  test('clear() removes file', () => {
    storage.save(sampleJwt);
    expect(fs.existsSync(filePath)).toBe(true);

    storage.clear();
    expect(fs.existsSync(filePath)).toBe(false);
  });

  test('clear() does not throw when file does not exist', () => {
    expect(() => storage.clear()).not.toThrow();
  });

  test('load() returns null after clear()', () => {
    storage.save(sampleJwt);
    storage.clear();
    expect(storage.load()).toBeNull();
  });
});
