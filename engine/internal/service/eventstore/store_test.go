package eventstore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"testing"

	_ "github.com/glebarez/go-sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/syntheticinc/bytebrew/engine/api/proto/gen"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=journal_mode(WAL)")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestStore_New(t *testing.T) {
	db := newTestDB(t)
	store, err := New(db)
	require.NoError(t, err)
	require.NotNil(t, store)
}

func TestStore_AppendAndGetAll(t *testing.T) {
	db := newTestDB(t)
	store, err := New(db)
	require.NoError(t, err)

	event := &pb.SessionEvent{
		SessionId: "sess-1",
		Type:      pb.SessionEventType_SESSION_EVENT_ANSWER,
		Content:   "Hello",
		AgentId:   "supervisor",
	}
	jsonData := map[string]interface{}{
		"type":    "MessageCompleted",
		"content": "Hello",
	}

	id, err := store.Append("sess-1", "answer", event, jsonData)
	require.NoError(t, err)
	assert.Equal(t, int64(1), id)

	events, err := store.GetAll("sess-1")
	require.NoError(t, err)
	require.Len(t, events, 1)

	assert.Equal(t, int64(1), events[0].ID)
	assert.Equal(t, "sess-1", events[0].SessionID)
	assert.Equal(t, "answer", events[0].EventType)
	assert.Equal(t, "Hello", events[0].Proto.GetContent())
	assert.Equal(t, "1", events[0].Proto.GetEventId()) // Set from auto-increment ID
	assert.Equal(t, "MessageCompleted", events[0].JSON["type"])
}

func TestStore_GetAfter(t *testing.T) {
	db := newTestDB(t)
	store, err := New(db)
	require.NoError(t, err)

	// Insert 3 events
	for i := 0; i < 3; i++ {
		_, err := store.Append("sess-1", "answer", &pb.SessionEvent{
			Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
			Content: "msg",
		}, map[string]interface{}{"type": "MessageCompleted"})
		require.NoError(t, err)
	}

	// GetAfter(0) returns all
	events, err := store.GetAfter("sess-1", 0)
	require.NoError(t, err)
	assert.Len(t, events, 3)

	// GetAfter(1) returns events 2 and 3
	events, err = store.GetAfter("sess-1", 1)
	require.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, int64(2), events[0].ID)
	assert.Equal(t, int64(3), events[1].ID)

	// GetAfter(3) returns nothing (no events after last)
	events, err = store.GetAfter("sess-1", 3)
	require.NoError(t, err)
	assert.Len(t, events, 0)
}

func TestStore_GetAfterUnknownID(t *testing.T) {
	db := newTestDB(t)
	store, err := New(db)
	require.NoError(t, err)

	_, err = store.Append("sess-1", "answer", &pb.SessionEvent{
		Type: pb.SessionEventType_SESSION_EVENT_ANSWER,
	}, map[string]interface{}{"type": "MessageCompleted"})
	require.NoError(t, err)

	// Unknown lastEventID — returns all events (safe fallback)
	events, err := store.GetAfter("sess-1", 999)
	require.NoError(t, err)
	assert.Len(t, events, 1)
}

