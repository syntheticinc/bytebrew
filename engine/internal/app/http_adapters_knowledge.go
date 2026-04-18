package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	deliveryhttp "github.com/syntheticinc/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/indexing"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/configrepo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools"
	svcknowledge "github.com/syntheticinc/bytebrew/engine/internal/service/knowledge"
	"gorm.io/gorm"
)

// --- Legacy agent-scoped adapters (kept for backward compatibility) ---

// knowledgeUploadHTTPAdapter bridges svcknowledge.UploadService to deliveryhttp.KnowledgeFileUploader.
type knowledgeUploadHTTPAdapter struct {
	svc *svcknowledge.UploadService
}

func (a *knowledgeUploadHTTPAdapter) UploadFile(ctx context.Context, tenantID, agentName, fileName, fileType string, fileSize int64, fileHash string, content []byte) (*deliveryhttp.KnowledgeFileResponse, error) {
	resp, err := a.svc.UploadFile(ctx, tenantID, agentName, fileName, fileType, fileSize, fileHash, content)
	if err != nil {
		return nil, err
	}
	return svcFileToHTTP(resp), nil
}

// knowledgeFileListerHTTPAdapter bridges svcknowledge.UploadService to deliveryhttp.KnowledgeFileLister.
// Uses KB-scoped queries by resolving agent name → linked KB IDs.
type knowledgeFileListerHTTPAdapter struct {
	svc    *svcknowledge.UploadService
	kbRepo *configrepo.GORMKnowledgeBaseRepository
}

func (a *knowledgeFileListerHTTPAdapter) ListFiles(ctx context.Context, agentName string) ([]deliveryhttp.KnowledgeFileResponse, error) {
	kbIDs, err := a.kbRepo.ListKBsByAgentName(ctx, agentName)
	if err != nil || len(kbIDs) == 0 {
		return []deliveryhttp.KnowledgeFileResponse{}, nil
	}
	var allFiles []deliveryhttp.KnowledgeFileResponse
	for _, kbID := range kbIDs {
		files, err := a.svc.ListFilesByKB(ctx, kbID)
		if err != nil {
			return nil, err
		}
		allFiles = append(allFiles, svcFilesToHTTP(files)...)
	}
	return allFiles, nil
}

func (a *knowledgeFileListerHTTPAdapter) DeleteFile(ctx context.Context, agentName, fileID string) error {
	kbIDs, err := a.kbRepo.ListKBsByAgentName(ctx, agentName)
	if err != nil || len(kbIDs) == 0 {
		return fmt.Errorf("file not found")
	}
	for _, kbID := range kbIDs {
		if err := a.svc.DeleteFileByKB(ctx, kbID, fileID); err == nil {
			return nil
		}
	}
	return fmt.Errorf("file not found")
}

func (a *knowledgeFileListerHTTPAdapter) ReindexFile(ctx context.Context, agentName, fileID string) error {
	// Legacy reindex not supported without KB context
	return fmt.Errorf("reindex requires KB-scoped endpoint")
}

// --- KB-scoped adapters (new many-to-many architecture) ---

// kbStoreAdapter bridges GORMKnowledgeBaseRepository to deliveryhttp.KBStore.
type kbStoreAdapter struct {
	repo *configrepo.GORMKnowledgeBaseRepository
	db   *gorm.DB // for counting files and resolving agents
}

func (a *kbStoreAdapter) Create(ctx context.Context, name, description, embeddingModelID, tenantID string) (*deliveryhttp.KnowledgeBaseInfo, error) {
	kb := &models.KnowledgeBase{
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if embeddingModelID != "" {
		kb.EmbeddingModelID = &embeddingModelID
	}
	if err := a.repo.Create(ctx, kb); err != nil {
		return nil, err
	}
	return a.toInfo(ctx, kb)
}

func (a *kbStoreAdapter) Update(ctx context.Context, id, name, description, embeddingModelID string) (*deliveryhttp.KnowledgeBaseInfo, error) {
	kb, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if kb == nil {
		return nil, nil
	}
	kb.Name = name
	kb.Description = description
	if embeddingModelID != "" {
		kb.EmbeddingModelID = &embeddingModelID
	} else {
		kb.EmbeddingModelID = nil
	}
	kb.UpdatedAt = time.Now()
	if err := a.repo.Update(ctx, kb); err != nil {
		return nil, err
	}
	return a.toInfo(ctx, kb)
}

func (a *kbStoreAdapter) GetByID(ctx context.Context, id string) (*deliveryhttp.KnowledgeBaseInfo, error) {
	kb, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if kb == nil {
		return nil, nil
	}
	return a.toInfo(ctx, kb)
}

func (a *kbStoreAdapter) List(ctx context.Context) ([]deliveryhttp.KnowledgeBaseInfo, error) {
	kbs, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]deliveryhttp.KnowledgeBaseInfo, 0, len(kbs))
	for i := range kbs {
		info, err := a.toInfo(ctx, &kbs[i])
		if err != nil {
			continue
		}
		result = append(result, *info)
	}
	return result, nil
}

