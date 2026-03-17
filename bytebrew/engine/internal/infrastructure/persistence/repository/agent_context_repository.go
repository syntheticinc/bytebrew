package repository

import (
	"context"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/adapters"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AgentContextRepository provides CRUD for agent context snapshots
type AgentContextRepository struct {
	db *gorm.DB
}

// NewAgentContextRepository creates a new AgentContextRepository
func NewAgentContextRepository(db *gorm.DB) *AgentContextRepository {
	return &AgentContextRepository{db: db}
}

// Save performs upsert by agent_id (one snapshot per agent per session)
func (r *AgentContextRepository) Save(ctx context.Context, snapshot *domain.AgentContextSnapshot) error {
	model := adapters.AgentContextSnapshotToModel(snapshot)
	if model.ID == uuid.Nil {
		model.ID = uuid.New()
	}
	model.UpdatedAt = time.Now()

	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "agent_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"session_id", "context_data", "step_number", "token_count", "status", "updated_at", "schema_version",
		}),
	}).Create(model)

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.CodeInternal, "save agent context snapshot")
	}

	snapshot.ID = model.ID.String()
	return nil
}

// Load loads snapshot by session+agent ID
func (r *AgentContextRepository) Load(ctx context.Context, sessionID, agentID string) (*domain.AgentContextSnapshot, error) {
	sessID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidInput, "invalid session ID")
	}

	var model models.AgentContextSnapshot
	result := r.db.WithContext(ctx).Where("session_id = ? AND agent_id = ?", sessID, agentID).First(&model)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil // Not found = fresh start
		}
		return nil, errors.Wrap(result.Error, errors.CodeInternal, "load agent context snapshot")
	}

	return adapters.AgentContextSnapshotFromModel(&model), nil
}

// Delete removes snapshot by session+agent ID
func (r *AgentContextRepository) Delete(ctx context.Context, sessionID, agentID string) error {
	sessID, err := uuid.Parse(sessionID)
	if err != nil {
		return errors.Wrap(err, errors.CodeInvalidInput, "invalid session ID")
	}

	result := r.db.WithContext(ctx).Where("session_id = ? AND agent_id = ?", sessID, agentID).Delete(&models.AgentContextSnapshot{})
	if result.Error != nil {
		return errors.Wrap(result.Error, errors.CodeInternal, "delete agent context snapshot")
	}
	return nil
}

// FindActive returns all snapshots with status "active"
func (r *AgentContextRepository) FindActive(ctx context.Context) ([]*domain.AgentContextSnapshot, error) {
	var dbModels []models.AgentContextSnapshot
	result := r.db.WithContext(ctx).Where("status = ?", string(domain.AgentContextStatusActive)).Find(&dbModels)
	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.CodeInternal, "find active snapshots")
	}

	snapshots := make([]*domain.AgentContextSnapshot, 0, len(dbModels))
	for i := range dbModels {
		snapshots = append(snapshots, adapters.AgentContextSnapshotFromModel(&dbModels[i]))
	}
	return snapshots, nil
}
