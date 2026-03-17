package adapters

import (
	"encoding/json"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// TaskArtifactToModel converts domain TaskArtifact to persistence model
func TaskArtifactToModel(artifact *domain.TaskArtifact) *models.TaskArtifact {
	if artifact == nil {
		return nil
	}

	var id uuid.UUID
	if artifact.ID != "" {
		id, _ = uuid.Parse(artifact.ID)
	}

	taskID, _ := uuid.Parse(artifact.TaskID)

	// Convert map to JSON
	contentJSON, _ := json.Marshal(artifact.Content)

	return &models.TaskArtifact{
		ID:             id,
		TaskID:         taskID,
		ArtifactType:   artifact.ArtifactType,
		Content:        datatypes.JSON(contentJSON),
		FilePath:       artifact.FilePath,
		CreatedByAgent: artifact.CreatedByAgent,
		CreatedAt:      artifact.CreatedAt,
		UpdatedAt:      artifact.UpdatedAt,
	}
}

// TaskArtifactFromModel converts persistence model to domain TaskArtifact
func TaskArtifactFromModel(model *models.TaskArtifact) (*domain.TaskArtifact, error) {
	if model == nil {
		return nil, nil
	}

	// Convert JSON to map
	var content map[string]interface{}
	if err := json.Unmarshal([]byte(model.Content), &content); err != nil {
		content = make(map[string]interface{})
	}

	artifact := &domain.TaskArtifact{
		ID:             model.ID.String(),
		TaskID:         model.TaskID.String(),
		ArtifactType:   model.ArtifactType,
		Content:        content,
		FilePath:       model.FilePath,
		CreatedByAgent: model.CreatedByAgent,
		CreatedAt:      model.CreatedAt,
		UpdatedAt:      model.UpdatedAt,
	}

	return artifact, nil
}
