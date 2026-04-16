package models

import "time"

// SessionEventLogModel maps to the "session_event_log" table.
// Stores session events for reliable replay on reconnect.
// Not to be confused with SessionEventModel (admin dashboard events).
type SessionEventLogModel struct {
	ID        string    `gorm:"primaryKey;type:uuid"`
	SessionID string    `gorm:"type:varchar(36);not null;index:idx_session_event_log_lookup"`
	EventType string    `gorm:"type:varchar(50);not null"`
	ProtoData []byte    `gorm:"type:bytea;not null"`
	JSONData  string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (SessionEventLogModel) TableName() string { return "session_event_log" }
