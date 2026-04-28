package models

import (
	"encoding/json"
	"time"
)

// MessageModel maps to the "messages" table.
// Stores chronological events for agent sessions: messages, tool calls, reasoning.
// Origin (chat/cron/webhook) is derivable via session → schema.
type MessageModel struct {
	ID        string          `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  string          `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001';index:idx_messages_tenant_session_chrono,priority:1" json:"tenant_id"`
	SessionID string          `gorm:"type:uuid;not null;index:idx_messages_tenant_session_chrono,priority:2;index:idx_messages_call_id,priority:1"`
	EventType string          `gorm:"type:varchar(20);not null"`
	AgentID   *string         `gorm:"type:uuid;index"`
	CallID    string          `gorm:"type:varchar(100);index:idx_messages_call_id,priority:2"`
	Payload   json.RawMessage `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt time.Time       `gorm:"autoCreateTime;index:idx_messages_tenant_session_chrono,priority:3"`
}

func (MessageModel) TableName() string { return "messages" }
