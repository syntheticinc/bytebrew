import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import { ByteBrewHome } from '../ByteBrewHome';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';

describe('ByteBrewHome', () => {
  test('dir() returns path ending with .bytebrew', () => {
    const dir = ByteBrewHome.dir();
    expect(dir.endsWith('.bytebrew') || dir.endsWith('.bytebrew/')).toBe(true);
  });

  test('dir() uses HOME or USERPROFILE env variable', () => {
    const home = process.env.HOME || process.env.USERPROFILE;
    const dir = ByteBrewHome.dir();
    expect(dir).toBe(path.join(home!, '.bytebrew'));
  });

  test('authFile() returns path to auth.json inside .bytebrew', () => {
    const authFile = ByteBrewHome.authFile();
    expect(authFile).toBe(path.join(ByteBrewHome.dir(), 'auth.json'));
  });

  test('licenseFile() returns path to license.jwt inside .bytebrew', () => {
    const licenseFile = ByteBrewHome.licenseFile();
    expect(licenseFile).toBe(path.join(ByteBrewHome.dir(), 'license.jwt'));
  });

  describe('ensureDir()', () => {
    let tempDir: string;
    let originalHome: string | undefined;
    let originalUserProfile: string | undefined;

    beforeEach(() => {
      tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'bytebrewhome-test-'));
      originalHome = process.env.HOME;
      originalUserProfile = process.env.USERPROFILE;
      // Override HOME to use temp dir
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

    test('ensureDir() creates .bytebrew directory', () => {
      const expectedDir = path.join(tempDir, '.bytebrew');
      expect(fs.existsSync(expectedDir)).toBe(false);

      ByteBrewHome.ensureDir();

      expect(fs.existsSync(expectedDir)).toBe(true);
      expect(fs.statSync(expectedDir).isDirectory()).toBe(true);
    });

    test('ensureDir() is idempotent', () => {
      ByteBrewHome.ensureDir();
      // Should not throw when called again
      expect(() => ByteBrewHome.ensureDir()).not.toThrow();
    });
  });

  describe('dataDir()', () => {
    test('returns a non-empty string', () => {
      const dir = ByteBrewHome.dataDir();
      expect(typeof dir).toBe('string');
      expect(dir.length).toBeGreaterThan(0);
    });

    test('returns platform-appropriate path', () => {
      const dir = ByteBrewHome.dataDir();
      if (process.platform === 'win32') {
        expect(dir).toContain('AppData');
      } else if (process.platform === 'darwin') {
        expect(dir).toContain('Application Support');
      } else {
        // Linux: either XDG_DATA_HOME or ~/.local/share
        expect(dir).toMatch(/\.local[\\/]share|XDG_DATA_HOME/);
      }
    });
  });
});
