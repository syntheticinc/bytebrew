import { describe, it, expect, beforeEach, afterEach } from 'bun:test';
import path from 'path';
import fs from 'fs/promises';
import os from 'os';
import { isDevMode, UpdateApplier } from '../UpdateApplier';

describe('UpdateApplier', () => {
  describe('isDevMode', () => {
    it('returns true when running via bun', () => {
      // When running tests via bun, process.execPath is the bun binary
      const name = path.basename(process.execPath).toLowerCase();
      const expected = name.startsWith('bun') || name.startsWith('node');
      expect(isDevMode()).toBe(expected);
    });

    it('detects dev mode based on execPath basename', () => {
      // In test environment, execPath is always bun or node
      expect(isDevMode()).toBe(true);
    });
  });

  describe('cleanupOldFiles', () => {
    let tmpDir: string;

    beforeEach(async () => {
      tmpDir = path.join(os.tmpdir(), `vector-applier-test-${Date.now()}`);
      await fs.mkdir(tmpDir, { recursive: true });
    });

    afterEach(async () => {
      await fs.rm(tmpDir, { recursive: true, force: true });
    });

    it('removes stale backup directory older than 7 days', async () => {
      const backupDir = path.join(tmpDir, 'backups');
      await fs.mkdir(backupDir, { recursive: true });
      await fs.writeFile(path.join(backupDir, 'vector-srv.bak'), 'binary-data');

      // Artificially age the directory to 8 days ago
      const eightDaysAgo = new Date(Date.now() - 8 * 24 * 60 * 60 * 1000);
      await fs.utimes(backupDir, eightDaysAgo, eightDaysAgo);

      const applier = new UpdateApplier({ backupDir });
      await applier.cleanupOldFiles();

      const exists = await fs.access(backupDir).then(() => true).catch(() => false);
      expect(exists).toBe(false);
    });

    it('keeps recent backup directory', async () => {
      const backupDir = path.join(tmpDir, 'backups');
      await fs.mkdir(backupDir, { recursive: true });
      await fs.writeFile(path.join(backupDir, 'vector-srv.bak'), 'binary-data');

      const applier = new UpdateApplier({ backupDir });
      await applier.cleanupOldFiles();

      // Directory was just created — mtime is fresh, should be kept
      const exists = await fs.access(backupDir).then(() => true).catch(() => false);
      expect(exists).toBe(true);
    });

    it('does not throw when backup directory does not exist', async () => {
      const backupDir = path.join(tmpDir, 'nonexistent');

      const applier = new UpdateApplier({ backupDir });

      // Should not throw — gracefully handles missing directory
      await applier.cleanupOldFiles();
    });
  });

  describe('applyPending', () => {
    it('returns { applied: false } when no pending update', async () => {
      const mockProvider = {
        getPendingUpdate: async () => null,
        cleanStaging: async () => {},
      };

      const applier = new UpdateApplier({
        backupDir: path.join(os.tmpdir(), `vector-applier-noop-${Date.now()}`),
        stagingProvider: mockProvider,
      });
      const result = await applier.applyPending();

      expect(result.applied).toBe(false);
      expect(result.version).toBeUndefined();
    });
  });
});
