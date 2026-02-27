/**
 * Downloads GitHub Release assets, verifies SHA256 checksums,
 * stages binaries in ~/.bytebrew/updates/staging/ for later apply.
 */

import path from 'path';
import fs from 'fs/promises';
import { ByteBrewHome } from '../config/ByteBrewHome.js';
import { detectPlatform, serverAssetName, clientAssetName } from './PlatformDetector.js';
import { extractTarGz, extractZip } from './ArchiveExtractor.js';
import type { UpdateInfo, ReleaseAsset } from './UpdateChecker.js';

const DOWNLOAD_TIMEOUT_MS = 120_000;
const CHECKSUMS_TIMEOUT_MS = 30_000;

interface StagingManifest {
  version: string;
  timestamp: number;
  complete: boolean;
}

export interface StagedUpdate {
  version: string;
  serverBinaryPath: string;
  clientBinaryPath: string;
}

export type ProgressCallback = (info: { component: string; phase: string }) => void;

/**
 * Verify SHA256 checksum of a buffer against expected checksums map.
 * Skips verification if no checksum entry exists for the given filename.
 */
export function verifySha256(buffer: Buffer, filename: string, checksums: Map<string, string>): void {
  const expected = checksums.get(filename);
  if (!expected) return;

  const hasher = new Bun.CryptoHasher('sha256');
  hasher.update(buffer);
  const actual = hasher.digest('hex');

  if (actual !== expected) {
    throw new Error(`Checksum mismatch for ${filename}: expected ${expected}, got ${actual}`);
  }
}

export class UpdateDownloader {
  private readonly stagingDir: string;

  constructor(stagingDir?: string) {
    this.stagingDir = stagingDir ?? path.join(ByteBrewHome.dir(), 'updates', 'staging');
  }

  /**
   * Download server and client binaries, verify checksums, stage for apply.
   */
  async download(updateInfo: UpdateInfo, onProgress?: ProgressCallback): Promise<StagedUpdate> {
    const platform = detectPlatform();

    await fs.mkdir(this.stagingDir, { recursive: true });

    // Write incomplete manifest
    await this.writeManifest({ version: updateInfo.latestVersion, timestamp: Date.now(), complete: false });

    // Parse checksums
    let checksums = new Map<string, string>();
    if (updateInfo.checksumsUrl) {
      onProgress?.({ component: 'checksums', phase: 'downloading' });
      checksums = await this.downloadChecksums(updateInfo.checksumsUrl);
    }

    // Determine expected asset names
    const serverName = serverAssetName(updateInfo.latestVersion, platform);
    const clientName = clientAssetName(updateInfo.latestVersion, platform);

    // Find assets in release
    const serverAsset = updateInfo.assets.find(a => a.name === serverName);
    const clientAsset = updateInfo.assets.find(a => a.name === clientName);

    if (!serverAsset) {
      await this.cleanStaging();
      throw new Error(`Server asset not found: ${serverName}. Available: ${updateInfo.assets.map(a => a.name).join(', ')}`);
    }
    if (!clientAsset) {
      await this.cleanStaging();
      throw new Error(`Client asset not found: ${clientName}. Available: ${updateInfo.assets.map(a => a.name).join(', ')}`);
    }

    // Download and extract server
    const serverBinaryName = `vector-srv${platform.binaryExt}`;
    const serverDest = path.join(this.stagingDir, serverBinaryName);
    await this.downloadAndExtract(serverAsset, serverName, checksums, 'vector-srv', serverDest, platform.archiveExt, onProgress, 'server');

    // Download and extract client
    const clientBinaryName = `vector${platform.binaryExt}`;
    const clientDest = path.join(this.stagingDir, clientBinaryName);
    await this.downloadAndExtract(clientAsset, clientName, checksums, 'vector', clientDest, platform.archiveExt, onProgress, 'client');

    // chmod on unix
    if (platform.binaryExt === '') {
      await fs.chmod(serverDest, 0o755);
      await fs.chmod(clientDest, 0o755);
    }

    // Mark staging as complete
    await this.writeManifest({ version: updateInfo.latestVersion, timestamp: Date.now(), complete: true });

    return {
      version: updateInfo.latestVersion,
      serverBinaryPath: serverDest,
      clientBinaryPath: clientDest,
    };
  }

