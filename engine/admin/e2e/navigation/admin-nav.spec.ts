// §1.6 Navigation — all sidebar links visible post-onboarding
// TC: NAV-01 | No SCC tags

import { test, expect, ENGINE_API } from '../fixtures';

const EXPECTED_NAV_ITEMS = [
  /overview/i,
  /agents/i,
  /schemas/i,
  /models/i,
  /mcp/i,
  /knowledge/i,
  /api.?key/i,
  /settings/i,
  /widget/i,
];

// Bypass OnboardingGate by provisioning a model for the new tenant before
// navigating to any admin page. OnboardingGate redirects to /onboarding when
// GET /models returns an empty list — creating a model here lets the normal
// admin surface render.
async function seedModel(page: import('@playwright/test').Page) {
  const token = await page.evaluate(() => localStorage.getItem('jwt') ?? '');
  if (!token) return;
  await page.request.post(`${ENGINE_API}/models`, {
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    data: {
      name: `nav-seed-${Date.now()}`,
      type: 'openrouter',
      kind: 'chat',
      model_name: 'openai/gpt-4o-mini',
      api_key: 'sk-or-nav-test',
      base_url: 'https://openrouter.ai/api/v1',
    },
  });
}

test.describe('Admin navigation — sidebar links', () => {
  // BUG-11 resolved: admin APIClient constructor reads localStorage.getItem('jwt')
  // synchronously at init, so bootstrapAuth() finds an authenticated client and
  // skips the redirect path — the fixture's addInitScript sets 'jwt' before any
  // page scripts execute, which works for both local and external build modes.
  test('all expected sidebar links are visible', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    await seedModel(page);
    await page.goto('/admin/');
    await page.waitForLoadState('networkidle');

    // Try nav/aside first; fall back to full body text (SPA may not use semantic nav)
    const nav = page.locator('nav, aside, [role="navigation"]').first();
    let navText = '';
    try {
      navText = await nav.textContent({ timeout: 5000 }) ?? '';
    } catch {
      navText = await page.textContent('body') ?? '';
    }

    for (const pattern of EXPECTED_NAV_ITEMS) {
      expect.soft(navText).toMatch(pattern);
    }
  });

  test('each sidebar link is clickable and does not 404', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    await seedModel(page);
    await page.goto('/admin/');
    await page.waitForLoadState('networkidle');

    // Sidebar uses <aside> with NavLink elements that have relative hrefs like /agents
    // Try both absolute and relative link patterns
    const navLinks = page.locator('aside a, nav a[href]');
    const count = await navLinks.count();
    expect(count).toBeGreaterThan(0);

    for (let i = 0; i < Math.min(count, 10); i++) {
      const href = await navLinks.nth(i).getAttribute('href');
      if (href) {
        const res = await page.request.get(href.startsWith('http') ? href : `http://localhost:18082${href.startsWith('/admin') ? '' : '/admin'}${href}`);
        expect.soft(res.status()).not.toBe(404);
      }
    }
  });
});
