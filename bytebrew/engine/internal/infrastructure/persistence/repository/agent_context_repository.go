package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/adapters"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/errors"
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

// Save performs upsert by (session_id, agent_id) — one snapshot per agent per session
func (r *AgentContextRepository) Save(ctx context.Context, snapshot *domain.AgentContextSnapshot) error {
	model := adapters.AgentContextSnapshotToModel(snapshot)
	if model.ID == "" {
		model.ID = uuid.New().String()
	}
	model.UpdatedAt = time.Now()

	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "session_id"}, {Name: "agent_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"context_data", "step_number", "token_count", "status", "updated_at", "schema_version",
		}),
	}).Create(model)

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.CodeInternal, "save agent context snapshot")
	}

	snapshot.ID = model.ID
	return nil
}

// Load loads snapshot by session+agent ID
func (r *AgentContextRepository) Load(ctx context.Context, sessionID, agentID string) (*domain.AgentContextSnapshot, error) {
	var model models.RuntimeAgentContextModel
	result := r.db.WithContext(ctx).Where("session_id = ? AND agent_id = ?", sessionID, agentID).First(&model)
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
	result := r.db.WithContext(ctx).Where("session_id = ? AND agent_id = ?", sessionID, agentID).Delete(&models.RuntimeAgentContextModel{})
	if result.Error != nil {
		return errors.Wrap(result.Error, errors.CodeInternal, "delete agent context snapshot")
	}
	return nil
}

// FindActive returns all snapshots with status "active"
func (r *AgentContextRepository) FindActive(ctx context.Context) ([]*domain.AgentContextSnapshot, error) {
	var dbModels []models.RuntimeAgentContextModel
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
