package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockModelService struct {
	listFunc   func(ctx context.Context) ([]ModelResponse, error)
	createFunc func(ctx context.Context, req CreateModelRequest) (*ModelResponse, error)
	updateFunc func(ctx context.Context, name string, req CreateModelRequest) (*ModelResponse, error)
	deleteFunc func(ctx context.Context, name string) error
	verifyFunc func(ctx context.Context, name string) (*ModelVerifyResult, error)
}

func (m *mockModelService) ListModels(ctx context.Context) ([]ModelResponse, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx)
	}
	return nil, nil
}

func (m *mockModelService) CreateModel(ctx context.Context, req CreateModelRequest) (*ModelResponse, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockModelService) UpdateModel(ctx context.Context, name string, req CreateModelRequest) (*ModelResponse, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, name, req)
	}
	return nil, nil
}

func (m *mockModelService) DeleteModel(ctx context.Context, name string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, name)
	}
	return nil
}

func (m *mockModelService) VerifyModel(ctx context.Context, name string) (*ModelVerifyResult, error) {
	if m.verifyFunc != nil {
		return m.verifyFunc(ctx, name)
	}
	return nil, nil
}

func TestModelHandler_Verify(t *testing.T) {
	tests := []struct {
		name           string
		modelName      string
		verifyFunc     func(ctx context.Context, name string) (*ModelVerifyResult, error)
		wantStatus     int
		wantResult     *ModelVerifyResult
		wantErrMessage string
	}{
		{
			name:      "successful verification with known provider",
			modelName: "gpt-4",
			verifyFunc: func(ctx context.Context, name string) (*ModelVerifyResult, error) {
				return &ModelVerifyResult{
					Connectivity:   "ok",
					ToolCalling:    "skipped",
					ResponseTimeMs: 150,
					ModelName:      "gpt-4",
					Provider:       "openai",
				}, nil
			},
			wantStatus: http.StatusOK,
			wantResult: &ModelVerifyResult{
				Connectivity:   "ok",
				ToolCalling:    "skipped",
				ResponseTimeMs: 150,
				ModelName:      "gpt-4",
				Provider:       "openai",
			},
		},
		{
			name:      "connectivity error",
			modelName: "bad-model",
			verifyFunc: func(ctx context.Context, name string) (*ModelVerifyResult, error) {
				errMsg := "connectivity check failed: connection refused"
				return &ModelVerifyResult{
					Connectivity: "error",
					ToolCalling:  "skipped",
					ModelName:    "bad-model",
					Provider:     "ollama",
					Error:        &errMsg,
				}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:      "model not found",
			modelName: "nonexistent",
			verifyFunc: func(ctx context.Context, name string) (*ModelVerifyResult, error) {
				return nil, fmt.Errorf("model not found: nonexistent")
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: "model not found: nonexistent",
		},
		{
			name:      "tool calling supported",
			modelName: "llama3",
			verifyFunc: func(ctx context.Context, name string) (*ModelVerifyResult, error) {
				return &ModelVerifyResult{
					Connectivity:   "ok",
					ToolCalling:    "supported",
					ResponseTimeMs: 1200,
					ModelName:      "llama3",
					Provider:       "ollama",
				}, nil
			},
			wantStatus: http.StatusOK,
			wantResult: &ModelVerifyResult{
				Connectivity:   "ok",
				ToolCalling:    "supported",
				ResponseTimeMs: 1200,
				ModelName:      "llama3",
				Provider:       "ollama",
			},
		},
		{
			name:      "tool calling not detected",
			modelName: "phi3",
			verifyFunc: func(ctx context.Context, name string) (*ModelVerifyResult, error) {
				return &ModelVerifyResult{
					Connectivity:   "ok",
					ToolCalling:    "not_detected",
					ResponseTimeMs: 800,
					ModelName:      "phi3",
					Provider:       "ollama",
				}, nil
			},
			wantStatus: http.StatusOK,
			wantResult: &ModelVerifyResult{
				Connectivity:   "ok",
				ToolCalling:    "not_detected",
				ResponseTimeMs: 800,
				ModelName:      "phi3",
				Provider:       "ollama",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockModelService{verifyFunc: tt.verifyFunc}
			handler := NewModelHandler(svc)

			// Use chi router to inject URL params.
			r := chi.NewRouter()
			r.Mount("/api/v1/models", handler.Routes())

			req := httptest.NewRequest(http.MethodPost, "/api/v1/models/"+tt.modelName+"/verify", nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			if tt.wantResult != nil {
				var result ModelVerifyResult
				err := json.NewDecoder(rec.Body).Decode(&result)
				require.NoError(t, err)
				assert.Equal(t, tt.wantResult.Connectivity, result.Connectivity)
				assert.Equal(t, tt.wantResult.ToolCalling, result.ToolCalling)
				assert.Equal(t, tt.wantResult.ModelName, result.ModelName)
				assert.Equal(t, tt.wantResult.Provider, result.Provider)
			}

			if tt.wantErrMessage != "" {
				var errResp map[string]string
				err := json.NewDecoder(rec.Body).Decode(&errResp)
				require.NoError(t, err)
				assert.Contains(t, errResp["error"], tt.wantErrMessage)
			}
		})
	}
}

func TestModelHandler_Verify_ErrorField(t *testing.T) {
	errMsg := "connection refused"
	svc := &mockModelService{
		verifyFunc: func(ctx context.Context, name string) (*ModelVerifyResult, error) {
			return &ModelVerifyResult{
				Connectivity: "error",
				ToolCalling:  "skipped",
				ModelName:    "test",
				Provider:     "ollama",
				Error:        &errMsg,
			}, nil
		},
	}
	handler := NewModelHandler(svc)
	r := chi.NewRouter()
	r.Mount("/api/v1/models", handler.Routes())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/models/test/verify", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result ModelVerifyResult
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Connectivity)
	require.NotNil(t, result.Error)
	assert.Equal(t, "connection refused", *result.Error)
}
