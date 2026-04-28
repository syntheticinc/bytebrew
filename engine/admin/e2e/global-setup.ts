// Pre-warm Vite dev servers behind Caddy before any spec runs. The first hit
// to a Vite-served route triggers on-demand compilation that can take 5-15s,
// which racks up against per-test timeouts even when the engine and SPA logic
// are healthy. Fetching the SPA shells once up-front pays the cold-compile
// cost in setup so individual specs see warm responses.
//
// In CI (production-built nginx-served SPAs through `make`-style stacks) these
// requests just hit cached static files, so the warm-up is effectively free.

import type { FullConfig } from '@playwright/test';

const ROUTES_TO_WARM = [
  '/login',
  '/register',
  '/dashboard',
  '/admin/',
  '/',
];

export default async function globalSetup(config: FullConfig): Promise<void> {
  const baseURL =
    config.projects[0]?.use?.baseURL ??
    process.env.PLAYWRIGHT_BASE_URL ??
    'http://localhost:18082';

  await Promise.all(
    ROUTES_TO_WARM.map(async (path) => {
      const url = `${baseURL}${path}`;
      try {
        const ctl = new AbortController();
        const t = setTimeout(() => ctl.abort(), 60_000);
        await fetch(url, { signal: ctl.signal }).finally(() => clearTimeout(t));
      } catch {
        // Vite may briefly 5xx during first-compile; the next test's own
        // navigation will retry. Swallowing here keeps setup non-blocking.
      }
    }),
  );
}
