package repository

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type projectFileRepository struct {
	db *gorm.DB
}

// NewProjectFileRepository creates a new ProjectFileRepository
func NewProjectFileRepository(db *gorm.DB) *projectFileRepository {
	return &projectFileRepository{db: db}
}

func (r *projectFileRepository) Create(ctx context.Context, file *models.ProjectFile) error {
	if err := r.db.WithContext(ctx).Create(file).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to create project file")
	}
	return nil
}

func (r *projectFileRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ProjectFile, error) {
	var file models.ProjectFile
	if err := r.db.WithContext(ctx).First(&file, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "project file not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get project file by id")
	}
	return &file, nil
}

func (r *projectFileRepository) GetByProjectID(ctx context.Context, projectID uuid.UUID) ([]*models.ProjectFile, error) {
	var files []*models.ProjectFile
	if err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Order("file_path ASC").Find(&files).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get project files by project id")
	}
	return files, nil
}

func (r *projectFileRepository) GetByProjectAndPath(ctx context.Context, projectID uuid.UUID, filePath string) (*models.ProjectFile, error) {
	var file models.ProjectFile
	if err := r.db.WithContext(ctx).Where("project_id = ? AND file_path = ?", projectID, filePath).First(&file).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "project file not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get project file by project and path")
	}
	return &file, nil
}

func (r *projectFileRepository) Update(ctx context.Context, file *models.ProjectFile) error {
	if err := r.db.WithContext(ctx).Save(file).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to update project file")
	}
	return nil
}

func (r *projectFileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.ProjectFile{}, "id = ?", id).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to delete project file")
	}
	return nil
}
