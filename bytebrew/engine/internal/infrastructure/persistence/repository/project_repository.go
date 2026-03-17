package repository

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/adapters"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type projectRepository struct {
	db *gorm.DB
}

// NewProjectRepository creates a new project repository
func NewProjectRepository(db *gorm.DB) *projectRepository {
	return &projectRepository{db: db}
}

func (r *projectRepository) Create(ctx context.Context, project *domain.Project) error {
	model := adapters.ProjectToModel(project)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to create project")
	}

	// Update domain entity with generated ID
	project.ID = model.ID.String()
	return nil
}

func (r *projectRepository) GetByID(ctx context.Context, id string) (*domain.Project, error) {
	projectID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidInput, "invalid project id")
	}

	var model models.Project
	if err := r.db.WithContext(ctx).First(&model, "id = ?", projectID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "project not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get project by id")
	}

	return adapters.ProjectFromModel(&model)
}

func (r *projectRepository) GetByProjectKey(ctx context.Context, projectKey string) (*domain.Project, error) {
	var model models.Project
	if err := r.db.WithContext(ctx).Where("project_key = ?", projectKey).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "project not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get project by key")
	}

	return adapters.ProjectFromModel(&model)
}

func (r *projectRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Project, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidInput, "invalid user id")
	}

	var models []*models.Project
	if err := r.db.WithContext(ctx).Where("user_id = ?", uid).Find(&models).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get projects by user id")
	}

	projects := make([]*domain.Project, 0, len(models))
	for _, model := range models {
		project, err := adapters.ProjectFromModel(model)
		if err != nil {
			return nil, errors.Wrap(err, errors.CodeInternal, "failed to convert project")
		}
		projects = append(projects, project)
	}

	return projects, nil
}

func (r *projectRepository) Update(ctx context.Context, project *domain.Project) error {
	model := adapters.ProjectToModel(project)

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to update project")
	}
	return nil
}

func (r *projectRepository) Delete(ctx context.Context, id string) error {
	projectID, err := uuid.Parse(id)
	if err != nil {
		return errors.Wrap(err, errors.CodeInvalidInput, "invalid project id")
	}

	if err := r.db.WithContext(ctx).Delete(&models.Project{}, "id = ?", projectID).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to delete project")
	}
	return nil
}
