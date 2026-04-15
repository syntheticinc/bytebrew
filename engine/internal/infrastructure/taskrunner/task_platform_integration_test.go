package taskrunner_test

// End-to-end integration test for the cron-driven autonomous task platform.
//
// Path under test (production wiring; only the chat model is mocked):
//
//   triggerTaskCreator.CreateFromTrigger     (simulated cron tick)
//     -> EngineTaskManagerAdapter.CreateTask (real, GORM/SQLite)
//     -> GORMTriggerRepository.MarkFired     (real — stamps last_fired_at)
//     -> TaskWorker.Submit                   (real, channel queue)
//     -> TaskExecutor.Execute                (real)
//          -> sessionRegistry.CreateSession  (fake — emits ANSWER + STOPPED)
//          -> waitForCompletion              (real event loop)
//     -> EngineTaskManagerAdapter.CompleteTask
//
// V2 (§4.2): the on-complete webhook feature is removed — terminal
// transitions do not fan out to external URLs. last_fired_at is the only
// observable side-effect the test asserts on the trigger row.
//
// The cron timer itself is intentionally bypassed (NewCronScheduler standard cron does
// not support sub-minute schedules and we do not want to wait 60s in CI). Instead we
// call CreateFromTrigger directly — the scheduler's only job is timing, and the
// "cron tick → CreateFromTrigger" arrow is a tiny `cron.AddFunc` callback already
// covered by cron/v3 unit tests upstream.

