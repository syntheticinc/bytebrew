package models

import (
	"encoding/json"
	"time"
)

// MessageModel maps to the "messages" table.
// Stores chronological events for agent sessions: messages, tool calls, reasoning.
type MessageModel struct {
	ID        string          `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SessionID string          `gorm:"type:uuid;not null;index:idx_messages_session"`
	EventType string          `gorm:"type:varchar(20);not null"`
	AgentID   string          `gorm:"type:varchar(100);index:idx_messages_agent"`
	CallID    string          `gorm:"type:varchar(100)"`
	Payload   json.RawMessage `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt time.Time       `gorm:"autoCreateTime;index:idx_messages_created"`
}

func (MessageModel) TableName() string { return "messages" }
