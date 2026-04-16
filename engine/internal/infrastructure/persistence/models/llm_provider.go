package models

import "time"

// LLMProviderModel maps to the "models" table (LLM provider configuration).
type LLMProviderModel struct {
	ID              string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name            string    `gorm:"uniqueIndex;not null"`
	Type            string    `gorm:"type:varchar(30);not null"`
	BaseURL         string    `gorm:"type:varchar(500)"`
	ModelName       string    `gorm:"type:varchar(255);not null"`
	APIKeyEncrypted string    `gorm:"type:varchar(1000)"`
	APIVersion      string    `gorm:"type:varchar(30);default:''"`
	EmbeddingDim    int       `gorm:"type:int;default:0"` // >0 for embedding models (e.g. 1536 for text-embedding-3-small)
	TenantID        string    `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001'" json:"tenant_id"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

func (LLMProviderModel) TableName() string { return "models" }
