package models

import "time"

// AuditLogModel maps to the "audit_logs" table.
type AuditLogModel struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	OccurredAt time.Time `gorm:"column:occurred_at;not null;default:now();index"`
	ActorType string    `gorm:"type:varchar(20);not null;index"`
	ActorUserID *string `gorm:"column:actor_user_id;type:uuid"`
	Action    string    `gorm:"type:varchar(50);not null;index"`
	Resource  string    `gorm:"type:varchar(500)"`
	Details   string    `gorm:"type:jsonb"`
	SessionID *string   `gorm:"type:uuid;index"`
	TaskID    *string   `gorm:"type:uuid;index"`
	TenantID  string    `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001'" json:"tenant_id"`

	Session *SessionModel `gorm:"foreignKey:SessionID"`
	Task    *TaskModel    `gorm:"foreignKey:TaskID"`
}

func (AuditLogModel) TableName() string { return "audit_logs" }