import (
	"context"
	"sync"
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
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/taskrunner"
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

	// V2 (§4.1): triggers carry their type-specific config inside a single
	// jsonb column. SQLite has no jsonb — TEXT is a faithful stand-in
	// because the Scan/Value pair on TriggerConfig round-trips via
	// encoding/json.
	require.NoError(t, db.Exec(`
CREATE TABLE triggers (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	title TEXT NOT NULL,
	agent_id TEXT,
	schema_id TEXT,
	description TEXT,
	enabled INTEGER NOT NULL DEFAULT 1,
	config TEXT NOT NULL DEFAULT '{}',
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

// seedAgentAndTrigger inserts an agent + a cron trigger.
// Returns the trigger id (caller uses it to assert last_fired_at is stamped).
func seedAgentAndTrigger(t *testing.T, db *gorm.DB, agentName string) string {
	t.Helper()
	agentID := uuid.NewString()
	require.NoError(t, db.Exec(
		`INSERT INTO agents (id, name, system_prompt, lifecycle, tool_execution, max_steps, max_context_size, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		agentID, agentName, "you are a test agent", "persistent", "sequential", 1, 4000, time.Now(), time.Now(),
	).Error)

	triggerID := uuid.NewString()
	cfg, _ := models.TriggerConfig{Schedule: "0 * * * *"}.Value()
	require.NoError(t, db.Exec(
		`INSERT INTO triggers (id, type, title, agent_id, enabled, config, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		triggerID, models.TriggerTypeCron, "Hourly health check", agentID, 1, cfg, time.Now(), time.Now(),
	).Error)

	return triggerID
}

// startPlatform builds the production object graph (adapter + worker + executor +
// trigger creator) and ensures Stop is wired to t.Cleanup.
type platform struct {
	db          *gorm.DB
	taskRepo    *configrepo.GORMTaskRepository
	triggerRepo *configrepo.GORMTriggerRepository
	adapter     *taskrunner.EngineTaskManagerAdapter
	worker      *task.TaskWorker
	creator     task.TaskCreator
	registry    *fakeSessionRegistry
}

func startPlatform(t *testing.T, registry *fakeSessionRegistry) *platform {
	t.Helper()
	db := setupPlatformDB(t)
	taskRepo := configrepo.NewGORMTaskRepository(db)
	triggerRepo := configrepo.NewGORMTriggerRepository(db)

	adapter := taskrunner.NewEngineTaskManagerAdapter(taskRepo)

	executor := taskrunner.NewTaskExecutor(adapter, registry, registry, 5*time.Second)
	worker := taskrunner.StartBackgroundWorker(executor, 2)
	require.NotNil(t, worker, "worker must start with a non-nil executor")

	creator := taskrunner.NewTriggerTaskCreator(adapter, worker, triggerRepo)

	t.Cleanup(func() {
		worker.Stop()
	})

	return &platform{
		db:          db,
		taskRepo:    taskRepo,
		triggerRepo: triggerRepo,
		adapter:     adapter,
		worker:      worker,
		creator:     creator,
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

// waitForTaskStatus polls the task row until it reaches the expected status or times out.
// We poll instead of sleep because the worker -> executor chain is asynchronous and we
// need a deterministic synchronisation point on DB state.
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

// TestTaskPlatform_CronFire_MarksLastFiredAt verifies the full happy path:
// simulated cron tick -> task created in DB -> MarkFired stamps last_fired_at
// -> worker picks it up -> executor runs the (fake) agent -> task transitions
// to completed.
func TestTaskPlatform_CronFire_MarksLastFiredAt(t *testing.T) {
	registry := newFakeSessionRegistry("health probes passed", behaviorAnswer)
	p := startPlatform(t, registry)

	const agentName = "health-agent"
	triggerID := seedAgentAndTrigger(t, p.db, agentName)

	before := time.Now().UTC().Add(-time.Second)
	taskID := p.fireCron(t, triggerID, agentName)

	// Task reaches "completed" in the DB.
	completed := waitForTaskStatus(t, p.taskRepo, taskID, domain.EngineTaskStatusCompleted, 5*time.Second)
	assert.NotNil(t, completed.CompletedAt, "CompletedAt must be set on terminal transition")
	assert.Equal(t, "health probes passed", completed.Result)
	assert.Equal(t, domain.TaskSourceCron, completed.Source)
	assert.Equal(t, triggerID, completed.SourceID)

	// §4.1: every cron fire stamps last_fired_at. The window must include
	// the moment we invoked the creator.
	trigger, err := p.triggerRepo.GetByID(context.Background(), triggerID)
	require.NoError(t, err)
	require.NotNil(t, trigger.LastFiredAt, "last_fired_at must be set after cron fire")
	assert.True(t, !trigger.LastFiredAt.UTC().Before(before) && !trigger.LastFiredAt.UTC().After(time.Now().UTC().Add(time.Second)),
		"last_fired_at %s must fall in the test window", trigger.LastFiredAt)
}

// TestTaskPlatform_AutonomousNeedsInput_AutoFails verifies that when an agent
// stops without producing an answer (the closest analogue of "needs_input" for
// V2 cron tasks), the task is auto-failed. MarkFired still stamps because the
// trigger did fire — the agent's failure is orthogonal.
//
// V2 has no ProvideInput HTTP path, so any cron task that ends up unable to
// answer must reach a terminal failed state so operators see it in the UI.
func TestTaskPlatform_AutonomousNeedsInput_AutoFails(t *testing.T) {
	registry := newFakeSessionRegistry("", behaviorNeedsInput)
	p := startPlatform(t, registry)

	const agentName = "stuck-agent"
	triggerID := seedAgentAndTrigger(t, p.db, agentName)

	taskID := p.fireCron(t, triggerID, agentName)

	// Task ends up failed (executor.markFailed path).
	failed := waitForTaskStatus(t, p.taskRepo, taskID, domain.EngineTaskStatusFailed, 5*time.Second)
	assert.NotEmpty(t, failed.Error, "Error column must carry a human-readable reason")
	assert.Contains(t, failed.Error, "without producing a final answer",
		"failure reason must explain why the autonomous run could not finish")

	// last_fired_at still stamped — the trigger did fire, regardless of the
	// agent's subsequent failure.
	trigger, err := p.triggerRepo.GetByID(context.Background(), triggerID)
	require.NoError(t, err)
	require.NotNil(t, trigger.LastFiredAt, "last_fired_at must still be set when the run fails")
}
