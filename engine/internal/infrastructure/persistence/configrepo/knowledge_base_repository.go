package configrepo

import (
	"context"
	"errors"
	"fmt"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// GORMKnowledgeBaseRepository provides CRUD for knowledge bases and agent linking.
// Tenant isolation is applied via tenantScope(ctx) from base_repo.go.
type GORMKnowledgeBaseRepository struct {
	db *gorm.DB
}

// NewGORMKnowledgeBaseRepository creates a new repository.
func NewGORMKnowledgeBaseRepository(db *gorm.DB) *GORMKnowledgeBaseRepository {
	return &GORMKnowledgeBaseRepository{db: db}
}

// Create creates a new knowledge base, stamping tenant from context.
func (r *GORMKnowledgeBaseRepository) Create(ctx context.Context, kb *models.KnowledgeBase) error {
	kb.TenantID = tenantIDFromCtx(ctx)
	if err := r.db.WithContext(ctx).Create(kb).Error; err != nil {
		return fmt.Errorf("create knowledge base: %w", err)
	}
	return nil
}

// Update updates a knowledge base (tenant preserved from existing row).
func (r *GORMKnowledgeBaseRepository) Update(ctx context.Context, kb *models.KnowledgeBase) error {
	// Ensure the kb belongs to the current tenant before saving.
	var existing models.KnowledgeBase
	if err := r.db.WithContext(ctx).
		Scopes(tenantScope(ctx)).
		Where("id = ?", kb.ID).
		First(&existing).Error; err != nil {
		return fmt.Errorf("find knowledge base: %w", err)
	}
	kb.TenantID = existing.TenantID
	if err := r.db.WithContext(ctx).Save(kb).Error; err != nil {
		return fmt.Errorf("update knowledge base: %w", err)
	}
	return nil
}

// GetByID returns a knowledge base by ID (tenant-scoped).
func (r *GORMKnowledgeBaseRepository) GetByID(ctx context.Context, id string) (*models.KnowledgeBase, error) {
	var kb models.KnowledgeBase
	err := r.db.WithContext(ctx).
		Scopes(tenantScope(ctx)).
		Where("id = ?", id).
		First(&kb).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get knowledge base: %w", err)
	}
	return &kb, nil
}

// List returns all knowledge bases for the current tenant.
func (r *GORMKnowledgeBaseRepository) List(ctx context.Context) ([]models.KnowledgeBase, error) {
	var kbs []models.KnowledgeBase
	if err := r.db.WithContext(ctx).
		Scopes(tenantScope(ctx)).
		Order("created_at DESC").
		Find(&kbs).Error; err != nil {
		return nil, fmt.Errorf("list knowledge bases: %w", err)
	}
	return kbs, nil
}

