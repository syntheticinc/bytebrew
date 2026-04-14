package taskrunner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/configrepo"
	"github.com/syntheticinc/bytebrew/engine/internal/service/task"
)

// ---------------------------------------------------------------------------
// DB helpers
// ---------------------------------------------------------------------------

// setupHookTestDB creates an in-memory SQLite DB with tasks + triggers tables.
// Uses a unique named shared-cache URI so concurrent goroutines (e.g. the hook's
// detached fire() goroutines) share the same in-memory database instance.
func setupHookTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Each test gets a unique DB name to avoid cross-test interference.
	dbName := "file:" + t.Name() + uuid.New().String() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
CREATE TABLE tasks (
	id TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	description TEXT,
	acceptance_criteria TEXT,
	agent_name TEXT NOT NULL,
	source TEXT NOT NULL,
	source_id TEXT,
	user_id TEXT,
	session_id TEXT,
	parent_task_id TEXT,
	depth INTEGER NOT NULL DEFAULT 0,
	status TEXT NOT NULL DEFAULT 'pending',
	mode TEXT NOT NULL DEFAULT 'interactive',
	priority INTEGER NOT NULL DEFAULT 0,
	assigned_agent_id TEXT,
	blocked_by TEXT,
	result TEXT,
	error TEXT,
	created_at DATETIME,
	updated_at DATETIME,
	approved_at DATETIME,
	started_at DATETIME,
	completed_at DATETIME
)`).Error)

	require.NoError(t, db.Exec(`
CREATE TABLE triggers (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	title TEXT NOT NULL,
	agent_id TEXT,
	schema_id TEXT,
	schedule TEXT,
	webhook_path TEXT,
	description TEXT,
	enabled INTEGER NOT NULL DEFAULT 1,
	on_complete_url TEXT,
	on_complete_headers TEXT,
	last_fired_at DATETIME,
	created_at DATETIME,
	updated_at DATETIME
)`).Error)

	return db
}

// newHookNotifier builds a CompletionNotifier with an injected HTTP client so
// tests can control timeout and target a fake server without modifying production code.
func newHookNotifier(httpClient *http.Client) *task.CompletionNotifier {
	// CompletionNotifier's fields are unexported, so we rely on the package
	// constructor and then swap via functional option — but since there is no
	// option, we use the default constructor for integration tests and rely on
	// httptest server being fast enough within the 30 s client timeout.
	// For timeout-specific tests we use a custom notifier built via
	// newFastNotifier below.
	_ = httpClient
	return task.NewCompletionNotifier()
}

// newFastNotifier returns a CompletionNotifier with a short backoff base (5ms)
// so retry tests complete in milliseconds instead of seconds.
func newFastNotifier(clientTimeout time.Duration, maxRetries int) *task.CompletionNotifier {
	return task.NewCompletionNotifierWithOptions(clientTimeout, maxRetries, 5*time.Millisecond)
}

// insertTask inserts a domain.EngineTask directly into SQLite for test setup.
func insertTask(t *testing.T, db *gorm.DB, tsk *domain.EngineTask) {
	t.Helper()
	now := time.Now()
	tsk.CreatedAt = now
	tsk.UpdatedAt = now
	err := db.Exec(`
INSERT INTO tasks
  (id, title, agent_name, source, source_id, status, mode, result,
   acceptance_criteria, blocked_by, depth, priority,
   created_at, updated_at, started_at, completed_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		tsk.ID.String(), tsk.Title, tsk.AgentName, string(tsk.Source), tsk.SourceID,
		string(tsk.Status), string(tsk.Mode), tsk.Result,
		"[]", "[]", tsk.Depth, tsk.Priority,
		tsk.CreatedAt, tsk.UpdatedAt,
		tsk.StartedAt, tsk.CompletedAt,
	).Error
	require.NoError(t, err)
}

// insertTrigger inserts a trigger row directly into SQLite.
func insertTrigger(t *testing.T, db *gorm.DB, id, triggerType, onCompleteURL, onCompleteHeaders string) {
	t.Helper()
	err := db.Exec(`
INSERT INTO triggers (id, type, title, enabled, on_complete_url, on_complete_headers, created_at, updated_at)
VALUES (?,?,?,1,?,?,?,?)`,
		id, triggerType, "test-trigger", onCompleteURL, onCompleteHeaders, time.Now(), time.Now(),
	).Error
	require.NoError(t, err)
}

