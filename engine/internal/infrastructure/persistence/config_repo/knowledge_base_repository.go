package config_repo

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// GORMKnowledgeBaseRepository provides CRUD for knowledge bases and agent linking.
type GORMKnowledgeBaseRepository struct {
	db *gorm.DB
}

// NewGORMKnowledgeBaseRepository creates a new repository.
func NewGORMKnowledgeBaseRepository(db *gorm.DB) *GORMKnowledgeBaseRepository {
	return &GORMKnowledgeBaseRepository{db: db}
}

func (r *GORMKnowledgeBaseRepository) tenantID(ctx context.Context) string {
	tid := domain.TenantIDFromContext(ctx)
	if tid == "" {
		return "default"
	}
	return tid
}

// Create creates a new knowledge base.
func (r *GORMKnowledgeBaseRepository) Create(ctx context.Context, kb *models.KnowledgeBase) error {
	if err := r.db.WithContext(ctx).Create(kb).Error; err != nil {
		return fmt.Errorf("create knowledge base: %w", err)
	}
	return nil
}

// Update updates a knowledge base.
func (r *GORMKnowledgeBaseRepository) Update(ctx context.Context, kb *models.KnowledgeBase) error {
	if err := r.db.WithContext(ctx).Save(kb).Error; err != nil {
		return fmt.Errorf("update knowledge base: %w", err)
	}
	return nil
}

// GetByID returns a knowledge base by ID (tenant-scoped).
func (r *GORMKnowledgeBaseRepository) GetByID(ctx context.Context, id string) (*models.KnowledgeBase, error) {
	tenantID := r.tenantID(ctx)
	var kb models.KnowledgeBase
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		First(&kb).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get knowledge base: %w", err)
	}
	return &kb, nil
}

// List returns all knowledge bases for the current tenant.
func (r *GORMKnowledgeBaseRepository) List(ctx context.Context) ([]models.KnowledgeBase, error) {
	tenantID := r.tenantID(ctx)
	var kbs []models.KnowledgeBase
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		Find(&kbs).Error; err != nil {
		return nil, fmt.Errorf("list knowledge bases: %w", err)
	}
	return kbs, nil
}

// Delete removes a knowledge base and its join table entries (documents cascade via KB handler).
func (r *GORMKnowledgeBaseRepository) Delete(ctx context.Context, id string) error {
	tenantID := r.tenantID(ctx)

	// Remove agent links
	if err := r.db.WithContext(ctx).
		Where("knowledge_base_id = ?", id).
		Delete(&models.KnowledgeBaseAgent{}).Error; err != nil {
		return fmt.Errorf("delete KB agent links: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		Delete(&models.KnowledgeBase{}).Error; err != nil {
		return fmt.Errorf("delete knowledge base: %w", err)
	}
	return nil
}

// LinkAgent links an agent (by name) to a knowledge base.
func (r *GORMKnowledgeBaseRepository) LinkAgent(ctx context.Context, kbID, agentName string) error {
	link := models.KnowledgeBaseAgent{
		KnowledgeBaseID: kbID,
		AgentName:       agentName,
	}
	if err := r.db.WithContext(ctx).
		Where("knowledge_base_id = ? AND agent_name = ?", kbID, agentName).
		FirstOrCreate(&link).Error; err != nil {
		return fmt.Errorf("link agent to KB: %w", err)
	}
	return nil
}

// UnlinkAgent removes the link between an agent and a knowledge base.
func (r *GORMKnowledgeBaseRepository) UnlinkAgent(ctx context.Context, kbID, agentName string) error {
	if err := r.db.WithContext(ctx).
		Where("knowledge_base_id = ? AND agent_name = ?", kbID, agentName).
		Delete(&models.KnowledgeBaseAgent{}).Error; err != nil {
		return fmt.Errorf("unlink agent from KB: %w", err)
	}
	return nil
}

// ListLinkedAgents returns agent names linked to a knowledge base.
func (r *GORMKnowledgeBaseRepository) ListLinkedAgents(ctx context.Context, kbID string) ([]string, error) {
	var names []string
	if err := r.db.WithContext(ctx).
		Model(&models.KnowledgeBaseAgent{}).
		Where("knowledge_base_id = ?", kbID).
		Pluck("agent_name", &names).Error; err != nil {
		return nil, fmt.Errorf("list linked agents: %w", err)
	}
	return names, nil
}

// ListKBsByAgentName returns knowledge base IDs linked to an agent (by name).
func (r *GORMKnowledgeBaseRepository) ListKBsByAgentName(ctx context.Context, agentName string) ([]string, error) {
	var kbIDs []string
	if err := r.db.WithContext(ctx).
		Model(&models.KnowledgeBaseAgent{}).
		Where("agent_name = ?", agentName).
		Pluck("knowledge_base_id", &kbIDs).Error; err != nil {
		return nil, fmt.Errorf("list KBs by agent: %w", err)
	}
	return kbIDs, nil
}

// GetKBsWithEmbeddingModel returns knowledge bases with their embedding model info for an agent.
// Used by the knowledge_search tool to resolve per-KB embedding models.
func (r *GORMKnowledgeBaseRepository) GetKBsWithEmbeddingModel(ctx context.Context, agentName string) ([]models.KnowledgeBase, error) {
	kbIDs, err := r.ListKBsByAgentName(ctx, agentName)
	if err != nil || len(kbIDs) == 0 {
		return nil, err
	}
	tenantID := r.tenantID(ctx)
	var kbs []models.KnowledgeBase
	if err := r.db.WithContext(ctx).
		Where("id IN ? AND tenant_id = ?", kbIDs, tenantID).
		Find(&kbs).Error; err != nil {
		return nil, fmt.Errorf("get KBs with embedding model: %w", err)
	}
	return kbs, nil
}
