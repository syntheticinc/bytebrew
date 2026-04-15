package taskrunner_test

// End-to-end integration test for the cron-driven autonomous task platform.
//
// Path under test (production wiring; only the chat model and webhook server are mocked):
//
//   triggerTaskCreator.CreateFromTrigger     (simulated cron tick)
//     -> EngineTaskManagerAdapter.CreateTask (real, GORM/SQLite)
//     -> TaskWorker.Submit                   (real, channel queue)
//     -> TaskExecutor.Execute                (real)
//          -> sessionRegistry.CreateSession  (fake — emits ANSWER + STOPPED)
//          -> waitForCompletion              (real event loop)
//     -> EngineTaskManagerAdapter.CompleteTask
//          -> TaskCompletionHook.OnCompleted (real)
//               -> CompletionNotifier.Notify (real http.Client)
//                    -> POST mock webhook server (real httptest.NewServer)
//
// The cron timer itself is intentionally bypassed (NewCronScheduler standard cron does
// not support sub-minute schedules and we do not want to wait 60s in CI). Instead we
// call CreateFromTrigger directly — the scheduler's only job is timing, and the
// "cron tick → CreateFromTrigger" arrow is a tiny `cron.AddFunc` callback already
// covered by cron/v3 unit tests upstream.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

	pb "github.com/syntheticinc/bytebrew/engine/api/proto/gen"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/configrepo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/taskrunner"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/engine/internal/service/task"
)

// --- Fake session machinery ---
//
// The TaskExecutor talks to a session registry + a session processor.  In production
// these are wired to the Eino ReAct agent. For the integration test we replace them
// with a fake that, when StartProcessing is called, pushes a fixed (ANSWER, STOPPED)
// pair onto the subscribe channel for that session — exactly what a real agent run
// would emit on success.
//
// behavior controls what events the fake emits, so the same registry can drive both
// the happy-path and the "needs_input" autonomous-failure path.

type fakeBehavior int

const (
	behaviorAnswer     fakeBehavior = iota // emits ANSWER + STOPPED
	behaviorNeedsInput                     // emits no answer; executor will mark the task needs_input via SetTaskStatus
)

// fakeSessionRegistry implements both sessionRegistryForExecutor and sessionProcessorForExecutor.
// Each session id gets its own buffered channel so multiple parallel runs do not cross-talk.
type fakeSessionRegistry struct {
	mu       sync.Mutex
	channels map[string]chan *pb.SessionEvent
	answer   string
	behavior fakeBehavior

	// hook fired right before StartProcessing emits — gives tests a place to mutate
	// task state (e.g. transition to needs_input) before the executor exits its loop.
	preEmit func(sessionID string)
}

func newFakeSessionRegistry(answer string, b fakeBehavior) *fakeSessionRegistry {
	return &fakeSessionRegistry{
		channels: make(map[string]chan *pb.SessionEvent),
		answer:   answer,
		behavior: b,
	}
}

func (f *fakeSessionRegistry) channel(sessionID string) chan *pb.SessionEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	ch, ok := f.channels[sessionID]
	if !ok {
		ch = make(chan *pb.SessionEvent, 8)
		f.channels[sessionID] = ch
	}
	return ch
}

func (f *fakeSessionRegistry) CreateSession(sessionID, _, _, _, _, _ string) {
	_ = f.channel(sessionID)
}

func (f *fakeSessionRegistry) Subscribe(sessionID string) (<-chan *pb.SessionEvent, func()) {
	return f.channel(sessionID), func() {}
}

func (f *fakeSessionRegistry) EnqueueMessage(_ string, _ string) error { return nil }

func (f *fakeSessionRegistry) RemoveSession(sessionID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if ch, ok := f.channels[sessionID]; ok {
		close(ch)
		delete(f.channels, sessionID)
	}
}

// StartProcessing simulates the agent producing output.
// Done in a goroutine so the executor can subscribe and receive the events in its loop.
func (f *fakeSessionRegistry) StartProcessing(_ context.Context, sessionID string) {
	ch := f.channel(sessionID)
	if f.preEmit != nil {
		f.preEmit(sessionID)
	}
	go func() {
		switch f.behavior {
		case behaviorAnswer:
			ch <- &pb.SessionEvent{Type: pb.SessionEventType_SESSION_EVENT_ANSWER, Content: f.answer}
			ch <- &pb.SessionEvent{Type: pb.SessionEventType_SESSION_EVENT_PROCESSING_STOPPED}
		case behaviorNeedsInput:
			// Emit only STOPPED with no prior ANSWER — the executor's waitForCompletion
			// returns "agent stopped without producing a final answer" which makes the
			// run fail. That is the de-facto autonomous-needs-input outcome with the
			// current TaskExecutor, since there is no ProvideInput path in V2.
			ch <- &pb.SessionEvent{Type: pb.SessionEventType_SESSION_EVENT_PROCESSING_STOPPED}
		}
	}()
}

