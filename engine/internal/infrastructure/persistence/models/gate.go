package models

import "time"

// GateModel maps to the "gates" table.
type GateModel struct {
	ID            uint      `gorm:"primaryKey"`
	SchemaID      uint      `gorm:"not null;index"`
	Name          string    `gorm:"type:varchar(255);not null"`
	ConditionType string    `gorm:"type:varchar(50);not null;default:all"`
	Config        string    `gorm:"type:text"`       // JSON
	MaxIterations int       `gorm:"not null;default:0"`
	Timeout       int       `gorm:"not null;default:0"` // seconds
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`

	Schema SchemaModel `gorm:"foreignKey:SchemaID"`
}

func (GateModel) TableName() string { return "gates" }
