package models

import "time"

// AuditLogModel maps to the "audit_logs" table.
//
// Actor identity: populated from the request context's user_sub (JWT sub or
// synthetic local-admin sub). No FK — identity is external. Older schemas had
// actor_user_id (uuid FK to users); that column was dropped in migration 002.
type AuditLogModel struct {
	ID         string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID   string    `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001';index:idx_audit_tenant_time,priority:1" json:"tenant_id"`
	OccurredAt time.Time `gorm:"column:occurred_at;not null;default:now();index:idx_audit_tenant_time,priority:2"`
	ActorType  string    `gorm:"type:varchar(20);not null;index"`
	ActorSub   *string   `gorm:"column:actor_sub;type:varchar(255);index" json:"actor_sub"`
	Action     string    `gorm:"type:varchar(50);not null;index"`
	Resource   string    `gorm:"type:varchar(500)"`
	Details    string    `gorm:"type:jsonb"`
	SessionID  *string   `gorm:"type:uuid;index"`
	TaskID     *string   `gorm:"type:uuid;index"`

	Session *SessionModel `gorm:"foreignKey:SessionID"`
	Task    *TaskModel    `gorm:"foreignKey:TaskID"`
}

func (AuditLogModel) TableName() string { return "audit_logs" }