func (f *fakeSessionRegistry) StopProcessing(_ string) {}

// --- Helpers ---

// setupPlatformDB returns an in-memory SQLite DB with the tasks + triggers + agents tables
// created via portable DDL (the production GORM tags use Postgres-only `gen_random_uuid()`
// defaults that SQLite rejects on AutoMigrate).
//
// CRITICAL: we cap the connection pool at 1. With pool > 1 and `:memory:`, every new
// SQLite connection gets its OWN empty database — so the GORM schema is invisible to
// the goroutines spun up by the worker / completion hook (each grabs a fresh
// connection from the pool). Using a shared single connection makes the DB behave
// like a single in-process database, which is what every test here needs.
func setupPlatformDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

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

	// Minimal agents table — only Name + ID are read by triggerTaskMetadata via Preload("Agent").
	require.NoError(t, db.Exec(`
CREATE TABLE agents (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	model_id TEXT,
	system_prompt TEXT,
	lifecycle TEXT,
	tool_execution TEXT,
	max_steps INTEGER,
	max_context_size INTEGER,
	created_at DATETIME,
	updated_at DATETIME
)`).Error)

	return db
}

// seedAgentAndTrigger inserts an agent + a cron trigger with the given on-complete webhook URL.
// Returns the trigger id (caller uses it to assert CompletionPayload.TriggerID).
func seedAgentAndTrigger(t *testing.T, db *gorm.DB, agentName, webhookURL string) string {
	t.Helper()
	agentID := uuid.NewString()
	require.NoError(t, db.Exec(
		`INSERT INTO agents (id, name, system_prompt, lifecycle, tool_execution, max_steps, max_context_size, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		agentID, agentName, "you are a test agent", "persistent", "sequential", 1, 4000, time.Now(), time.Now(),
	).Error)

	triggerID := uuid.NewString()
	require.NoError(t, db.Exec(
		`INSERT INTO triggers (id, type, title, agent_id, schedule, enabled, on_complete_url, on_complete_headers, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		triggerID, models.TriggerTypeCron, "Hourly health check", agentID, "0 * * * *", 1, webhookURL, "", time.Now(), time.Now(),
	).Error)

	return triggerID
}

// receivedWebhook captures one POST from the executor for assertions.
type receivedWebhook struct {
	statusCode int
	payload    task.CompletionPayload
	headers    http.Header
}

// newWebhookServer returns a server that publishes each incoming POST onto the
// returned channel. The handler can be customised to fail on the first N requests
// (used by the retry test). respond is called for every request with the attempt
// counter (1-indexed); it must write any non-2xx status it wants the notifier to see.
func newWebhookServer(t *testing.T, respond func(attempt int, w http.ResponseWriter)) (*httptest.Server, <-chan receivedWebhook) {
	t.Helper()
	out := make(chan receivedWebhook, 8)
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := int(attempts.Add(1))
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var p task.CompletionPayload
		_ = json.Unmarshal(body, &p)
		// Snapshot headers before responding (response writes invalidate the request).
		hdr := r.Header.Clone()
		if respond != nil {
			respond(attempt, w)
			// Drain a synthetic 2xx if the responder didn't write one explicitly — but
			// httptest defaults to 200 if the handler returns without writing, so we
			// don't need anything here. We just need to capture the response status the
			// notifier will observe.
		}
		// We can't read the response status the notifier sees from here directly;
		// what matters for assertions is the payload + headers + attempt count, and
		// the test-side respond() determines the http response.
		out <- receivedWebhook{statusCode: http.StatusOK, payload: p, headers: hdr}
	}))
	t.Cleanup(srv.Close)
	return srv, out
}

// startPlatform builds the production object graph (adapter + worker + executor + hook +
// notifier + trigger creator) and ensures Stop is wired to t.Cleanup.
type platform struct {
	db          *gorm.DB
	taskRepo    *configrepo.GORMTaskRepository
	triggerRepo *configrepo.GORMTriggerRepository
	adapter     *taskrunner.EngineTaskManagerAdapter
	worker      *task.TaskWorker
	creator     task.TaskCreator
	hook        *taskrunner.TaskCompletionHook
	registry    *fakeSessionRegistry
}

