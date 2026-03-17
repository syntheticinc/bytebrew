export interface PlatformInfo {
  os: 'linux' | 'darwin' | 'windows';
  arch: 'amd64' | 'arm64';
  archiveExt: '.tar.gz' | '.zip';
  binaryExt: '' | '.exe';
}

/**
 * Detect current platform for asset selection.
 * Maps Node.js process.platform/arch to Go-style naming.
 */
export function detectPlatform(): PlatformInfo {
  const os = mapOS(process.platform);
  const arch = mapArch(process.arch);
  const archiveExt = os === 'windows' ? '.zip' : '.tar.gz';
  const binaryExt = os === 'windows' ? '.exe' : '';

  return { os, arch, archiveExt, binaryExt } as PlatformInfo;
}

function mapOS(platform: string): PlatformInfo['os'] {
  if (platform === 'win32') return 'windows';
  if (platform === 'darwin') return 'darwin';
  return 'linux';
}

function mapArch(arch: string): PlatformInfo['arch'] {
  if (arch === 'arm64') return 'arm64';
  return 'amd64';
}

/**
 * Server asset name in GitHub Release.
 * Convention: vector-srv_{version}_{os}_{arch}.{ext}
 * Version without 'v' prefix.
 */
export function serverAssetName(version: string, platform: PlatformInfo): string {
  return `vector-srv_${version}_${platform.os}_${platform.arch}${platform.archiveExt}`;
}

/**
 * Client asset name in GitHub Release.
 * Convention: vector_{version}_{os}_{arch}.{ext}
 */
export function clientAssetName(version: string, platform: PlatformInfo): string {
  return `vector_${version}_${platform.os}_${platform.arch}${platform.archiveExt}`;
}
