import { describe, it, expect, beforeEach, afterEach, mock } from 'bun:test';
import { UpdateChecker, type UpdateInfo } from '../UpdateChecker';

describe('UpdateChecker', () => {
  let originalFetch: typeof globalThis.fetch;
  let fetchMock: ReturnType<typeof mock>;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    fetchMock = mock(() => Promise.resolve(new Response('{}', { status: 200 })));
    globalThis.fetch = fetchMock as unknown as typeof fetch;
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  function makeGitHubResponse(
    tagName: string,
    htmlUrl: string,
    assets: Array<{ name: string; browser_download_url: string; size: number }> = [],
  ): Response {
    return new Response(
      JSON.stringify({ tag_name: tagName, html_url: htmlUrl, assets }),
      { status: 200, headers: { 'Content-Type': 'application/json' } },
    );
  }

  // -- isNewer (semver comparison) -------------------------------------------

  describe('isNewer', () => {
    const checker = new UpdateChecker('0.0.0');

    const cases: Array<{ latest: string; current: string; expected: boolean }> = [
      { latest: '0.3.0', current: '0.2.0', expected: true },
      { latest: '0.2.0', current: '0.3.0', expected: false },
      { latest: '0.2.0', current: '0.2.0', expected: false },
      { latest: '1.0.0', current: '0.9.9', expected: true },
      { latest: '2.0.0', current: '1.9.9', expected: true },
      { latest: '0.2.1', current: '0.2.0', expected: true },
      { latest: '0.0.1', current: '0.0.0', expected: true },
      { latest: '0.0.0', current: '0.0.1', expected: false },
      { latest: '1.0.0', current: '1.0.0', expected: false },
      // Partial versions (missing parts default to 0)
      { latest: '1', current: '0.0.0', expected: true },
      { latest: '1.2', current: '1.1.0', expected: true },
      { latest: '0', current: '0.0.0', expected: false },
    ];

    for (const { latest, current, expected } of cases) {
      it(`${latest} > ${current} = ${expected}`, () => {
        expect(checker.isNewer(latest, current)).toBe(expected);
      });
    }
  });

  // -- check() ---------------------------------------------------------------

  describe('check()', () => {
    it('returns update info when newer version available', async () => {
      fetchMock.mockReturnValueOnce(
        Promise.resolve(makeGitHubResponse('v0.3.0', 'https://github.com/releases/v0.3.0')),
      );

      const checker = new UpdateChecker('0.2.0');
      const result = await checker.check();

      expect(result).not.toBeNull();
      expect(result!.currentVersion).toBe('0.2.0');
      expect(result!.latestVersion).toBe('0.3.0');
      expect(result!.updateAvailable).toBe(true);
      expect(result!.downloadUrl).toBe('https://github.com/releases/v0.3.0');
    });

    it('returns updateAvailable=false when current matches latest', async () => {
      fetchMock.mockReturnValueOnce(
        Promise.resolve(makeGitHubResponse('v0.2.0', 'https://github.com/releases/v0.2.0')),
      );

      const checker = new UpdateChecker('0.2.0');
      const result = await checker.check();

      expect(result).not.toBeNull();
      expect(result!.updateAvailable).toBe(false);
      expect(result!.currentVersion).toBe('0.2.0');
      expect(result!.latestVersion).toBe('0.2.0');
    });

    it('returns updateAvailable=false when current is newer than latest', async () => {
      fetchMock.mockReturnValueOnce(
        Promise.resolve(makeGitHubResponse('v0.2.0', 'https://github.com/releases/v0.2.0')),
      );

      const checker = new UpdateChecker('0.3.0');
      const result = await checker.check();

      expect(result).not.toBeNull();
      expect(result!.updateAvailable).toBe(false);
    });

    it('strips "v" prefix from tag_name', async () => {
      fetchMock.mockReturnValueOnce(
        Promise.resolve(makeGitHubResponse('v1.2.3', 'https://example.com')),
      );

      const checker = new UpdateChecker('1.0.0');
      const result = await checker.check();

      expect(result).not.toBeNull();
      expect(result!.latestVersion).toBe('1.2.3');
    });

    it('handles tag_name without "v" prefix', async () => {
      fetchMock.mockReturnValueOnce(
        Promise.resolve(makeGitHubResponse('1.2.3', 'https://example.com')),
      );

      const checker = new UpdateChecker('1.0.0');
      const result = await checker.check();

      expect(result).not.toBeNull();
      expect(result!.latestVersion).toBe('1.2.3');
      expect(result!.updateAvailable).toBe(true);
    });

    it('returns null on non-200 response', async () => {
      fetchMock.mockReturnValueOnce(
        Promise.resolve(new Response('Not Found', { status: 404 })),
      );

      const checker = new UpdateChecker('0.2.0');
      const result = await checker.check();

      expect(result).toBeNull();
    });

    it('returns null on network error', async () => {
      fetchMock.mockReturnValueOnce(
        Promise.reject(new Error('network timeout')),
      );

      const checker = new UpdateChecker('0.2.0');
      const result = await checker.check();

      expect(result).toBeNull();
    });

    it('returns null on invalid JSON response', async () => {
      fetchMock.mockReturnValueOnce(
        Promise.resolve(new Response('not json', { status: 200 })),
      );

      const checker = new UpdateChecker('0.2.0');
      const result = await checker.check();

      expect(result).toBeNull();
    });

    it('calls GitHub API with correct URL and headers', async () => {
      fetchMock.mockReturnValueOnce(
        Promise.resolve(makeGitHubResponse('v0.1.0', 'https://example.com')),
      );

      const checker = new UpdateChecker('0.1.0', 'TestOrg/TestRepo');
      await checker.check();

      expect(fetchMock).toHaveBeenCalledTimes(1);
      const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
      expect(url).toBe('https://api.github.com/repos/TestOrg/TestRepo/releases/latest');
      expect((init.headers as Record<string, string>)['Accept']).toBe(
        'application/vnd.github.v3+json',
      );
    });

    it('uses default repo when none specified', async () => {
      fetchMock.mockReturnValueOnce(
        Promise.resolve(makeGitHubResponse('v0.1.0', 'https://example.com')),
      );

      const checker = new UpdateChecker('0.1.0');
      await checker.check();

      const [url] = fetchMock.mock.calls[0] as [string, RequestInit];
      expect(url).toBe(
        'https://api.github.com/repos/syntheticinc/bytebrew/releases/latest',
      );
    });

    it('returns null on fetch abort/timeout', async () => {
      fetchMock.mockReturnValueOnce(
        Promise.reject(new DOMException('The operation was aborted', 'AbortError')),
      );

      const checker = new UpdateChecker('0.1.0');
      const result = await checker.check();

      expect(result).toBeNull();
    });
  });

  // -- assets parsing -----------------------------------------------------------

  describe('assets parsing', () => {
    it('parses release assets', async () => {
      const assets = [
        { name: 'vector-srv_0.3.0_linux_amd64.tar.gz', browser_download_url: 'https://example.com/srv.tar.gz', size: 15000000 },
        { name: 'checksums.txt', browser_download_url: 'https://example.com/checksums.txt', size: 500 },
      ];
      fetchMock.mockReturnValueOnce(
        Promise.resolve(makeGitHubResponse('v0.3.0', 'https://github.com/releases/v0.3.0', assets)),
      );

      const checker = new UpdateChecker('0.2.0');
      const result = await checker.check();

      expect(result).not.toBeNull();
      expect(result!.assets).toHaveLength(2);
      expect(result!.assets[0].name).toBe('vector-srv_0.3.0_linux_amd64.tar.gz');
      expect(result!.assets[0].browserDownloadUrl).toBe('https://example.com/srv.tar.gz');
      expect(result!.assets[0].size).toBe(15000000);
    });

    it('finds checksumsUrl', async () => {
      const assets = [
        { name: 'vector_0.3.0_linux_amd64.tar.gz', browser_download_url: 'https://example.com/v.tar.gz', size: 70000000 },
        { name: 'checksums.txt', browser_download_url: 'https://example.com/checksums.txt', size: 500 },
      ];
      fetchMock.mockReturnValueOnce(
        Promise.resolve(makeGitHubResponse('v0.3.0', 'https://github.com/releases/v0.3.0', assets)),
      );

      const checker = new UpdateChecker('0.2.0');
      const result = await checker.check();

      expect(result).not.toBeNull();
      expect(result!.checksumsUrl).toBe('https://example.com/checksums.txt');
    });

    it('returns empty assets when none in response', async () => {
      fetchMock.mockReturnValueOnce(
        Promise.resolve(makeGitHubResponse('v0.3.0', 'https://github.com/releases/v0.3.0')),
      );

      const checker = new UpdateChecker('0.2.0');
      const result = await checker.check();

      expect(result).not.toBeNull();
      expect(result!.assets).toEqual([]);
      expect(result!.checksumsUrl).toBeUndefined();
    });
  });
});