func startPlatform(t *testing.T, registry *fakeSessionRegistry) *platform {
	t.Helper()
	db := setupPlatformDB(t)
	taskRepo := configrepo.NewGORMTaskRepository(db)
	triggerRepo := configrepo.NewGORMTriggerRepository(db)

	adapter := taskrunner.NewEngineTaskManagerAdapter(taskRepo)

	notifier := task.NewCompletionNotifier()
	hook := taskrunner.NewTaskCompletionHook(taskRepo, triggerRepo, notifier)
	adapter.SetCompletionHook(hook)

	executor := taskrunner.NewTaskExecutor(adapter, registry, registry, 5*time.Second)
	worker := taskrunner.StartBackgroundWorker(executor, 2)
	require.NotNil(t, worker, "worker must start with a non-nil executor")

	creator := taskrunner.NewTriggerTaskCreator(adapter, worker)

	t.Cleanup(func() {
		worker.Stop()
		hook.Stop()
	})

	return &platform{
		db:          db,
		taskRepo:    taskRepo,
		triggerRepo: triggerRepo,
		adapter:     adapter,
		worker:      worker,
		creator:     creator,
		hook:        hook,
		registry:    registry,
	}
}

// fireCron simulates one cron tick. Equivalent to what task.CronScheduler does inside
// its cron.AddFunc callback — except synchronous so the test can wait deterministically.
func (p *platform) fireCron(t *testing.T, triggerID, agentName string) uuid.UUID {
	t.Helper()
	taskID, err := p.creator.CreateFromTrigger(context.Background(), task.TriggerTaskParams{
		Title:       "Hourly health check",
		Description: "Run health probes",
		AgentName:   agentName,
		Source:      string(domain.TaskSourceCron),
		SourceID:    triggerID,
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, taskID)
	return taskID
}

// waitForWebhook returns the first delivery on ch, or fails the test on timeout.
func waitForWebhook(t *testing.T, ch <-chan receivedWebhook, timeout time.Duration) receivedWebhook {
	t.Helper()
	select {
	case w := <-ch:
		return w
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for webhook delivery after %s", timeout)
		return receivedWebhook{}
	}
}

// waitForTaskStatus polls the task row until it reaches the expected status or times out.
// We poll instead of sleep because the worker -> executor -> hook chain is asynchronous
// and we need a deterministic synchronisation point on DB state.
func waitForTaskStatus(t *testing.T, repo *configrepo.GORMTaskRepository, taskID uuid.UUID, want domain.EngineTaskStatus, timeout time.Duration) *domain.EngineTask {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		got, err := repo.GetByID(context.Background(), taskID)
		if err == nil && got != nil && got.Status == want {
			return got
		}
		time.Sleep(20 * time.Millisecond)
	}
	got, _ := repo.GetByID(context.Background(), taskID)
	if got != nil {
		t.Fatalf("task %s never reached status %s (last status=%s, error=%q)", taskID, want, got.Status, got.Error)
	}
	t.Fatalf("task %s never reached status %s (task not found)", taskID, want)
	return nil
}

// --- Tests ---

// TestTaskPlatform_CronToWebhookE2E verifies the full happy path:
// simulated cron tick -> task created in DB -> worker picks it up -> executor runs
// the (fake) agent -> task transitions to completed -> webhook fires with payload.
func TestTaskPlatform_CronToWebhookE2E(t *testing.T) {
	srv, hookCh := newWebhookServer(t, func(_ int, w http.ResponseWriter) {
		w.WriteHeader(http.StatusOK)
	})

	registry := newFakeSessionRegistry("health probes passed", behaviorAnswer)
	p := startPlatform(t, registry)

	const agentName = "health-agent"
	triggerID := seedAgentAndTrigger(t, p.db, agentName, srv.URL)

	taskID := p.fireCron(t, triggerID, agentName)

	// 1. Task reaches "completed" in the DB.
	completed := waitForTaskStatus(t, p.taskRepo, taskID, domain.EngineTaskStatusCompleted, 5*time.Second)
	assert.NotNil(t, completed.CompletedAt, "CompletedAt must be set on terminal transition")
	assert.Equal(t, "health probes passed", completed.Result)
	assert.Equal(t, domain.TaskSourceCron, completed.Source)
	assert.Equal(t, triggerID, completed.SourceID)

	// 2. Webhook receives the completion payload.
	got := waitForWebhook(t, hookCh, 5*time.Second)

	// Verify payload structure end-to-end.
	assert.Equal(t, taskID.String(), got.payload.TaskID, "task id round-trips as canonical UUID string")
	_, err := uuid.Parse(got.payload.TaskID)
	assert.NoError(t, err, "TaskID is a valid UUID")
	assert.Equal(t, "completed", got.payload.Status)
	assert.Equal(t, "health probes passed", got.payload.Result)
	assert.Equal(t, triggerID, got.payload.TriggerID)
	assert.Equal(t, agentName, got.payload.AgentName)
	assert.NotEmpty(t, got.payload.Timestamp)
	assert.Equal(t, "application/json", got.headers.Get("Content-Type"))
}

