import fs from "fs/promises";
import type { GithubReleaseSpec, InstallResult } from "../types.js";
import type { BinDirectory } from "../BinDirectory.js";
import { getLogger } from "../../../../lib/logger.js";
import { extractTarGz, extractZip } from "../../../server/ArchiveExtractor.js";

const DOWNLOAD_TIMEOUT_MS = 60_000;

interface GithubRelease {
  assets: { name: string; browser_download_url: string }[];
}

/**
 * Downloads LSP server binaries from GitHub Releases.
 * Supports .tar.gz, .zip, and bare binary assets.
 */
export async function installFromGithubRelease(
  spec: GithubReleaseSpec,
  binDir: BinDirectory,
): Promise<InstallResult> {
  const logger = getLogger();

  const assetPattern = spec.assetSelector(process.platform, process.arch);
  if (!assetPattern) {
    return { success: false, error: `Unsupported platform: ${process.platform}/${process.arch}` };
  }

  await binDir.ensureExists();

  logger.info(`[LSP] github: fetching release for ${spec.repo}`, { pattern: assetPattern });

  try {
    // 1. Fetch latest release
    const releaseUrl = `https://api.github.com/repos/${spec.repo}/releases/latest`;
    const response = await fetchWithTimeout(releaseUrl, {
      headers: { Accept: "application/vnd.github.v3+json" },
    });
    if (!response.ok) {
      return { success: false, error: `GitHub API error: ${response.status} ${response.statusText}` };
    }
    const release: GithubRelease = await response.json();

    // 2. Find matching asset
    const asset = release.assets.find((a) => a.name.includes(assetPattern));
    if (!asset) {
      const available = release.assets.map((a) => a.name).join(", ");
      return { success: false, error: `No asset matching "${assetPattern}". Available: ${available.slice(0, 300)}` };
    }

    logger.info(`[LSP] github: downloading ${asset.name}`);

    // 3. Download
    const downloadResponse = await fetchWithTimeout(asset.browser_download_url);
    if (!downloadResponse.ok) {
      return { success: false, error: `Download failed: ${downloadResponse.status}` };
    }
    const buffer = Buffer.from(await downloadResponse.arrayBuffer());

    // 4. Extract based on extension
    const binaryName = spec.binaryName || guessServerBinaryName(spec.repo);
    const binPath = binDir.getPath();

    if (asset.name.endsWith(".tar.gz") || asset.name.endsWith(".tgz")) {
      await extractTarGz(buffer, binPath, binaryName);
    } else if (asset.name.endsWith(".zip")) {
      await extractZip(buffer, binPath, binaryName);
    } else {
      // Bare binary
      const dest = binDir.binaryPath(binaryName);
      await fs.writeFile(dest, buffer);
    }

    // 5. Set executable permission
    if (process.platform !== "win32") {
      const dest = binDir.binaryPath(binaryName);
      await fs.chmod(dest, 0o755);
    }

    return { success: true, binaryPath: binDir.binaryPath(binaryName) };
  } catch (err) {
    return { success: false, error: String(err) };
  }
}

function guessServerBinaryName(repo: string): string {
  // "rust-lang/rust-analyzer" -> "rust-analyzer"
  // "clangd/clangd" -> "clangd"
  return repo.split("/").pop() || "server";
}

async function fetchWithTimeout(
  url: string,
  init?: RequestInit,
): Promise<Response> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), DOWNLOAD_TIMEOUT_MS);
  try {
    return await fetch(url, { ...init, signal: controller.signal });
  } finally {
    clearTimeout(timer);
  }
}
