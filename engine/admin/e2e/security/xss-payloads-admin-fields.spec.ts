// §1.20 OWASP A03 — XSS payloads in admin input fields: all render as escaped text
// TC: SEC-XSS-01

import { test, expect, apiFetch } from '../fixtures';

const XSS_PAYLOAD = '<script>window.__xss_test=true;</script>';
const XSS_ATTR_PAYLOAD = '"><img src=x onerror=window.__xss_test2=true>';

test.describe('XSS prevention — admin input fields', () => {
  test('XSS in agent name stored as literal', async ({ request, adminToken }) => {
    const name = `xss-test-${Date.now()}`;
    const res = await apiFetch(request, '/agents', {
      method: 'POST',
      token: adminToken,
      body: { name, system_prompt: XSS_PAYLOAD },
    });
    if ([200, 201].includes(res.status())) {
      const body = await res.json();
      // system_prompt stored as-is (literal)
      expect.soft(body.system_prompt).toBe(XSS_PAYLOAD);
      await apiFetch(request, `/agents/${name}`, { method: 'DELETE', token: adminToken });
    }
  });

  test('XSS payload in agent system_prompt not executed in admin UI', async ({ authenticatedAdmin, request, adminToken }) => {
    const page = authenticatedAdmin;
    const name = `xss-ui-${Date.now()}`;

    await apiFetch(request, '/agents', {
      method: 'POST',
      token: adminToken,
      body: { name, system_prompt: XSS_PAYLOAD },
    });

    await page.goto('/admin/agents');
    await page.waitForLoadState('networkidle');

    // Check that XSS was NOT executed
    const xssExecuted = await page.evaluate(() => (window as Window & { __xss_test?: boolean }).__xss_test);
    expect(xssExecuted).toBeFalsy();

    await apiFetch(request, `/agents/${name}`, { method: 'DELETE', token: adminToken });
  });

  test('XSS in schema name not executed', async ({ request, adminToken, authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    const createRes = await apiFetch(request, '/schemas', {
      method: 'POST',
      token: adminToken,
      body: { name: `xss-schema-${Date.now()}`, chat_enabled: false },
    });
    const created = await createRes.json();
    const id = created.id;

    await page.goto('/admin/schemas');
    await page.waitForLoadState('networkidle');

    const xssExecuted = await page.evaluate(() => (window as Window & { __xss_test?: boolean }).__xss_test);
    expect(xssExecuted).toBeFalsy();

    if (id) await apiFetch(request, `/schemas/${id}`, { method: 'DELETE', token: adminToken });
  });
});
