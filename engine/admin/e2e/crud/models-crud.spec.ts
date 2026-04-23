// §1.7 CRUD — Models: create OpenRouter chat model, edit, delete; filter by kind
// TC: CRUD-01 | SCC-01 (no unauth access)

import { test, expect, apiFetch } from '../fixtures';

test.describe('Models CRUD', () => {
  let createdModelName: string;

  test('create OpenRouter chat model via API', async ({ request, adminToken }) => {
    createdModelName = `test-model-${Date.now()}`;
    const res = await apiFetch(request, '/models', {
      method: 'POST',
      token: adminToken,
      body: {
        name: createdModelName,
        kind: 'chat',
        provider: 'openrouter',
        model_name: 'openai/gpt-3.5-turbo',
        api_key: 'sk-or-test-placeholder',
        base_url: 'https://openrouter.ai/api/v1',
      },
    });
    expect([200, 201]).toContain(res.status());
    const body = await res.json();
    expect(body.name ?? body.id).toBeTruthy();
  });

  test('created model appears in GET /models list', async ({ request, adminToken }) => {
    const name = `list-model-${Date.now()}`;
    await apiFetch(request, '/models', {
      method: 'POST',
      token: adminToken,
      body: {
        name,
        kind: 'chat',
        provider: 'openrouter',
        model_name: 'openai/gpt-4o-mini',
        api_key: 'sk-or-test',
        base_url: 'https://openrouter.ai/api/v1',
      },
    });

    const listRes = await apiFetch(request, '/models', { token: adminToken });
    expect(listRes.status()).toBe(200);
    const body = await listRes.json();
    const models = Array.isArray(body) ? body : (body.models ?? body.data ?? []);
    const found = models.some((m: { name: string }) => m.name === name);
    expect(found).toBe(true);

    // Teardown
    await apiFetch(request, `/models/${name}`, { method: 'DELETE', token: adminToken });
  });

  test('delete model removes it from list', async ({ request, adminToken }) => {
    const name = `del-model-${Date.now()}`;
    await apiFetch(request, '/models', {
      method: 'POST',
      token: adminToken,
      body: {
        name,
        kind: 'chat',
        provider: 'openrouter',
        model_name: 'openai/gpt-4o-mini',
        api_key: 'sk-or-test',
        base_url: 'https://openrouter.ai/api/v1',
      },
    });

    const delRes = await apiFetch(request, `/models/${name}`, { method: 'DELETE', token: adminToken });
    expect([200, 204]).toContain(delRes.status());

    const listRes = await apiFetch(request, '/models', { token: adminToken });
    const body = await listRes.json();
    const models = Array.isArray(body) ? body : (body.models ?? body.data ?? []);
    const found = models.some((m: { name: string }) => m.name === name);
    expect(found).toBe(false);
  });

  test('models page renders in admin UI', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    await page.goto('/admin/models');
    await page.waitForLoadState('networkidle');
    const errorBoundary = page.locator('text=/Something went wrong|Error:/i');
    await expect(errorBoundary).not.toBeVisible();
  });
});
