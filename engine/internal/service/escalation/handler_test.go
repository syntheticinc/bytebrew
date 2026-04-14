package escalation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCapabilityReader struct {
	config *Config
	err    error
}

func (m *mockCapabilityReader) GetEscalationConfig(_ context.Context, _ string) (*Config, error) {
	return m.config, m.err
}

func TestNewHandler(t *testing.T) {
	h := NewHandler(&mockCapabilityReader{})

	require.NotNil(t, h)
	assert.NotNil(t, h.httpClient)
	assert.NotNil(t, h.reader)
}

func TestEscalate(t *testing.T) {
	tests := []struct {
		name       string
		reader     *mockCapabilityReader
		wantResult string
		wantErr    bool
	}{
		{
			name:       "nil config defaults to transfer_to_user",
			reader:     &mockCapabilityReader{config: nil},
			wantResult: "Escalation triggered: transfer_to_user. The conversation has been flagged for human review. Reason: user is upset",
		},
		{
			name:       "empty action defaults to transfer_to_user",
			reader:     &mockCapabilityReader{config: &Config{Action: ""}},
			wantResult: "Escalation triggered: transfer_to_user. The conversation has been flagged for human review. Reason: user is upset",
		},
		{
			name:       "explicit transfer_to_user",
			reader:     &mockCapabilityReader{config: &Config{Action: "transfer_to_user"}},
			wantResult: "Escalation triggered: transfer_to_user. The conversation has been flagged for human review. Reason: user is upset",
		},
		{
			name:       "notify_webhook with no URL",
			reader:     &mockCapabilityReader{config: &Config{Action: "notify_webhook"}},
			wantResult: "Escalation triggered: notify_webhook (no webhook URL configured — skipped)",
		},
		{
			name:    "reader returns error",
			reader:  &mockCapabilityReader{err: fmt.Errorf("db connection failed")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(tt.reader)

			result, err := h.Escalate(context.Background(), "sess-1", "support-bot", "user is upset")
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "read escalation config")
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

func TestEscalate_WebhookSuccess(t *testing.T) {
	var receivedPayload map[string]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	h := NewHandler(&mockCapabilityReader{
		config: &Config{Action: "notify_webhook", WebhookURL: srv.URL},
	})

	result, err := h.Escalate(context.Background(), "sess-42", "sales-agent", "customer wants refund")

	require.NoError(t, err)
	assert.Equal(t, "Escalation triggered: notify_webhook (webhook notified successfully)", result)

	assert.Equal(t, "escalation", receivedPayload["event"])
	assert.Equal(t, "sales-agent", receivedPayload["agent"])
	assert.Equal(t, "sess-42", receivedPayload["session_id"])
	assert.Equal(t, "customer wants refund", receivedPayload["reason"])
}

func TestEscalate_WebhookReturns500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	h := NewHandler(&mockCapabilityReader{
		config: &Config{Action: "notify_webhook", WebhookURL: srv.URL},
	})

	result, err := h.Escalate(context.Background(), "sess-1", "bot", "issue")

	require.NoError(t, err)
	assert.Contains(t, result, "webhook failed")
	assert.Contains(t, result, "500")
}
