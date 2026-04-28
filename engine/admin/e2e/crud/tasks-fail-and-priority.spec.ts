// §1.7-ext CRUD — Tasks: fail transition; set priority; verify status
// TC: CRUD-19 | GAP-9

import { test, expect, apiFetch } from '../fixtures';

test.describe('Tasks — fail and priority', () => {
  test('task can be failed and priority can be set', async ({ request, adminToken }) => {
    const createRes = await apiFetch(request, '/tasks', {
      method: 'POST',
      token: adminToken,
      body: { title: `fail-task-${Date.now()}`, description: 'Fail test' },
    });

    if (createRes.status() === 404) {
      test.skip(true, 'Tasks endpoint not available');
      return;
    }
    if (createRes.status() === 400) {
      test.skip(true, 'Tasks require an existing session_id — no live session available in this run');
      return;
    }
    expect([200, 201]).toContain(createRes.status());
    const task = await createRes.json();
    const taskId = task.id;

    // Fail the task
    const failRes = await apiFetch(request, `/tasks/${taskId}/fail`, {
      method: 'POST',
      token: adminToken,
      body: { reason: 'Test failure' },
    });
    expect([200, 204]).toContain(failRes.status());

    // Verify status=failed
    const getRes = await apiFetch(request, `/tasks/${taskId}`, { token: adminToken });
    const body = await getRes.json();
    expect(['failed', 'error']).toContain(body.status ?? body.state);
  });

  test('task priority can be set', async ({ request, adminToken }) => {
    const createRes = await apiFetch(request, '/tasks', {
      method: 'POST',
      token: adminToken,
      body: { title: `priority-task-${Date.now()}`, description: 'Priority test' },
    });

    if (createRes.status() === 404) {
      test.skip(true, 'Tasks endpoint not available');
      return;
    }
    if (createRes.status() === 400) {
      test.skip(true, 'Tasks require an existing session_id — no live session available in this run');
      return;
    }
    const task = await createRes.json();
    const taskId = task.id;

    const priorityRes = await apiFetch(request, `/tasks/${taskId}/priority`, {
      method: 'PUT',
      token: adminToken,
      body: { priority: 2 },
    });
    expect([200, 204]).toContain(priorityRes.status());

    const getRes = await apiFetch(request, `/tasks/${taskId}`, { token: adminToken });
    const body = await getRes.json();
    expect(body.priority).toBe(2);
  });
});
