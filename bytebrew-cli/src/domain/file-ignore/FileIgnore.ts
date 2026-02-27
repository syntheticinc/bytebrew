// FileIgnore - centralized file/directory ignore logic
// Combines built-in patterns, default patterns, and gitignore rules
import ignore, { Ignore } from 'ignore';

/** Directories/files that should ALWAYS be ignored (never useful for analysis) */
const ALWAYS_IGNORE = new Set([
  '.git', '.svn', '.hg',
  'node_modules', 'vendor',
  '__pycache__',
  '.idea', '.vscode',
  '.DS_Store', 'Thumbs.db',
  // Lock files
  'package-lock.json', 'yarn.lock', 'pnpm-lock.yaml', 'bun.lockb',
]);

/** Directories that are ignored by default but can be overridden by user config */
const DEFAULT_IGNORE = new Set([
  'dist', 'build', 'out', 'target', 'bin', 'obj',
  '.bytebrew', '.next', '.nuxt',
  'coverage',
  '.cache',
  '.venv', 'venv', 'env',
]);

export class FileIgnore {
  private alwaysIgnore: Set<string>;
  private defaultIgnore: Set<string>;
  private gitignore: Ignore;

  constructor(gitignorePatterns?: string[]) {
    this.alwaysIgnore = ALWAYS_IGNORE;
    this.defaultIgnore = DEFAULT_IGNORE;
    this.gitignore = ignore();

    if (gitignorePatterns) {
      this.gitignore.add(gitignorePatterns);
    }

    // Add default ignore as gitignore patterns too (for relative path matching)
    this.gitignore.add([...DEFAULT_IGNORE]);
  }

  /**
   * Fast check by name only (no path context needed).
   * Use for directory traversal where only the entry name is available.
   */
  shouldIgnoreName(name: string): boolean {
    if (this.alwaysIgnore.has(name)) {
      return true;
    }
    if (this.defaultIgnore.has(name)) {
      return true;
    }
    // Hidden files/dirs (except . and ..)
    if (name.startsWith('.') && name !== '.' && name !== '..') {
      return true;
    }
    return false;
  }

  /**
   * Full check using relative path and gitignore rules.
   * relativePath should be relative to project root (e.g. "src/utils/file.ts").
   */
  shouldIgnore(relativePath: string, isDirectory?: boolean): boolean {
    // Normalize: backslashes → forward slashes, strip trailing slash
    let normalized = relativePath.replace(/\\/g, '/').replace(/\/+$/, '');

    // Extract the name (last segment)
    const segments = normalized.split('/');
    const name = segments[segments.length - 1];

    // Fast path: check by name
    if (name && this.shouldIgnoreName(name)) {
      return true;
    }

    // Check gitignore patterns (handles full path matching)
    const pathToCheck = isDirectory ? normalized + '/' : normalized;
    return this.gitignore.ignores(pathToCheck);
  }
}
