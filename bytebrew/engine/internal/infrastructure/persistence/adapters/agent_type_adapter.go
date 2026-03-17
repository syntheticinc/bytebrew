package adapters

import (
	"encoding/json"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// AgentTypeToModel converts domain AgentType to persistence model
func AgentTypeToModel(agentType *domain.AgentType) *models.AgentType {
	if agentType == nil {
		return nil
	}

	var id uuid.UUID
	if agentType.ID != "" {
		id, _ = uuid.Parse(agentType.ID)
	}

	// Convert tools slice to JSON
	toolsJSON, _ := json.Marshal(agentType.Tools)

	return &models.AgentType{
		ID:           id,
		Code:         agentType.Code,
		Name:         agentType.Name,
		Description:  agentType.Description,
		SystemPrompt: agentType.SystemPrompt,
		Tools:        datatypes.JSON(toolsJSON),
	}
}

// AgentTypeFromModel converts persistence model to domain AgentType
func AgentTypeFromModel(model *models.AgentType) (*domain.AgentType, error) {
	if model == nil {
		return nil, nil
	}

	// Convert JSON to tools slice
	var tools []string
	if err := json.Unmarshal([]byte(model.Tools), &tools); err != nil {
		tools = []string{}
	}

	agentType := &domain.AgentType{
		ID:           model.ID.String(),
		Code:         model.Code,
		Name:         model.Name,
		Description:  model.Description,
		SystemPrompt: model.SystemPrompt,
		Tools:        tools,
	}

	return agentType, nil
}
