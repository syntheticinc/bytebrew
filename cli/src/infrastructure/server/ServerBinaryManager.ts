import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';
import { ByteBrewHome } from '../config/ByteBrewHome.js';

/**
 * Finds the bytebrew-srv binary in known locations.
 *
 * Search order:
 * 1. Same directory as CLI binary (co-located install)
 * 2. User data directory (managed installs)
 * 3. System PATH (manual install)
 *
 * Does NOT download — that is Plan 07 (Build + Auto-Update).
 */
export class ServerBinaryManager {
  private readonly searchPaths: string[];

  constructor() {
    const ext = process.platform === 'win32' ? '.exe' : '';
    const binaryName = `bytebrew-srv${ext}`;

    this.searchPaths = [
      // Same directory as CLI binary
      path.join(path.dirname(process.execPath), binaryName),
      // User data dir (managed installs)
      path.join(ByteBrewHome.dataDir(), 'bytebrew', 'bin', binaryName),
    ];
  }

  /**
   * Find server binary. Returns absolute path or null if not found.
   */
  findBinary(): string | null {
    // Check known paths first
    for (const p of this.searchPaths) {
      if (fs.existsSync(p)) return p;
    }

    // Check PATH via 'which' / 'where'
    return this.findInPath();
  }

  /**
   * Get installed version by running `bytebrew-srv --version`.
   */
  getVersion(binaryPath: string): string | null {
    try {
      return execSync(`"${binaryPath}" --version`, {
        encoding: 'utf-8',
        stdio: ['pipe', 'pipe', 'pipe'],
        timeout: 5000,
      }).trim();
    } catch {
      return null;
    }
  }

  private findInPath(): string | null {
    try {
      const cmd = process.platform === 'win32' ? 'where bytebrew-srv' : 'which bytebrew-srv';
      const result = execSync(cmd, {
        encoding: 'utf-8',
        stdio: ['pipe', 'pipe', 'pipe'],
        timeout: 5000,
      }).trim();
      if (result) {
        // 'where' on Windows may return multiple lines; take the first
        const firstLine = result.split('\n')[0].trim();
        if (firstLine && fs.existsSync(firstLine)) {
          return firstLine;
        }
      }
    } catch {
      // Not in PATH
    }
    return null;
  }
}
