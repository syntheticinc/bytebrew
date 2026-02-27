// Integration test: agentId flow from ToolCallInfo → PermissionRequest → auto-approval
import { describe, it, expect, beforeAll, afterAll } from 'bun:test';
import { ToolExecutorAdapter } from '../ToolExecutorAdapter.js';
import { ToolCallInfo } from '../../../domain/entities/Message.js';
import { PermissionApproval } from '../../permission/PermissionApproval.js';
import path from 'path';
import fs from 'fs/promises';
import os from 'os';

const TEST_PROJECT_ROOT = path.join(os.tmpdir(), 'vector-test-executor-agent-id-' + Date.now());

describe('ToolExecutorAdapter - agentId propagation', () => {
  beforeAll(async () => {
    // Set headless mode to block (so only auto-approval allows execution)
    PermissionApproval.setHeadlessMode('block');

    await fs.mkdir(TEST_PROJECT_ROOT, { recursive: true });

    // Create config with 'ask' for all operations
    const config = {
      permissions: {
        bash: {
          defaultAction: 'ask' as const,
          rules: [],
        },
        read: 'ask' as const,
        edit: 'ask' as const,
        list: 'ask' as const,
      },
      defaultTimeoutSeconds: 30,
      maxTimeoutSeconds: 300,
    };

    await fs.writeFile(
      path.join(TEST_PROJECT_ROOT, 'vector-permissions.json'),
      JSON.stringify(config, null, 2)
    );

    // Create test file for reading
    await fs.writeFile(
      path.join(TEST_PROJECT_ROOT, 'test.txt'),
      'test content'
    );
  });

  afterAll(async () => {
    await fs.rm(TEST_PROJECT_ROOT, { recursive: true, force: true });
    PermissionApproval.reset();
  });

  it('auto-approves read_file for code agents', async () => {
    const adapter = new ToolExecutorAdapter(TEST_PROJECT_ROOT);

    const toolCall: ToolCallInfo = {
      callId: 'test-read-1',
      toolName: 'read_file',
      arguments: {
        file_path: path.join(TEST_PROJECT_ROOT, 'test.txt'),
      },
      agentId: 'code-agent-backend-001',
    };

    const result = await adapter.execute(toolCall);

    // Should be allowed (auto-approved for code agent)
    expect(result.error).toBeUndefined();
    expect(result.result).toContain('test content');
  });

  it('auto-approves edit_file for code agents', async () => {
    const adapter = new ToolExecutorAdapter(TEST_PROJECT_ROOT);

    const toolCall: ToolCallInfo = {
      callId: 'test-edit-1',
      toolName: 'edit_file',
      arguments: {
        file_path: path.join(TEST_PROJECT_ROOT, 'test.txt'),
        old_string: 'test',
        new_string: 'modified',
      },
      agentId: 'code-agent-reviewer-xyz',
    };

    const result = await adapter.execute(toolCall);

    // Should be allowed (no PERMISSION_DENIED error)
    expect(result.error).toBeUndefined();
    expect(result.result).toContain('Edit applied');
  });

  it('auto-approves write_file for code agents', async () => {
    const adapter = new ToolExecutorAdapter(TEST_PROJECT_ROOT);

    const toolCall: ToolCallInfo = {
      callId: 'test-write-1',
      toolName: 'write_file',
      arguments: {
        file_path: path.join(TEST_PROJECT_ROOT, 'output.txt'),
        content: 'written by code agent',
      },
      agentId: 'code-agent-frontend-002',
    };

    const result = await adapter.execute(toolCall);

    // Should be allowed
    expect(result.error).toBeUndefined();
    expect(result.result).toContain('written');

    // Verify file was actually written
    const content = await fs.readFile(
      path.join(TEST_PROJECT_ROOT, 'output.txt'),
      'utf-8'
    );
    expect(content).toBe('written by code agent');
  });

  it('auto-approves get_project_tree for code agents', async () => {
    const adapter = new ToolExecutorAdapter(TEST_PROJECT_ROOT);

    const toolCall: ToolCallInfo = {
      callId: 'test-list-1',
      toolName: 'get_project_tree',
      arguments: {
        root: TEST_PROJECT_ROOT,
      },
      agentId: 'code-agent-analyzer-001',
    };

    const result = await adapter.execute(toolCall);

    // Should be allowed (no PERMISSION_DENIED error)
    expect(result.error).toBeUndefined();
    // Should return tree structure with test file
    expect(result.result).toContain('test.txt');
  });

  it('auto-approves execute_command for code agents', async () => {
    const adapter = new ToolExecutorAdapter(TEST_PROJECT_ROOT);

    const toolCall: ToolCallInfo = {
      callId: 'test-cmd-1',
      toolName: 'execute_command',
      arguments: {
        command: 'echo hello',
      },
      agentId: 'code-agent-reviewer-003',
    };

    const result = await adapter.execute(toolCall);

    // Should be allowed
    expect(result.error).toBeUndefined();
    expect(result.result).toContain('hello');
  });
});
