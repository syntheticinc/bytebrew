import { describe, it, expect, beforeEach, afterEach } from 'bun:test';
import path from 'path';
import fs from 'fs/promises';
import os from 'os';
import { UpdateDownloader, verifySha256 } from '../UpdateDownloader';

describe('UpdateDownloader', () => {
  let tmpDir: string;
  let downloader: UpdateDownloader;

  beforeEach(async () => {
    tmpDir = path.join(os.tmpdir(), `vector-test-${Date.now()}-${Math.random().toString(36).slice(2)}`);
    await fs.mkdir(tmpDir, { recursive: true });
    downloader = new UpdateDownloader(tmpDir);
  });

  afterEach(async () => {
    await fs.rm(tmpDir, { recursive: true, force: true });
  });

  // -- getPendingUpdate ---------------------------------------------------------

  describe('getPendingUpdate', () => {
    it('returns null when no manifest exists', async () => {
      const result = await downloader.getPendingUpdate();
      expect(result).toBeNull();
    });

    it('returns null when manifest is incomplete', async () => {
      await fs.writeFile(
        path.join(tmpDir, 'manifest.json'),
        JSON.stringify({ version: '0.3.0', timestamp: Date.now(), complete: false }),
      );
      const result = await downloader.getPendingUpdate();
      expect(result).toBeNull();
    });

    it('returns null when manifest is complete but binaries missing', async () => {
      await fs.writeFile(
        path.join(tmpDir, 'manifest.json'),
        JSON.stringify({ version: '0.3.0', timestamp: Date.now(), complete: true }),
      );
      const result = await downloader.getPendingUpdate();
      expect(result).toBeNull();
    });

    it('returns StagedUpdate when manifest is complete and binaries exist', async () => {
      await fs.writeFile(
        path.join(tmpDir, 'manifest.json'),
        JSON.stringify({ version: '0.3.0', timestamp: Date.now(), complete: true }),
      );
      // Create fake binaries matching current platform
      const ext = process.platform === 'win32' ? '.exe' : '';
      await fs.writeFile(path.join(tmpDir, `vector-srv${ext}`), 'fake-server');
      await fs.writeFile(path.join(tmpDir, `vector${ext}`), 'fake-client');

      const result = await downloader.getPendingUpdate();
      expect(result).not.toBeNull();
      expect(result!.version).toBe('0.3.0');
      expect(result!.serverBinaryPath).toContain(`vector-srv${ext}`);
      expect(result!.clientBinaryPath).toContain(`vector${ext}`);
    });

    it('returns correct absolute paths for binaries', async () => {
      await fs.writeFile(
        path.join(tmpDir, 'manifest.json'),
        JSON.stringify({ version: '1.0.0', timestamp: Date.now(), complete: true }),
      );
      const ext = process.platform === 'win32' ? '.exe' : '';
      await fs.writeFile(path.join(tmpDir, `vector-srv${ext}`), 'server');
      await fs.writeFile(path.join(tmpDir, `vector${ext}`), 'client');

      const result = await downloader.getPendingUpdate();
      expect(result).not.toBeNull();
      // Paths should be absolute and within staging dir
      expect(path.isAbsolute(result!.serverBinaryPath)).toBe(true);
      expect(path.isAbsolute(result!.clientBinaryPath)).toBe(true);
      expect(result!.serverBinaryPath.startsWith(tmpDir)).toBe(true);
      expect(result!.clientBinaryPath.startsWith(tmpDir)).toBe(true);
    });
  });

  // -- cleanStaging -------------------------------------------------------------

  describe('cleanStaging', () => {
    it('removes staging directory', async () => {
      await fs.writeFile(path.join(tmpDir, 'manifest.json'), '{}');
      await downloader.cleanStaging();

      const exists = await fs.access(tmpDir).then(() => true).catch(() => false);
      expect(exists).toBe(false);
    });

    it('does not throw when staging directory does not exist', async () => {
      await fs.rm(tmpDir, { recursive: true, force: true });
      // Should not throw
      await downloader.cleanStaging();
    });

    it('removes nested files and directories', async () => {
      const subDir = path.join(tmpDir, 'nested', 'deep');
      await fs.mkdir(subDir, { recursive: true });
      await fs.writeFile(path.join(subDir, 'file.txt'), 'content');
      await fs.writeFile(path.join(tmpDir, 'manifest.json'), '{}');

      await downloader.cleanStaging();

      const exists = await fs.access(tmpDir).then(() => true).catch(() => false);
      expect(exists).toBe(false);
    });
  });

  // -- verifySha256 (standalone function) -----------------------------------------

  describe('verifySha256', () => {
    it('passes when checksum matches', () => {
      const buffer = Buffer.from('hello world');
      const hasher = new Bun.CryptoHasher('sha256');
      hasher.update(buffer);
      const expectedHash = hasher.digest('hex');

      const checksums = new Map<string, string>();
      checksums.set('test.tar.gz', expectedHash);

      // Should not throw
      verifySha256(buffer, 'test.tar.gz', checksums);
    });

    it('throws on checksum mismatch', () => {
      const buffer = Buffer.from('hello world');
      const checksums = new Map<string, string>();
      checksums.set('test.tar.gz', 'a'.repeat(64));

      expect(() => {
        verifySha256(buffer, 'test.tar.gz', checksums);
      }).toThrow('Checksum mismatch');
    });

    it('throws with filename in error message on mismatch', () => {
      const buffer = Buffer.from('data');
      const checksums = new Map<string, string>();
      checksums.set('myfile.zip', 'b'.repeat(64));

      expect(() => {
        verifySha256(buffer, 'myfile.zip', checksums);
      }).toThrow('myfile.zip');
    });

    it('skips verification when no checksum available for file', () => {
      const buffer = Buffer.from('hello world');
      const checksums = new Map<string, string>();
      // No entry for 'unknown.tar.gz'

      // Should not throw
      verifySha256(buffer, 'unknown.tar.gz', checksums);
    });

    it('skips verification when checksums map is empty', () => {
      const buffer = Buffer.from('anything');
      const checksums = new Map<string, string>();

      // Should not throw
      verifySha256(buffer, 'file.tar.gz', checksums);
    });

    it('verifies correctly with different buffer contents', () => {
      const buffer1 = Buffer.from('content A');
      const buffer2 = Buffer.from('content B');

      const hasher = new Bun.CryptoHasher('sha256');
      hasher.update(buffer1);
      const hashOfA = hasher.digest('hex');

      const checksums = new Map<string, string>();
      checksums.set('file.tar.gz', hashOfA);

      // buffer1 should pass
      verifySha256(buffer1, 'file.tar.gz', checksums);

      // buffer2 should fail
      expect(() => {
        verifySha256(buffer2, 'file.tar.gz', checksums);
      }).toThrow('Checksum mismatch');
    });
  });

  // -- constructor defaults -----------------------------------------------------

  describe('constructor', () => {
    it('accepts custom staging directory', () => {
      const custom = new UpdateDownloader('/tmp/custom-staging');
      // Verify it doesn't throw and creates an instance
      expect(custom).toBeDefined();
    });
  });
});
