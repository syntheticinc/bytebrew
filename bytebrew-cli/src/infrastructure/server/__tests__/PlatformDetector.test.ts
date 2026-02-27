import { describe, it, expect } from 'bun:test';
import {
  detectPlatform,
  serverAssetName,
  clientAssetName,
  type PlatformInfo,
} from '../PlatformDetector';

describe('PlatformDetector', () => {
  describe('detectPlatform', () => {
    it('returns valid PlatformInfo for current platform', () => {
      const p = detectPlatform();
      expect(['linux', 'darwin', 'windows']).toContain(p.os);
      expect(['amd64', 'arm64']).toContain(p.arch);
      expect(['.tar.gz', '.zip']).toContain(p.archiveExt);
      expect(['', '.exe']).toContain(p.binaryExt);
    });

    it('uses .zip for windows, .tar.gz for others', () => {
      const p = detectPlatform();
      if (p.os === 'windows') {
        expect(p.archiveExt).toBe('.zip');
        expect(p.binaryExt).toBe('.exe');
      } else {
        expect(p.archiveExt).toBe('.tar.gz');
        expect(p.binaryExt).toBe('');
      }
    });
  });

  describe('serverAssetName', () => {
    it('formats correctly for linux amd64', () => {
      const p: PlatformInfo = { os: 'linux', arch: 'amd64', archiveExt: '.tar.gz', binaryExt: '' };
      expect(serverAssetName('0.3.0', p)).toBe('vector-srv_0.3.0_linux_amd64.tar.gz');
    });

    it('formats correctly for windows amd64', () => {
      const p: PlatformInfo = { os: 'windows', arch: 'amd64', archiveExt: '.zip', binaryExt: '.exe' };
      expect(serverAssetName('1.0.0', p)).toBe('vector-srv_1.0.0_windows_amd64.zip');
    });

    it('formats correctly for darwin arm64', () => {
      const p: PlatformInfo = { os: 'darwin', arch: 'arm64', archiveExt: '.tar.gz', binaryExt: '' };
      expect(serverAssetName('0.3.0', p)).toBe('vector-srv_0.3.0_darwin_arm64.tar.gz');
    });
  });

  describe('clientAssetName', () => {
    it('formats correctly for linux amd64', () => {
      const p: PlatformInfo = { os: 'linux', arch: 'amd64', archiveExt: '.tar.gz', binaryExt: '' };
      expect(clientAssetName('0.3.0', p)).toBe('vector_0.3.0_linux_amd64.tar.gz');
    });

    it('formats correctly for windows amd64', () => {
      const p: PlatformInfo = { os: 'windows', arch: 'amd64', archiveExt: '.zip', binaryExt: '.exe' };
      expect(clientAssetName('1.0.0', p)).toBe('vector_1.0.0_windows_amd64.zip');
    });

    it('formats correctly for darwin arm64', () => {
      const p: PlatformInfo = { os: 'darwin', arch: 'arm64', archiveExt: '.tar.gz', binaryExt: '' };
      expect(clientAssetName('0.3.0', p)).toBe('vector_0.3.0_darwin_arm64.tar.gz');
    });
  });
});
