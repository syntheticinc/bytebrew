// §1.6 Navigation — empty agents/schemas/KBs shows "Create first…" CTA
// TC: NAV-03 | No SCC tags

import { test, expect, apiFetch } from '../fixtures';

test.describe('Admin — empty state CTAs', () => {
  test('agents page shows content (CTA or list)', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    await page.goto('/admin/agents');
    await page.waitForLoadState('networkidle');

    // Either a "create" CTA or a list of agents should be present
    const createCta = page.locator('button:has-text("Create"), button:has-text("New"), a:has-text("Create"), a:has-text("New agent")');
    const agentList = page.locator('[data-testid="agent-row"], [class*="agent-item"], table tbody tr');
    const ctaCount = await createCta.count();
    const listCount = await agentList.count();
    expect(ctaCount + listCount).toBeGreaterThan(0);
  });

  test('schemas page shows content (CTA or list)', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    await page.goto('/admin/schemas');
    await page.waitForLoadState('networkidle');

    const createCta = page.locator('button:has-text("Create"), button:has-text("New"), a:has-text("Create schema"), a:has-text("New schema")');
    const schemaList = page.locator('[data-testid="schema-row"], [class*="schema-item"], table tbody tr');
    const ctaCount = await createCta.count();
    const listCount = await schemaList.count();
    expect(ctaCount + listCount).toBeGreaterThan(0);
  });

  test('knowledge page shows content (CTA or list)', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    await page.goto('/admin/knowledge');
    await page.waitForLoadState('networkidle');

    const createCta = page.locator('button:has-text("Create"), button:has-text("New"), a:has-text("Create"), a:has-text("Upload")');
    const kbList = page.locator('[data-testid="kb-row"], [class*="kb-item"], table tbody tr');
    const ctaCount = await createCta.count();
    const listCount = await kbList.count();
    expect(ctaCount + listCount).toBeGreaterThan(0);
  });
});
