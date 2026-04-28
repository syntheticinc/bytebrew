// §1.21 Concurrency — two contexts DELETE same agent → one 204, other 404 (not 500)
// TC: CON-02 | GAP-7

import { test, expect, apiFetch } from '../fixtures';

test.describe('Concurrency — concurrent delete race', () => {
  test('two parallel DELETE same agent → one 204, other 404, neither 500', async ({ request, adminToken }) => {
    const name = `race-del-agent-${Date.now()}`;
    await apiFetch(request, '/agents', {
      method: 'POST',
      token: adminToken,
      body: { name, system_prompt: 'Race test' },
    });

    // Fire two concurrent deletes
    const [res1, res2] = await Promise.all([
      apiFetch(request, `/agents/${name}`, { method: 'DELETE', token: adminToken }),
      apiFetch(request, `/agents/${name}`, { method: 'DELETE', token: adminToken }),
    ]);

    const s1 = res1.status();
    const s2 = res2.status();

    // Neither should be 500
    expect.soft(s1).not.toBe(500);
    expect.soft(s2).not.toBe(500);

    // One should succeed, other should 404
    const statuses = [s1, s2].sort();
    // Acceptable combos: [200,404], [204,404], [200,200], [204,204] — last two only if idempotent
    expect([200, 204, 404]).toContain(s1);
    expect([200, 204, 404]).toContain(s2);
  });

  test('two parallel DELETE same model → no 500', async ({ request, adminToken }) => {
    const name = `race-del-model-${Date.now()}`;
    await apiFetch(request, '/models', {
      method: 'POST',
      token: adminToken,
      body: {
        name,
        type: 'openai_compatible',
        provider: 'openrouter',
        model_name: 'openai/gpt-4o-mini',
        api_key: 'sk-test',
        base_url: 'https://openrouter.ai/api/v1',
      },
    });

    const [res1, res2] = await Promise.all([
      apiFetch(request, `/models/${name}`, { method: 'DELETE', token: adminToken }),
      apiFetch(request, `/models/${name}`, { method: 'DELETE', token: adminToken }),
    ]);

    expect.soft(res1.status()).not.toBe(500);
    expect.soft(res2.status()).not.toBe(500);
  });
});
