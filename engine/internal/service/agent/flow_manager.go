package agent

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/pkg/config"
)

// FlowManager manages flow configurations
type FlowManager struct {
	flows map[domain.FlowType]*domain.Flow
}

// NewFlowManager creates a FlowManager from config
func NewFlowManager(flowsCfg *config.FlowsConfig, prompts *config.PromptsConfig) (*FlowManager, error) {
	if flowsCfg == nil {
		return nil, fmt.Errorf("flows config is required")
	}

	flows := make(map[domain.FlowType]*domain.Flow)
	for flowType := range flowsCfg.Flows {
		flow, err := flowsCfg.ToDomainFlow(flowType, prompts)
		if err != nil {
			return nil, fmt.Errorf("create flow %s: %w", flowType, err)
		}
		flows[domain.FlowType(flowType)] = flow
	}

	return &FlowManager{flows: flows}, nil
}

// GetFlow returns flow configuration by type
func (m *FlowManager) GetFlow(ctx context.Context, flowType domain.FlowType) (*domain.Flow, error) {
	flow, ok := m.flows[flowType]
	if !ok {
		return nil, fmt.Errorf("unknown flow type: %s", flowType)
	}
	return flow, nil
}