func (a *kbStoreAdapter) Delete(ctx context.Context, id string) error {
	return a.repo.Delete(ctx, id)
}

func (a *kbStoreAdapter) LinkAgent(ctx context.Context, kbID, agentName string) error {
	// Resolve agent name → ID
	agentID, err := a.resolveAgentID(ctx, agentName)
	if err != nil {
		return err
	}
	return a.repo.LinkAgent(ctx, kbID, agentID)
}

func (a *kbStoreAdapter) UnlinkAgent(ctx context.Context, kbID, agentName string) error {
	// Resolve agent name → ID
	agentID, err := a.resolveAgentID(ctx, agentName)
	if err != nil {
		return err
	}
	return a.repo.UnlinkAgent(ctx, kbID, agentID)
}

func (a *kbStoreAdapter) resolveAgentID(ctx context.Context, agentName string) (string, error) {
	var agentID string
	if err := a.db.WithContext(ctx).
		Raw("SELECT id FROM agents WHERE name = ?", agentName).
		Scan(&agentID).Error; err != nil || agentID == "" {
		return "", fmt.Errorf("agent %q not found", agentName)
	}
	return agentID, nil
}

func (a *kbStoreAdapter) toInfo(ctx context.Context, kb *models.KnowledgeBase) (*deliveryhttp.KnowledgeBaseInfo, error) {
	agentIDs, _ := a.repo.ListLinkedAgentIDs(ctx, kb.ID)
	// Resolve agent IDs to names for the API response
	agents := make([]string, 0, len(agentIDs))
	for _, id := range agentIDs {
		var name string
		if err := a.db.WithContext(ctx).
			Raw("SELECT name FROM agents WHERE id = ?", id).
			Scan(&name).Error; err == nil && name != "" {
			agents = append(agents, name)
		}
	}

	var fileCount int64
	a.db.WithContext(ctx).Model(&models.KnowledgeDocument{}).
		Where("knowledge_base_id = ?", kb.ID).Count(&fileCount)

	embModelID := ""
	if kb.EmbeddingModelID != nil {
		embModelID = *kb.EmbeddingModelID
	}

	return &deliveryhttp.KnowledgeBaseInfo{
		ID:               kb.ID,
		Name:             kb.Name,
		Description:      kb.Description,
		EmbeddingModelID: embModelID,
		FileCount:        int(fileCount),
		LinkedAgents:     agents,
		CreatedAt:        kb.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        kb.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// kbFileManagerAdapter bridges svcknowledge.UploadService to deliveryhttp.KBFileManager.
type kbFileManagerAdapter struct {
	svc *svcknowledge.UploadService
}

func (a *kbFileManagerAdapter) ListFiles(ctx context.Context, kbID string) ([]deliveryhttp.KnowledgeFileResponse, error) {
	files, err := a.svc.ListFilesByKB(ctx, kbID)
	if err != nil {
		return nil, err
	}
	return svcFilesToHTTP(files), nil
}

func (a *kbFileManagerAdapter) UploadFile(ctx context.Context, tenantID, kbID, embeddingModelID, fileName, fileType string, fileSize int64, fileHash string, content []byte) (*deliveryhttp.KnowledgeFileResponse, error) {
	resp, err := a.svc.UploadFileToKB(ctx, tenantID, kbID, embeddingModelID, fileName, fileType, fileSize, fileHash, content)
	if err != nil {
		return nil, err
	}
	return svcFileToHTTP(resp), nil
}

func (a *kbFileManagerAdapter) DeleteFile(ctx context.Context, kbID, fileID string) error {
	return a.svc.DeleteFileByKB(ctx, kbID, fileID)
}

func (a *kbFileManagerAdapter) ReindexFile(ctx context.Context, kbID, embeddingModelID, fileID string) error {
	return a.svc.ReindexFileByKB(ctx, kbID, embeddingModelID, fileID)
}

func (a *kbFileManagerAdapter) DeleteAllFiles(ctx context.Context, kbID string) error {
	return a.svc.DeleteAllByKB(ctx, kbID)
}

// --- Embedding model resolvers ---

// embeddingModelResolver resolves the embedding model from an agent's knowledge capability config.
type embeddingModelResolver struct {
	db *gorm.DB
}

func (r *embeddingModelResolver) ResolveEmbeddingModel(ctx context.Context, agentName string) (*svcknowledge.EmbeddingModelInfo, error) {
	var agentID string
	if err := r.db.WithContext(ctx).
		Raw("SELECT id FROM agents WHERE name = ?", agentName).
		Scan(&agentID).Error; err != nil || agentID == "" {
		return nil, fmt.Errorf("agent %q not found", agentName)
	}

	var cap models.CapabilityModel
	if err := r.db.WithContext(ctx).
		Where("agent_id = ? AND type = ?", agentID, "knowledge").
		First(&cap).Error; err != nil {
		return nil, fmt.Errorf("no knowledge capability for agent %q", agentName)
	}

	var config map[string]interface{}
	if cap.Config != "" {
		if err := json.Unmarshal([]byte(cap.Config), &config); err != nil {
			return nil, fmt.Errorf("parse capability config: %w", err)
		}
	}

	embModelID, _ := config["embedding_model_id"].(string)
	if embModelID == "" {
		return nil, fmt.Errorf("no embedding_model_id in knowledge config")
	}

	return resolveEmbeddingModelByID(r.db, ctx, embModelID)
}

// kbEmbeddingResolver resolves embedding model from a model ID (for KB-scoped operations).
type kbEmbeddingResolver struct {
	db *gorm.DB
}

func (r *kbEmbeddingResolver) ResolveByModelID(ctx context.Context, modelID string) (*svcknowledge.EmbeddingModelInfo, error) {
	return resolveEmbeddingModelByID(r.db, ctx, modelID)
}

// resolveEmbeddingModelByID loads an embedding model from the DB by its ID.
// DBML models.type enum is {ollama, openai_compatible, anthropic, azure_openai}
// and does NOT include "embedding". Embedding-capable models are identified
// by a positive config.embedding_dim jsonb field instead.
func resolveEmbeddingModelByID(db *gorm.DB, ctx context.Context, modelID string) (*svcknowledge.EmbeddingModelInfo, error) {
	var llm models.LLMProviderModel
	if err := db.WithContext(ctx).
		Where("id = ? AND (config->>'embedding_dim')::int > 0", modelID).
		First(&llm).Error; err != nil {
		return nil, fmt.Errorf("embedding model %q not found or config.embedding_dim not set", modelID)
	}
	return &svcknowledge.EmbeddingModelInfo{
		BaseURL:      llm.BaseURL,
		APIKey:       llm.APIKeyEncrypted,
		ModelName:    llm.ModelName,
		EmbeddingDim: llm.EmbeddingDim(),
	}, nil
}

// knowledgeEmbedderResolverAdapter bridges embeddingModelResolver to tools.KnowledgeEmbedderResolver.
type knowledgeEmbedderResolverAdapter struct {
	resolver *embeddingModelResolver
}

func (a *knowledgeEmbedderResolverAdapter) ResolveEmbedder(ctx context.Context, agentName string) (tools.KnowledgeEmbedder, error) {
	info, err := a.resolver.ResolveEmbeddingModel(ctx, agentName)
	if err != nil || info == nil {
		return nil, err
	}
	return indexing.NewOpenAIEmbeddingsClient(info.BaseURL, info.APIKey, info.ModelName, info.EmbeddingDim), nil
}

// --- Conversion helpers ---

func svcFileToHTTP(f *svcknowledge.FileResponse) *deliveryhttp.KnowledgeFileResponse {
	return &deliveryhttp.KnowledgeFileResponse{
		ID:         f.ID,
		FileName:   f.FileName,
		FileType:   f.FileType,
		FileSize:   f.FileSize,
		Status:     f.Status,
		StatusMsg:  f.StatusMsg,
		ChunkCount: f.ChunkCount,
		CreatedAt:  f.CreatedAt,
		IndexedAt:  f.IndexedAt,
	}
}

func svcFilesToHTTP(files []svcknowledge.FileResponse) []deliveryhttp.KnowledgeFileResponse {
	result := make([]deliveryhttp.KnowledgeFileResponse, len(files))
	for i, f := range files {
		result[i] = deliveryhttp.KnowledgeFileResponse{
			ID:         f.ID,
			FileName:   f.FileName,
			FileType:   f.FileType,
			FileSize:   f.FileSize,
			Status:     f.Status,
			StatusMsg:  f.StatusMsg,
			ChunkCount: f.ChunkCount,
			CreatedAt:  f.CreatedAt,
			IndexedAt:  f.IndexedAt,
		}
	}
	return result
}
