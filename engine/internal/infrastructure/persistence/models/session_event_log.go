package models

import "time"

// SessionEventLogModel maps to the "session_event_log" table.
// Stores session events for reliable replay on reconnect.
// JSON representation is generated on-the-fly from ProtoData (json_data column dropped in migration 029).
type SessionEventLogModel struct {
	ID        string    `gorm:"primaryKey;type:uuid"`
	SessionID string    `gorm:"type:varchar(36);not null;index:idx_session_event_log_lookup"`
	EventType string    `gorm:"type:varchar(50);not null"`
	ProtoData []byte    `gorm:"type:bytea;not null"`
	TenantID  string    `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001'" json:"tenant_id"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (SessionEventLogModel) TableName() string { return "session_event_log" }
