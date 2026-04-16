package domain

import (
	"fmt"
	"time"
)

// Memory represents a long-term memory entry scoped to a schema.
// Memory is cross-session by definition: agents in the same schema
// share memories across all sessions.
type Memory struct {
	ID        string
	TenantID  string
	SchemaID  string
	UserID    string
	Content   string
	Metadata  map[string]string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewMemory creates a new Memory with validation.
func NewMemory(schemaID, userID, content string) (*Memory, error) {
	mem := &Memory{
		SchemaID:  schemaID,
		UserID:    userID,
		Content:   content,
		Metadata:  make(map[string]string),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := mem.Validate(); err != nil {
		return nil, err
	}
	return mem, nil
}

// Validate validates the Memory.
func (m *Memory) Validate() error {
	if m.SchemaID == "" {
		return fmt.Errorf("schema_id is required")
	}
	if m.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if m.Content == "" {
		return fmt.Errorf("content is required")
	}
	return nil
}

// AddMetadata adds metadata to the memory.
func (m *Memory) AddMetadata(key, value string) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]string)
	}
	m.Metadata[key] = value
	m.UpdatedAt = time.Now()
}

// GetMetadata retrieves metadata from the memory.
func (m *Memory) GetMetadata(key string) (string, bool) {
	if m.Metadata == nil {
		return "", false
	}
	value, ok := m.Metadata[key]
	return value, ok
}

// MemoryConfig holds per-schema memory configuration.
// Embedded in the Memory capability config.
type MemoryConfig struct {
	MaxEntries int // 0 = unlimited
}

// DefaultMemoryConfig returns default memory configuration.
func DefaultMemoryConfig() MemoryConfig {
	return MemoryConfig{
		MaxEntries: 0, // unlimited by default (AC-MEM-RET-01)
	}
}