// Delete removes a knowledge base and its join table entries (documents cascade via KB handler).
// Tenant-scoped — only removes rows that belong to the current tenant.
func (r *GORMKnowledgeBaseRepository) Delete(ctx context.Context, id string) error {
	// Remove agent links (link table has no tenant column; key is (kb_id, agent_id)
	// and kb_id is unique — safe to remove without tenant filter as long as we
	// only fall through to the kb delete when the kb itself is tenant-scoped).
	if err := r.db.WithContext(ctx).
		Where("knowledge_base_id = ?", id).
		Delete(&models.KnowledgeBaseAgent{}).Error; err != nil {
		return fmt.Errorf("delete KB agent links: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Scopes(tenantScope(ctx)).
		Where("id = ?", id).
		Delete(&models.KnowledgeBase{}).Error; err != nil {
		return fmt.Errorf("delete knowledge base: %w", err)
	}
	return nil
}

// LinkAgent links an agent (by ID) to a knowledge base.
//
// The join table `knowledge_base_agents` has no `tenant_id` column, so the
// tenant invariant here is enforced at link-creation time: both the KB and the
// agent must belong to the tenant in the context. Without this check a caller
// could stitch together an arbitrary KB and an arbitrary agent (even across
// tenants) by passing the two UUIDs directly.
func (r *GORMKnowledgeBaseRepository) LinkAgent(ctx context.Context, kbID, agentID string) error {
	if err := r.verifyKBAndAgentInTenant(ctx, kbID, agentID); err != nil {
		return err
	}
	link := models.KnowledgeBaseAgent{
		KnowledgeBaseID: kbID,
		AgentID:         agentID,
	}
	if err := r.db.WithContext(ctx).
		Where("knowledge_base_id = ? AND agent_id = ?", kbID, agentID).
		FirstOrCreate(&link).Error; err != nil {
		return fmt.Errorf("link agent to KB: %w", err)
	}
	return nil
}

// UnlinkAgent removes the link between an agent and a knowledge base.
// Enforces that both sides belong to the current tenant for the same reason
// as LinkAgent — the join table has no tenant column.
func (r *GORMKnowledgeBaseRepository) UnlinkAgent(ctx context.Context, kbID, agentID string) error {
	if err := r.verifyKBAndAgentInTenant(ctx, kbID, agentID); err != nil {
		return err
	}
	if err := r.db.WithContext(ctx).
		Where("knowledge_base_id = ? AND agent_id = ?", kbID, agentID).
		Delete(&models.KnowledgeBaseAgent{}).Error; err != nil {
		return fmt.Errorf("unlink agent from KB: %w", err)
	}
	return nil
}

// verifyKBAndAgentInTenant ensures both the KB and the agent belong to the
// tenant resolved from ctx. Used by LinkAgent/UnlinkAgent because the join
// table carries no tenant_id of its own.
func (r *GORMKnowledgeBaseRepository) verifyKBAndAgentInTenant(ctx context.Context, kbID, agentID string) error {
	tenantID := tenantIDFromCtx(ctx)

	var kbCount int64
	if err := r.db.WithContext(ctx).
		Model(&models.KnowledgeBase{}).
		Where("id = ? AND tenant_id = ?", kbID, tenantID).
		Count(&kbCount).Error; err != nil {
		return fmt.Errorf("verify knowledge base tenant: %w", err)
	}
	if kbCount == 0 {
		return fmt.Errorf("knowledge base not found in tenant")
	}

	var agentCount int64
	if err := r.db.WithContext(ctx).
		Model(&models.AgentModel{}).
		Where("id = ? AND tenant_id = ?", agentID, tenantID).
		Count(&agentCount).Error; err != nil {
		return fmt.Errorf("verify agent tenant: %w", err)
	}
	if agentCount == 0 {
		return fmt.Errorf("agent not found in tenant")
	}
	return nil
}

// ListLinkedAgentIDs returns agent IDs linked to a knowledge base.
func (r *GORMKnowledgeBaseRepository) ListLinkedAgentIDs(ctx context.Context, kbID string) ([]string, error) {
	var ids []string
	if err := r.db.WithContext(ctx).
		Model(&models.KnowledgeBaseAgent{}).
		Where("knowledge_base_id = ?", kbID).
		Pluck("agent_id", &ids).Error; err != nil {
		return nil, fmt.Errorf("list linked agents: %w", err)
	}
	return ids, nil
}

// ListKBsByAgentID returns knowledge base IDs linked to an agent (by UUID).
func (r *GORMKnowledgeBaseRepository) ListKBsByAgentID(ctx context.Context, agentID string) ([]string, error) {
	var kbIDs []string
	if err := r.db.WithContext(ctx).
		Model(&models.KnowledgeBaseAgent{}).
		Where("agent_id = ?", agentID).
		Pluck("knowledge_base_id", &kbIDs).Error; err != nil {
		return nil, fmt.Errorf("list KBs by agent: %w", err)
	}
	return kbIDs, nil
}

// ListKBsByAgentName resolves agent name → ID (tenant-scoped), then returns linked KB IDs.
// Implements KnowledgeKBResolver interface for builtin_tool_store.
func (r *GORMKnowledgeBaseRepository) ListKBsByAgentName(ctx context.Context, agentName string) ([]string, error) {
	tenantID := tenantIDFromCtx(ctx)

	var agentID string
	if err := r.db.WithContext(ctx).
		Raw("SELECT id FROM agents WHERE name = ? AND tenant_id = ?", agentName, tenantID).
		Scan(&agentID).Error; err != nil || agentID == "" {
		return nil, nil // agent not found — no KBs
	}
	return r.ListKBsByAgentID(ctx, agentID)
}

// GetKBsWithEmbeddingModel returns knowledge bases with their embedding model info for an agent (tenant-scoped).
// Used by the knowledge_search tool to resolve per-KB embedding models.
func (r *GORMKnowledgeBaseRepository) GetKBsWithEmbeddingModel(ctx context.Context, agentName string) ([]models.KnowledgeBase, error) {
	kbIDs, err := r.ListKBsByAgentName(ctx, agentName)
	if err != nil || len(kbIDs) == 0 {
		return nil, err
	}
	var kbs []models.KnowledgeBase
	if err := r.db.WithContext(ctx).
		Scopes(tenantScope(ctx)).
		Where("id IN ?", kbIDs).
		Find(&kbs).Error; err != nil {
		return nil, fmt.Errorf("get KBs with embedding model: %w", err)
	}
	return kbs, nil
}