func TestStore_GetAllEmptySession(t *testing.T) {
	db := newTestDB(t)
	store, err := New(db)
	require.NoError(t, err)

	events, err := store.GetAll("nonexistent")
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestStore_SessionIsolation(t *testing.T) {
	db := newTestDB(t)
	store, err := New(db)
	require.NoError(t, err)

	_, err = store.Append("sess-1", "answer", &pb.SessionEvent{Type: pb.SessionEventType_SESSION_EVENT_ANSWER}, map[string]interface{}{"type": "A"})
	require.NoError(t, err)
	_, err = store.Append("sess-2", "answer", &pb.SessionEvent{Type: pb.SessionEventType_SESSION_EVENT_ANSWER}, map[string]interface{}{"type": "B"})
	require.NoError(t, err)

	events1, err := store.GetAll("sess-1")
	require.NoError(t, err)
	assert.Len(t, events1, 1)

	events2, err := store.GetAll("sess-2")
	require.NoError(t, err)
	assert.Len(t, events2, 1)
}

func TestStore_CleanupSession(t *testing.T) {
	db := newTestDB(t)
	store, err := New(db)
	require.NoError(t, err)

	_, err = store.Append("sess-1", "answer", &pb.SessionEvent{Type: pb.SessionEventType_SESSION_EVENT_ANSWER}, map[string]interface{}{"type": "A"})
	require.NoError(t, err)
	_, err = store.Append("sess-2", "answer", &pb.SessionEvent{Type: pb.SessionEventType_SESSION_EVENT_ANSWER}, map[string]interface{}{"type": "B"})
	require.NoError(t, err)

	err = store.CleanupSession("sess-1")
	require.NoError(t, err)

	events1, err := store.GetAll("sess-1")
	require.NoError(t, err)
	assert.Empty(t, events1)

	// sess-2 is untouched
	events2, err := store.GetAll("sess-2")
	require.NoError(t, err)
	assert.Len(t, events2, 1)
}

func TestStore_MonotonicIDs(t *testing.T) {
	db := newTestDB(t)
	store, err := New(db)
	require.NoError(t, err)

	id1, err := store.Append("s", "a", &pb.SessionEvent{Type: pb.SessionEventType_SESSION_EVENT_ANSWER}, map[string]interface{}{})
	require.NoError(t, err)
	id2, err := store.Append("s", "a", &pb.SessionEvent{Type: pb.SessionEventType_SESSION_EVENT_ANSWER}, map[string]interface{}{})
	require.NoError(t, err)
	id3, err := store.Append("s", "a", &pb.SessionEvent{Type: pb.SessionEventType_SESSION_EVENT_ANSWER}, map[string]interface{}{})
	require.NoError(t, err)

	assert.True(t, id1 < id2)
	assert.True(t, id2 < id3)
}

// TC-ES-07: Concurrent Append + GetAfter — 10 goroutines append simultaneously,
// all events persisted with monotonically increasing IDs.
func TestStore_ConcurrentAppend(t *testing.T) {
	// Use file-based DB for concurrent access (in-memory with cache=shared
	// can lose schema across connections opened by database/sql pool).
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "concurrent.db")
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	store, err := New(db)
	require.NoError(t, err)

	const numGoroutines = 10
	var wg sync.WaitGroup
	ids := make([]int64, numGoroutines)
	errs := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			id, appendErr := store.Append("sess-concurrent", "answer", &pb.SessionEvent{
				Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
				Content: fmt.Sprintf("msg-%d", idx),
			}, map[string]interface{}{"type": "MessageCompleted", "idx": idx})
			ids[idx] = id
			errs[idx] = appendErr
		}(i)
	}
	wg.Wait()

	// All appends should succeed
	for i, appendErr := range errs {
		require.NoError(t, appendErr, "goroutine %d should not error", i)
	}

	// GetAll returns all 10 events
	events, err := store.GetAll("sess-concurrent")
	require.NoError(t, err)
	assert.Len(t, events, numGoroutines)

	// IDs should be monotonically increasing
	sort.Slice(ids, func(a, b int) bool { return ids[a] < ids[b] })
	for i := 1; i < len(ids); i++ {
		assert.True(t, ids[i-1] < ids[i], "IDs should be strictly increasing: %d < %d", ids[i-1], ids[i])
	}
}

// TC-ES-08: Events survive close + reopen DB — persistence across DB connections.
func TestStore_PersistenceAcrossReopen(t *testing.T) {
	// Use a temp file DB, not :memory:
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_events.db")

	// Phase 1: Create store, append 3 events, close
	db1, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	store1, err := New(db1)
	require.NoError(t, err)

	var originalIDs [3]int64
	for i := 0; i < 3; i++ {
		id, appendErr := store1.Append("sess-persist", "answer", &pb.SessionEvent{
			Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
			Content: fmt.Sprintf("event-%d", i),
		}, map[string]interface{}{"type": "MessageCompleted", "content": fmt.Sprintf("event-%d", i)})
		require.NoError(t, appendErr)
		originalIDs[i] = id
	}

	require.NoError(t, db1.Close())

	// Phase 2: Reopen and verify
	db2, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db2.Close()

	store2, err := New(db2)
	require.NoError(t, err)

	events, err := store2.GetAll("sess-persist")
	require.NoError(t, err)
	require.Len(t, events, 3)

	for i, evt := range events {
		assert.Equal(t, originalIDs[i], evt.ID, "event %d ID should match", i)
		assert.Equal(t, fmt.Sprintf("event-%d", i), evt.Proto.GetContent())
	}
}

