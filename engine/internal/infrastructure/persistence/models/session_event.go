package models

import "time"

// SessionEventModel maps to the "session_events" table.
type SessionEventModel struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SessionID string    `gorm:"type:varchar(36);not null;index:idx_session_event_id"`
	EventType string    `gorm:"type:varchar(50);not null"`
	Payload   string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`

	Session *SessionModel `gorm:"foreignKey:SessionID"`
}

func (SessionEventModel) TableName() string { return "session_events" }
