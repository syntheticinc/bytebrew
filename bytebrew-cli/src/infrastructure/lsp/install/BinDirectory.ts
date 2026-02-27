import path from "path";
import fs from "fs/promises";
import fsSync from "fs";
import { ByteBrewHome } from '../../config/ByteBrewHome.js';

const APP_NAME = "vector";

/**
 * Manages the local binary directory for auto-installed LSP servers.
 *
 * Paths per platform:
 *   Linux:   ~/.local/share/vector/bin
 *   macOS:   ~/Library/Application Support/vector/bin
 *   Windows: %APPDATA%/vector/bin
 */
export class BinDirectory {
  private readonly binPath: string;

  constructor() {
    this.binPath = path.join(ByteBrewHome.dataDir(), APP_NAME, "bin");
  }

  getPath(): string {
    return this.binPath;
  }

  async ensureExists(): Promise<void> {
    await fs.mkdir(this.binPath, { recursive: true });
  }

  /**
   * Find a binary in the managed directory.
   * Checks: <binDir>/<name>, <binDir>/node_modules/.bin/<name>
   * On Windows also checks .exe and .cmd extensions.
   */
  async hasBinary(name: string): Promise<string | undefined> {
    const candidates = this.getCandidatePaths(name);
    for (const candidate of candidates) {
      try {
        await fs.access(candidate);
        return candidate;
      } catch {
        // not found, try next
      }
    }
    return undefined;
  }

  /**
   * Synchronous check — used by whichBin() in the hot path.
   * Returns the binary path if found, null otherwise.
   */
  hasBinarySync(name: string): string | null {
    const candidates = this.getCandidatePaths(name);
    for (const candidate of candidates) {
      try {
        fsSync.accessSync(candidate);
        return candidate;
      } catch {
        // not found, try next
      }
    }
    return null;
  }

  /** Get the full path for a binary name (with .exe on Windows) */
  binaryPath(name: string): string {
    const ext = process.platform === "win32" ? ".exe" : "";
    return path.join(this.binPath, name + ext);
  }

  private getCandidatePaths(name: string): string[] {
    const isWin = process.platform === "win32";
    const ext = isWin ? ".exe" : "";
    const candidates: string[] = [
      // Direct binary in bin dir
      path.join(this.binPath, name + ext),
      // npm/bun installed packages
      path.join(this.binPath, "node_modules", ".bin", name + ext),
    ];
    if (isWin) {
      // bun/npm also creates .cmd wrappers on Windows
      candidates.push(
        path.join(this.binPath, "node_modules", ".bin", name + ".cmd"),
      );
    }
    return candidates;
  }
}
