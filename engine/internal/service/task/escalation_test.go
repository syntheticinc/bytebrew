package task

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEscalationDetector_Check(t *testing.T) {
	detector := NewEscalationDetector()

	tests := []struct {
		name     string
		response string
		triggers []string
		want     string
	}{
		{
			name:     "exact match",
			response: "I need help from a human",
			triggers: []string{"need help", "escalate"},
			want:     "need help",
		},
		{
			name:     "case insensitive match",
			response: "PLEASE ESCALATE THIS",
			triggers: []string{"escalate"},
			want:     "escalate",
		},
		{
			name:     "no match",
			response: "Everything is fine",
			triggers: []string{"need help", "escalate"},
			want:     "",
		},
		{
			name:     "empty response",
			response: "",
			triggers: []string{"help"},
			want:     "",
		},
		{
			name:     "empty triggers",
			response: "need help",
			triggers: nil,
			want:     "",
		},
		{
			name:     "first match wins",
			response: "I need help, please escalate",
			triggers: []string{"escalate", "need help"},
			want:     "escalate",
		},
		{
			name:     "partial word match",
			response: "unescalated issue",
			triggers: []string{"escalat"},
			want:     "escalat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detector.Check(tt.response, tt.triggers)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEscalationWebhookSender_Success(t *testing.T) {
	var called atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Add(1)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sender := NewEscalationWebhookSender()
	err := sender.Send(context.Background(), srv.URL, EscalationWebhookPayload{
		SessionID: "sess-1",
		TaskID:    42,
		Reason:    "need help",
		AgentName: "test-agent",
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), called.Load())
}

func TestEscalationWebhookSender_RetryOnError(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sender := &EscalationWebhookSender{
		client:     &http.Client{Timeout: 5 * time.Second},
		maxRetries: 3,
		baseDelay:  time.Millisecond, // fast retries for test
	}

	err := sender.Send(context.Background(), srv.URL, EscalationWebhookPayload{
		SessionID: "sess-1",
		TaskID:    1,
		Reason:    "escalate",
	})
	require.NoError(t, err)
	assert.Equal(t, int32(3), attempts.Load())
}

func TestEscalationWebhookSender_AllRetriesFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	sender := &EscalationWebhookSender{
		client:     &http.Client{Timeout: 5 * time.Second},
		maxRetries: 2,
		baseDelay:  time.Millisecond,
	}

	err := sender.Send(context.Background(), srv.URL, EscalationWebhookPayload{
		SessionID: "sess-1",
		TaskID:    1,
		Reason:    "help",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed after 3 retries")
}

func TestEscalationWebhookSender_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	sender := &EscalationWebhookSender{
		client:     &http.Client{Timeout: 5 * time.Second},
		maxRetries: 5,
		baseDelay:  time.Second, // long delay to trigger context cancel
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := sender.Send(ctx, srv.URL, EscalationWebhookPayload{})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}
