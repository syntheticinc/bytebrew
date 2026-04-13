package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	deliveryhttp "github.com/syntheticinc/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/llm"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/config_repo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	pkgerrors "github.com/syntheticinc/bytebrew/engine/pkg/errors"
)

// ModelCacheInvalidator allows invalidating cached model clients when models are modified.
type ModelCacheInvalidator interface {
	Invalidate(modelID string)
}

// modelServiceHTTPAdapter bridges GORMLLMProviderRepository to the http.ModelService interface.
type modelServiceHTTPAdapter struct {
	repo       *config_repo.GORMLLMProviderRepository
	modelCache ModelCacheInvalidator
}

func (m *modelServiceHTTPAdapter) ListModels(ctx context.Context) ([]deliveryhttp.ModelResponse, error) {
	providers, err := m.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}

	result := make([]deliveryhttp.ModelResponse, 0, len(providers))
	for _, p := range providers {
		result = append(result, deliveryhttp.ModelResponse{
			ID:           p.ID,
			Name:         p.Name,
			Type:         p.Type,
			BaseURL:      p.BaseURL,
			ModelName:    p.ModelName,
			HasAPIKey:    p.APIKeyEncrypted != "",
			APIVersion:   p.APIVersion,
			EmbeddingDim: p.EmbeddingDim,
			CreatedAt:    p.CreatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

func (m *modelServiceHTTPAdapter) CreateModel(ctx context.Context, req deliveryhttp.CreateModelRequest) (*deliveryhttp.ModelResponse, error) {
	provider := &models.LLMProviderModel{
		Name:            req.Name,
		Type:            req.Type,
		BaseURL:         req.BaseURL,
		ModelName:       req.ModelName,
		APIKeyEncrypted: req.APIKey,
		APIVersion:      req.APIVersion,
		EmbeddingDim:    req.EmbeddingDim,
	}

	if err := m.repo.Create(ctx, provider); err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, pkgerrors.AlreadyExists(fmt.Sprintf("model with name %q already exists", req.Name))
		}
		return nil, fmt.Errorf("create model: %w", err)
	}

	return &deliveryhttp.ModelResponse{
		ID:           provider.ID,
		Name:         provider.Name,
		Type:         provider.Type,
		BaseURL:      provider.BaseURL,
		ModelName:    provider.ModelName,
		HasAPIKey:    provider.APIKeyEncrypted != "",
		APIVersion:   provider.APIVersion,
		EmbeddingDim: provider.EmbeddingDim,
		CreatedAt:    provider.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (m *modelServiceHTTPAdapter) UpdateModel(ctx context.Context, name string, req deliveryhttp.CreateModelRequest) (*deliveryhttp.ModelResponse, error) {
	providers, err := m.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list models for update: %w", err)
	}

	var existing *models.LLMProviderModel
	for i := range providers {
		if providers[i].Name == name {
			existing = &providers[i]
			break
		}
	}
	if existing == nil {
		return nil, pkgerrors.NotFound(fmt.Sprintf("model not found: %s", name))
	}

	update := &models.LLMProviderModel{
		Name:         req.Name,
		Type:         req.Type,
		BaseURL:      req.BaseURL,
		ModelName:    req.ModelName,
		APIVersion:   req.APIVersion,
		EmbeddingDim: req.EmbeddingDim,
	}
	// Only update API key if provided (empty means keep existing).
	if req.APIKey != "" {
		update.APIKeyEncrypted = req.APIKey
	}

	if err := m.repo.Update(ctx, existing.ID, update); err != nil {
		return nil, fmt.Errorf("update model: %w", err)
	}

	// Invalidate cached client so next access picks up changes.
	if m.modelCache != nil {
		m.modelCache.Invalidate(existing.ID)
	}

	hasKey := existing.APIKeyEncrypted != ""
	if req.APIKey != "" {
		hasKey = true
	}

	respName := req.Name
	if respName == "" {
		respName = existing.Name
	}

	return &deliveryhttp.ModelResponse{
		ID:           existing.ID,
		Name:         respName,
		Type:         req.Type,
		BaseURL:      req.BaseURL,
		ModelName:    req.ModelName,
		HasAPIKey:    hasKey,
		APIVersion:   req.APIVersion,
		EmbeddingDim: req.EmbeddingDim,
		CreatedAt:    existing.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (m *modelServiceHTTPAdapter) DeleteModel(ctx context.Context, name string) error {
	providers, err := m.repo.List(ctx)
	if err != nil {
		return fmt.Errorf("list models for delete: %w", err)
	}

	for _, p := range providers {
		if p.Name == name {
			if err := m.repo.Delete(ctx, p.ID); err != nil {
				return err
			}
			if m.modelCache != nil {
				m.modelCache.Invalidate(p.ID)
			}
			return nil
		}
	}
	return pkgerrors.NotFound(fmt.Sprintf("model not found: %s", name))
}

func (m *modelServiceHTTPAdapter) VerifyModel(ctx context.Context, name string) (*deliveryhttp.ModelVerifyResult, error) {
	providers, err := m.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list models for verify: %w", err)
	}

	var dbModel *models.LLMProviderModel
	for i := range providers {
		if providers[i].Name == name {
			dbModel = &providers[i]
			break
		}
	}
	if dbModel == nil {
		return nil, pkgerrors.NotFound(fmt.Sprintf("model not found: %s", name))
	}

	client, err := llm.CreateClientFromDBModel(*dbModel)
	if err != nil {
		errMsg := fmt.Sprintf("failed to create client: %s", err.Error())
		return &deliveryhttp.ModelVerifyResult{
			Connectivity: "error",
			ToolCalling:  "skipped",
			ModelName:    dbModel.ModelName,
			Provider:     dbModel.Type,
			Error:        &errMsg,
		}, nil
	}

	verifyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	vr := llm.VerifyModel(verifyCtx, client, dbModel.ModelName, dbModel.Type)
	return &deliveryhttp.ModelVerifyResult{
		Connectivity:   vr.Connectivity,
		ToolCalling:    vr.ToolCalling,
		ResponseTimeMs: vr.ResponseTimeMs,
		ModelName:      vr.ModelName,
		Provider:       vr.Provider,
		Error:          vr.Error,
	}, nil
}
