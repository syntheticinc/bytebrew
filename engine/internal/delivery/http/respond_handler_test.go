package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSessionResponder struct {
	sessions   map[string]bool
	lastCallID string
	lastReply  string
}

func (m *mockSessionResponder) HasSession(sessionID string) bool {
	return m.sessions[sessionID]
}

func (m *mockSessionResponder) SendAskUserReply(sessionID, callID, reply string) {
	m.lastCallID = callID
	m.lastReply = reply
}

func newRespondRouter(handler *RespondHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/api/v1/sessions/{id}/respond", handler.Respond)
	return r
}

func TestRespondHandler_Respond(t *testing.T) {
	tests := []struct {
		name           string
		sessionID      string
		body           interface{}
		sessions       map[string]bool
		wantStatus     int
		wantCallID     string
		wantReply      string
		wantErrContain string
	}{
		{
			name:      "successful respond with answers",
			sessionID: "sess-1",
			body: respondRequest{
				CallID:  "call-42",
				Answers: []string{"iOS", "yes"},
			},
			sessions:   map[string]bool{"sess-1": true},
			wantStatus: http.StatusOK,
			wantCallID: "call-42",
			wantReply:  `["iOS","yes"]`,
		},
		{
			name:      "successful respond with empty answers",
			sessionID: "sess-1",
			body: respondRequest{
				CallID:  "call-1",
				Answers: []string{},
			},
			sessions:   map[string]bool{"sess-1": true},
			wantStatus: http.StatusOK,
			wantCallID: "call-1",
			wantReply:  `[]`,
		},
		{
			name:           "session not found",
			sessionID:      "nonexistent",
			body:           respondRequest{CallID: "call-1", Answers: []string{"yes"}},
			sessions:       map[string]bool{},
			wantStatus:     http.StatusNotFound,
			wantErrContain: "session not found",
		},
		{
			name:           "missing call_id",
			sessionID:      "sess-1",
			body:           respondRequest{CallID: "", Answers: []string{"yes"}},
			sessions:       map[string]bool{"sess-1": true},
			wantStatus:     http.StatusBadRequest,
			wantErrContain: "call_id is required",
		},
		{
			name:           "invalid body",
			sessionID:      "sess-1",
			body:           "not json",
			sessions:       map[string]bool{"sess-1": true},
			wantStatus:     http.StatusBadRequest,
			wantErrContain: "invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSessionResponder{sessions: tt.sessions}
			handler := NewRespondHandler(mock)
			router := newRespondRouter(handler)

			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				var err error
				bodyBytes, err = json.Marshal(v)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost,
				"/api/v1/sessions/"+tt.sessionID+"/respond",
				bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(context.Background())

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)

			if tt.wantErrContain != "" {
				assert.Contains(t, rr.Body.String(), tt.wantErrContain)
				return
			}

			assert.Equal(t, tt.wantCallID, mock.lastCallID)
			assert.Equal(t, tt.wantReply, mock.lastReply)

			var resp map[string]string
			err := json.Unmarshal(rr.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, "ok", resp["status"])
		})
	}
}
