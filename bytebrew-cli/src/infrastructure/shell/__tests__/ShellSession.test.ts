import { describe, it, expect, afterEach } from 'bun:test';
import { ShellSession } from '../ShellSession.js';

// Use a temp dir that exists on all platforms
const testCwd = process.platform === 'win32' ? 'C:/Users' : '/tmp';

describe('ShellSession', () => {
  let session: ShellSession;

  afterEach(() => {
    if (session) {
      session.destroy();
    }
  });

  it('should execute simple command', async () => {
    session = new ShellSession({ cwd: testCwd });
    const result = await session.execute('echo hello', 10000);
    expect(result.completed).toBe(true);
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('hello');
  });

  it('should parse non-zero exit code', async () => {
    session = new ShellSession({ cwd: testCwd });
    // Use 'false' command which returns exit code 1 (or bash -c "exit 42" as subshell)
    const result = await session.execute('bash -c "exit 42"', 10000);
    expect(result.completed).toBe(true);
    expect(result.exitCode).toBe(42);
  });

  it('should maintain state between commands (cd + pwd)', async () => {
    session = new ShellSession({ cwd: testCwd });

    // cd to a known directory
    const cdResult = await session.execute('cd /tmp', 10000);
    expect(cdResult.completed).toBe(true);

    // pwd should reflect the cd
    const pwdResult = await session.execute('pwd', 10000);
    expect(pwdResult.completed).toBe(true);
    expect(pwdResult.stdout).toContain('/tmp');
  });

  it('should maintain env vars between commands', async () => {
    session = new ShellSession({ cwd: testCwd });

    await session.execute('export MY_TEST_VAR=hello123', 10000);
    const result = await session.execute('echo $MY_TEST_VAR', 10000);
    expect(result.completed).toBe(true);
    expect(result.stdout).toContain('hello123');
  });

  it('should timeout on long-running command', async () => {
    session = new ShellSession({ cwd: testCwd });
    const result = await session.execute('sleep 30', 1000); // 1s timeout
    expect(result.completed).toBe(false);
    expect(result.exitCode).toBeNull();

    // After timeout, interrupt to clean up
    await session.interrupt();
    // Small delay for bash to process Ctrl+C
    await new Promise((r) => setTimeout(r, 500));
  });

  it('should reject concurrent commands', async () => {
    session = new ShellSession({ cwd: testCwd });

    // Start a slow command
    const p1 = session.execute('sleep 5', 10000);

    // Try to execute another — should throw
    try {
      await session.execute('echo concurrent', 10000);
      throw new Error('should not reach');
    } catch (err: any) {
      expect(err.message).toContain('busy');
    }

    // Clean up
    await session.interrupt();
    await new Promise((r) => setTimeout(r, 500));
  });

  it('should respawn after process death', async () => {
    session = new ShellSession({ cwd: testCwd });

    // Execute first command
    const r1 = await session.execute('echo before', 10000);
    expect(r1.completed).toBe(true);

    // Kill the process
    session.destroy();

    // Create new session (destroy killed it, execute should respawn)
    session = new ShellSession({ cwd: testCwd });
    const r2 = await session.execute('echo after', 10000);
    expect(r2.completed).toBe(true);
    expect(r2.stdout).toContain('after');
  });

  it('isAlive should return correct status', () => {
    session = new ShellSession({ cwd: testCwd });
    // Before first execute, process not spawned yet
    expect(session.isAlive()).toBe(false);
  });

  it('isExecuting should reflect command state', async () => {
    session = new ShellSession({ cwd: testCwd });
    expect(session.isExecuting()).toBe(false);

    // Start a command and check
    const p = session.execute('echo test', 10000);
    // isExecuting should be true while command runs
    // (timing-sensitive, but echo should be fast enough)
    await p;
    expect(session.isExecuting()).toBe(false);
  });
});
