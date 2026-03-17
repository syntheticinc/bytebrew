package llm

import (
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

// ModelSelector selects a ChatModel based on FlowType.
// Allows different agent roles (Supervisor, Coder, Reviewer, Tester)
// to use different LLM models.
// Also supports named model resolution for per-agent model configuration.
type ModelSelector struct {
	models       map[domain.FlowType]model.ToolCallingChatModel
	defaultModel model.ToolCallingChatModel
	modelNames   map[domain.FlowType]string
	defaultName  string
	namedModels  map[string]model.ToolCallingChatModel
}

// NewModelSelector creates a new ModelSelector with a default model.
func NewModelSelector(defaultModel model.ToolCallingChatModel, defaultName string) *ModelSelector {
	return &ModelSelector{
		models:       make(map[domain.FlowType]model.ToolCallingChatModel),
		defaultModel: defaultModel,
		modelNames:   make(map[domain.FlowType]string),
		defaultName:  defaultName,
		namedModels:  make(map[string]model.ToolCallingChatModel),
	}
}

// SetModel sets a specific model for a given flow type.
func (s *ModelSelector) SetModel(flowType domain.FlowType, m model.ToolCallingChatModel, name string) {
	s.models[flowType] = m
	s.modelNames[flowType] = name
}

// Select returns the ChatModel for the given flow type.
// Falls back to default if no specific model is configured.
func (s *ModelSelector) Select(flowType domain.FlowType) model.ToolCallingChatModel {
	if m, ok := s.models[flowType]; ok {
		return m
	}
	return s.defaultModel
}

// ModelName returns the model name for the given flow type.
// Falls back to default name if no specific name is configured.
func (s *ModelSelector) ModelName(flowType domain.FlowType) string {
	if name, ok := s.modelNames[flowType]; ok {
		return name
	}
	return s.defaultName
}

// RegisterNamedModel registers a model under a given name for per-agent resolution.
// Agents configured with a model name (e.g., "llama-4") can resolve it via ResolveByName.
func (s *ModelSelector) RegisterNamedModel(name string, m model.ToolCallingChatModel) {
	s.namedModels[name] = m
}

// ResolveByName returns a model registered under the given name.
// Returns an error if the name is not found.
func (s *ModelSelector) ResolveByName(name string) (model.ToolCallingChatModel, error) {
	m, ok := s.namedModels[name]
	if !ok {
		return nil, fmt.Errorf("named model %q not registered", name)
	}
	return m, nil
}

// NamedModelCount returns the number of registered named models.
func (s *ModelSelector) NamedModelCount() int {
	return len(s.namedModels)
}
