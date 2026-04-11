package models

// AgentEscalation maps to the "agent_escalation" table (1:1 per agent).
type AgentEscalation struct {
	ID         string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	AgentID    string `gorm:"type:uuid;uniqueIndex;not null"`
	Action     string `gorm:"type:varchar(30);not null;default:transfer_to_human"`
	WebhookURL string `gorm:"type:varchar(500)"`

	Agent    AgentModel                `gorm:"foreignKey:AgentID"`
	Triggers []AgentEscalationTrigger  `gorm:"foreignKey:EscalationID"`
}

func (AgentEscalation) TableName() string { return "agent_escalation" }

// AgentEscalationTrigger maps to the "agent_escalation_triggers" table.
type AgentEscalationTrigger struct {
	ID           string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	EscalationID string `gorm:"type:uuid;not null;index;uniqueIndex:idx_escalation_keyword"`
	Keyword      string `gorm:"type:varchar(255);not null;index;uniqueIndex:idx_escalation_keyword"`

	Escalation AgentEscalation `gorm:"foreignKey:EscalationID"`
}

func (AgentEscalationTrigger) TableName() string { return "agent_escalation_triggers" }
