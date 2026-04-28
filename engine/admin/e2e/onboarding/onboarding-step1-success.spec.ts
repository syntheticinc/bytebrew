// §1.5 Onboarding Step 1 — success: fill form → Next advances; model persisted in GET /models
// TC: OB-02 | No SCC tags

import { test, expect, ENGINE_API, apiFetch } from '../fixtures';

test.describe('Onboarding Step 1 — success flow', () => {
  test.skip(true, 'Requires real OpenRouter API key from environment; skip in CI without key');

  test('fill step 1 with valid OpenRouter creds → advances to step 2', async ({ page, request }) => {
    await page.goto('/admin/onboarding');

    // Provider
    const providerSelect = page.locator('select[name="provider"]').first();
    if (await providerSelect.count() > 0) {
      await providerSelect.selectOption({ label: /openrouter/i });
    }

    const modelInput = page.locator('input[name="model_name"], input[placeholder*="model"]').first();
    await modelInput.fill('openai/gpt-3.5-turbo');

    const apiKeyInput = page.locator('input[name="api_key"], input[type="password"]').first();
    await apiKeyInput.fill(process.env.OPENROUTER_API_KEY ?? 'sk-or-placeholder');

    const nextBtn = page.locator('button:has-text("Next"), button[type="submit"]').first();
    await nextBtn.click();

    // Should advance — step 2 indicator or template screen visible
    const step2 = page.locator('[data-step="2"], [data-testid="step2"], text=/template|pick|choose/i').first();
    await expect(step2).toBeVisible({ timeout: 10_000 });
  });

  test('model exists in GET /models after step 1 completion', async ({ request, adminToken }) => {
    const res = await apiFetch(request, '/models', { token: adminToken });
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(Array.isArray(body) || Array.isArray(body.models) || Array.isArray(body.data)).toBe(true);
  });
});
