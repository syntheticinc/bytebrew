import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import { ByteBrewConfig } from '../ByteBrewConfig';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';

describe('ByteBrewConfig', () => {
  let tempDir: string;
  let originalHome: string | undefined;
  let originalUserProfile: string | undefined;

  beforeEach(() => {
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'bytebrew-config-test-'));
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
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  test('load() returns empty object when file does not exist', () => {
    const config = new ByteBrewConfig();
    expect(config.load()).toEqual({});
  });

  test('save() creates config file with JSON content', () => {
    const config = new ByteBrewConfig();
    config.save({ bridge_url: 'https://bridge.example.com' });

    const filePath = path.join(tempDir, '.bytebrew', 'config.json');
    expect(fs.existsSync(filePath)).toBe(true);

    const raw = fs.readFileSync(filePath, 'utf-8');
    const parsed = JSON.parse(raw);
    expect(parsed.bridge_url).toBe('https://bridge.example.com');
  });

  test('getBridgeUrl() returns undefined when no config exists', () => {
    const config = new ByteBrewConfig();
    expect(config.getBridgeUrl()).toBeUndefined();
  });

  test('setBridgeUrl() saves and round-trips correctly', () => {
    const config = new ByteBrewConfig();
    config.setBridgeUrl('wss://bridge.example.com:8443');

    const config2 = new ByteBrewConfig();
    expect(config2.getBridgeUrl()).toBe('wss://bridge.example.com:8443');
  });

  test('load() returns empty object when file has invalid JSON', () => {
    const configDir = path.join(tempDir, '.bytebrew');
    fs.mkdirSync(configDir, { recursive: true });
    fs.writeFileSync(path.join(configDir, 'config.json'), '{not valid json!!!', 'utf-8');

    const config = new ByteBrewConfig();
    expect(config.load()).toEqual({});
  });
});
