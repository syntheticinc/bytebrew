package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixRegexEscapes(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{
			name:    "backspace from JSON \\b decoded as word boundary",
			pattern: "\x08AgentEvent\x08",
			want:    `\bAgentEvent\b`,
		},
		{
			name:    "multiple backspace characters",
			pattern: "\x08foo\x08 and \x08bar\x08",
			want:    `\bfoo\b and \bbar\b`,
		},
		{
			name:    "pattern without backspace unchanged",
			pattern: `func.*User`,
			want:    `func.*User`,
		},
		{
			name:    "already correct literal \\b unchanged",
			pattern: `\bword\b`,
			want:    `\bword\b`,
		},
		{
			name:    "empty pattern unchanged",
			pattern: "",
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fixRegexEscapes(tt.pattern)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGrepSearchTool_BackspacePatternFixed(t *testing.T) {
	proxy := &mockClientOperationsProxy{}
	var capturedPattern string
	proxy.grepSearchFunc = func(ctx context.Context, sessionID, pattern string, limit int32, fileTypes []string, ignoreCase bool) (string, error) {
		capturedPattern = pattern
		return "result", nil
	}

	tool := NewGrepSearchTool(proxy, "session-1")

	// Simulate what json.Unmarshal produces when LLM sends {"pattern": "\bSymbol\b"}:
	// JSON spec defines \b as backspace (0x08). We encode this as valid JSON using json.Marshal.
	args := GrepSearchArgs{Pattern: "\x08Symbol\x08"}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	_, err = tool.InvokableRun(context.Background(), string(argsJSON))
	require.NoError(t, err)

	assert.Equal(t, `\bSymbol\b`, capturedPattern, "backspace chars should be replaced with \\b")
}

func TestNewGrepSearchTool(t *testing.T) {
	proxy := &mockClientOperationsProxy{}
	tool := NewGrepSearchTool(proxy, "session-1")
	require.NotNil(t, tool)
}

func TestGrepSearchTool_Info(t *testing.T) {
	proxy := &mockClientOperationsProxy{}
	tool := NewGrepSearchTool(proxy, "session-1")

	info, err := tool.Info(context.Background())
	require.NoError(t, err)
	require.NotNil(t, info)

	assert.Equal(t, "grep_search", info.Name)
	assert.Contains(t, info.Desc, "regex")
	assert.Contains(t, info.Desc, "ripgrep")
	assert.NotNil(t, info.ParamsOneOf)
}

func TestGrepSearchTool_InvokableRun(t *testing.T) {
	tests := []struct {
		name           string
		args           string
		setupProxy     func(*mockClientOperationsProxy)
		wantContains   string
		wantErr        bool
		checkProxyCall func(*testing.T, *mockClientOperationsProxy)
	}{
		{
			name: "successful search",
			args: `{"pattern": "TODO:", "limit": 50}`,
			setupProxy: func(m *mockClientOperationsProxy) {
				m.grepSearchFunc = func(ctx context.Context, sessionID, pattern string, limit int32, fileTypes []string, ignoreCase bool) (string, error) {
					return "src/main.go:10\n  // TODO: implement", nil
				}
			},
			wantContains: "TODO: implement",
		},
		{
			name: "with ignore_case",
			args: `{"pattern": "error", "ignore_case": true, "limit": 100}`,
			setupProxy: func(m *mockClientOperationsProxy) {
				m.grepSearchFunc = func(ctx context.Context, sessionID, pattern string, limit int32, fileTypes []string, ignoreCase bool) (string, error) {
					assert.True(t, ignoreCase, "ignoreCase should be true")
					assert.Equal(t, int32(100), limit, "limit should be 100")
					return "main.go:5\n  Error handling", nil
				}
			},
			wantContains: "Error handling",
		},
		{
			name: "with include filter",
			args: `{"pattern": "func", "include": "*.go,*.ts", "limit": 20}`,
			setupProxy: func(m *mockClientOperationsProxy) {
				m.grepSearchFunc = func(ctx context.Context, sessionID, pattern string, limit int32, fileTypes []string, ignoreCase bool) (string, error) {
					assert.Equal(t, []string{"*.go", "*.ts"}, fileTypes, "fileTypes should match")
					assert.False(t, ignoreCase, "ignoreCase should be false by default")
					return "file.go:1\n  func test()", nil
				}
			},
			wantContains: "func test()",
		},
		{
			name:         "invalid JSON - returns soft error",
			args:         `{invalid json}`,
			wantContains: "[ERROR] Invalid arguments",
		},
		{
			name:         "empty pattern - returns soft error",
			args:         `{"pattern": ""}`,
			wantContains: "[ERROR] pattern is required",
		},
		{
			name: "default limit applied",
			args: `{"pattern": "test"}`,
			setupProxy: func(m *mockClientOperationsProxy) {
				m.grepSearchFunc = func(ctx context.Context, sessionID, pattern string, limit int32, fileTypes []string, ignoreCase bool) (string, error) {
					assert.Equal(t, int32(100), limit, "default limit should be 100")
					return "result", nil
				}
			},
			wantContains: "result",
		},
		{
			name: "timeout error - soft error",
			args: `{"pattern": ".*"}`,
			setupProxy: func(m *mockClientOperationsProxy) {
				m.grepSearchFunc = func(ctx context.Context, sessionID, pattern string, limit int32, fileTypes []string, ignoreCase bool) (string, error) {
					return "", assert.AnError
				}
			},
			wantContains: "[ERROR]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy := &mockClientOperationsProxy{}
			if tt.setupProxy != nil {
				tt.setupProxy(proxy)
			}

			tool := NewGrepSearchTool(proxy, "session-1")

			result, err := tool.InvokableRun(context.Background(), tt.args)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.wantContains != "" {
				assert.Contains(t, result, tt.wantContains)
			}

			if tt.checkProxyCall != nil {
				tt.checkProxyCall(t, proxy)
			}
		})
	}
}
