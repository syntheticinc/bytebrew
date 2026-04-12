package app

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/engine/internal/service/capability"
	"github.com/syntheticinc/bytebrew/engine/internal/service/escalation"
)

// escalationConfigAdapter bridges capability.CapabilityReader to escalation.CapabilityReader.
// Depends on interface (capability.CapabilityReader), not concrete repo (DIP).
type escalationConfigAdapter struct {
	reader capability.CapabilityReader
}

func (a *escalationConfigAdapter) GetEscalationConfig(ctx context.Context, agentName string) (*escalation.Config, error) {
	records, err := a.reader.ListEnabledByAgent(ctx, agentName)
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
