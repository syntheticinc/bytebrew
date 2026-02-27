package models

import (
	"time"

	"github.com/google/uuid"
)

// Project represents a user's codebase project
type Project struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index"`
	Name       string    `gorm:"type:varchar(255);not null"`
	RootPath   string    `gorm:"type:varchar(500);not null"`
	ProjectKey string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	CreatedAt  time.Time `gorm:"not null;default:now()"`
	UpdatedAt  time.Time `gorm:"not null;default:now()"`

	// Relationships
	User         User          `gorm:"foreignKey:UserID"`
	ProjectFiles []ProjectFile `gorm:"foreignKey:ProjectID"`
	ChatSessions []ChatSession `gorm:"foreignKey:ProjectID"`
}

// TableName specifies the table name for Project model
func (Project) TableName() string {
	return "project"
}
