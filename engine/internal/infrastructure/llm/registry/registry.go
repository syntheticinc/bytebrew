// Package registry provides a built-in catalog of known AI models and providers.
// The registry is read-only and backed by embedded Go maps (no database).
package registry

// ModelTier classifies models by capability level.
type ModelTier int

const (
	// Tier1 models are suitable for orchestrator agents with full agent capabilities.
	Tier1 ModelTier = 1
	// Tier2 models are suitable for sub-agents with simple tool calling.
	Tier2 ModelTier = 2
	// Tier3 models are utility-grade, suitable for classification but not agents.
	Tier3 ModelTier = 3
)

// ModelInfo describes a known AI model with its capabilities and pricing.
type ModelInfo struct {
	ID             string    `json:"id"`
	DisplayName    string    `json:"display_name"`
	Provider       string    `json:"provider"`
	Tier           ModelTier `json:"tier"`
	ContextWindow  int       `json:"context_window"`
	MaxOutput      int       `json:"max_output"`
	SupportsTools  bool      `json:"supports_tools"`
	SupportsVision bool      `json:"supports_vision"`
	PricingInput   float64   `json:"pricing_input"`
	PricingOutput  float64   `json:"pricing_output"`
	Description    string    `json:"description"`
	RecommendedFor []string  `json:"recommended_for"`
}

// ProviderInfo describes a known LLM provider.
type ProviderInfo struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	AuthType    string `json:"auth_type"`
	Website     string `json:"website"`
}

// ModelFilters controls which models are returned by ListModels.
type ModelFilters struct {
	Provider      string
	Tier          *ModelTier
	SupportsTools *bool
}

// Registry holds the built-in catalog of models and providers.
type Registry struct {
	providers map[string]ProviderInfo
	models    map[string]ModelInfo
}

// New creates a Registry populated with the built-in model and provider data.
func New() *Registry {
	return &Registry{
		providers: builtinProviders(),
		models:    builtinModels(),
	}
}

// ListProviders returns all known providers in no particular order.
func (r *Registry) ListProviders() []ProviderInfo {
	result := make([]ProviderInfo, 0, len(r.providers))
	for _, p := range r.providers {
		result = append(result, p)
	}
	return result
}

// ListModels returns models matching the given filters.
// An empty filter returns all models.
func (r *Registry) ListModels(filters ModelFilters) []ModelInfo {
	result := make([]ModelInfo, 0, len(r.models))
	for _, m := range r.models {
		if !matchesFilters(m, filters) {
			continue
		}
		result = append(result, m)
	}
	return result
}

// GetModel returns the model with the given ID, or nil if not found.
func (r *Registry) GetModel(id string) *ModelInfo {
	m, ok := r.models[id]
	if !ok {
		return nil
	}
	return &m
}

// GetProvider returns the provider with the given ID, or nil if not found.
func (r *Registry) GetProvider(id string) *ProviderInfo {
	p, ok := r.providers[id]
	if !ok {
		return nil
	}
	return &p
}

func matchesFilters(m ModelInfo, f ModelFilters) bool {
	if f.Provider != "" && m.Provider != f.Provider {
		return false
	}
	if f.Tier != nil && m.Tier != *f.Tier {
		return false
	}
	if f.SupportsTools != nil && m.SupportsTools != *f.SupportsTools {
		return false
	}
	return true
}
