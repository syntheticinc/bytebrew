package models

// RuntimeConfigKV maps to the "runtime_config" table.
// Stores key-value pairs for server identity and other config data.
type RuntimeConfigKV struct {
	Key   string `gorm:"primaryKey;type:varchar(255)"`
	Value string `gorm:"type:text;not null"`
}

func (RuntimeConfigKV) TableName() string { return "runtime_config" }
