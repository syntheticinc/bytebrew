package models

import "time"

// RuntimeSessionModel maps to the "runtime_sessions" table.
// Stores domain.Session data (legacy work-session with ProjectKey).
// Not to be confused with SessionModel (admin dashboard sessions).
type RuntimeSessionModel struct {
	ID             string    `gorm:"primaryKey;type:varchar(36)"`
	ProjectKey     string    `gorm:"type:varchar(255);not null;index"`
	Status         string    `gorm:"type:varchar(20);not null;default:active;index"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
	LastActivityAt time.Time `gorm:"not null"`
}

func (RuntimeSessionModel) TableName() string { return "runtime_sessions" }
