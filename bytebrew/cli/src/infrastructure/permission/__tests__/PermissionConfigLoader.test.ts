import { describe, it, expect, beforeEach, afterEach } from 'bun:test';
import fs from 'fs/promises';
import path from 'path';
import os from 'os';
import { loadPermissionConfig, getDefaultPermissionConfig, savePermissionConfig } from '../PermissionConfigLoader.js';

// Use temp directory for test configs
const TEST_ROOT = path.join(os.tmpdir(), 'vector-permission-test-' + Date.now());
const BYTEBREW_DIR = path.join(TEST_ROOT, '.bytebrew');

beforeEach(async () => {
  await fs.mkdir(BYTEBREW_DIR, { recursive: true });
});

afterEach(async () => {
  try {
    await fs.rm(TEST_ROOT, { recursive: true, force: true });
  } catch {}
});

describe('PermissionConfigLoader', () => {
  describe('getDefaultPermissionConfig', () => {
    it('should return a valid default config', () => {
      const config = getDefaultPermissionConfig();

      expect(config.permissions.allow.length).toBeGreaterThan(0);
      expect(config.permissions.deny.length).toBeGreaterThan(0);
    });

    it('should have allow rules in Bash(pattern) format', () => {
      const config = getDefaultPermissionConfig();
      const bashAllowRules = config.permissions.allow.filter(r => r.startsWith('Bash('));

      expect(bashAllowRules.length).toBeGreaterThan(0);
      expect(bashAllowRules).toContain('Bash(npm *)');
      expect(bashAllowRules).toContain('Bash(go *)');
      expect(bashAllowRules).toContain('Bash(git status)');
    });

    it('should have deny rules for dangerous commands', () => {
      const config = getDefaultPermissionConfig();

      expect(config.permissions.deny).toContain('Bash(sudo *)');
      expect(config.permissions.deny).toContain('Bash(rm -rf /)');
      expect(config.permissions.deny).toContain('Bash(git push *)');
      expect(config.permissions.deny).toContain('Bash(shutdown *)');
    });

    it('should have Read and Edit in allow list', () => {
      const config = getDefaultPermissionConfig();

      expect(config.permissions.allow).toContain('Read');
      expect(config.permissions.allow).toContain('Edit');
    });
  });

  describe('loadPermissionConfig', () => {
    it('should return defaults when no config file exists', async () => {
      const config = await loadPermissionConfig(TEST_ROOT);

      expect(config.permissions.allow.length).toBeGreaterThan(0);
      expect(config.permissions.deny.length).toBeGreaterThan(0);
    });

    it('should load project-level settings.local.json', async () => {
      const customConfig = {
        permissions: {
          allow: ['Bash(custom *)'],
          deny: ['Bash(forbidden *)'],
        },
      };

      await fs.writeFile(
        path.join(BYTEBREW_DIR, 'settings.local.json'),
        JSON.stringify(customConfig),
        'utf-8'
      );

      const config = await loadPermissionConfig(TEST_ROOT);

      expect(config.permissions.allow).toEqual(['Bash(custom *)']);
      expect(config.permissions.deny).toEqual(['Bash(forbidden *)']);
    });

    it('should merge partial config with defaults', async () => {
      const partialConfig = {
        permissions: {
          allow: ['Bash(custom *)'],
        },
      };

      await fs.writeFile(
        path.join(BYTEBREW_DIR, 'settings.local.json'),
        JSON.stringify(partialConfig),
        'utf-8'
      );

      const config = await loadPermissionConfig(TEST_ROOT);

      // Overridden
      expect(config.permissions.allow).toEqual(['Bash(custom *)']);
      // Defaults preserved for deny
      expect(config.permissions.deny.length).toBeGreaterThan(0);
    });

    it('should handle settings.local.json with other fields', async () => {
      const settingsWithOtherFields = {
        some_other_field: 'value',
        permissions: {
          allow: ['Read'],
          deny: [],
        },
      };

      await fs.writeFile(
        path.join(BYTEBREW_DIR, 'settings.local.json'),
        JSON.stringify(settingsWithOtherFields),
        'utf-8'
      );

      const config = await loadPermissionConfig(TEST_ROOT);

      expect(config.permissions.allow).toEqual(['Read']);
      expect(config.permissions.deny).toEqual([]);
    });
  });

  describe('savePermissionConfig', () => {
    it('should save config to .bytebrew/settings.local.json', async () => {
      const config = getDefaultPermissionConfig();
      config.permissions.allow = ['Bash(test *)'];

      await savePermissionConfig(config, TEST_ROOT);

      const content = await fs.readFile(
        path.join(BYTEBREW_DIR, 'settings.local.json'),
        'utf-8'
      );
      const saved = JSON.parse(content);
      expect(saved.permissions.allow).toEqual(['Bash(test *)']);
    });

    it('should create .bytebrew directory if not exists', async () => {
      const newRoot = path.join(os.tmpdir(), 'vector-perm-save-' + Date.now());
      const config = getDefaultPermissionConfig();

      try {
        await savePermissionConfig(config, newRoot);

        const content = await fs.readFile(
          path.join(newRoot, '.bytebrew', 'settings.local.json'),
          'utf-8'
        );
        expect(JSON.parse(content)).toBeTruthy();
      } finally {
        await fs.rm(newRoot, { recursive: true, force: true }).catch(() => {});
      }
    });

    it('should preserve other fields in settings.local.json', async () => {
      // Write initial config with other fields
      await fs.writeFile(
        path.join(BYTEBREW_DIR, 'settings.local.json'),
        JSON.stringify({ some_field: 'value', other: 123 }),
        'utf-8'
      );

      // Save permissions
      const config = getDefaultPermissionConfig();
      config.permissions.allow = ['Read'];
      await savePermissionConfig(config, TEST_ROOT);

      // Read back
      const content = await fs.readFile(
        path.join(BYTEBREW_DIR, 'settings.local.json'),
        'utf-8'
      );
      const saved = JSON.parse(content);

      expect(saved.some_field).toBe('value');
      expect(saved.other).toBe(123);
      expect(saved.permissions.allow).toEqual(['Read']);
    });
  });

  describe('deepCopyConfig', () => {
    it('should return a deep copy that does not mutate original', () => {
      const config1 = getDefaultPermissionConfig();
      const config2 = getDefaultPermissionConfig();

      config1.permissions.allow.push('Bash(new-rule)');

      expect(config2.permissions.allow).not.toContain('Bash(new-rule)');
    });
  });
});
