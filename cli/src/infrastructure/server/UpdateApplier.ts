import path from 'path';
import fs from 'fs/promises';
import { ByteBrewHome } from '../config/ByteBrewHome.js';
import { ServerBinaryManager } from './ServerBinaryManager.js';
import { UpdateDownloader, type StagedUpdate } from './UpdateDownloader.js';
import { detectPlatform } from './PlatformDetector.js';

/**
 * Detect if running via bun/node (dev mode) vs compiled binary.
 * In dev mode, we skip client binary replacement.
 */
export function isDevMode(): boolean {
  const name = path.basename(process.execPath).toLowerCase();
  return name.startsWith('bun') || name.startsWith('node');
}

/** Consumer-side interface for staged update provider */
interface StagedUpdateProvider {
  getPendingUpdate(): Promise<StagedUpdate | null>;
  cleanStaging(): Promise<void>;
}

/** Consumer-side interface for locating server binary */
interface BinaryLocator {
  findBinary(): string | null;
}

/**
 * Applies a previously staged update by swapping binaries in place.
 *
 * Flow:
 * 1. Check for pending staged update
 * 2. Backup current binaries to backupDir
 * 3. Replace server binary (always)
 * 4. Replace client binary (skip in dev mode)
 * 5. Clean staging directory
 *
 * Windows special handling:
 * - Running binary is locked, so rename current → .old, then copy new.
 * - .old files are cleaned up on next startup via cleanupOldFiles().
 */
export class UpdateApplier {
  private readonly backupDir: string;
  private readonly stagingProvider: StagedUpdateProvider;
  private readonly binaryLocator: BinaryLocator;

  constructor(opts?: {
    backupDir?: string;
    stagingProvider?: StagedUpdateProvider;
    binaryLocator?: BinaryLocator;
  }) {
    this.backupDir = opts?.backupDir ?? path.join(ByteBrewHome.dir(), 'updates', 'backups');
    this.stagingProvider = opts?.stagingProvider ?? new UpdateDownloader();
    this.binaryLocator = opts?.binaryLocator ?? new ServerBinaryManager();
  }

  /**
   * Apply a previously staged update.
   * Returns { applied: true, version } if update was applied.
   * Returns { applied: false } if no pending update.
   */
  async applyPending(): Promise<{ applied: boolean; version?: string }> {
    const pending = await this.stagingProvider.getPendingUpdate();
    if (!pending) return { applied: false };

    await fs.mkdir(this.backupDir, { recursive: true });

    try {
      await this.applyServerBinary(pending.serverBinaryPath);

      if (!isDevMode()) {
        await this.applyClientBinary(pending.clientBinaryPath);
      }

      await this.stagingProvider.cleanStaging();

      return { applied: true, version: pending.version };
    } catch (err) {
      await this.stagingProvider.cleanStaging();
      throw err;
    }
  }

  /**
   * Clean up .old files from previous Windows self-update.
   * Also remove stale backups (older than 7 days).
   * Called on every startup — must never block or throw.
   */
  async cleanupOldFiles(): Promise<void> {
    // Windows: remove .old file left from previous client self-update
    if (process.platform === 'win32') {
      const oldPath = process.execPath + '.old';
      await fs.rm(oldPath, { force: true }).catch(() => {});
    }

    // Remove stale backups
    try {
      const stat = await fs.stat(this.backupDir);
      const age = Date.now() - stat.mtimeMs;
      const sevenDays = 7 * 24 * 60 * 60 * 1000;
      if (age > sevenDays) {
        await fs.rm(this.backupDir, { recursive: true, force: true });
      }
    } catch {
      // Backup dir doesn't exist — nothing to clean
    }
  }

  // --- Private ---

  private async applyServerBinary(stagedPath: string): Promise<void> {
    const platform = detectPlatform();
    let currentPath = this.binaryLocator.findBinary();

    // If not installed anywhere, install to data dir
    if (!currentPath) {
      const dataDir = ByteBrewHome.dataDir();
      const binDir = path.join(dataDir, 'vector', 'bin');
      await fs.mkdir(binDir, { recursive: true });
      currentPath = path.join(binDir, `vector-srv${platform.binaryExt}`);
    }

    // Backup current binary (ignore if no existing binary — first install)
    const backupPath = path.join(this.backupDir, `vector-srv${platform.binaryExt}.bak`);
    try {
      await fs.copyFile(currentPath, backupPath);
    } catch {
      // No existing binary to backup — first install
    }

    // Replace with staged binary
    await fs.copyFile(stagedPath, currentPath);
    if (platform.binaryExt === '') {
      await fs.chmod(currentPath, 0o755);
    }
  }

  private async applyClientBinary(stagedPath: string): Promise<void> {
    const currentPath = process.execPath;
    const platform = detectPlatform();
    const backupPath = path.join(this.backupDir, `vector${platform.binaryExt}.bak`);

    if (process.platform !== 'win32') {
      // Unix: backup then overwrite
      await fs.copyFile(currentPath, backupPath);
      await fs.copyFile(stagedPath, currentPath);
      await fs.chmod(currentPath, 0o755);
      return;
    }

    // Windows: running binary is locked — rename current to .old, then copy new
    const oldPath = currentPath + '.old';
    await fs.rename(currentPath, oldPath);
    await fs.copyFile(stagedPath, currentPath);
  }
}
