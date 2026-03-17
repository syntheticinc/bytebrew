package task

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTriggerProvider struct {
	triggers map[string]*WebhookTrigger
}

func (m *mockTriggerProvider) FindWebhookTrigger(_ context.Context, path string) (*WebhookTrigger, error) {
	t, ok := m.triggers[path]
	if !ok {
		return nil, fmt.Errorf("trigger not found for path: %s", path)
	}
	return t, nil
}

func TestWebhookService_HandleWebhook(t *testing.T) {
	baseTrigger := &WebhookTrigger{
		Title:       "Default Title",
		Description: "Default Description",
		AgentName:   "test-agent",
		Path:        "/hooks/deploy",
	}

	tests := []struct {
		name        string
		path        string
		body        []byte
		wantTitle   string
		wantDesc    string
		wantErr     bool
		errContains string
	}{
		{
			name:      "no body uses trigger defaults",
			path:      "/hooks/deploy",
			body:      nil,
			wantTitle: "Default Title",
			wantDesc:  "Default Description",
		},
		{
			name:      "empty body uses trigger defaults",
			path:      "/hooks/deploy",
			body:      []byte{},
			wantTitle: "Default Title",
			wantDesc:  "Default Description",
		},
		{
			name:      "body overrides description only",
			path:      "/hooks/deploy",
			body:      mustJSON(t, map[string]string{"description": "Custom Desc"}),
			wantTitle: "Default Title",
			wantDesc:  "Custom Desc",
		},
		{
			name:      "body overrides title only",
			path:      "/hooks/deploy",
			body:      mustJSON(t, map[string]string{"title": "Custom Title"}),
			wantTitle: "Custom Title",
			wantDesc:  "Default Description",
		},
		{
			name:      "body overrides both",
			path:      "/hooks/deploy",
			body:      mustJSON(t, map[string]string{"title": "T", "description": "D"}),
			wantTitle: "T",
			wantDesc:  "D",
		},
		{
			name:      "invalid json uses trigger defaults",
			path:      "/hooks/deploy",
			body:      []byte("not-json"),
			wantTitle: "Default Title",
			wantDesc:  "Default Description",
		},
		{
			name:        "unknown path returns error",
			path:        "/hooks/unknown",
			body:        nil,
			wantErr:     true,
			errContains: "find webhook trigger",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &mockTriggerProvider{
				triggers: map[string]*WebhookTrigger{
					"/hooks/deploy": baseTrigger,
				},
			}
			creator := &mockTaskCreator{}
			svc := NewWebhookService(provider, creator)

			taskID, err := svc.HandleWebhook(context.Background(), tt.path, tt.body)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err)
			assert.NotZero(t, taskID)

			calls := creator.getCalls()
			require.Len(t, calls, 1)
			assert.Equal(t, tt.wantTitle, calls[0].Title)
			assert.Equal(t, tt.wantDesc, calls[0].Description)
			assert.Equal(t, "test-agent", calls[0].AgentName)
			assert.Equal(t, "webhook", calls[0].Source)
			assert.Equal(t, tt.path, calls[0].SourceID)
		})
	}
}

func TestWebhookService_CreatorError(t *testing.T) {
	provider := &mockTriggerProvider{
		triggers: map[string]*WebhookTrigger{
			"/hooks/x": {Title: "T", Description: "D", AgentName: "a", Path: "/hooks/x"},
		},
	}
	creator := &mockTaskCreator{err: assert.AnError}
	svc := NewWebhookService(provider, creator)

	_, err := svc.HandleWebhook(context.Background(), "/hooks/x", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create task from webhook")
}

func mustJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return data
}