// hookFixture builds a fully wired TaskCompletionHook using the provided DB and notifier.
func hookFixture(db *gorm.DB, n *task.CompletionNotifier) *TaskCompletionHook {
	taskRepo := configrepo.NewGORMTaskRepository(db)
	triggerRepo := configrepo.NewGORMTriggerRepository(db)
	return NewTaskCompletionHook(taskRepo, triggerRepo, n)
}

// waitForWebhook polls the counter until it reaches the expected value or times out.
func waitForWebhook(t *testing.T, counter *atomic.Int32, expected int32, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if counter.Load() >= expected {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d webhooks, got %d", expected, counter.Load())
}

// ---------------------------------------------------------------------------
// fire() — task not found
// ---------------------------------------------------------------------------

func TestCompletionHook_Fire_TaskNotFound_NoOp(t *testing.T) {
	db := setupHookTestDB(t)
	h := hookFixture(db, task.NewCompletionNotifier())

	// OnCompleted with a UUID that does not exist in the DB — must not panic,
	// must complete quickly (fire() logs and returns early).
	done := make(chan struct{})
	go func() {
		defer close(done)
		h.OnCompleted(context.Background(), uuid.New())
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("OnCompleted did not complete within 2s for missing task")
	}
	h.Stop()
}

// ---------------------------------------------------------------------------
// fire() — task has no SourceID (created from agent / API, not a trigger)
// ---------------------------------------------------------------------------

func TestCompletionHook_Fire_NoSourceID_SkipsWebhook(t *testing.T) {
	db := setupHookTestDB(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tsk := &domain.EngineTask{
		ID:        uuid.New(),
		Title:     "no-source",
		AgentName: "agent1",
		Source:    domain.TaskSourceAgent, // no SourceID
		SourceID:  "",
		Status:    domain.EngineTaskStatusCompleted,
		Mode:      domain.TaskModeBackground,
	}
	insertTask(t, db, tsk)

	h := hookFixture(db, task.NewCompletionNotifier())
	h.OnCompleted(context.Background(), tsk.ID)
	h.Stop()

	assert.Equal(t, int32(0), hits.Load(), "webhook must not fire when SourceID is empty")
}

// ---------------------------------------------------------------------------
// fire() — source is not cron/webhook (e.g. agent) → skip
// ---------------------------------------------------------------------------

func TestCompletionHook_Fire_NonTriggerSource_SkipsWebhook(t *testing.T) {
	db := setupHookTestDB(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	triggerID := uuid.New().String()
	insertTrigger(t, db, triggerID, "cron", srv.URL, "")

	for _, src := range []domain.TaskSource{domain.TaskSourceAgent, domain.TaskSourceAPI, domain.TaskSourceDashboard} {
		src := src
		t.Run(string(src), func(t *testing.T) {
			tsk := &domain.EngineTask{
				ID:        uuid.New(),
				Title:     "non-trigger-src",
				AgentName: "agent1",
				Source:    src,
				SourceID:  triggerID,
				Status:    domain.EngineTaskStatusCompleted,
				Mode:      domain.TaskModeBackground,
			}
			insertTask(t, db, tsk)
			h := hookFixture(db, task.NewCompletionNotifier())
			h.OnCompleted(context.Background(), tsk.ID)
			h.Stop()
		})
	}

	assert.Equal(t, int32(0), hits.Load(), "webhook must not fire for non-cron/webhook sources")
}

// ---------------------------------------------------------------------------
// fire() — trigger not found in DB → no panic
// ---------------------------------------------------------------------------

func TestCompletionHook_Fire_TriggerNotFound_NoOp(t *testing.T) {
	db := setupHookTestDB(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Task references a trigger ID that doesn't exist in the triggers table.
	tsk := &domain.EngineTask{
		ID:        uuid.New(),
		Title:     "orphan-trigger",
		AgentName: "agent1",
		Source:    domain.TaskSourceCron,
		SourceID:  uuid.New().String(), // non-existent trigger
		Status:    domain.EngineTaskStatusCompleted,
		Mode:      domain.TaskModeBackground,
	}
	insertTask(t, db, tsk)

	h := hookFixture(db, task.NewCompletionNotifier())
	h.OnCompleted(context.Background(), tsk.ID)
	h.Stop()

	assert.Equal(t, int32(0), hits.Load(), "webhook must not fire when trigger is not found")
}

// ---------------------------------------------------------------------------
// fire() — trigger has empty OnCompleteURL → skip
// ---------------------------------------------------------------------------

func TestCompletionHook_Fire_EmptyOnCompleteURL_SkipsWebhook(t *testing.T) {
	db := setupHookTestDB(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	triggerID := uuid.New().String()
	insertTrigger(t, db, triggerID, "cron", "", "") // empty URL

	tsk := &domain.EngineTask{
		ID:        uuid.New(),
		Title:     "no-url",
		AgentName: "agent1",
		Source:    domain.TaskSourceCron,
		SourceID:  triggerID,
		Status:    domain.EngineTaskStatusCompleted,
		Mode:      domain.TaskModeBackground,
	}
	insertTask(t, db, tsk)

	h := hookFixture(db, task.NewCompletionNotifier())
	h.OnCompleted(context.Background(), tsk.ID)
	h.Stop()

	assert.Equal(t, int32(0), hits.Load(), "webhook must not fire when OnCompleteURL is empty")
}

// ---------------------------------------------------------------------------
// fire() — happy path: webhook delivered successfully (200 OK)
// ---------------------------------------------------------------------------

func TestCompletionHook_Fire_HappyPath_WebhookDelivered(t *testing.T) {
	db := setupHookTestDB(t)

	var (
		hits    atomic.Int32
		gotBody []byte
		mu      sync.Mutex
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		hits.Add(1)
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		gotBody = buf
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	triggerID := uuid.New().String()
	insertTrigger(t, db, triggerID, "webhook", srv.URL, "")

	now := time.Now()
	started := now.Add(-500 * time.Millisecond)
	completed := now
	tsk := &domain.EngineTask{
		ID:          uuid.New(),
		Title:       "happy",
		AgentName:   "support-agent",
		Source:      domain.TaskSourceWebhook,
		SourceID:    triggerID,
		Status:      domain.EngineTaskStatusCompleted,
		Mode:        domain.TaskModeBackground,
		Result:      "done successfully",
		StartedAt:   &started,
		CompletedAt: &completed,
	}
	insertTask(t, db, tsk)

	h := hookFixture(db, task.NewCompletionNotifier())
	h.OnCompleted(context.Background(), tsk.ID)
	h.Stop()

	require.Equal(t, int32(1), hits.Load(), "webhook must be called exactly once")

	mu.Lock()
	body := gotBody
	mu.Unlock()

	var payload task.CompletionPayload
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, tsk.ID.String(), payload.TaskID)
	assert.Equal(t, string(tsk.Status), payload.Status)
	assert.Equal(t, tsk.Result, payload.Result)
	assert.Equal(t, triggerID, payload.TriggerID)
	assert.Equal(t, tsk.AgentName, payload.AgentName)
	assert.GreaterOrEqual(t, payload.DurationMs, int64(0))
}

// ---------------------------------------------------------------------------
// fire() — payload: DurationMs computed from StartedAt/CompletedAt
// ---------------------------------------------------------------------------

func TestCompletionHook_Fire_DurationMs_Computed(t *testing.T) {
	db := setupHookTestDB(t)

	var (
		mu      sync.Mutex
		gotBody []byte
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		mu.Lock()
		gotBody = buf
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	triggerID := uuid.New().String()
	insertTrigger(t, db, triggerID, "cron", srv.URL, "")

	started := time.Now().Add(-2 * time.Second)
	completed := started.Add(1500 * time.Millisecond)
	tsk := &domain.EngineTask{
		ID:          uuid.New(),
		Title:       "duration-test",
		AgentName:   "cron-agent",
		Source:      domain.TaskSourceCron,
		SourceID:    triggerID,
		Status:      domain.EngineTaskStatusCompleted,
		Mode:        domain.TaskModeBackground,
		StartedAt:   &started,
		CompletedAt: &completed,
	}
	insertTask(t, db, tsk)

	h := hookFixture(db, task.NewCompletionNotifier())
	h.OnCompleted(context.Background(), tsk.ID)
	h.Stop()

	mu.Lock()
	body := gotBody
	mu.Unlock()

	var payload task.CompletionPayload
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, int64(1500), payload.DurationMs, "duration must reflect CompletedAt-StartedAt in milliseconds")
}

// ---------------------------------------------------------------------------
// fire() — payload: DurationMs is 0 when StartedAt or CompletedAt is nil
// ---------------------------------------------------------------------------

func TestCompletionHook_Fire_DurationMs_ZeroWhenTimestampsMissing(t *testing.T) {
	db := setupHookTestDB(t)

	var mu sync.Mutex
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		mu.Lock()
		gotBody = buf
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	triggerID := uuid.New().String()
	insertTrigger(t, db, triggerID, "cron", srv.URL, "")

	tsk := &domain.EngineTask{
		ID:          uuid.New(),
		Title:       "no-timestamps",
		AgentName:   "cron-agent",
		Source:      domain.TaskSourceCron,
		SourceID:    triggerID,
		Status:      domain.EngineTaskStatusCompleted,
		Mode:        domain.TaskModeBackground,
		StartedAt:   nil,
		CompletedAt: nil,
	}
	insertTask(t, db, tsk)

	h := hookFixture(db, task.NewCompletionNotifier())
	h.OnCompleted(context.Background(), tsk.ID)
	h.Stop()

	mu.Lock()
	body := gotBody
	mu.Unlock()

	var payload task.CompletionPayload
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, int64(0), payload.DurationMs)
}

// ---------------------------------------------------------------------------
// fire() — payload: Timestamp is RFC3339 format
// ---------------------------------------------------------------------------

func TestCompletionHook_Fire_Timestamp_IsRFC3339(t *testing.T) {
	db := setupHookTestDB(t)

	var mu sync.Mutex
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		mu.Lock()
		gotBody = buf
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	triggerID := uuid.New().String()
	insertTrigger(t, db, triggerID, "webhook", srv.URL, "")

	tsk := &domain.EngineTask{
		ID:        uuid.New(),
		Title:     "ts-test",
		AgentName: "agent",
		Source:    domain.TaskSourceWebhook,
		SourceID:  triggerID,
		Status:    domain.EngineTaskStatusCompleted,
		Mode:      domain.TaskModeBackground,
	}
	insertTask(t, db, tsk)

	before := time.Now().UTC().Truncate(time.Second)
	h := hookFixture(db, task.NewCompletionNotifier())
	h.OnCompleted(context.Background(), tsk.ID)
	h.Stop()
	after := time.Now().UTC().Add(time.Second)

	mu.Lock()
	body := gotBody
	mu.Unlock()

	var payload task.CompletionPayload
	require.NoError(t, json.Unmarshal(body, &payload))

	ts, err := time.Parse(time.RFC3339, payload.Timestamp)
	require.NoError(t, err, "Timestamp must parse as RFC3339")
	assert.True(t, !ts.Before(before) && !ts.After(after),
		"Timestamp %s must be between %s and %s", payload.Timestamp, before, after)
}

// ---------------------------------------------------------------------------
// fire() — headers from OnCompleteHeaders are forwarded to the request
// ---------------------------------------------------------------------------

func TestCompletionHook_Fire_Headers_ForwardedToRequest(t *testing.T) {
	db := setupHookTestDB(t)

	var (
		mu         sync.Mutex
		gotHeaders http.Header
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotHeaders = r.Header.Clone()
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	triggerID := uuid.New().String()
	headers := map[string]string{
		"Authorization": "Bearer secret-token-123",
		"X-Custom-Key":  "my-value",
		"X-Tenant-ID":   "tenant-42",
	}
	headersJSON, err := json.Marshal(headers)
	require.NoError(t, err)
	insertTrigger(t, db, triggerID, "webhook", srv.URL, string(headersJSON))

	tsk := &domain.EngineTask{
		ID:        uuid.New(),
		Title:     "headers-test",
		AgentName: "agent",
		Source:    domain.TaskSourceWebhook,
		SourceID:  triggerID,
		Status:    domain.EngineTaskStatusCompleted,
		Mode:      domain.TaskModeBackground,
	}
	insertTask(t, db, tsk)

	h := hookFixture(db, task.NewCompletionNotifier())
	h.OnCompleted(context.Background(), tsk.ID)
	h.Stop()

	mu.Lock()
	recv := gotHeaders
	mu.Unlock()

	require.NotNil(t, recv, "must have received headers")
	assert.Equal(t, "Bearer secret-token-123", recv.Get("Authorization"))
	assert.Equal(t, "my-value", recv.Get("X-Custom-Key"))
	assert.Equal(t, "tenant-42", recv.Get("X-Tenant-ID"))
	assert.Equal(t, "application/json", recv.Get("Content-Type"))
}

// ---------------------------------------------------------------------------
// fire() — malformed OnCompleteHeaders JSON → no headers sent but webhook fires
// ---------------------------------------------------------------------------

func TestCompletionHook_Fire_MalformedHeaders_WebhookStillFires(t *testing.T) {
	db := setupHookTestDB(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	triggerID := uuid.New().String()
	insertTrigger(t, db, triggerID, "cron", srv.URL, `not-valid-json{{`)

	tsk := &domain.EngineTask{
		ID:        uuid.New(),
		Title:     "bad-headers",
		AgentName: "agent",
		Source:    domain.TaskSourceCron,
		SourceID:  triggerID,
		Status:    domain.EngineTaskStatusCompleted,
		Mode:      domain.TaskModeBackground,
	}
	insertTask(t, db, tsk)

	h := hookFixture(db, task.NewCompletionNotifier())
	h.OnCompleted(context.Background(), tsk.ID)
	h.Stop()

	assert.Equal(t, int32(1), hits.Load(), "webhook must still fire even with malformed headers JSON")
}

// ---------------------------------------------------------------------------
// Concurrent: multiple tasks complete simultaneously — all webhooks fire
// ---------------------------------------------------------------------------

func TestCompletionHook_Concurrent_AllWebhooksFire(t *testing.T) {
	const numTasks = 10
	db := setupHookTestDB(t)

	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	triggerID := uuid.New().String()
	insertTrigger(t, db, triggerID, "webhook", srv.URL, "")

	h := hookFixture(db, task.NewCompletionNotifier())

	for i := 0; i < numTasks; i++ {
		tsk := &domain.EngineTask{
			ID:        uuid.New(),
			Title:     "concurrent-task",
			AgentName: "agent",
			Source:    domain.TaskSourceWebhook,
			SourceID:  triggerID,
			Status:    domain.EngineTaskStatusCompleted,
			Mode:      domain.TaskModeBackground,
		}
		insertTask(t, db, tsk)
		h.OnCompleted(context.Background(), tsk.ID)
	}

	h.Stop()
	assert.Equal(t, int32(numTasks), hits.Load(), "every concurrent task must have its webhook delivered")
}

// ---------------------------------------------------------------------------
// Stop() during in-flight webhooks: no panic, wg drains correctly
// ---------------------------------------------------------------------------

func TestCompletionHook_Stop_WaitsForInflightWebhooks(t *testing.T) {
	db := setupHookTestDB(t)

	// Server that introduces a small delay to allow Stop() to be called mid-flight.
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(20 * time.Millisecond)
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	triggerID := uuid.New().String()
	insertTrigger(t, db, triggerID, "cron", srv.URL, "")

	const numTasks = 3
	h := hookFixture(db, task.NewCompletionNotifier())
	for i := 0; i < numTasks; i++ {
		tsk := &domain.EngineTask{
			ID:        uuid.New(),
			Title:     "inflight",
			AgentName: "agent",
			Source:    domain.TaskSourceCron,
			SourceID:  triggerID,
			Status:    domain.EngineTaskStatusCompleted,
			Mode:      domain.TaskModeBackground,
		}
		insertTask(t, db, tsk)
		h.OnCompleted(context.Background(), tsk.ID)
	}

	// Stop while goroutines are still in-flight.
	stopDone := make(chan struct{})
	go func() {
		h.Stop()
		close(stopDone)
	}()

	select {
	case <-stopDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not return within 5s")
	}

	assert.Equal(t, int32(numTasks), hits.Load(), "all in-flight webhooks must complete before Stop returns")
}

// ---------------------------------------------------------------------------
// After Stop(): new OnCompleted calls are dropped
// ---------------------------------------------------------------------------

func TestCompletionHook_AfterStop_NewCallsDropped(t *testing.T) {
	db := setupHookTestDB(t)

	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	triggerID := uuid.New().String()
	insertTrigger(t, db, triggerID, "webhook", srv.URL, "")

	h := hookFixture(db, task.NewCompletionNotifier())
	h.Stop() // stop immediately, before any tasks

	// These must be dropped (stopped=true).
	for i := 0; i < 5; i++ {
		tsk := &domain.EngineTask{
			ID:        uuid.New(),
			Title:     "post-stop",
			AgentName: "agent",
			Source:    domain.TaskSourceWebhook,
			SourceID:  triggerID,
			Status:    domain.EngineTaskStatusCompleted,
			Mode:      domain.TaskModeBackground,
		}
		insertTask(t, db, tsk)
		h.OnCompleted(context.Background(), tsk.ID)
	}

	// Brief wait to confirm no goroutines sneak through.
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(0), hits.Load(), "no webhook must fire after Stop()")
}

// ---------------------------------------------------------------------------
// Both cron and webhook sources trigger the hook
// ---------------------------------------------------------------------------

func TestCompletionHook_Fire_BothCronAndWebhookSources(t *testing.T) {
	for _, src := range []domain.TaskSource{domain.TaskSourceCron, domain.TaskSourceWebhook} {
		src := src
		t.Run(string(src), func(t *testing.T) {
			db := setupHookTestDB(t)
			var hits atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			triggerID := uuid.New().String()
			insertTrigger(t, db, triggerID, string(src), srv.URL, "")

			tsk := &domain.EngineTask{
				ID:        uuid.New(),
				Title:     "src-test",
				AgentName: "agent",
				Source:    src,
				SourceID:  triggerID,
				Status:    domain.EngineTaskStatusCompleted,
				Mode:      domain.TaskModeBackground,
			}
			insertTask(t, db, tsk)

			h := hookFixture(db, task.NewCompletionNotifier())
			h.OnCompleted(context.Background(), tsk.ID)
			h.Stop()

			assert.Equal(t, int32(1), hits.Load(), "webhook must fire for source %s", src)
		})
	}
}

// ---------------------------------------------------------------------------
// CompletionNotifier: 500 → retries → eventual success on 3rd attempt
// ---------------------------------------------------------------------------

func TestCompletionNotifier_Retry_SuccessOnThirdAttempt(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Use short backoff notifier to keep test fast (50ms, 100ms instead of 1s, 2s).
	n := newFastNotifier(5*time.Second, 3)
	err := n.Notify(context.Background(), srv.URL, nil, task.CompletionPayload{TaskID: "test"})
	require.NoError(t, err)
	assert.Equal(t, int32(3), attempts.Load(), "must have retried until 3rd attempt succeeds")
}

// ---------------------------------------------------------------------------
// CompletionNotifier: 500 × 3 → exhausts retries → returns error
// ---------------------------------------------------------------------------

func TestCompletionNotifier_Retry_ExhaustsOnPersistent500(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	n := newFastNotifier(5*time.Second, 3)
	err := n.Notify(context.Background(), srv.URL, nil, task.CompletionPayload{TaskID: "test"})
	require.Error(t, err)
	assert.Equal(t, int32(3), attempts.Load(), "must have made exactly 3 attempts")
}

// ---------------------------------------------------------------------------
// CompletionNotifier: 502 → retry → success on 3rd attempt
// ---------------------------------------------------------------------------

func TestCompletionNotifier_Retry_502ThenSuccess(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := newFastNotifier(5*time.Second, 3)
	err := n.Notify(context.Background(), srv.URL, nil, task.CompletionPayload{TaskID: "test"})
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// CompletionNotifier: 400 → no retry (non-retryable), immediate fail
// ---------------------------------------------------------------------------

func TestCompletionNotifier_NoRetry_On400(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	n := newFastNotifier(5*time.Second, 3)
	err := n.Notify(context.Background(), srv.URL, nil, task.CompletionPayload{TaskID: "test"})
	require.Error(t, err)
	assert.Equal(t, int32(1), attempts.Load(), "400 must not be retried")
}

// ---------------------------------------------------------------------------
// CompletionNotifier: 401/403/404 → no retry
// ---------------------------------------------------------------------------

func TestCompletionNotifier_NoRetry_On4xxCodes(t *testing.T) {
	codes := []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound}
	for _, code := range codes {
		code := code
		t.Run(http.StatusText(code), func(t *testing.T) {
			var attempts atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts.Add(1)
				w.WriteHeader(code)
			}))
			defer srv.Close()

			n := newFastNotifier(5*time.Second, 3)
			err := n.Notify(context.Background(), srv.URL, nil, task.CompletionPayload{TaskID: "test"})
			require.Error(t, err)
			assert.Equal(t, int32(1), attempts.Load(), "status %d must not be retried", code)
		})
	}
}

// ---------------------------------------------------------------------------
// CompletionNotifier: 200 on first attempt → success, no retries
// ---------------------------------------------------------------------------

func TestCompletionNotifier_Success_On200(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := newFastNotifier(5*time.Second, 3)
	err := n.Notify(context.Background(), srv.URL, nil, task.CompletionPayload{TaskID: "test"})
	require.NoError(t, err)
	assert.Equal(t, int32(1), attempts.Load(), "must succeed on first attempt with 200")
}

// ---------------------------------------------------------------------------
// CompletionNotifier: context cancelled during retry → returns error fast
// ---------------------------------------------------------------------------

func TestCompletionNotifier_ContextCancelled_DuringRetry(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately after first call.
	n := newFastNotifier(5*time.Second, 3)

	// Run in goroutine and cancel context mid-retry.
	errCh := make(chan error, 1)
	go func() {
		errCh <- n.Notify(ctx, srv.URL, nil, task.CompletionPayload{TaskID: "test"})
	}()

	// Give time for first attempt, then cancel.
	time.Sleep(20 * time.Millisecond)
	cancel()

	err := <-errCh
	require.Error(t, err, "must return error when context is cancelled")
}

// ---------------------------------------------------------------------------
// CompletionNotifier: webhook server slow (exceeds client timeout) → error
// ---------------------------------------------------------------------------

func TestCompletionNotifier_Timeout_SlowServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than the client timeout.
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Client timeout of 50ms — much shorter than server delay.
	n := newFastNotifier(50*time.Millisecond, 1)
	err := n.Notify(context.Background(), srv.URL, nil, task.CompletionPayload{TaskID: "timeout-test"})
	require.Error(t, err, "must return error when server exceeds client timeout")
}

// ---------------------------------------------------------------------------
// CompletionNotifier: headers are forwarded correctly
// ---------------------------------------------------------------------------

func TestCompletionNotifier_Headers_ForwardedToRequest(t *testing.T) {
	var (
		mu         sync.Mutex
		gotHeaders http.Header
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotHeaders = r.Header.Clone()
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := newFastNotifier(5*time.Second, 1)
	headers := map[string]string{
		"Authorization": "Basic dXNlcjpwYXNz",
		"X-Api-Key":     "key-abc",
	}
	err := n.Notify(context.Background(), srv.URL, headers, task.CompletionPayload{TaskID: "h-test"})
	require.NoError(t, err)

	mu.Lock()
	recv := gotHeaders
	mu.Unlock()

	assert.Equal(t, "Basic dXNlcjpwYXNz", recv.Get("Authorization"))
	assert.Equal(t, "key-abc", recv.Get("X-Api-Key"))
	assert.Equal(t, "application/json", recv.Get("Content-Type"))
}

// ---------------------------------------------------------------------------
// CompletionNotifier: payload is valid JSON in request body
// ---------------------------------------------------------------------------

func TestCompletionNotifier_Payload_IsValidJSON(t *testing.T) {
	var (
		mu      sync.Mutex
		gotBody []byte
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		mu.Lock()
		gotBody = buf[:n]
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := newFastNotifier(5*time.Second, 1)
	sent := task.CompletionPayload{
		TaskID:     uuid.New().String(),
		Status:     "completed",
		Result:     "all done",
		DurationMs: 1234,
		TriggerID:  "trig-1",
		AgentName:  "my-agent",
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
	err := n.Notify(context.Background(), srv.URL, nil, sent)
	require.NoError(t, err)

	mu.Lock()
	body := gotBody
	mu.Unlock()

	var received task.CompletionPayload
	require.NoError(t, json.Unmarshal(body, &received))
	assert.Equal(t, sent.TaskID, received.TaskID)
	assert.Equal(t, sent.Status, received.Status)
	assert.Equal(t, sent.Result, received.Result)
	assert.Equal(t, sent.DurationMs, received.DurationMs)
	assert.Equal(t, sent.TriggerID, received.TriggerID)
	assert.Equal(t, sent.AgentName, received.AgentName)
}
