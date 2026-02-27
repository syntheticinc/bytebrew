// Test agentId propagation and auto-approval for code agents
import { describe, it, expect, beforeAll, afterAll } from 'bun:test';
import { PermissionService } from '../PermissionService.js';
import { PermissionRequest } from '../../../domain/permission/Permission.js';
import { PermissionApproval } from '../../../infrastructure/permission/PermissionApproval.js';
import path from 'path';
import fs from 'fs/promises';

const TEST_PROJECT_ROOT = path.join(import.meta.dir, '__test_data__', 'agent-id-test');

describe('PermissionService - agentId auto-approval', () => {
  beforeAll(async () => {
    // Set headless mode to block by default (so we can test auto-approval)
    PermissionApproval.setHeadlessMode('block');
    await fs.mkdir(path.join(TEST_PROJECT_ROOT, '.bytebrew'), { recursive: true });

    // Create config with empty allow/deny (so everything asks by default)
    const config = {
      permissions: {
        allow: [],
        deny: [],
      },
    };

    await fs.writeFile(
      path.join(TEST_PROJECT_ROOT, '.bytebrew', 'settings.local.json'),
      JSON.stringify(config, null, 2)
    );
  });

  afterAll(async () => {
    await fs.rm(TEST_PROJECT_ROOT, { recursive: true, force: true });
    PermissionApproval.reset();
  });

  it('auto-approves bash commands for code agents', async () => {
    const service = new PermissionService(TEST_PROJECT_ROOT);

    const request: PermissionRequest = {
      type: 'bash',
      value: 'ls -la',
      agentId: 'code-agent-backend-001',
    };

    const result = await service.check(request);

    expect(result.allowed).toBe(true);
    // Should not show 'reason' because it was auto-approved, not denied
    expect((result as any).reason).toBeUndefined();
  });

  it('auto-approves read operations for code agents', async () => {
    const service = new PermissionService(TEST_PROJECT_ROOT);

    const request: PermissionRequest = {
      type: 'read',
      value: '/path/to/file.ts',
      agentId: 'code-agent-frontend-002',
    };

    const result = await service.check(request);

    expect(result.allowed).toBe(true);
  });

  it('auto-approves edit operations for code agents', async () => {
    const service = new PermissionService(TEST_PROJECT_ROOT);

    const request: PermissionRequest = {
      type: 'edit',
      value: '/path/to/file.go',
      agentId: 'code-agent-reviewer-003',
    };

    const result = await service.check(request);

    expect(result.allowed).toBe(true);
  });

  it('does NOT auto-approve deny actions even for code agents', async () => {
    // First, create config with deny rule
    const denyConfig = {
      permissions: {
        allow: [],
        deny: ['Bash(rm -rf *)'],
      },
    };

    const testRoot = path.join(import.meta.dir, '__test_data__', 'deny-test');
    await fs.mkdir(path.join(testRoot, '.bytebrew'), { recursive: true });
    await fs.writeFile(
      path.join(testRoot, '.bytebrew', 'settings.local.json'),
      JSON.stringify(denyConfig, null, 2)
    );

    const service = new PermissionService(testRoot);

    const request: PermissionRequest = {
      type: 'bash',
      value: 'rm -rf /',
      agentId: 'code-agent-backend-001',
    };

    const result = await service.check(request);

    // Even with code agent ID, deny rules should still block
    expect(result.allowed).toBe(false);
    expect((result as any).reason).toContain('deny');

    await fs.rm(testRoot, { recursive: true, force: true });
  });
});
