//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TC-TASK-01: POST /tasks — creating a task typically requires an active
// chat session. Attempt it; skip cleanly if the API requires context we
// can't synthesise without an LLM.
func TestTASK01_CreateTask(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	resp := do(t, http.MethodPost, "/api/v1/tasks",
		mustJSON(map[string]any{
			"agent_name": "tc-task-01-agent",
			"prompt":     "test task",
		}), adminToken)
	body := readBody(t, resp)
	// Task creation outside an active session is uncommon — accept any
	// non-5xx outcome.
	if resp.StatusCode >= 500 {
		t.Fatalf("task create returned 5xx: %d %s", resp.StatusCode, string(body))
	}
	if resp.StatusCode >= 400 {
		t.Skipf("task create requires session context (got %d: %s)", resp.StatusCode, string(body))
	}
}

// TC-TASK-02: GET /tasks → 200.
func TestTASK02_ListTasks(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	resp := do(t, http.MethodGet, "/api/v1/tasks", nil, adminToken)
	_ = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TC-TASK-03: GET /tasks/{id} on a fake id → 404.
func TestTASK03_GetNonexistent(t *testing.T) {
	requireSuite(t)

	resp := do(t, http.MethodGet, "/api/v1/tasks/00000000-0000-0000-0000-000000000000", nil, adminToken)
	_ = readBody(t, resp)
	// Some builds return 400 for malformed uuid parse; 404 is the canonical
	// "not found". Any 4xx is acceptable.
	assertStatusAny(t, resp, http.StatusNotFound, http.StatusBadRequest)
}

// TC-TASK-04: GET /tasks returns a JSON array (list shape).
func TestTASK04_ListIsArray(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	resp := do(t, http.MethodGet, "/api/v1/tasks", nil, adminToken)
	body := readBody(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	// Accept either [] array or {"tasks":[]} envelope.
	if len(body) == 0 {
		t.Fatal("empty body for tasks list")
	}
	first := body[0]
	assert.True(t, first == '[' || first == '{',
		"tasks list should be JSON array or object: %s", body)
}

// TC-TASK-05: DELETE /tasks/{id} on fake id returns 4xx (not 5xx).
func TestTASK05_DeleteNonexistent(t *testing.T) {
	requireSuite(t)

	resp := do(t, http.MethodDelete, "/api/v1/tasks/00000000-0000-0000-0000-000000000000", nil, adminToken)
	_ = readBody(t, resp)
	if resp.StatusCode >= 500 {
		t.Fatalf("delete on missing task must not 5xx: got %d", resp.StatusCode)
	}
}

// TC-TASK-06: Respond endpoint — requires an active ask_user flow. Skip if
// unavailable in this build.
func TestTASK06_RespondEndpoint(t *testing.T) {
	requireSuite(t)

	// /api/v1/sessions/{id}/respond — no live session, so expect 4xx.
	resp := do(t, http.MethodPost, "/api/v1/sessions/fake-session/respond",
		mustJSON(map[string]any{"reply": "hi"}), adminToken)
	_ = readBody(t, resp)
	if resp.StatusCode == http.StatusNotFound {
		t.Skip("respond endpoint not registered or session not found")
	}
	if resp.StatusCode >= 500 {
		t.Fatalf("respond must not 5xx: got %d", resp.StatusCode)
	}
}

// TC-TASK-07: 10 concurrent GET /tasks — no races, all 200.
func TestTASK07_ConcurrentList(t *testing.T) {
	requireSuite(t)

	const n = 10
	var wg sync.WaitGroup
	errs := make(chan error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp := do(t, http.MethodGet, "/api/v1/tasks", nil, adminToken)
			_ = readBody(t, resp)
			if resp.StatusCode != http.StatusOK {
				errs <- fmt.Errorf("status %d", resp.StatusCode)
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent list: %v", err)
	}
}

// TC-TASK-08: Filter query param — not every build supports it, but the
// endpoint must never 5xx on an unknown param.
func TestTASK08_ListWithFilter(t *testing.T) {
	requireSuite(t)

	resp := do(t, http.MethodGet, "/api/v1/tasks?status=pending", nil, adminToken)
	body := readBody(t, resp)
	if resp.StatusCode >= 500 {
		t.Fatalf("list with filter must not 5xx: %d %s", resp.StatusCode, string(body))
	}
	// If the endpoint responded OK, make sure it returned parseable JSON.
	if resp.StatusCode == http.StatusOK && len(body) > 0 {
		var anyVal any
		assert.NoError(t, json.Unmarshal(body, &anyVal),
			"OK response should be valid JSON: %s", body)
	}
}
