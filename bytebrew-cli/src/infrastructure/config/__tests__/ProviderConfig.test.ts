import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import {
  readProviderConfig,
  writeProviderConfig,
  readModelsConfig,
  writeModelOverride,
  resetModelOverrides,
  isValidProviderMode,
} from '../ProviderConfig';

describe('ProviderConfig', () => {
  let tempDir: string;
  let originalHome: string | undefined;
  let originalUserProfile: string | undefined;

  beforeEach(() => {
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'provider-config-test-'));
    originalHome = process.env.HOME;
    originalUserProfile = process.env.USERPROFILE;
    process.env.HOME = tempDir;
    process.env.USERPROFILE = tempDir;
  });

  afterEach(() => {
    if (originalHome !== undefined) {
      process.env.HOME = originalHome;
    } else {
      delete process.env.HOME;
    }
    if (originalUserProfile !== undefined) {
      process.env.USERPROFILE = originalUserProfile;
    } else {
      delete process.env.USERPROFILE;
    }
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true });
    }
  });

  describe('isValidProviderMode', () => {
    test('returns true for valid modes', () => {
      expect(isValidProviderMode('proxy')).toBe(true);
      expect(isValidProviderMode('byok')).toBe(true);
      expect(isValidProviderMode('auto')).toBe(true);
    });

    test('returns false for invalid modes', () => {
      expect(isValidProviderMode('invalid')).toBe(false);
      expect(isValidProviderMode('')).toBe(false);
      expect(isValidProviderMode('PROXY')).toBe(false);
    });
  });

  describe('readProviderConfig', () => {
    test('returns defaults when no config file exists', () => {
      const config = readProviderConfig();
      expect(config.mode).toBe('auto');
      expect(config.cloudApiUrl).toBeUndefined();
    });

    test('reads mode from config file', () => {
      const configDir = path.join(tempDir, '.bytebrew');
      fs.mkdirSync(configDir, { recursive: true });
      fs.writeFileSync(
        path.join(configDir, 'provider.json'),
        JSON.stringify({ provider: { mode: 'byok' } }),
        'utf-8'
      );

      const config = readProviderConfig();
      expect(config.mode).toBe('byok');
    });

    test('reads cloudApiUrl from config file', () => {
      const configDir = path.join(tempDir, '.bytebrew');
      fs.mkdirSync(configDir, { recursive: true });
      fs.writeFileSync(
        path.join(configDir, 'provider.json'),
        JSON.stringify({ provider: { mode: 'proxy', cloudApiUrl: 'http://custom:8080' } }),
        'utf-8'
      );

      const config = readProviderConfig();
      expect(config.mode).toBe('proxy');
      expect(config.cloudApiUrl).toBe('http://custom:8080');
    });

    test('handles corrupt JSON gracefully', () => {
      const configDir = path.join(tempDir, '.bytebrew');
      fs.mkdirSync(configDir, { recursive: true });
      fs.writeFileSync(
        path.join(configDir, 'provider.json'),
        '{ invalid json',
        'utf-8'
      );

      const config = readProviderConfig();
      expect(config.mode).toBe('auto');
    });
  });

  describe('writeProviderConfig', () => {
    test('writes mode to config file', () => {
      writeProviderConfig({ mode: 'proxy' });

      const raw = fs.readFileSync(path.join(tempDir, '.bytebrew', 'provider.json'), 'utf-8');
      const config = JSON.parse(raw);
      expect(config.provider.mode).toBe('proxy');
    });

    test('preserves existing settings on partial update', () => {
      writeProviderConfig({ mode: 'proxy', cloudApiUrl: 'http://test:8080' });
      writeProviderConfig({ mode: 'byok' });

      const config = readProviderConfig();
      expect(config.mode).toBe('byok');
      expect(config.cloudApiUrl).toBe('http://test:8080');
    });

    test('creates .bytebrew directory if missing', () => {
      const bytebrewDir = path.join(tempDir, '.bytebrew');
      expect(fs.existsSync(bytebrewDir)).toBe(false);

      writeProviderConfig({ mode: 'proxy' });

      expect(fs.existsSync(bytebrewDir)).toBe(true);
    });
  });

  describe('readModelsConfig', () => {
    test('returns empty overrides when no config exists', () => {
      const models = readModelsConfig();
      expect(models.overrides).toEqual({});
    });

    test('reads model overrides from config', () => {
      const configDir = path.join(tempDir, '.bytebrew');
      fs.mkdirSync(configDir, { recursive: true });
      fs.writeFileSync(
        path.join(configDir, 'provider.json'),
        JSON.stringify({ models: { overrides: { reviewer: 'glm-5', coder: 'glm-4.7' } } }),
        'utf-8'
      );

      const models = readModelsConfig();
      expect(models.overrides).toEqual({ reviewer: 'glm-5', coder: 'glm-4.7' });
    });
  });

  describe('writeModelOverride', () => {
    test('writes a single model override', () => {
      writeModelOverride('reviewer', 'glm-5');

      const models = readModelsConfig();
      expect(models.overrides.reviewer).toBe('glm-5');
    });

    test('preserves existing overrides when adding new', () => {
      writeModelOverride('reviewer', 'glm-5');
      writeModelOverride('coder', 'glm-4.7');

      const models = readModelsConfig();
      expect(models.overrides).toEqual({ reviewer: 'glm-5', coder: 'glm-4.7' });
    });

    test('overwrites existing override for same role', () => {
      writeModelOverride('reviewer', 'glm-4.7');
      writeModelOverride('reviewer', 'glm-5');

      const models = readModelsConfig();
      expect(models.overrides.reviewer).toBe('glm-5');
    });

    test('preserves provider settings when writing model override', () => {
      writeProviderConfig({ mode: 'proxy' });
      writeModelOverride('reviewer', 'glm-5');

      const config = readProviderConfig();
      expect(config.mode).toBe('proxy');

      const models = readModelsConfig();
      expect(models.overrides.reviewer).toBe('glm-5');
    });
  });

  describe('resetModelOverrides', () => {
    test('clears all model overrides', () => {
      writeModelOverride('reviewer', 'glm-5');
      writeModelOverride('coder', 'glm-4.7');

      resetModelOverrides();

      const models = readModelsConfig();
      expect(models.overrides).toEqual({});
    });

    test('preserves provider settings when resetting models', () => {
      writeProviderConfig({ mode: 'byok' });
      writeModelOverride('reviewer', 'glm-5');

      resetModelOverrides();

      const config = readProviderConfig();
      expect(config.mode).toBe('byok');
    });

    test('is safe to call when no overrides exist', () => {
      expect(() => resetModelOverrides()).not.toThrow();
    });
  });
});