// TestTaskPlatform_AutonomousNeedsInput_AutoFails verifies that when an agent
// stops without producing an answer (the closest analogue of "needs_input" for
// V2 cron tasks), the task is auto-failed and the failure webhook fires.
//
// V2 has no ProvideInput HTTP path, so any cron task that ends up unable to
// answer must reach a terminal failed state so operators see it in the UI.
func TestTaskPlatform_AutonomousNeedsInput_AutoFails(t *testing.T) {
	srv, hookCh := newWebhookServer(t, func(_ int, w http.ResponseWriter) {
		w.WriteHeader(http.StatusOK)
	})

	registry := newFakeSessionRegistry("", behaviorNeedsInput)
	p := startPlatform(t, registry)

	const agentName = "stuck-agent"
	triggerID := seedAgentAndTrigger(t, p.db, agentName, srv.URL)

	taskID := p.fireCron(t, triggerID, agentName)

	// Task ends up failed (executor.markFailed path).
	failed := waitForTaskStatus(t, p.taskRepo, taskID, domain.EngineTaskStatusFailed, 5*time.Second)
	assert.NotEmpty(t, failed.Error, "Error column must carry a human-readable reason")
	assert.Contains(t, failed.Error, "without producing a final answer",
		"failure reason must explain why the autonomous run could not finish")

	got := waitForWebhook(t, hookCh, 5*time.Second)
	assert.Equal(t, "failed", got.payload.Status, "failed runs must still notify the webhook")
	assert.Equal(t, triggerID, got.payload.TriggerID)
	assert.Equal(t, taskID.String(), got.payload.TaskID)
}

// TestTaskPlatform_WebhookRetry_OnTransient500 verifies that a transient 5xx on
// the first delivery attempt is retried and the webhook is eventually delivered.
//
// Notifier policy: 3 attempts total with 1s, 2s backoff. We force a 500 on attempt 1
// and a 200 on attempt 2 — total wall clock ~1s + processing.
func TestTaskPlatform_WebhookRetry_OnTransient500(t *testing.T) {
	var attemptStatuses sync.Map // attempt -> http status served
	srv, hookCh := newWebhookServer(t, func(attempt int, w http.ResponseWriter) {
		if attempt == 1 {
			attemptStatuses.Store(attempt, http.StatusInternalServerError)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		attemptStatuses.Store(attempt, http.StatusOK)
		w.WriteHeader(http.StatusOK)
	})

	registry := newFakeSessionRegistry("retry-result", behaviorAnswer)
	p := startPlatform(t, registry)

	const agentName = "retry-agent"
	triggerID := seedAgentAndTrigger(t, p.db, agentName, srv.URL)

	taskID := p.fireCron(t, triggerID, agentName)
	waitForTaskStatus(t, p.taskRepo, taskID, domain.EngineTaskStatusCompleted, 5*time.Second)

	// Drain at least 2 attempts: the failed one then the successful one.
	first := waitForWebhook(t, hookCh, 5*time.Second)
	second := waitForWebhook(t, hookCh, 5*time.Second)

	// Both attempts target the same task with status=completed; the notifier just retries
	// the same payload.
	for _, w := range []receivedWebhook{first, second} {
		assert.Equal(t, "completed", w.payload.Status)
		assert.Equal(t, taskID.String(), w.payload.TaskID)
		assert.Equal(t, triggerID, w.payload.TriggerID)
	}

	// Defensive: ensure attempt 1 was a 500 (not a flaky reordering).
	v1, ok := attemptStatuses.Load(1)
	require.True(t, ok)
	assert.Equal(t, http.StatusInternalServerError, v1)
	v2, ok := attemptStatuses.Load(2)
	require.True(t, ok)
	assert.Equal(t, http.StatusOK, v2)
}

// fmtDuration is a tiny test helper to keep failure messages readable.
// (Kept here in case future assertions need it; harmless if unused.)
var _ = fmt.Sprintf
