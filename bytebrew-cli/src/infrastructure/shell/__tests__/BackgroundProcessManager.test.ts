import { describe, it, expect, afterEach } from 'bun:test';
import { BackgroundProcessManager } from '../BackgroundProcessManager.js';

const testCwd = process.platform === 'win32' ? 'C:/Users' : '/tmp';

describe('BackgroundProcessManager', () => {
  let manager: BackgroundProcessManager;

  afterEach(async () => {
    if (manager) {
      await manager.disposeAll();
    }
  });

  it('should spawn a background process and return info', () => {
    manager = new BackgroundProcessManager();
    const info = manager.spawn('echo hello && sleep 10', testCwd);

    expect(info.id).toBe('bg-1');
    expect(info.command).toBe('echo hello && sleep 10');
    expect(info.pid).toBeGreaterThan(0);
    expect(info.status).toBe('running');
  });

  it('should auto-increment IDs', () => {
    manager = new BackgroundProcessManager();
    const p1 = manager.spawn('sleep 10', testCwd);
    const p2 = manager.spawn('sleep 10', testCwd);
    expect(p1.id).toBe('bg-1');
    expect(p2.id).toBe('bg-2');
  });

  it('should list all processes', () => {
    manager = new BackgroundProcessManager();
    manager.spawn('sleep 10', testCwd);
    manager.spawn('sleep 10', testCwd);

    const list = manager.list();
    expect(list).toHaveLength(2);
    expect(list[0].id).toBe('bg-1');
    expect(list[1].id).toBe('bg-2');
  });

  it('should read output from process', async () => {
    manager = new BackgroundProcessManager();
    manager.spawn('echo "test output"', testCwd);

    // Wait for output to be captured
    await new Promise((r) => setTimeout(r, 1000));

    const output = manager.readOutput('bg-1');
    expect(output).not.toBeNull();
    expect(output!).toContain('test output');
  });

  it('should kill a process', async () => {
    manager = new BackgroundProcessManager();
    manager.spawn('sleep 999', testCwd);

    const killed = await manager.kill('bg-1');
    expect(killed).toBe(true);

    // Wait for exit event
    await new Promise((r) => setTimeout(r, 500));

    const info = manager.get('bg-1');
    expect(info?.status).toBe('exited');
  });

  it('should return false for killing non-existent process', async () => {
    manager = new BackgroundProcessManager();
    const killed = await manager.kill('bg-999');
    expect(killed).toBe(false);
  });

  it('should update status when process exits naturally', async () => {
    manager = new BackgroundProcessManager();
    manager.spawn('echo done', testCwd);

    // Wait for process to exit
    await new Promise((r) => setTimeout(r, 1000));

    const info = manager.get('bg-1');
    expect(info?.status).toBe('exited');
    expect(info?.exitCode).toBe(0);
  });

  it('readOutput returns null for non-existent process', () => {
    manager = new BackgroundProcessManager();
    expect(manager.readOutput('bg-999')).toBeNull();
  });

  it('disposeAll should kill all running processes', async () => {
    manager = new BackgroundProcessManager();
    manager.spawn('sleep 999', testCwd);
    manager.spawn('sleep 999', testCwd);

    await manager.disposeAll();

    expect(manager.list()).toHaveLength(0);
  });
});
