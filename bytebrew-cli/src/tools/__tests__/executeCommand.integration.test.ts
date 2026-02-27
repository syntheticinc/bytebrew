import { describe, it, expect, beforeEach, afterEach } from 'bun:test';
import { ExecuteCommandTool } from '../executeCommand.js';
import { ShellSessionManager } from '../../infrastructure/shell/ShellSessionManager.js';

const testCwd = process.platform === 'win32' ? 'C:/Users' : '/tmp';

describe('ExecuteCommandTool integration', () => {
  let tool: ExecuteCommandTool;
  let manager: ShellSessionManager;

  beforeEach(() => {
    manager = new ShellSessionManager();
    tool = new ExecuteCommandTool(testCwd, manager);
  });

  afterEach(async () => {
    await manager.disposeAll();
  });

  describe('foreground execution', () => {
    it('should execute command and return output', async () => {
      const result = await tool.execute({ command: 'echo "test output"' });

      expect(result.error).toBeUndefined();
      expect(result.result).toContain('test output');
      expect(result.summary).toBe('exit 0');
    });

    it('should handle non-zero exit code', async () => {
      const result = await tool.execute({ command: 'bash -c "exit 42"' });

      expect(result.error).toBeUndefined();
      expect(result.result).toContain('[Command exited with code 42]');
      expect(result.summary).toBe('exit 42');
    });

    it('should timeout and interrupt long-running command', async () => {
      const result = await tool.execute({
        command: 'sleep 10',
        timeout: '1', // 1 second
      });

      expect(result.summary).toBe('timed out');
      expect(result.result).toContain('[Command timed out after 1s — interrupted]');
      expect(result.result).toContain('[Use background=true');
    });

    it('should reject concurrent foreground commands', async () => {
      // Start a long-running command
      const promise1 = tool.execute({ command: 'sleep 2' });

      // Try to execute another while first is running
      const result2 = await tool.execute({ command: 'echo "should fail"' });

      expect(result2.error).toBeDefined();
      expect(result2.error?.message).toContain('Shell session busy');
      expect(result2.result).toContain('[ERROR]');
      expect(result2.result).toContain('Shell session is busy');

      // Wait for first command to complete
      await promise1;
    });

    it('should maintain state between foreground commands (cd + pwd)', async () => {
      // cd to a known directory
      await tool.execute({ command: 'cd /tmp' });

      // pwd should reflect the cd
      const pwdResult = await tool.execute({ command: 'pwd' });
      expect(pwdResult.result).toContain('/tmp');
    });
  });

  describe('background execution', () => {
    it('should spawn background process and return immediately', async () => {
      const result = await tool.execute({
        command: 'sleep 5 && echo "done"',
        background: 'true',
      });

      expect(result.error).toBeUndefined();
      expect(result.result).toContain('Started background process');
      expect(result.result).toMatch(/bg-\d+/);
      expect(result.summary).toMatch(/started bg-\d+/);
    });

    it('should allow reading background process output', async () => {
      // Start background process
      const startResult = await tool.execute({
        command: 'echo "background output"',
        background: 'true',
      });

      const bgIdMatch = startResult.result.match(/bg-(\d+)/);
      expect(bgIdMatch).not.toBeNull();
      const bgId = `bg-${bgIdMatch![1]}`;

      // Wait for process to complete
      await new Promise(r => setTimeout(r, 200));

      // Read output
      const readResult = await tool.execute({
        bg_action: 'read',
        bg_id: bgId,
      });

      expect(readResult.error).toBeUndefined();
      expect(readResult.result).toContain('background output');
    });

    it('should list all background processes', async () => {
      // Start 2 background processes
      await tool.execute({ command: 'sleep 2', background: 'true' });
      await tool.execute({ command: 'sleep 2', background: 'true' });

      const listResult = await tool.execute({ bg_action: 'list' });

      expect(listResult.error).toBeUndefined();
      expect(listResult.result).toContain('Background processes:');
      expect(listResult.result).toContain('bg-');
      expect(listResult.summary).toContain('2 processes');
    });

    it('should kill background process', async () => {
      // Start long-running background process
      const startResult = await tool.execute({
        command: 'sleep 30',
        background: 'true',
      });

      const bgIdMatch = startResult.result.match(/bg-(\d+)/);
      const bgId = `bg-${bgIdMatch![1]}`;

      // Kill it
      const killResult = await tool.execute({
        bg_action: 'kill',
        bg_id: bgId,
      });

      expect(killResult.error).toBeUndefined();
      expect(killResult.result).toContain('killed');
      expect(killResult.summary).toContain('killed');

      // Wait for cleanup
      await new Promise(r => setTimeout(r, 100));

      // Check it's no longer running
      const readResult = await tool.execute({
        bg_action: 'read',
        bg_id: bgId,
      });

      // Should show exited status
      expect(readResult.result).toContain('[Process exited');
    });
  });

  describe('bg_action validation', () => {
    it('should require bg_id for read action', async () => {
      const result = await tool.execute({ bg_action: 'read' });

      expect(result.result).toContain('bg_id is required');
      expect(result.result).toContain('bg_action="list"');
    });

    it('should return error for unknown bg_action', async () => {
      const result = await tool.execute({ bg_action: 'invalid' });

      expect(result.error).toBeDefined();
      expect(result.error?.message).toContain('Unknown bg_action');
      expect(result.result).toContain('[ERROR]');
      expect(result.result).toContain('Valid actions: list, read, kill');
    });

    it('should return informative message for non-existent bg_id', async () => {
      const result = await tool.execute({
        bg_action: 'read',
        bg_id: 'bg-999',
      });

      expect(result.error).toBeUndefined();
      expect(result.result).toContain('not found');
      expect(result.result).toContain('bg_action="list"');
    });
  });

  describe('legacy mode (without ShellSessionManager)', () => {
    it('should fallback to execa for foreground commands', async () => {
      const toolWithoutManager = new ExecuteCommandTool(testCwd);

      const result = await toolWithoutManager.execute({
        command: 'echo "legacy mode"',
      });

      expect(result.error).toBeUndefined();
      expect(result.result).toContain('legacy mode');
    });

    it('should return error for background execution without manager', async () => {
      const toolWithoutManager = new ExecuteCommandTool(testCwd);

      const result = await toolWithoutManager.execute({
        command: 'sleep 1',
        background: 'true',
      });

      expect(result.error).toBeDefined();
      expect(result.error?.message).toContain('ShellSessionManager not available');
      expect(result.result).toContain('[ERROR]');
    });

    it('should return error for bg_action without manager', async () => {
      const toolWithoutManager = new ExecuteCommandTool(testCwd);

      const result = await toolWithoutManager.execute({ bg_action: 'list' });

      expect(result.error).toBeDefined();
      expect(result.error?.message).toContain('ShellSessionManager not available');
      expect(result.result).toContain('[ERROR]');
    });
  });
});
