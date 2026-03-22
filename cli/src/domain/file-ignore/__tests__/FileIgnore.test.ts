import { describe, it, expect } from 'bun:test';
import { FileIgnore } from '../FileIgnore.js';

describe('FileIgnore', () => {
  describe('shouldIgnoreName', () => {
    const fi = new FileIgnore();

    it('should ignore .git', () => {
      expect(fi.shouldIgnoreName('.git')).toBe(true);
    });

    it('should ignore node_modules', () => {
      expect(fi.shouldIgnoreName('node_modules')).toBe(true);
    });

    it('should ignore vendor', () => {
      expect(fi.shouldIgnoreName('vendor')).toBe(true);
    });

    it('should ignore __pycache__', () => {
      expect(fi.shouldIgnoreName('__pycache__')).toBe(true);
    });

    it('should ignore .idea', () => {
      expect(fi.shouldIgnoreName('.idea')).toBe(true);
    });

    it('should ignore .DS_Store', () => {
      expect(fi.shouldIgnoreName('.DS_Store')).toBe(true);
    });

    it('should ignore Thumbs.db', () => {
      expect(fi.shouldIgnoreName('Thumbs.db')).toBe(true);
    });

    it('should ignore lock files', () => {
      expect(fi.shouldIgnoreName('package-lock.json')).toBe(true);
      expect(fi.shouldIgnoreName('yarn.lock')).toBe(true);
      expect(fi.shouldIgnoreName('pnpm-lock.yaml')).toBe(true);
      expect(fi.shouldIgnoreName('bun.lockb')).toBe(true);
    });

    it('should ignore default directories (dist, build, etc)', () => {
      expect(fi.shouldIgnoreName('dist')).toBe(true);
      expect(fi.shouldIgnoreName('build')).toBe(true);
      expect(fi.shouldIgnoreName('out')).toBe(true);
      expect(fi.shouldIgnoreName('target')).toBe(true);
      expect(fi.shouldIgnoreName('.bytebrew')).toBe(true);
      expect(fi.shouldIgnoreName('coverage')).toBe(true);
    });

    it('should ignore hidden files (starting with dot)', () => {
      expect(fi.shouldIgnoreName('.env')).toBe(true);
      expect(fi.shouldIgnoreName('.env.local')).toBe(true);
      expect(fi.shouldIgnoreName('.hidden')).toBe(true);
    });

    it('should NOT ignore . and ..', () => {
      expect(fi.shouldIgnoreName('.')).toBe(false);
      expect(fi.shouldIgnoreName('..')).toBe(false);
    });

    it('should NOT ignore regular files', () => {
      expect(fi.shouldIgnoreName('src')).toBe(false);
      expect(fi.shouldIgnoreName('main.ts')).toBe(false);
      expect(fi.shouldIgnoreName('README.md')).toBe(false);
      expect(fi.shouldIgnoreName('index.tsx')).toBe(false);
    });

    it('should NOT ignore regular directories', () => {
      expect(fi.shouldIgnoreName('internal')).toBe(false);
      expect(fi.shouldIgnoreName('cmd')).toBe(false);
      expect(fi.shouldIgnoreName('lib')).toBe(false);
    });
  });

  describe('shouldIgnore with gitignore patterns', () => {
    it('should respect gitignore patterns', () => {
      const fi = new FileIgnore(['*.log', 'tmp/']);

      expect(fi.shouldIgnore('server.log')).toBe(true);
      expect(fi.shouldIgnore('logs/app.log')).toBe(true);
      expect(fi.shouldIgnore('tmp/', true)).toBe(true);
    });

    it('should match relative paths via gitignore', () => {
      const fi = new FileIgnore(['docs/generated/']);

      expect(fi.shouldIgnore('docs/generated/', true)).toBe(true);
      expect(fi.shouldIgnore('docs/README.md')).toBe(false);
    });

    it('should still apply always-ignore rules with gitignore', () => {
      const fi = new FileIgnore(['*.log']);

      expect(fi.shouldIgnore('node_modules', true)).toBe(true);
      expect(fi.shouldIgnore('.git', true)).toBe(true);
    });

    it('should work without gitignore patterns', () => {
      const fi = new FileIgnore();

      expect(fi.shouldIgnore('src/main.ts')).toBe(false);
      expect(fi.shouldIgnore('node_modules', true)).toBe(true);
    });
  });

  describe('shouldIgnore with backslashes (Windows paths)', () => {
    it('should normalize backslashes to forward slashes', () => {
      const fi = new FileIgnore(['logs/']);

      expect(fi.shouldIgnore('logs\\app.log')).toBe(true);
    });
  });

  describe('directory detection', () => {
    it('should ignore default-ignore directories by name', () => {
      const fi = new FileIgnore();

      expect(fi.shouldIgnore('dist', true)).toBe(true);
      expect(fi.shouldIgnore('build', true)).toBe(true);
      expect(fi.shouldIgnore('.next', true)).toBe(true);
    });
  });
});
