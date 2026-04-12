package app

import (
	"context"
	"encoding/json"
	"fmt"

	deliveryhttp "github.com/syntheticinc/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/indexing"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools"
	svcknowledge "github.com/syntheticinc/bytebrew/engine/internal/service/knowledge"
	"gorm.io/gorm"
)

// knowledgeUploadHTTPAdapter bridges svcknowledge.UploadService to deliveryhttp.KnowledgeFileUploader.
type knowledgeUploadHTTPAdapter struct {
	svc *svcknowledge.UploadService
}

func (a *knowledgeUploadHTTPAdapter) UploadFile(ctx context.Context, tenantID, agentName, fileName, fileType string, fileSize int64, fileHash string, content []byte) (*deliveryhttp.KnowledgeFileResponse, error) {
	resp, err := a.svc.UploadFile(ctx, tenantID, agentName, fileName, fileType, fileSize, fileHash, content)
	if err != nil {
		return nil, err
	}
	return &deliveryhttp.KnowledgeFileResponse{
		ID:         resp.ID,
		FileName:   resp.FileName,
		FileType:   resp.FileType,
		FileSize:   resp.FileSize,
		Status:     resp.Status,
		StatusMsg:  resp.StatusMsg,
		ChunkCount: resp.ChunkCount,
		CreatedAt:  resp.CreatedAt,
		IndexedAt:  resp.IndexedAt,
	}, nil
}

// knowledgeFileListerHTTPAdapter bridges svcknowledge.UploadService to deliveryhttp.KnowledgeFileLister.
type knowledgeFileListerHTTPAdapter struct {
	svc *svcknowledge.UploadService
}

func (a *knowledgeFileListerHTTPAdapter) ListFiles(ctx context.Context, agentName string) ([]deliveryhttp.KnowledgeFileResponse, error) {
	files, err := a.svc.ListFiles(ctx, agentName)
	if err != nil {
		return nil, err
	}
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
	return result, nil
}

func (a *knowledgeFileListerHTTPAdapter) DeleteFile(ctx context.Context, agentName, fileID string) error {
	return a.svc.DeleteFile(ctx, agentName, fileID)
}

func (a *knowledgeFileListerHTTPAdapter) ReindexFile(ctx context.Context, agentName, fileID string) error {
	return a.svc.ReindexFile(ctx, agentName, fileID)
}

// embeddingModelResolver resolves the embedding model from an agent's knowledge capability config.
// Implements svcknowledge.EmbeddingModelResolver.
type embeddingModelResolver struct {
	db *gorm.DB
}

// knowledgeEmbedderResolverAdapter bridges embeddingModelResolver to tools.KnowledgeEmbedderResolver.
// Resolves per-agent embedding model from capability config for knowledge_search tool at runtime.
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

func (r *embeddingModelResolver) ResolveEmbeddingModel(ctx context.Context, agentName string) (*svcknowledge.EmbeddingModelInfo, error) {
	// Find agent ID
	var agentID string
	if err := r.db.WithContext(ctx).
		Raw("SELECT id FROM agents WHERE name = ?", agentName).
		Scan(&agentID).Error; err != nil || agentID == "" {
		return nil, fmt.Errorf("agent %q not found", agentName)
	}

	// Find knowledge capability config
	var cap models.CapabilityModel
	if err := r.db.WithContext(ctx).
		Where("agent_id = ? AND type = ?", agentID, "knowledge").
		First(&cap).Error; err != nil {
		return nil, fmt.Errorf("no knowledge capability for agent %q", agentName)
	}

	// Parse config to get embedding_model_id
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

	// Load embedding model from DB
	var llm models.LLMProviderModel
	if err := r.db.WithContext(ctx).Where("id = ? AND type = ?", embModelID, "embedding").First(&llm).Error; err != nil {
		return nil, fmt.Errorf("embedding model %q not found or not type=embedding", embModelID)
	}

	return &svcknowledge.EmbeddingModelInfo{
		BaseURL:      llm.BaseURL,
		APIKey:       llm.APIKeyEncrypted,
		ModelName:    llm.ModelName,
		EmbeddingDim: llm.EmbeddingDim,
	}, nil
}

