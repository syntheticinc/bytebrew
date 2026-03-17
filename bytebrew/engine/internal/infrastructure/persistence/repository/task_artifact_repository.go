package repository

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type taskArtifactRepository struct {
	db *gorm.DB
}

// NewTaskArtifactRepository creates a new TaskArtifactRepository
func NewTaskArtifactRepository(db *gorm.DB) *taskArtifactRepository {
	return &taskArtifactRepository{db: db}
}

func (r *taskArtifactRepository) Create(ctx context.Context, artifact *models.TaskArtifact) error {
	if err := r.db.WithContext(ctx).Create(artifact).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to create task artifact")
	}
	return nil
}

func (r *taskArtifactRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.TaskArtifact, error) {
	var artifact models.TaskArtifact
	if err := r.db.WithContext(ctx).First(&artifact, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "task artifact not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get task artifact by id")
	}
	return &artifact, nil
}

func (r *taskArtifactRepository) GetByTaskID(ctx context.Context, taskID uuid.UUID) ([]*models.TaskArtifact, error) {
	var artifacts []*models.TaskArtifact
	if err := r.db.WithContext(ctx).Where("task_id = ?", taskID).Order("created_at DESC").Find(&artifacts).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get task artifacts by task id")
	}
	return artifacts, nil
}

func (r *taskArtifactRepository) GetLatestByTaskAndType(ctx context.Context, taskID uuid.UUID, artifactType string) (*models.TaskArtifact, error) {
	var artifact models.TaskArtifact
	if err := r.db.WithContext(ctx).
		Where("task_id = ? AND artifact_type = ?", taskID, artifactType).
		Order("created_at DESC").
		First(&artifact).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "task artifact not found")
		}
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to get latest task artifact")
	}
	return &artifact, nil
}

func (r *taskArtifactRepository) Update(ctx context.Context, artifact *models.TaskArtifact) error {
	if err := r.db.WithContext(ctx).Save(artifact).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to update task artifact")
	}
	return nil
}

func (r *taskArtifactRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.TaskArtifact{}, "id = ?", id).Error; err != nil {
		return errors.Wrap(err, errors.CodeInternal, "failed to delete task artifact")
	}
	return nil
}
