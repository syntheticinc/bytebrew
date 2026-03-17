import { describe, it, expect, beforeEach } from 'bun:test';
import { PermissionApproval } from '../PermissionApproval.js';

beforeEach(() => {
  PermissionApproval.reset();
});

describe('PermissionApproval', () => {
  describe('headless mode', () => {
    it('should default to block behavior', async () => {
      const result = await PermissionApproval.requestApproval({
        type: 'bash',
        value: 'some command',
      });
      expect(result.approved).toBe(false);
      expect(result.remember).toBe(false);
    });

    it('should allow-once when configured', async () => {
      PermissionApproval.setHeadlessMode('allow-once');

      const result = await PermissionApproval.requestApproval({
        type: 'bash',
        value: 'some command',
      });
      expect(result.approved).toBe(true);
      expect(result.remember).toBe(false);
    });

    it('should allow-remember when configured', async () => {
      PermissionApproval.setHeadlessMode('allow-remember');

      const result = await PermissionApproval.requestApproval({
        type: 'bash',
        value: 'some command',
      });
      expect(result.approved).toBe(true);
      expect(result.remember).toBe(true);
    });

    it('should block when configured', async () => {
      PermissionApproval.setHeadlessMode('block');

      const result = await PermissionApproval.requestApproval({
        type: 'edit',
        value: '/some/file.ts',
      });
      expect(result.approved).toBe(false);
    });
  });

  describe('interactive mode', () => {
    it('should call approval callback', async () => {
      let receivedRequest: any = null;

      PermissionApproval.setInteractiveMode(async (request) => {
        receivedRequest = request;
        return { approved: true, remember: false };
      });

      const result = await PermissionApproval.requestApproval({
        type: 'bash',
        value: 'npm install',
      });

      expect(result.approved).toBe(true);
      expect(receivedRequest).toBeTruthy();
      expect(receivedRequest.type).toBe('bash');
      expect(receivedRequest.value).toBe('npm install');
    });

    it('should handle rejection from callback', async () => {
      PermissionApproval.setInteractiveMode(async () => {
        return { approved: false, remember: false };
      });

      const result = await PermissionApproval.requestApproval({
        type: 'edit',
        value: '/file.ts',
      });

      expect(result.approved).toBe(false);
    });

    it('should return not approved when no callback set', async () => {
      // Manually set interactive mode without callback
      PermissionApproval.setInteractiveMode(null as any);
      // Force interactive mode by directly accessing
      (PermissionApproval as any).mode = 'interactive';
      (PermissionApproval as any).approvalCallback = null;

      const result = await PermissionApproval.requestApproval({
        type: 'bash',
        value: 'test',
      });

      expect(result.approved).toBe(false);
    });
  });

  describe('isInteractive', () => {
    it('should be false by default', () => {
      expect(PermissionApproval.isInteractive()).toBe(false);
    });

    it('should be true after setInteractiveMode', () => {
      PermissionApproval.setInteractiveMode(async () => ({ approved: true, remember: false }));
      expect(PermissionApproval.isInteractive()).toBe(true);
    });

    it('should be false after setHeadlessMode', () => {
      PermissionApproval.setInteractiveMode(async () => ({ approved: true, remember: false }));
      PermissionApproval.setHeadlessMode();
      expect(PermissionApproval.isInteractive()).toBe(false);
    });
  });

  describe('reset', () => {
    it('should reset to headless block mode', () => {
      PermissionApproval.setInteractiveMode(async () => ({ approved: true, remember: false }));
      PermissionApproval.reset();

      expect(PermissionApproval.isInteractive()).toBe(false);
      expect(PermissionApproval.getHeadlessBehavior()).toBe('block');
    });
  });
});
