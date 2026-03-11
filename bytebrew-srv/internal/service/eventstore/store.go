package eventstore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
	"google.golang.org/protobuf/proto"
)

// StoredEvent represents a persisted session event with both proto and JSON representations.
type StoredEvent struct {
	ID        int64
	SessionID string
	EventType string
	Proto     *pb.SessionEvent
	JSON      map[string]interface{}
}

// Store persists session events in SQLite for reliable replay on reconnect.
type Store struct {
	db *sql.DB
}

// New creates a new event store and ensures the schema exists.
func New(db *sql.DB) (*Store, error) {
	if err := createSchema(db); err != nil {
		return nil, fmt.Errorf("create event store schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Append persists a session event and returns the auto-increment ID.
// The proto is marshaled WITHOUT EventId (unknown pre-insert). Callers should
// set event.EventId = strconv.FormatInt(id, 10) after Append returns.
func (s *Store) Append(sessionID, eventType string, event *pb.SessionEvent, jsonData map[string]interface{}) (int64, error) {
	protoBytes, err := proto.Marshal(event)
	if err != nil {
		return 0, fmt.Errorf("marshal proto: %w", err)
	}

	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		return 0, fmt.Errorf("marshal json: %w", err)
	}

	result, err := s.db.Exec(
		`INSERT INTO session_events (session_id, event_type, proto_data, json_data) VALUES (?, ?, ?, ?)`,
		sessionID, eventType, protoBytes, string(jsonBytes),
	)
	if err != nil {
		return 0, fmt.Errorf("insert event: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id: %w", err)
	}

	return id, nil
}

// GetAfter returns all events for a session with ID > lastEventID.
// If lastEventID is 0 or no row with that ID exists for the session,
// all events for the session are returned (safe fallback).
func (s *Store) GetAfter(sessionID string, lastEventID int64) ([]StoredEvent, error) {
	if lastEventID <= 0 {
		return s.GetAll(sessionID)
	}

	// Check if lastEventID actually exists for this session.
	var exists int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM session_events WHERE session_id = ? AND id = ?`,
		sessionID, lastEventID,
	).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check last event id: %w", err)
	}

	if exists == 0 {
		// Unknown lastEventID — return all events as safe fallback.
		return s.GetAll(sessionID)
	}

	rows, err := s.db.Query(
		`SELECT id, session_id, event_type, proto_data, json_data FROM session_events WHERE session_id = ? AND id > ? ORDER BY id`,
		sessionID, lastEventID,
	)
	if err != nil {
		return nil, fmt.Errorf("query events after %d: %w", lastEventID, err)
	}
	defer rows.Close()

	return scanEvents(rows)
}

// GetAll returns all events for a session ordered by ID.
func (s *Store) GetAll(sessionID string) ([]StoredEvent, error) {
	rows, err := s.db.Query(
		`SELECT id, session_id, event_type, proto_data, json_data FROM session_events WHERE session_id = ? ORDER BY id`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query all events: %w", err)
	}
	defer rows.Close()

	return scanEvents(rows)
}

// CleanupSession deletes all events for a session.
func (s *Store) CleanupSession(sessionID string) error {
	_, err := s.db.Exec(`DELETE FROM session_events WHERE session_id = ?`, sessionID)
	if err != nil {
		return fmt.Errorf("cleanup session events: %w", err)
	}
	return nil
}

func createSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS session_events (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id  TEXT    NOT NULL,
			event_type  TEXT    NOT NULL,
			proto_data  BLOB   NOT NULL,
			json_data   TEXT    NOT NULL,
			created_at  INTEGER NOT NULL DEFAULT (unixepoch())
		)
	`)
	if err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_session_events_lookup ON session_events (session_id, id)`)
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}

	return nil
}

func scanEvents(rows *sql.Rows) ([]StoredEvent, error) {
	var events []StoredEvent

	for rows.Next() {
		var (
			id         int64
			sessionID  string
			eventType  string
			protoBytes []byte
			jsonStr    string
		)

		if err := rows.Scan(&id, &sessionID, &eventType, &protoBytes, &jsonStr); err != nil {
			return nil, fmt.Errorf("scan event row: %w", err)
		}

		pbEvent := &pb.SessionEvent{}
		if err := proto.Unmarshal(protoBytes, pbEvent); err != nil {
			return nil, fmt.Errorf("unmarshal proto for event %d: %w", id, err)
		}
		// Set EventId from the auto-increment ID (not stored in proto_data).
		pbEvent.EventId = strconv.FormatInt(id, 10)

		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &jsonData); err != nil {
			return nil, fmt.Errorf("unmarshal json for event %d: %w", id, err)
		}

		events = append(events, StoredEvent{
			ID:        id,
			SessionID: sessionID,
			EventType: eventType,
			Proto:     pbEvent,
			JSON:      jsonData,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate event rows: %w", err)
	}

	return events, nil
}
