package models

import "time"

// RuntimeMessageModel maps to the "runtime_messages" table.
// Stores domain.Message data for agent conversation history.
type RuntimeMessageModel struct {
	ID          string    `gorm:"primaryKey;type:varchar(36)"`
	SessionID   string    `gorm:"type:varchar(36);not null;index:idx_rt_msg_session"`
	MessageType string    `gorm:"type:varchar(50);not null"`
	Sender      string    `gorm:"type:varchar(255)"`
	AgentID     string    `gorm:"type:varchar(100);index:idx_rt_msg_session_agent"`
	Content     string    `gorm:"type:text;not null"`
	Metadata    string    `gorm:"type:text"` // JSON blob
	CreatedAt   time.Time `gorm:"autoCreateTime;index:idx_rt_msg_created"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (RuntimeMessageModel) TableName() string { return "runtime_messages" }
