package models

import "time"

// SettingModel maps to the "settings" key-value table.
type SettingModel struct {
	Key       string    `gorm:"primaryKey;type:varchar(255)"`
	Value     string    `gorm:"type:text;not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (SettingModel) TableName() string { return "settings" }
