package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// TriggerConfig is the type-specific configuration carried inside
// `triggers.config` (jsonb). Fields are type-dependent; unused fields are
// simply omitted:
//
//   - cron:    {"schedule": "0 */5 * * *"}
//   - webhook: {"webhook_path": "/hooks/foo"}
//   - chat:    {} (may carry future fields like allowed_domains)
//
// See docs/architecture/agent-first-runtime.md §4.1.
type TriggerConfig struct {
	Schedule    string `json:"schedule,omitempty"`
	WebhookPath string `json:"webhook_path,omitempty"`
}

// Scan implements sql.Scanner for jsonb columns.
func (c *TriggerConfig) Scan(value interface{}) error {
	if value == nil {
		*c = TriggerConfig{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("scan trigger config: unsupported type %T", value)
	}
	if len(bytes) == 0 {
		*c = TriggerConfig{}
		return nil
	}
	return json.Unmarshal(bytes, c)
}

// Value implements driver.Valuer for jsonb columns.
func (c TriggerConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// TriggerModel maps to the "triggers" table.
//
// V2: type-specific fields live in `Config` (jsonb). The legacy flat
// `schedule` / `webhook_path` columns and the `on_complete_url` /
// `on_complete_headers` webhook feature are removed entirely.
// See docs/architecture/agent-first-runtime.md §4.1 / §4.2.
type TriggerModel struct {
	ID          string        `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Type        string        `gorm:"type:varchar(10);not null;index"`
	Title       string        `gorm:"type:varchar(255);not null"`
	AgentID     *string       `gorm:"type:uuid;index"`
	SchemaID    *string       `gorm:"type:uuid;index;constraint:OnDelete:CASCADE"`
	Description string        `gorm:"type:text"`
	Enabled     bool          `gorm:"not null;default:true"`
	Config      TriggerConfig `gorm:"type:jsonb;not null;default:'{}'"`
	LastFiredAt *time.Time
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`

	Agent  AgentModel  `gorm:"foreignKey:AgentID"`
	Schema SchemaModel `gorm:"foreignKey:SchemaID"`
}

func (TriggerModel) TableName() string { return "triggers" }
