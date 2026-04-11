package models

import "time"

// RuntimeSessionEventModel maps to the "runtime_session_events" table.
// Stores session events for reliable replay on reconnect.
// Not to be confused with SessionEventModel (admin dashboard events).
type RuntimeSessionEventModel struct {
	ID        string    `gorm:"primaryKey;type:uuid"`
	SessionID string    `gorm:"type:varchar(36);not null;index:idx_runtime_session_event_lookup"`
	EventType string    `gorm:"type:varchar(50);not null"`
	ProtoData []byte    `gorm:"type:bytea;not null"`
	JSONData  string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (RuntimeSessionEventModel) TableName() string { return "runtime_session_events" }
