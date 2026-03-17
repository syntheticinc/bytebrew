package models

import "time"

// LLMProviderModel maps to the "models" table (LLM provider configuration).
type LLMProviderModel struct {
	ID              uint      `gorm:"primaryKey"`
	Name            string    `gorm:"uniqueIndex;not null"`
	Type            string    `gorm:"type:varchar(30);not null"`
	BaseURL         string    `gorm:"type:varchar(500)"`
	ModelName       string    `gorm:"type:varchar(255);not null"`
	APIKeyEncrypted string    `gorm:"type:varchar(1000)"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

func (LLMProviderModel) TableName() string { return "models" }
