package adapters

import (
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
)

// ProjectFileToModel converts domain ProjectFile to persistence model
func ProjectFileToModel(file *domain.ProjectFile) *models.ProjectFile {
	if file == nil {
		return nil
	}

	var id uuid.UUID
	if file.ID != "" {
		id, _ = uuid.Parse(file.ID)
	}

	projectID, _ := uuid.Parse(file.ProjectID)

	return &models.ProjectFile{
		ID:          id,
		ProjectID:   projectID,
		FilePath:    file.FilePath,
		Description: file.Description,
		Language:    file.Language,
		SizeBytes:   file.SizeBytes,
		CreatedAt:   file.CreatedAt,
		UpdatedAt:   file.UpdatedAt,
	}
}

// ProjectFileFromModel converts persistence model to domain ProjectFile
func ProjectFileFromModel(model *models.ProjectFile) (*domain.ProjectFile, error) {
	if model == nil {
		return nil, nil
	}

	file := &domain.ProjectFile{
		ID:          model.ID.String(),
		ProjectID:   model.ProjectID.String(),
		FilePath:    model.FilePath,
		Description: model.Description,
		Language:    model.Language,
		SizeBytes:   model.SizeBytes,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}

	return file, nil
}
