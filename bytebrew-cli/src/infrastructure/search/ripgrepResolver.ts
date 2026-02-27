import { BinDirectory } from '../lsp/install/BinDirectory.js';
import { installFromGithubRelease } from '../lsp/install/strategies/GithubReleaseStrategy.js';
import type { GithubReleaseSpec } from '../lsp/install/types.js';

const RIPGREP_SPEC: GithubReleaseSpec = {
  type: 'github-release',
  repo: 'BurntSushi/ripgrep',
  assetSelector: (platform, arch) => {
    const cpu = arch === 'arm64' ? 'aarch64' : 'x86_64';
    if (platform === 'linux') return `ripgrep-${cpu}-unknown-linux-musl`;
    if (platform === 'darwin') return `ripgrep-${cpu}-apple-darwin`;
    if (platform === 'win32') return `ripgrep-${cpu}-pc-windows-msvc`;
    return undefined;
  },
  binaryName: 'rg',
};

let cachedRgPath: string | null = null;
let resolvePromise: Promise<string | null> | null = null;
let downloadFailedUntil = 0;

/**
 * Resolve the path to the `rg` binary.
 * 1. Check system PATH
 * 2. Check managed bin directory (~/.local/share/vector/bin/)
 * 3. Auto-install from GitHub releases if not found
 *
 * Concurrent calls are coalesced — only one resolve runs at a time.
 * Returns the absolute path to `rg`, or null if unavailable.
 */
export async function resolveRgBinary(): Promise<string | null> {
  if (cachedRgPath !== null) return cachedRgPath;

  // Coalesce concurrent calls into a single resolve
  if (resolvePromise !== null) return resolvePromise;

  resolvePromise = doResolve();
  const result = await resolvePromise;
  resolvePromise = null;
  return result;
}

async function doResolve(): Promise<string | null> {
  // 1. Check system PATH
  const systemRg = Bun.which('rg');
  if (systemRg) {
    cachedRgPath = systemRg;
    return cachedRgPath;
  }

  const binDir = new BinDirectory();

  // 2. Check managed bin directory
  const managedRg = await binDir.hasBinary('rg');
  if (managedRg) {
    cachedRgPath = managedRg;
    return cachedRgPath;
  }

  // 3. Auto-install from GitHub releases
  if (process.env.BYTEBREW_DISABLE_LSP_DOWNLOAD === 'true') {
    return null;
  }

  // Skip download if a recent attempt failed (retry after 15 min)
  if (Date.now() < downloadFailedUntil) {
    return null;
  }

  try {
    const result = await installFromGithubRelease(RIPGREP_SPEC, binDir);
    if (result.success && result.binaryPath) {
      cachedRgPath = result.binaryPath;
      return cachedRgPath;
    }
  } catch {
    // Install failed — cache failure for 15 minutes
  }

  downloadFailedUntil = Date.now() + 15 * 60 * 1000;
  return null;
}
