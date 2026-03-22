package eventstore

import (
	"encoding/json"
	"fmt"
	"strconv"

	pb "github.com/syntheticinc/bytebrew/engine/api/proto/gen"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// StoredEvent represents a persisted session event with both proto and JSON representations.
type StoredEvent struct {
	ID        int64
	SessionID string
	EventType string
	Proto     *pb.SessionEvent
	JSON      map[string]interface{}
}

// Store persists session events in PostgreSQL (GORM) for reliable replay on reconnect.
type Store struct {
	db *gorm.DB
}

// New creates a new event store.
func New(db *gorm.DB) (*Store, error) {
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

	m := models.RuntimeSessionEventModel{
		SessionID: sessionID,
		EventType: eventType,
		ProtoData: protoBytes,
		JSONData:  string(jsonBytes),
	}

	if err := s.db.Create(&m).Error; err != nil {
		return 0, fmt.Errorf("insert event: %w", err)
	}

	return int64(m.ID), nil
}

// GetAfter returns all events for a session with ID > lastEventID.
// If lastEventID is 0 or no row with that ID exists for the session,
// all events for the session are returned (safe fallback).
func (s *Store) GetAfter(sessionID string, lastEventID int64) ([]StoredEvent, error) {
	if lastEventID <= 0 {
		return s.GetAll(sessionID)
	}

	// Check if lastEventID actually exists for this session.
	var exists int64
	if err := s.db.Model(&models.RuntimeSessionEventModel{}).
		Where("session_id = ? AND id = ?", sessionID, lastEventID).
		Count(&exists).Error; err != nil {
		return nil, fmt.Errorf("check last event id: %w", err)
	}

	if exists == 0 {
		return s.GetAll(sessionID)
	}

	var ms []models.RuntimeSessionEventModel
	if err := s.db.
		Where("session_id = ? AND id > ?", sessionID, lastEventID).
		Order("id").
		Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("query events after %d: %w", lastEventID, err)
	}

	return scanEventModels(ms)
}

// GetAll returns all events for a session ordered by ID.
func (s *Store) GetAll(sessionID string) ([]StoredEvent, error) {
	var ms []models.RuntimeSessionEventModel
	if err := s.db.
		Where("session_id = ?", sessionID).
		Order("id").
		Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("query all events: %w", err)
	}

	return scanEventModels(ms)
}

// CleanupSession deletes all events for a session.
func (s *Store) CleanupSession(sessionID string) error {
	if err := s.db.Where("session_id = ?", sessionID).
		Delete(&models.RuntimeSessionEventModel{}).Error; err != nil {
		return fmt.Errorf("cleanup session events: %w", err)
	}
	return nil
}

func scanEventModels(ms []models.RuntimeSessionEventModel) ([]StoredEvent, error) {
	events := make([]StoredEvent, 0, len(ms))

	for _, m := range ms {
		pbEvent := &pb.SessionEvent{}
		if err := proto.Unmarshal(m.ProtoData, pbEvent); err != nil {
			return nil, fmt.Errorf("unmarshal proto for event %d: %w", m.ID, err)
		}
		pbEvent.EventId = strconv.FormatInt(int64(m.ID), 10)

		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(m.JSONData), &jsonData); err != nil {
			return nil, fmt.Errorf("unmarshal json for event %d: %w", m.ID, err)
		}

		events = append(events, StoredEvent{
			ID:        int64(m.ID),
			SessionID: m.SessionID,
			EventType: m.EventType,
			Proto:     pbEvent,
			JSON:      jsonData,
		})
	}

	return events, nil
}
