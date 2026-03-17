package task

import (
	"context"
	"encoding/json"
	"fmt"
)

// TriggerProvider looks up webhook triggers by path.
type TriggerProvider interface {
	FindWebhookTrigger(ctx context.Context, path string) (*WebhookTrigger, error)
}

// WebhookTrigger represents a configured webhook trigger.
type WebhookTrigger struct {
	Title       string
	Description string
	AgentName   string
	Path        string
}

// WebhookService handles webhook trigger matching and task creation.
type WebhookService struct {
	triggers TriggerProvider
	creator  TaskCreator
}

// NewWebhookService creates a new WebhookService.
func NewWebhookService(triggers TriggerProvider, creator TaskCreator) *WebhookService {
	return &WebhookService{triggers: triggers, creator: creator}
}

// HandleWebhook matches a webhook path to a trigger and creates a task.
// Fields from the request body (title, description) override trigger defaults.
func (s *WebhookService) HandleWebhook(ctx context.Context, path string, body []byte) (uint, error) {
	trigger, err := s.triggers.FindWebhookTrigger(ctx, path)
	if err != nil {
		return 0, fmt.Errorf("find webhook trigger for path %q: %w", path, err)
	}

	title := trigger.Title
	description := trigger.Description

	var bodyData struct {
		Description string `json:"description"`
		Title       string `json:"title"`
	}
	if len(body) > 0 {
		if jsonErr := json.Unmarshal(body, &bodyData); jsonErr == nil {
			if bodyData.Description != "" {
				description = bodyData.Description
			}
			if bodyData.Title != "" {
				title = bodyData.Title
			}
		}
	}

	taskID, err := s.creator.CreateFromTrigger(ctx, TriggerTaskParams{
		Title:       title,
		Description: description,
		AgentName:   trigger.AgentName,
		Source:      "webhook",
		SourceID:    path,
	})
	if err != nil {
		return 0, fmt.Errorf("create task from webhook: %w", err)
	}
	return taskID, nil
}
