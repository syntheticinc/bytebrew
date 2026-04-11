package models

import "time"

// AuditLogModel maps to the "audit_log" table.
type AuditLogModel struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Timestamp time.Time `gorm:"not null;default:now();index"`
	ActorType string    `gorm:"type:varchar(20);not null;index"`
	ActorID   string    `gorm:"type:varchar(255)"`
	Action    string    `gorm:"type:varchar(50);not null;index"`
	Resource  string    `gorm:"type:varchar(500)"`
	Details   string    `gorm:"type:text"`
	SessionID *string   `gorm:"type:varchar(36);index"`
	TaskID    *string   `gorm:"type:uuid;index"`

	Session *SessionModel `gorm:"foreignKey:SessionID"`
	Task    *TaskModel    `gorm:"foreignKey:TaskID"`
}

func (AuditLogModel) TableName() string { return "audit_log" }
