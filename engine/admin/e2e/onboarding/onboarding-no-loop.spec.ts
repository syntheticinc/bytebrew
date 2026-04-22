// §1.5 Onboarding — no loop: after completing wizard, navigating away does NOT re-open OnboardingGate
// TC: OB-05 | No SCC tags

import { test, expect } from '../fixtures';

test.describe('Onboarding gate — no re-open loop', () => {
  test('navigate to /admin/agents after onboarding completes does not redirect back to onboarding', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;

    // Navigate directly to agents page (assumes onboarding completed via fixture token)
    await page.goto('/admin/agents');
    await page.waitForLoadState('networkidle');

    const url = page.url();
    // Should NOT be redirected to onboarding
    expect(url).not.toContain('onboarding');
  });

  test('navigate to /admin/models does not trigger onboarding gate', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    await page.goto('/admin/models');
    await page.waitForLoadState('networkidle');
    expect(page.url()).not.toContain('onboarding');
  });

  test('navigate to /admin/schemas does not trigger onboarding gate', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    await page.goto('/admin/schemas');
    await page.waitForLoadState('networkidle');
    expect(page.url()).not.toContain('onboarding');
  });
});
