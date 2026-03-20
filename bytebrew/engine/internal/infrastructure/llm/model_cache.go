package llm

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"gorm.io/gorm"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
)

// ModelCache provides thread-safe caching of LLM clients resolved from the database.
// Clients are created lazily on first access and cached until explicitly invalidated.
type ModelCache struct {
	mu      sync.RWMutex
	clients map[uint]*cachedModel
	db      *gorm.DB
}

type cachedModel struct {
	client    model.ToolCallingChatModel
	name      string
	createdAt time.Time
}

// NewModelCache creates a new ModelCache backed by the given database.
func NewModelCache(db *gorm.DB) *ModelCache {
	return &ModelCache{
		clients: make(map[uint]*cachedModel),
		db:      db,
	}
}

// Get returns a cached model client or creates one from the database.
// Returns the client, the model display name, and any error.
func (c *ModelCache) Get(ctx context.Context, modelID uint) (model.ToolCallingChatModel, string, error) {
	c.mu.RLock()
	if cached, ok := c.clients[modelID]; ok {
		c.mu.RUnlock()
		return cached.client, cached.name, nil
	}
	c.mu.RUnlock()

	var dbModel models.LLMProviderModel
	if err := c.db.WithContext(ctx).First(&dbModel, modelID).Error; err != nil {
		return nil, "", fmt.Errorf("model ID %d not found: %w", modelID, err)
	}

	client, err := CreateClientFromDBModel(dbModel)
	if err != nil {
		return nil, "", fmt.Errorf("create client for model %q: %w", dbModel.Name, err)
	}

	c.mu.Lock()
	c.clients[modelID] = &cachedModel{
		client:    client,
		name:      dbModel.ModelName,
		createdAt: time.Now(),
	}
	c.mu.Unlock()

	slog.InfoContext(ctx, "model client created and cached",
		"model_id", modelID, "name", dbModel.Name, "model", dbModel.ModelName)

	return client, dbModel.ModelName, nil
}

// Invalidate removes a cached model client, forcing re-creation on next access.
func (c *ModelCache) Invalidate(modelID uint) {
	c.mu.Lock()
	delete(c.clients, modelID)
	c.mu.Unlock()

	slog.Info("model cache invalidated", "model_id", modelID)
}

// InvalidateAll clears the entire cache.
func (c *ModelCache) InvalidateAll() {
	c.mu.Lock()
	c.clients = make(map[uint]*cachedModel)
	c.mu.Unlock()

	slog.Info("model cache fully invalidated")
}

// anthropicTransport adds the required anthropic-version header to all requests.
type anthropicCacheTransport struct {
	base http.RoundTripper
}

func (t *anthropicCacheTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("anthropic-version", "2023-06-01")
	return t.base.RoundTrip(req)
}

// CreateClientFromDBModel creates a ToolCallingChatModel from a database LLMProviderModel record.
func CreateClientFromDBModel(m models.LLMProviderModel) (model.ToolCallingChatModel, error) {
	ctx := context.Background()

	switch m.Type {
	case "ollama":
		baseURL := m.BaseURL
		if strings.HasSuffix(baseURL, "/api") {
			baseURL = strings.TrimSuffix(baseURL, "/api") + "/v1"
		}
		if !strings.Contains(baseURL, "/v1") {
			baseURL = strings.TrimRight(baseURL, "/") + "/v1"
		}
		cfg := &openai.ChatModelConfig{
			BaseURL: baseURL,
			Model:   m.ModelName,
			APIKey:  "ollama",
		}
		return openai.NewChatModel(ctx, cfg)

	case "openai", "openai_compatible":
		cfg := &openai.ChatModelConfig{
			BaseURL: m.BaseURL,
			Model:   m.ModelName,
			APIKey:  m.APIKeyEncrypted,
		}
		return openai.NewChatModel(ctx, cfg)

	case "anthropic":
		baseURL := "https://api.anthropic.com/v1"
		if m.BaseURL != "" {
			baseURL = m.BaseURL
		}
		httpClient := &http.Client{}
		httpClient.Transport = &anthropicCacheTransport{
			base: http.DefaultTransport,
		}
		cfg := &openai.ChatModelConfig{
			BaseURL:    baseURL,
			Model:      m.ModelName,
			APIKey:     m.APIKeyEncrypted,
			HTTPClient: httpClient,
		}
		return openai.NewChatModel(ctx, cfg)

	default:
		return nil, fmt.Errorf("unsupported provider type: %s", m.Type)
	}
}
