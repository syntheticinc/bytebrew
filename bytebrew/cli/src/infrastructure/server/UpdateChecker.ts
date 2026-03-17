/**
 * Non-blocking update checker.
 * Checks GitHub Releases API for new version.
 * Shows notification if update available.
 */

export interface ReleaseAsset {
  name: string;
  browserDownloadUrl: string;
  size: number;
}

export interface UpdateInfo {
  currentVersion: string;
  latestVersion: string;
  updateAvailable: boolean;
  downloadUrl?: string;
  assets: ReleaseAsset[];
  checksumsUrl?: string;
}

export class UpdateChecker {
  private readonly repo: string;
  private readonly currentVersion: string;

  constructor(currentVersion: string, repo = 'syntheticinc/bytebrew') {
    this.currentVersion = currentVersion;
    this.repo = repo;
  }

  /**
   * Check for updates via GitHub Releases API.
   * Non-blocking -- returns null on any error (network, timeout, parse).
   */
  async check(): Promise<UpdateInfo | null> {
    try {
      const response = await fetch(
        `https://api.github.com/repos/${this.repo}/releases/latest`,
        {
          headers: { Accept: 'application/vnd.github.v3+json' },
          signal: AbortSignal.timeout(5000),
        },
      );

      if (!response.ok) return null;

      const data = (await response.json()) as {
        tag_name: string;
        html_url: string;
        assets?: Array<{
          name: string;
          browser_download_url: string;
          size: number;
        }>;
      };
      const latestVersion = data.tag_name.replace(/^v/, '');

      const assets: ReleaseAsset[] = (data.assets ?? []).map(a => ({
        name: a.name,
        browserDownloadUrl: a.browser_download_url,
        size: a.size,
      }));
      const checksumsAsset = assets.find(a => a.name === 'checksums.txt');

      return {
        currentVersion: this.currentVersion,
        latestVersion,
        updateAvailable: this.isNewer(latestVersion, this.currentVersion),
        downloadUrl: data.html_url,
        assets,
        checksumsUrl: checksumsAsset?.browserDownloadUrl,
      };
    } catch {
      // Network error, timeout, JSON parse error, etc.
      return null;
    }
  }

  /**
   * Semver comparison: returns true if latest > current.
   */
  isNewer(latest: string, current: string): boolean {
    const parse = (v: string): [number, number, number] => {
      const parts = v.split('.').map(Number);
      return [parts[0] || 0, parts[1] || 0, parts[2] || 0];
    };

    const [lMajor, lMinor, lPatch] = parse(latest);
    const [cMajor, cMinor, cPatch] = parse(current);

    if (lMajor !== cMajor) return lMajor > cMajor;
    if (lMinor !== cMinor) return lMinor > cMinor;
    return lPatch > cPatch;
  }
}
