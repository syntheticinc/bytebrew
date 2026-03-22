import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import { SessionStore } from './SessionStore';
import * as fs from 'fs';
import * as path from 'path';

const testRoot = path.join(process.cwd(), 'test-output', 'session-store-test');

describe('SessionStore', () => {
  beforeEach(() => {
    // Clean up test directory
    if (fs.existsSync(testRoot)) {
      fs.rmSync(testRoot, { recursive: true });
    }
  });

  afterEach(() => {
    // Clean up after tests
    if (fs.existsSync(testRoot)) {
      fs.rmSync(testRoot, { recursive: true });
    }
  });

  test('getLastSessionId returns null when file does not exist', () => {
    const store = new SessionStore(testRoot);
    expect(store.getLastSessionId()).toBeNull();
  });

  test('saveSessionId creates directory and saves ID', () => {
    const store = new SessionStore(testRoot);
    const sessionId = 'test-session-123';

    store.saveSessionId(sessionId);

    const filePath = path.join(testRoot, '.bytebrew', 'last_session');
    expect(fs.existsSync(filePath)).toBe(true);

    const content = fs.readFileSync(filePath, 'utf-8');
    expect(content).toBe(sessionId);
  });

  test('getLastSessionId returns saved session ID', () => {
    const store = new SessionStore(testRoot);
    const sessionId = 'test-session-456';

    store.saveSessionId(sessionId);
    const retrieved = store.getLastSessionId();

    expect(retrieved).toBe(sessionId);
  });

  test('saveSessionId overwrites previous session ID', () => {
    const store = new SessionStore(testRoot);
    const firstId = 'first-session';
    const secondId = 'second-session';

    store.saveSessionId(firstId);
    store.saveSessionId(secondId);

    const retrieved = store.getLastSessionId();
    expect(retrieved).toBe(secondId);
  });

  test('getLastSessionId trims whitespace', () => {
    const store = new SessionStore(testRoot);
    const sessionId = 'test-session-789';

    // Manually write with extra whitespace
    const filePath = path.join(testRoot, '.bytebrew', 'last_session');
    fs.mkdirSync(path.dirname(filePath), { recursive: true });
    fs.writeFileSync(filePath, `  ${sessionId}  \n`, 'utf-8');

    const retrieved = store.getLastSessionId();
    expect(retrieved).toBe(sessionId);
  });

  test('multiple SessionStore instances share same file', () => {
    const store1 = new SessionStore(testRoot);
    const store2 = new SessionStore(testRoot);
    const sessionId = 'shared-session';

    store1.saveSessionId(sessionId);
    const retrieved = store2.getLastSessionId();

    expect(retrieved).toBe(sessionId);
  });
});
