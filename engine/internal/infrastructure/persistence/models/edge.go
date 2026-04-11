package models

import "time"

// EdgeModel maps to the "edges" table.
type EdgeModel struct {
	ID              string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SchemaID        string    `gorm:"type:uuid;not null;index"`
	SourceAgentName string    `gorm:"type:varchar(255);not null"`
	TargetAgentName string    `gorm:"type:varchar(255);not null"`
	Type            string    `gorm:"type:varchar(50);not null;default:flow"`
	Config          string    `gorm:"type:text"` // JSON
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`

	Schema SchemaModel `gorm:"foreignKey:SchemaID"`
}

func (EdgeModel) TableName() string { return "edges" }
