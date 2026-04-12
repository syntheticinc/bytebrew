package app

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/config_repo"
	"github.com/syntheticinc/bytebrew/engine/internal/service/escalation"
)

// escalationConfigAdapter bridges GORMCapabilityRepository to escalation.CapabilityReader.
type escalationConfigAdapter struct {
	repo *config_repo.GORMCapabilityRepository
}

func (a *escalationConfigAdapter) GetEscalationConfig(ctx context.Context, agentName string) (*escalation.Config, error) {
	records, err := a.repo.ListEnabledByAgent(ctx, agentName)
	if err != nil {
		return nil, fmt.Errorf("list capabilities for %q: %w", agentName, err)
	}

	for _, r := range records {
		if r.Type != "escalation" {
			continue
		}
		cfg := &escalation.Config{}
		if v, ok := r.Config["action"].(string); ok {
			cfg.Action = v
		}
		if v, ok := r.Config["webhook_url"].(string); ok {
			cfg.WebhookURL = v
		}
		return cfg, nil
	}

	// No escalation capability configured — return default
	return &escalation.Config{Action: "transfer_to_user"}, nil
}