// TC-ES-11: Proto and JSON both correctly stored — verify dual representation.
func TestStore_ProtoAndJSONConsistency(t *testing.T) {
	db := newTestDB(t)
	store, err := New(db)
	require.NoError(t, err)

	protoEvent := &pb.SessionEvent{
		Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
		Content: "hello",
		AgentId: "supervisor",
	}
	jsonData := map[string]interface{}{
		"type":     "MessageCompleted",
		"content":  "hello",
		"agent_id": "supervisor",
	}

	_, err = store.Append("sess-dual", "answer", protoEvent, jsonData)
	require.NoError(t, err)

	events, err := store.GetAll("sess-dual")
	require.NoError(t, err)
	require.Len(t, events, 1)

	evt := events[0]

	// Verify proto fields
	assert.Equal(t, "hello", evt.Proto.GetContent())
	assert.Equal(t, pb.SessionEventType_SESSION_EVENT_ANSWER, evt.Proto.GetType())
	assert.Equal(t, "supervisor", evt.Proto.GetAgentId())

	// Verify JSON fields
	assert.Equal(t, "MessageCompleted", evt.JSON["type"])
	assert.Equal(t, "hello", evt.JSON["content"])
	assert.Equal(t, "supervisor", evt.JSON["agent_id"])

	// Verify JSON is valid by re-marshaling
	jsonBytes, err := json.Marshal(evt.JSON)
	require.NoError(t, err)
	var roundTrip map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonBytes, &roundTrip))
	assert.Equal(t, "MessageCompleted", roundTrip["type"])
}

// TC-ES-12: Different sessions don't interfere — extended isolation test with GetAfter.
func TestStore_SessionIsolationWithGetAfter(t *testing.T) {
	db := newTestDB(t)
	store, err := New(db)
	require.NoError(t, err)

	// Append 2 events to session-A
	idA1, err := store.Append("session-A", "answer", &pb.SessionEvent{
		Type: pb.SessionEventType_SESSION_EVENT_ANSWER, Content: "A1",
	}, map[string]interface{}{"type": "MessageCompleted", "content": "A1"})
	require.NoError(t, err)

	_, err = store.Append("session-A", "answer", &pb.SessionEvent{
		Type: pb.SessionEventType_SESSION_EVENT_ANSWER, Content: "A2",
	}, map[string]interface{}{"type": "MessageCompleted", "content": "A2"})
	require.NoError(t, err)

	// Append 3 events to session-B
	for i := 0; i < 3; i++ {
		_, err := store.Append("session-B", "answer", &pb.SessionEvent{
			Type: pb.SessionEventType_SESSION_EVENT_ANSWER, Content: fmt.Sprintf("B%d", i+1),
		}, map[string]interface{}{"type": "MessageCompleted"})
		require.NoError(t, err)
	}

	// GetAll returns correct counts per session
	eventsA, err := store.GetAll("session-A")
	require.NoError(t, err)
	assert.Len(t, eventsA, 2)

	eventsB, err := store.GetAll("session-B")
	require.NoError(t, err)
	assert.Len(t, eventsB, 3)

	// GetAfter for session-A after first event returns only the second
	afterA, err := store.GetAfter("session-A", idA1)
	require.NoError(t, err)
	require.Len(t, afterA, 1)
	assert.Equal(t, "A2", afterA[0].Proto.GetContent())

	// GetAfter for session-B with session-A's ID should not return session-A events.
	// Since idA1 doesn't exist in session-B, GetAfter falls back to GetAll for session-B.
	afterBWithAID, err := store.GetAfter("session-B", idA1)
	require.NoError(t, err)
	assert.Len(t, afterBWithAID, 3, "unknown ID for session-B should return all session-B events")

	// Verify no cross-contamination in content
	for _, evt := range eventsA {
		assert.Contains(t, evt.Proto.GetContent(), "A", "session-A events should have 'A' in content")
	}
	for _, evt := range eventsB {
		assert.Contains(t, evt.Proto.GetContent(), "B", "session-B events should have 'B' in content")
	}
}

