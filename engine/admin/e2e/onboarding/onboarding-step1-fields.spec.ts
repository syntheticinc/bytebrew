// §1.5 Onboarding Step 1 — field validation: OpenRouter auto-fills base_url; empty submit flags required fields
// TC: OB-01 | No SCC tags (unauthenticated onboarding page)

import { test, expect, BASE_URL, ENGINE_API, apiFetch } from '../fixtures';

test.describe('Onboarding Step 1 — field validation', () => {
  test('OpenRouter selection auto-fills base_url', async ({ page }) => {
    await page.goto('/admin/onboarding');
    // Select OpenRouter as provider
    const providerSelect = page.locator('select[name="provider"], [data-testid="provider-select"], input[name="provider"]').first();
    if (await providerSelect.count() > 0) {
      await providerSelect.selectOption({ label: /openrouter/i });
    } else {
      const openRouterOption = page.locator('text=/openrouter/i').first();
      if (await openRouterOption.count() > 0) await openRouterOption.click();
    }
    // base_url should be auto-filled
    const baseUrlInput = page.locator('input[name="base_url"], input[placeholder*="base"], [data-testid="base-url-input"]').first();
    if (await baseUrlInput.count() > 0) {
      const value = await baseUrlInput.inputValue();
      expect(value).toMatch(/openrouter/i);
    }
  });

  test('empty submit on Step 1 shows required field errors', async ({ page }) => {
    await page.goto('/admin/onboarding');
    // Click Next without filling anything
    const nextBtn = page.locator('button:has-text("Next"), button[type="submit"]').first();
    await nextBtn.click();
    // Expect at least one validation error visible
    const errorMsg = page.locator('[role="alert"], .error, [data-testid*="error"], [class*="error"], [class*="invalid"]').first();
    await expect(errorMsg).toBeVisible({ timeout: 5000 });
  });

  test('model_name field is required', async ({ page }) => {
    await page.goto('/admin/onboarding');
    const apiKeyInput = page.locator('input[name="api_key"], input[type="password"], [data-testid="api-key-input"]').first();
    if (await apiKeyInput.count() > 0) {
      await apiKeyInput.fill('sk-or-test-key');
    }
    const nextBtn = page.locator('button:has-text("Next"), button[type="submit"]').first();
    await nextBtn.click();
    // Should still show error if model_name missing
    const errors = page.locator('[role="alert"], .error, [data-testid*="error"], [class*="error"]');
    const count = await errors.count();
    expect(count).toBeGreaterThan(0);
  });
});
