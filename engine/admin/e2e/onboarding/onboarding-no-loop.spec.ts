// §1.5 Onboarding — no loop: after completing wizard, navigating away does NOT re-open OnboardingGate
// TC: OB-05 | No SCC tags

import { test, expect, ENGINE_API } from '../fixtures';

// OnboardingGate checks GET /models on every path change. Seed one model so the
// gate sees has-models state and does NOT redirect to /onboarding.
async function seedModel(page: import('@playwright/test').Page) {
  const token = await page.evaluate(() => localStorage.getItem('jwt') ?? '');
  if (!token) return;
  await page.request.post(`${ENGINE_API}/models`, {
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    data: {
      name: `ob-seed-${Date.now()}`,
      type: 'openrouter',
      kind: 'chat',
      model_name: 'openai/gpt-4o-mini',
      api_key: 'sk-or-ob-test',
      base_url: 'https://openrouter.ai/api/v1',
    },
  });
}

test.describe('Onboarding gate — no re-open loop', () => {
  test('navigate to /admin/agents after onboarding completes does not redirect back to onboarding', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    await seedModel(page);

    // Navigate directly to agents page (onboarding gate now satisfied by seeded model)
    await page.goto('/admin/agents');
    await page.waitForLoadState('networkidle');

    const url = page.url();
    // Should NOT be redirected to onboarding
    expect(url).not.toContain('onboarding');
  });

  test('navigate to /admin/models does not trigger onboarding gate', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    await seedModel(page);
    await page.goto('/admin/models');
    await page.waitForLoadState('networkidle');
    expect(page.url()).not.toContain('onboarding');
  });

  test('navigate to /admin/schemas does not trigger onboarding gate', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    await seedModel(page);
    await page.goto('/admin/schemas');
    await page.waitForLoadState('networkidle');
    expect(page.url()).not.toContain('onboarding');
  });
});
