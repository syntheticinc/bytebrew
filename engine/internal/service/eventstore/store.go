package eventstore

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	pb "github.com/syntheticinc/bytebrew/engine/api/proto/gen"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// StoredEvent represents a persisted session event with both proto and JSON representations.
type StoredEvent struct {
	ID        string
	SessionID string
	EventType string
	Proto     *pb.SessionEvent
	JSON      map[string]interface{}
	CreatedAt time.Time
}

// Store persists session events in PostgreSQL (GORM) for reliable replay on reconnect.
type Store struct {
	db *gorm.DB
}

// New creates a new event store.
func New(db *gorm.DB) (*Store, error) {
	return &Store{db: db}, nil
}

// Append persists a session event and returns the UUID.
// The proto is marshaled WITHOUT EventId (unknown pre-insert). Callers should
// set event.EventId = id after Append returns.
func (s *Store) Append(sessionID, eventType string, event *pb.SessionEvent, jsonData map[string]interface{}) (string, error) {
	protoBytes, err := proto.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("marshal proto: %w", err)
	}

	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		return "", fmt.Errorf("marshal json: %w", err)
	}

	m := models.RuntimeSessionEventModel{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		EventType: eventType,
		ProtoData: protoBytes,
		JSONData:  string(jsonBytes),
	}

	if err := s.db.Create(&m).Error; err != nil {
		return "", fmt.Errorf("insert event: %w", err)
	}

	return m.ID, nil
}

// GetAfter returns all events for a session created after the given timestamp.
// If afterCreatedAt is zero, all events for the session are returned.
func (s *Store) GetAfter(sessionID string, afterCreatedAt time.Time) ([]StoredEvent, error) {
	if afterCreatedAt.IsZero() {
		return s.GetAll(sessionID)
	}

	var ms []models.RuntimeSessionEventModel
	if err := s.db.
		Where("session_id = ? AND created_at > ?", sessionID, afterCreatedAt).
		Order("created_at ASC").
		Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("query events after %v: %w", afterCreatedAt, err)
	}

	return scanEventModels(ms)
}

// GetAll returns all events for a session ordered by creation time.
func (s *Store) GetAll(sessionID string) ([]StoredEvent, error) {
	var ms []models.RuntimeSessionEventModel
	if err := s.db.
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
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
			return nil, fmt.Errorf("unmarshal proto for event %s: %w", m.ID, err)
		}
		pbEvent.EventId = m.ID

		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(m.JSONData), &jsonData); err != nil {
			return nil, fmt.Errorf("unmarshal json for event %s: %w", m.ID, err)
		}

		events = append(events, StoredEvent{
			ID:        m.ID,
			SessionID: m.SessionID,
			EventType: m.EventType,
			Proto:     pbEvent,
			JSON:      jsonData,
			CreatedAt: m.CreatedAt,
		})
	}

	return events, nil
}
