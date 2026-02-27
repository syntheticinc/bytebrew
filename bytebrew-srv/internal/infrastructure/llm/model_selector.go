package llm

import (
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/cloudwego/eino/components/model"
)

// ModelSelector selects a ChatModel based on FlowType.
// Allows different agent roles (Supervisor, Coder, Reviewer, Tester)
// to use different LLM models.
type ModelSelector struct {
	models       map[domain.FlowType]model.ToolCallingChatModel
	defaultModel model.ToolCallingChatModel
	modelNames   map[domain.FlowType]string
	defaultName  string
}

// NewModelSelector creates a new ModelSelector with a default model.
func NewModelSelector(defaultModel model.ToolCallingChatModel, defaultName string) *ModelSelector {
	return &ModelSelector{
		models:       make(map[domain.FlowType]model.ToolCallingChatModel),
		defaultModel: defaultModel,
		modelNames:   make(map[domain.FlowType]string),
		defaultName:  defaultName,
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
