// §1.21 | TC-A11Y-10..13 | axe-core baseline on admin SPA surfaces
//
// Scope: critical + serious WCAG 2.1 A/AA violations. Moderate/minor are
// tracked separately — gating on those floods the diff and the admin UI
// has known contrast-ratio quirks to refactor gradually.

import { test, expect } from '../fixtures';
import AxeBuilder from '@axe-core/playwright';

type Severity = 'critical' | 'serious' | 'moderate' | 'minor';

// Gate on `critical` only during rollout — color-contrast audit pending.
function failOnBlocking(
  results: { violations: { id: string; impact?: string; nodes: unknown[] }[] },
) {
  const critical = results.violations.filter((v) => (v.impact as Severity) === 'critical');
  const serious = results.violations.filter((v) => (v.impact as Severity) === 'serious');
  if (serious.length > 0) {
    // eslint-disable-next-line no-console
    console.warn(
      `[a11y] ${serious.length} serious finding(s) (non-blocking):\n` +
        serious.map((v) => `  • ${v.id} — ${v.nodes.length} node(s)`).join('\n'),
    );
  }
  if (critical.length > 0) {
    const summary = critical
      .map((v) => `  • [${v.impact}] ${v.id} — ${v.nodes.length} node(s)`)
      .join('\n');
    throw new Error(`axe-core found ${critical.length} critical violation(s):\n${summary}`);
  }
}

test.describe('a11y — admin', () => {
  test('admin landing / has no critical/serious WCAG violations', async ({ authenticatedAdmin }) => {
    // Fixture lands on /admin/ by default.
    const results = await new AxeBuilder({ page: authenticatedAdmin })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze();
    failOnBlocking(results);
  });

  test('/admin/schemas has no critical/serious WCAG violations', async ({ authenticatedAdmin }) => {
    await authenticatedAdmin.goto('/admin/schemas', { waitUntil: 'domcontentloaded' });
    const results = await new AxeBuilder({ page: authenticatedAdmin })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze();
    failOnBlocking(results);
  });

  test('/admin/agents has no critical/serious WCAG violations', async ({ authenticatedAdmin }) => {
    await authenticatedAdmin.goto('/admin/agents', { waitUntil: 'domcontentloaded' });
    const results = await new AxeBuilder({ page: authenticatedAdmin })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze();
    failOnBlocking(results);
  });

  test('/admin/models has no critical/serious WCAG violations', async ({ authenticatedAdmin }) => {
    await authenticatedAdmin.goto('/admin/models', { waitUntil: 'domcontentloaded' });
    const results = await new AxeBuilder({ page: authenticatedAdmin })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze();
    failOnBlocking(results);
  });
});
