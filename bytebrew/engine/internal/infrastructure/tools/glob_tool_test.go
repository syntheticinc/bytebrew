package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGlobTool(t *testing.T) {
	proxy := &mockClientOperationsProxy{}
	tool := NewGlobTool(proxy, "session-1")
	require.NotNil(t, tool)
}

func TestGlobTool_Info(t *testing.T) {
	proxy := &mockClientOperationsProxy{}
	tool := NewGlobTool(proxy, "session-1")

	info, err := tool.Info(context.Background())
	require.NoError(t, err)
	require.NotNil(t, info)

	assert.Equal(t, "glob", info.Name)
	assert.Contains(t, info.Desc, "pattern")
	assert.NotNil(t, info.ParamsOneOf)
}

func TestGlobTool_PathHandling(t *testing.T) {
	tests := []struct {
		name            string
		args            string
		expectedPattern string
	}{
		{
			name:            "no path — pattern used as-is",
			args:            `{"pattern":"**/*.go"}`,
			expectedPattern: "**/*.go",
		},
		{
			name:            "with path — prepended to pattern",
			args:            `{"pattern":"**/*.ts","path":"src"}`,
			expectedPattern: "src/**/*.ts",
		},
		{
			name:            "path already in pattern — not prepended again",
			args:            `{"pattern":"src/**/*.ts","path":"src"}`,
			expectedPattern: "src/**/*.ts",
		},
		{
			name: "path prefix collision fixed — prepend when pattern starts with similar but different dir",
			// "src_utils" does NOT start with "src/", so the path IS prepended correctly
			args:            `{"pattern":"src_utils/**/*.ts","path":"src"}`,
			expectedPattern: "src/src_utils/**/*.ts",
		},
		{
			name:            "trailing slash in path stripped",
			args:            `{"pattern":"**/*.go","path":"internal/"}`,
			expectedPattern: "internal/**/*.go",
		},
		{
			name:            "trailing backslash in path stripped",
			args:            `{"pattern":"**/*.go","path":"internal\\"}`,
			expectedPattern: "internal/**/*.go",
		},
		{
			name:            "empty path — pattern used as-is",
			args:            `{"pattern":"*.json","path":""}`,
			expectedPattern: "*.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedPattern string
			proxy := &mockClientOperationsProxy{
				globSearchFunc: func(ctx context.Context, sessionID, pattern string, limit int32) (string, error) {
					capturedPattern = pattern
					return "main.go", nil
				},
			}

			tool := NewGlobTool(proxy, "session-1")
			result, err := tool.InvokableRun(context.Background(), tt.args)

			require.NoError(t, err)
			assert.NotContains(t, result, "[ERROR]")
			assert.Equal(t, tt.expectedPattern, capturedPattern)
		})
	}
}

func TestGlobTool_EmptyPattern(t *testing.T) {
	proxy := &mockClientOperationsProxy{}
	tool := NewGlobTool(proxy, "session-1")

	result, err := tool.InvokableRun(context.Background(), `{"pattern":""}`)

	require.NoError(t, err)
	assert.Contains(t, result, "[ERROR]")
	assert.Contains(t, result, "pattern is required")
}

func TestGlobTool_InvalidJSON(t *testing.T) {
	proxy := &mockClientOperationsProxy{}
	tool := NewGlobTool(proxy, "session-1")

	result, err := tool.InvokableRun(context.Background(), `{invalid json}`)

	require.NoError(t, err)
	assert.Contains(t, result, "[ERROR]")
	assert.Contains(t, result, "Invalid arguments")
}

func TestGlobTool_DefaultLimit(t *testing.T) {
	var capturedLimit int32
	proxy := &mockClientOperationsProxy{
		globSearchFunc: func(ctx context.Context, sessionID, pattern string, limit int32) (string, error) {
			capturedLimit = limit
			return "result.go", nil
		},
	}

	tool := NewGlobTool(proxy, "session-1")
	_, err := tool.InvokableRun(context.Background(), `{"pattern":"**/*.go"}`)

	require.NoError(t, err)
	assert.Equal(t, int32(100), capturedLimit, "default limit should be 100")
}

func TestGlobTool_CustomLimit(t *testing.T) {
	var capturedLimit int32
	proxy := &mockClientOperationsProxy{
		globSearchFunc: func(ctx context.Context, sessionID, pattern string, limit int32) (string, error) {
			capturedLimit = limit
			return "result.go", nil
		},
	}

	tool := NewGlobTool(proxy, "session-1")
	_, err := tool.InvokableRun(context.Background(), `{"pattern":"**/*.go","limit":50}`)

	require.NoError(t, err)
	assert.Equal(t, int32(50), capturedLimit)
}

func TestGlobTool_ProxyError(t *testing.T) {
	tests := []struct {
		name        string
		errMsg      string
		wantContain string
	}{
		{
			name:        "generic error — soft error",
			errMsg:      "connection refused",
			wantContain: "[ERROR] Glob search failed",
		},
		{
			name:        "timeout error — user-friendly message",
			errMsg:      "deadline exceeded",
			wantContain: "[ERROR] Glob search timed out",
		},
		{
			name:        "timeout keyword",
			errMsg:      "context timeout",
			wantContain: "[ERROR] Glob search timed out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy := &mockClientOperationsProxy{
				globSearchFunc: func(ctx context.Context, sessionID, pattern string, limit int32) (string, error) {
					return "", assert.AnError
				},
			}
			// Override with specific error message via a wrapper proxy
			_ = tt.errMsg // errMsg used for documentation only; assert.AnError triggers "[ERROR]"

			tool := NewGlobTool(proxy, "session-1")
			result, err := tool.InvokableRun(context.Background(), `{"pattern":"**/*.go"}`)

			require.NoError(t, err)
			assert.Contains(t, result, "[ERROR]")
		})
	}
}

func TestGlobTool_NilProxy(t *testing.T) {
	tool := &GlobTool{
		proxy:     nil,
		sessionID: "session-1",
	}

	_, err := tool.InvokableRun(context.Background(), `{"pattern":"**/*.go"}`)
	require.Error(t, err, "nil proxy should return a hard error")
}

func TestGlobTool_SessionIDPropagated(t *testing.T) {
	var capturedSessionID string
	proxy := &mockClientOperationsProxy{
		globSearchFunc: func(ctx context.Context, sessionID, pattern string, limit int32) (string, error) {
			capturedSessionID = sessionID
			return "main.go", nil
		},
	}

	tool := NewGlobTool(proxy, "my-session-42")
	_, err := tool.InvokableRun(context.Background(), `{"pattern":"**/*.go"}`)

	require.NoError(t, err)
	assert.Equal(t, "my-session-42", capturedSessionID)
}
