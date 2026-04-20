//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TC-CONC-01: 10 goroutines creating agents with unique names — no
// duplicate errors, all succeed.
func TestCONC01_UniqueAgentCreate(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	const n = 10
	var wg sync.WaitGroup
	statuses := make([]int, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp := do(t, http.MethodPost, "/api/v1/agents",
				mustJSON(map[string]any{
					"name":          fmt.Sprintf("tc-conc-01-%d", i),
					"system_prompt": "p",
				}), adminToken)
			_ = readBody(t, resp)
			statuses[i] = resp.StatusCode
		}(i)
	}
	wg.Wait()

	for i, s := range statuses {
		assert.True(t, s == http.StatusOK || s == http.StatusCreated,
			"goroutine %d got status %d", i, s)
	}
}

// TC-CONC-02: 5 goroutines doing GET /agents — all 200, no panics.
func TestCONC02_ConcurrentList(t *testing.T) {
	requireSuite(t)

	const n = 5
	var wg sync.WaitGroup
	errs := make(chan error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp := do(t, http.MethodGet, "/api/v1/agents", nil, adminToken)
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

// TC-CONC-03: Create 5 agents concurrently, list, count appearances.
func TestCONC03_ConcurrentCreateThenList(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	const n = 5
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp := do(t, http.MethodPost, "/api/v1/agents",
				mustJSON(map[string]any{
					"name":          fmt.Sprintf("tc-conc-03-%d", i),
					"system_prompt": "p",
				}), adminToken)
			_ = readBody(t, resp)
		}(i)
	}
	wg.Wait()

	listResp := do(t, http.MethodGet, "/api/v1/agents", nil, adminToken)
	body := readBody(t, listResp)
	require.Equal(t, http.StatusOK, listResp.StatusCode)

	for i := 0; i < n; i++ {
		expected := fmt.Sprintf("tc-conc-03-%d", i)
		assert.Contains(t, string(body), expected,
			"list must contain concurrently-created %s", expected)
	}
}

// TC-CONC-04: Concurrent create + delete on the same name — must not crash.
func TestCONC04_ConcurrentCreateDelete(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	name := "tc-conc-04-agent"
	var wg sync.WaitGroup

	// Create first, then fight.
	resp := do(t, http.MethodPost, "/api/v1/agents",
		mustJSON(map[string]any{"name": name, "system_prompt": "p"}), adminToken)
	_ = readBody(t, resp)
	assertStatusAny(t, resp, http.StatusOK, http.StatusCreated)

	// 3 recreators, 3 deleters — any mix of outcomes is acceptable as long
	// as no goroutine triggers a 5xx.
	const n = 3
	fives := make(chan int, 2*n)

	for i := 0; i < n; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			r := do(t, http.MethodPost, "/api/v1/agents",
				mustJSON(map[string]any{"name": name, "system_prompt": "p"}), adminToken)
			_ = readBody(t, r)
			if r.StatusCode >= 500 {
				fives <- r.StatusCode
			}
		}()
		go func() {
			defer wg.Done()
			r := do(t, http.MethodDelete, "/api/v1/agents/"+name, nil, adminToken)
			_ = readBody(t, r)
			if r.StatusCode >= 500 {
				fives <- r.StatusCode
			}
		}()
	}
	wg.Wait()
	close(fives)

	for s := range fives {
		t.Errorf("saw 5xx under concurrent create/delete: %d", s)
	}
}

// TC-CONC-05: 3 goroutines POSTing unique models — all 201.
func TestCONC05_ConcurrentModelCreate(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	const n = 3
	var wg sync.WaitGroup
	statuses := make([]int, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp := do(t, http.MethodPost, "/api/v1/models",
				mustJSON(map[string]any{
					"name":       fmt.Sprintf("tc-conc-05-%d", i),
					"type":       "openai_compatible",
					"provider":   "openrouter",
					"model_name": "test-model",
					"api_key":    "k",
					"base_url":   "https://api.test.com",
				}), adminToken)
			_ = readBody(t, resp)
			statuses[i] = resp.StatusCode
		}(i)
	}
	wg.Wait()

	for i, s := range statuses {
		assert.True(t, s == http.StatusOK || s == http.StatusCreated,
			"goroutine %d got status %d", i, s)
	}
}

// TC-CONC-06: Concurrent GET /schemas — stability check.
func TestCONC06_ConcurrentSchemasList(t *testing.T) {
	requireSuite(t)

	const n = 5
	var wg sync.WaitGroup
	errs := make(chan error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp := do(t, http.MethodGet, "/api/v1/schemas", nil, adminToken)
			_ = readBody(t, resp)
			if resp.StatusCode != http.StatusOK {
				errs <- fmt.Errorf("status %d", resp.StatusCode)
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent schemas list: %v", err)
	}
}
