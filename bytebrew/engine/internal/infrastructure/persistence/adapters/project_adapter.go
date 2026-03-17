package adapters

import (
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
)

// ProjectToModel converts domain Project to persistence model
func ProjectToModel(project *domain.Project) *models.Project {
	if project == nil {
		return nil
	}

	var id uuid.UUID
	if project.ID != "" {
		id, _ = uuid.Parse(project.ID)
	}

	userID, _ := uuid.Parse(project.UserID)

	return &models.Project{
		ID:         id,
		UserID:     userID,
		Name:       project.Name,
		RootPath:   project.RootPath,
		ProjectKey: project.ProjectKey,
		CreatedAt:  project.CreatedAt,
		UpdatedAt:  project.UpdatedAt,
	}
}

// ProjectFromModel converts persistence model to domain Project
func ProjectFromModel(model *models.Project) (*domain.Project, error) {
	if model == nil {
		return nil, nil
	}

	project := &domain.Project{
		ID:         model.ID.String(),
		UserID:     model.UserID.String(),
		Name:       model.Name,
		RootPath:   model.RootPath,
		ProjectKey: model.ProjectKey,
		CreatedAt:  model.CreatedAt,
		UpdatedAt:  model.UpdatedAt,
	}

	return project, nil
}
