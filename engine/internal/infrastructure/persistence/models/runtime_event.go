package models

import (
	"encoding/json"
	"time"
)

// RuntimeEventModel maps to the "runtime_events" table.
// Stores chronological events for agent sessions: messages, tool calls, reasoning.
type RuntimeEventModel struct {
	ID        string          `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SessionID string          `gorm:"type:uuid;not null;index:idx_rt_event_session"`
	EventType string          `gorm:"type:varchar(20);not null"`
	AgentID   string          `gorm:"type:varchar(100);index:idx_rt_event_agent"`
	CallID    string          `gorm:"type:varchar(100)"`
	Payload   json.RawMessage `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt time.Time       `gorm:"autoCreateTime;index:idx_rt_event_created"`
}

func (RuntimeEventModel) TableName() string { return "runtime_events" }
