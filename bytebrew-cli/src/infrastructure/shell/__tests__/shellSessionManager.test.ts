import { describe, it, expect, beforeEach, afterEach } from 'bun:test';
import { ShellSessionManager } from '../ShellSessionManager.js';

// Use existing directories that work across platforms
const testDir1 = process.platform === 'win32' ? 'C:/Users' : '/tmp';
const testDir2 = process.platform === 'win32' ? 'C:/Windows' : '/var';

describe('ShellSessionManager', () => {
  let manager: ShellSessionManager;

  beforeEach(() => {
    manager = new ShellSessionManager();
  });

  afterEach(async () => {
    await manager.disposeAll();
  });

  describe('getAvailableSession', () => {
    it('should create session lazily on first call', () => {
      const session = manager.getAvailableSession(testDir1, 'agent-1');
      expect(session).not.toBeNull();
    });

    it('should return same session when not busy', async () => {
      const s1 = manager.getAvailableSession(testDir1, 'agent-1');
      expect(s1).not.toBeNull();
      await s1!.execute('echo test', 5000);

      const s2 = manager.getAvailableSession(testDir1, 'agent-1');
      expect(s2).toBe(s1); // Same instance reused
    });

    it('should create separate pools for different agents', () => {
      const s1 = manager.getAvailableSession(testDir1, 'agent-1');
      const s2 = manager.getAvailableSession(testDir1, 'agent-2');
      expect(s1).not.toBe(s2); // Different pools
    });

    it('should create separate pools for different project roots', () => {
      const s1 = manager.getAvailableSession(testDir1, 'agent-1');
      const s2 = manager.getAvailableSession(testDir2, 'agent-1');
      expect(s1).not.toBe(s2);
    });

    it('should work without agentId (key = projectRoot)', async () => {
      const s1 = manager.getAvailableSession(testDir1);
      expect(s1).not.toBeNull();
      await s1!.execute('echo no-agent', 5000);

      const s2 = manager.getAvailableSession(testDir1);
      expect(s2).toBe(s1); // Reused
    });

    it('should create new session when existing is busy', async () => {
      const s1 = manager.getAvailableSession(testDir1, 'agent-1');
      expect(s1).not.toBeNull();

      // Start long command on session 1 (don't await — need it busy).
      // cancelPending in destroy() silently drops the promise, so we just fire-and-forget.
      void s1!.execute('sleep 30', 60000);

      // Ask for available — should get a new session (s1 is busy)
      const s2 = manager.getAvailableSession(testDir1, 'agent-1');
      expect(s2).not.toBeNull();
      expect(s2).not.toBe(s1);

      // Cleanup: disposeAll kills processes and cancels pending markers
    });

    it('should return null when all 3 sessions are busy', async () => {
      // Fill all 3 pool slots with busy sessions
      for (let i = 0; i < 3; i++) {
        const s = manager.getAvailableSession(testDir1, 'agent-1');
        expect(s).not.toBeNull();
        void s!.execute('sleep 30', 60000);
      }

      // 4th request — all busy, should return null
      const s4 = manager.getAvailableSession(testDir1, 'agent-1');
      expect(s4).toBeNull();

      // Cleanup: afterEach calls disposeAll
    });
  });

  describe('getBackgroundManager', () => {
    it('should return the same BackgroundProcessManager instance', () => {
      const bgManager1 = manager.getBackgroundManager();
      const bgManager2 = manager.getBackgroundManager();
      expect(bgManager1).toBe(bgManager2);
    });

    it('should spawn background processes', async () => {
      const bgManager = manager.getBackgroundManager();
      const proc = bgManager.spawn('echo "background test"', testDir1);

      expect(proc.id).toMatch(/^bg-\d+$/);
      expect(proc.pid).toBeGreaterThan(0);
      expect(proc.status).toBe('running');

      // Wait for process to complete
      await new Promise(r => setTimeout(r, 100));

      const output = bgManager.readOutput(proc.id);
      expect(output).toContain('background test');
    });
  });

  describe('disposeAll', () => {
    it('should destroy all sessions in pools', async () => {
      const s1 = manager.getAvailableSession(testDir1, 'agent-1');
      const s2 = manager.getAvailableSession(testDir2, 'agent-2');

      await s1!.execute('echo "test1"', 5000);
      await s2!.execute('echo "test2"', 5000);

      expect(s1!.isAlive()).toBe(true);
      expect(s2!.isAlive()).toBe(true);

      await manager.disposeAll();

      expect(s1!.isAlive()).toBe(false);
      expect(s2!.isAlive()).toBe(false);
    });

    it('should kill all background processes', async () => {
      const bgManager = manager.getBackgroundManager();
      bgManager.spawn('sleep 10', testDir1);
      bgManager.spawn('sleep 10', testDir1);

      const list = bgManager.list();
      expect(list.length).toBe(2);

      await manager.disposeAll();

      const listAfter = bgManager.list();
      expect(listAfter.length).toBe(0);
    });

    it('should allow re-use after dispose', async () => {
      const s1 = manager.getAvailableSession(testDir1, 'agent-1');
      await s1!.execute('echo "before"', 5000);

      await manager.disposeAll();

      // Get a new session for the same pool key
      const s2 = manager.getAvailableSession(testDir1, 'agent-1');
      expect(s2).not.toBeNull();
      const result = await s2!.execute('echo "after"', 5000);

      expect(result.completed).toBe(true);
      expect(result.stdout).toContain('after');
    });
  });
});
