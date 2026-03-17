package models

import "time"

// APITokenModel maps to the "api_tokens" table.
type APITokenModel struct {
	ID         uint       `gorm:"primaryKey"`
	Name       string     `gorm:"uniqueIndex;not null"`
	TokenHash  string     `gorm:"uniqueIndex;not null"`
	ScopesMask int        `gorm:"not null;default:0"`
	CreatedAt  time.Time  `gorm:"autoCreateTime"`
	LastUsedAt *time.Time
	RevokedAt  *time.Time
}

func (APITokenModel) TableName() string { return "api_tokens" }

// Scope bitmask constants.
const (
	ScopeChat       = 1
	ScopeTasks      = 2
	ScopeAgentsRead = 4
	ScopeConfig     = 8
	ScopeAdmin      = 16
)

// HasScope checks whether the token has the given scope bit set.
func (t *APITokenModel) HasScope(scope int) bool {
	return t.ScopesMask&scope != 0
}