  /**
   * Check if there's a pending staged update ready to apply.
   */
  async getPendingUpdate(): Promise<StagedUpdate | null> {
    const manifest = await this.readManifest();
    if (!manifest || !manifest.complete) return null;

    const platform = detectPlatform();
    const serverBinaryPath = path.join(this.stagingDir, `vector-srv${platform.binaryExt}`);
    const clientBinaryPath = path.join(this.stagingDir, `vector${platform.binaryExt}`);

    try {
      await fs.access(serverBinaryPath);
      await fs.access(clientBinaryPath);
    } catch {
      return null;
    }

    return {
      version: manifest.version,
      serverBinaryPath,
      clientBinaryPath,
    };
  }

  /**
   * Remove staging directory.
   */
  async cleanStaging(): Promise<void> {
    await fs.rm(this.stagingDir, { recursive: true, force: true });
  }

  // --- Private helpers ---

  private async downloadAndExtract(
    asset: ReleaseAsset,
    assetName: string,
    checksums: Map<string, string>,
    binaryBaseName: string,
    destPath: string,
    archiveExt: string,
    onProgress: ProgressCallback | undefined,
    componentLabel: string,
  ): Promise<void> {
    onProgress?.({ component: componentLabel, phase: 'downloading' });
    const buffer = await this.downloadAsset(asset);
    verifySha256(buffer, assetName, checksums);

    onProgress?.({ component: componentLabel, phase: 'extracting' });
    const extractDir = path.join(this.stagingDir, `${componentLabel}-extract`);
    await fs.mkdir(extractDir, { recursive: true });

    try {
      await this.extractArchive(buffer, extractDir, binaryBaseName, archiveExt);

      // Find the binary in extracted directory
      const ext = process.platform === 'win32' ? '.exe' : '';
      const binaryFileName = `${binaryBaseName}${ext}`;
      const srcPath = path.join(extractDir, binaryFileName);
      await fs.copyFile(srcPath, destPath);
    } finally {
      await fs.rm(extractDir, { recursive: true, force: true });
    }
  }

  private async downloadAsset(asset: ReleaseAsset): Promise<Buffer> {
    const response = await fetch(asset.browserDownloadUrl, {
      signal: AbortSignal.timeout(DOWNLOAD_TIMEOUT_MS),
    });
    if (!response.ok) {
      throw new Error(`Download failed: ${response.status} ${response.statusText} for ${asset.name}`);
    }
    return Buffer.from(await response.arrayBuffer());
  }

  private async downloadChecksums(url: string): Promise<Map<string, string>> {
    const response = await fetch(url, {
      signal: AbortSignal.timeout(CHECKSUMS_TIMEOUT_MS),
    });
    if (!response.ok) return new Map();

    const text = await response.text();
    const checksums = new Map<string, string>();
    // Format: "sha256hash  filename"
    for (const line of text.split('\n')) {
      const match = line.match(/^([a-f0-9]{64})\s+(.+)$/);
      if (match) {
        checksums.set(match[2].trim(), match[1]);
      }
    }
    return checksums;
  }

  private async extractArchive(
    buffer: Buffer,
    destDir: string,
    binaryName: string,
    archiveExt: string,
  ): Promise<void> {
    if (archiveExt === '.zip') {
      await extractZip(buffer, destDir, binaryName);
    } else {
      await extractTarGz(buffer, destDir, binaryName);
    }
  }

  private async writeManifest(manifest: StagingManifest): Promise<void> {
    const manifestPath = path.join(this.stagingDir, 'manifest.json');
    await fs.writeFile(manifestPath, JSON.stringify(manifest, null, 2));
  }

  private async readManifest(): Promise<StagingManifest | null> {
    try {
      const manifestPath = path.join(this.stagingDir, 'manifest.json');
      const content = await fs.readFile(manifestPath, 'utf-8');
      return JSON.parse(content) as StagingManifest;
    } catch {
      return null;
    }
  }
}
