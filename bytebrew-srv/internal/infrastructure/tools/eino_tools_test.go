package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
	pkgerrors "github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/errors"
)

// mockClientOperationsProxy implements ClientOperationsProxy interface for testing
type mockClientOperationsProxy struct {
	readFileFunc       func(ctx context.Context, sessionID, filePath string, startLine, endLine int32) (string, error)
	searchCodeFunc     func(ctx context.Context, sessionID, query, projectKey string, limit int32, minScore float32) ([]byte, error)
	getProjectTreeFunc func(ctx context.Context, sessionID, projectKey, path string, maxDepth int) (string, error)
	grepSearchFunc     func(ctx context.Context, sessionID, pattern string, limit int32, fileTypes []string, ignoreCase bool) (string, error)
	globSearchFunc     func(ctx context.Context, sessionID, pattern string, limit int32) (string, error)
	symbolSearchFunc   func(ctx context.Context, sessionID, symbolName string, limit int32, symbolTypes []string) (string, error)
	executeCommandFunc func(ctx context.Context, sessionID, command, cwd string, timeout int32) (string, error)
	writeFileFunc      func(ctx context.Context, sessionID, filePath, content string) (string, error)
	editFileFunc       func(ctx context.Context, sessionID, filePath, oldString, newString string, replaceAll bool) (string, error)
}

func (m *mockClientOperationsProxy) WriteFile(ctx context.Context, sessionID, filePath, content string) (string, error) {
	if m.writeFileFunc != nil {
		return m.writeFileFunc(ctx, sessionID, filePath, content)
	}
	return "File written", nil
}

func (m *mockClientOperationsProxy) EditFile(ctx context.Context, sessionID, filePath, oldString, newString string, replaceAll bool) (string, error) {
	if m.editFileFunc != nil {
		return m.editFileFunc(ctx, sessionID, filePath, oldString, newString, replaceAll)
	}
	return "Edit applied", nil
}

func (m *mockClientOperationsProxy) ReadFile(ctx context.Context, sessionID, filePath string, startLine, endLine int32) (string, error) {
	if m.readFileFunc != nil {
		return m.readFileFunc(ctx, sessionID, filePath, startLine, endLine)
	}
	return "mock file content", nil
}

func (m *mockClientOperationsProxy) SearchCode(ctx context.Context, sessionID, query, projectKey string, limit int32, minScore float32) ([]byte, error) {
	if m.searchCodeFunc != nil {
		return m.searchCodeFunc(ctx, sessionID, query, projectKey, limit, minScore)
	}
	results := []map[string]interface{}{
		{"file": "main.go", "content": "package main"},
	}
	return json.Marshal(results)
}

func (m *mockClientOperationsProxy) GetProjectTree(ctx context.Context, sessionID, projectKey, path string, maxDepth int) (string, error) {
	if m.getProjectTreeFunc != nil {
		return m.getProjectTreeFunc(ctx, sessionID, projectKey, path, maxDepth)
	}
	return `{"path":"project","name":"project","is_directory":true,"children":[{"path":"project/main.go","name":"main.go","is_directory":false}]}`, nil
}

func (m *mockClientOperationsProxy) GrepSearch(ctx context.Context, sessionID, pattern string, limit int32, fileTypes []string, ignoreCase bool) (string, error) {
	if m.grepSearchFunc != nil {
		return m.grepSearchFunc(ctx, sessionID, pattern, limit, fileTypes, ignoreCase)
	}
	return "main.go:10\n  func main() {", nil
}

func (m *mockClientOperationsProxy) GlobSearch(ctx context.Context, sessionID, pattern string, limit int32) (string, error) {
	if m.globSearchFunc != nil {
		return m.globSearchFunc(ctx, sessionID, pattern, limit)
	}
	return "main.go\nutils.go", nil
}

func (m *mockClientOperationsProxy) SymbolSearch(ctx context.Context, sessionID, symbolName string, limit int32, symbolTypes []string) (string, error) {
	if m.symbolSearchFunc != nil {
		return m.symbolSearchFunc(ctx, sessionID, symbolName, limit, symbolTypes)
	}
	return "[function] main\n  main.go:10-20", nil
}

func (m *mockClientOperationsProxy) ExecuteCommand(ctx context.Context, sessionID, command, cwd string, timeout int32) (string, error) {
	if m.executeCommandFunc != nil {
		return m.executeCommandFunc(ctx, sessionID, command, cwd, timeout)
	}
	return "mock command output", nil
}

func (m *mockClientOperationsProxy) ExecuteSubQueries(ctx context.Context, sessionID string, subQueries []*pb.SubQuery) ([]*pb.SubResult, error) {
	return nil, nil
}

func (m *mockClientOperationsProxy) AskUserQuestionnaire(ctx context.Context, sessionID, questionsJSON string) (string, error) {
	return `[{"question":"mock","answer":"mock answer"}]`, nil
}

func (m *mockClientOperationsProxy) LspRequest(ctx context.Context, sessionID, symbolName, operation string) (string, error) {
	return "", nil
}

func (m *mockClientOperationsProxy) ExecuteCommandFull(ctx context.Context, sessionID string, arguments map[string]string) (string, error) {
	// Extract command from arguments for backward compatibility
	command := arguments["command"]
	cwd := arguments["cwd"]
	timeout := int32(120)
	if m.executeCommandFunc != nil {
		return m.executeCommandFunc(ctx, sessionID, command, cwd, timeout)
	}
	return "mock command output", nil
}

func TestNewReadFileTool(t *testing.T) {
	proxy := &mockClientOperationsProxy{}
	tool := NewReadFileTool(proxy, "session-1")

	if tool == nil {
		t.Fatal("NewReadFileTool() returned nil")
	}

	readFileTool, ok := tool.(*ReadFileTool)
	if !ok {
		t.Error("NewReadFileTool() did not return *ReadFileTool")
	}

	if readFileTool.sessionID != "session-1" {
		t.Errorf("NewReadFileTool() sessionID = %v, want session-1", readFileTool.sessionID)
	}
}

func TestReadFileTool_Info(t *testing.T) {
	proxy := &mockClientOperationsProxy{}
	tool := NewReadFileTool(proxy, "session-1").(*ReadFileTool)

	ctx := context.Background()
	info, err := tool.Info(ctx)

	if err != nil {
		t.Errorf("Info() unexpected error: %v", err)
	}

	if info == nil {
		t.Fatal("Info() returned nil")
	}

	if info.Name != "read_file" {
		t.Errorf("Info() Name = %v, want read_file", info.Name)
	}

	if info.Desc == "" {
		t.Error("Info() Desc is empty")
	}
}

func TestReadFileTool_InvokableRun(t *testing.T) {
	tests := []struct {
		name        string
		args        string
		proxy       *mockClientOperationsProxy
		wantSoftErr bool // soft error = result contains "[ERROR]", err is nil
		wantValue   string
	}{
		{
			name: "successful read",
			args: `{"file_path": "/path/to/file.go"}`,
			proxy: &mockClientOperationsProxy{
				readFileFunc: func(ctx context.Context, sessionID, filePath string, startLine, endLine int32) (string, error) {
					// Leading slash is stripped by ReadFileTool
					if filePath != "path/to/file.go" {
						t.Errorf("ReadFile() filePath = %v, want path/to/file.go", filePath)
					}
					if sessionID != "session-1" {
						t.Errorf("ReadFile() sessionID = %v, want session-1", sessionID)
					}
					return "package main\n\nfunc main() {}", nil
				},
			},
			wantSoftErr: false,
			wantValue:   "package main\n\nfunc main() {}",
		},
		{
			name:        "invalid JSON - returns soft error so agent can retry",
			args:        `{invalid json}`,
			proxy:       &mockClientOperationsProxy{},
			wantSoftErr: true,
		},
		{
			name:        "empty file_path - returns soft error so agent can fix",
			args:        `{"file_path": ""}`,
			proxy:       &mockClientOperationsProxy{},
			wantSoftErr: true,
		},
		{
			name: "proxy error - returns soft error so agent can adapt",
			args: `{"file_path": "/path/to/file.go"}`,
			proxy: &mockClientOperationsProxy{
				readFileFunc: func(ctx context.Context, sessionID, filePath string, startLine, endLine int32) (string, error) {
					return "", errors.New("gRPC error")
				},
			},
			wantSoftErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := NewReadFileTool(tt.proxy, "session-1").(*ReadFileTool)
			ctx := context.Background()

			result, err := tool.InvokableRun(ctx, tt.args)

			// Tools should return soft errors (in result string) to let agent continue
			// Hard errors (err != nil) only for fatal infrastructure issues (nil proxy)
			if err != nil {
				t.Errorf("InvokableRun() unexpected hard error: %v (tools should return soft errors)", err)
				return
			}

			if tt.wantSoftErr {
				if !strings.Contains(result, "[ERROR]") {
					t.Errorf("InvokableRun() expected soft error containing [ERROR], got: %v", result)
				}
				return
			}

			if result != tt.wantValue {
				t.Errorf("InvokableRun() result = %v, want %v", result, tt.wantValue)
			}
		})
	}
}

func TestReadFileTool_NilProxy(t *testing.T) {
	tool := &ReadFileTool{
		proxy:     nil,
		sessionID: "session-1",
	}

	ctx := context.Background()
	args := `{"file_path": "/path/to/file.go"}`

	_, err := tool.InvokableRun(ctx, args)

	if err == nil {
		t.Error("InvokableRun() with nil proxy expected error, got nil")
	}

	if !pkgerrors.Is(err, pkgerrors.CodeInternal) {
		t.Errorf("InvokableRun() error code = %v, want %v", pkgerrors.GetCode(err), pkgerrors.CodeInternal)
	}
}

func TestNewSearchCodeTool(t *testing.T) {
	proxy := &mockClientOperationsProxy{}
	tool := NewSearchCodeTool(proxy, "session-1", "project-1")

	if tool == nil {
		t.Fatal("NewSearchCodeTool() returned nil")
	}

	searchTool, ok := tool.(*SearchCodeTool)
	if !ok {
		t.Error("NewSearchCodeTool() did not return *SearchCodeTool")
	}

	if searchTool.sessionID != "session-1" {
		t.Errorf("NewSearchCodeTool() sessionID = %v, want session-1", searchTool.sessionID)
	}

	if searchTool.projectKey != "project-1" {
		t.Errorf("NewSearchCodeTool() projectKey = %v, want project-1", searchTool.projectKey)
	}
}

func TestSearchCodeTool_Info(t *testing.T) {
	proxy := &mockClientOperationsProxy{}
	tool := NewSearchCodeTool(proxy, "session-1", "project-1").(*SearchCodeTool)

	ctx := context.Background()
	info, err := tool.Info(ctx)

	if err != nil {
		t.Errorf("Info() unexpected error: %v", err)
	}

	if info == nil {
		t.Fatal("Info() returned nil")
	}

	if info.Name != "search_code" {
		t.Errorf("Info() Name = %v, want search_code", info.Name)
	}
}

func TestSearchCodeTool_InvokableRun(t *testing.T) {
	tests := []struct {
		name        string
		args        string
		proxy       *mockClientOperationsProxy
		wantSoftErr bool // soft error = result contains "[ERROR]", err is nil
	}{
		{
			name: "successful search",
			args: `{"query": "function main", "limit": 10}`,
			proxy: &mockClientOperationsProxy{
				searchCodeFunc: func(ctx context.Context, sessionID, query, projectKey string, limit int32, minScore float32) ([]byte, error) {
					if query != "function main" {
						t.Errorf("SearchCode() query = %v, want 'function main'", query)
					}
					if limit != 10 {
						t.Errorf("SearchCode() limit = %v, want 10", limit)
					}
					results := []map[string]interface{}{
						{"file": "main.go", "content": "func main() {}"},
					}
					return json.Marshal(results)
				},
			},
			wantSoftErr: false,
		},
		{
			name: "default limit",
			args: `{"query": "function main"}`,
			proxy: &mockClientOperationsProxy{
				searchCodeFunc: func(ctx context.Context, sessionID, query, projectKey string, limit int32, minScore float32) ([]byte, error) {
					if limit != 5 {
						t.Errorf("SearchCode() limit = %v, want 5 (default)", limit)
					}
					return []byte("[]"), nil
				},
			},
			wantSoftErr: false,
		},
		{
			name:        "invalid JSON - returns soft error so agent can retry",
			args:        `{invalid}`,
			proxy:       &mockClientOperationsProxy{},
			wantSoftErr: true,
		},
		{
			name:        "empty query - returns soft error so agent can fix",
			args:        `{"query": ""}`,
			proxy:       &mockClientOperationsProxy{},
			wantSoftErr: true,
		},
		{
			name: "proxy error - returns soft error so agent can adapt",
			args: `{"query": "test"}`,
			proxy: &mockClientOperationsProxy{
				searchCodeFunc: func(ctx context.Context, sessionID, query, projectKey string, limit int32, minScore float32) ([]byte, error) {
					return nil, errors.New("search failed")
				},
			},
			wantSoftErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := NewSearchCodeTool(tt.proxy, "session-1", "project-1").(*SearchCodeTool)
			ctx := context.Background()

			result, err := tool.InvokableRun(ctx, tt.args)

			// Tools should return soft errors (in result string) to let agent continue
			if err != nil {
				t.Errorf("InvokableRun() unexpected hard error: %v (tools should return soft errors)", err)
				return
			}

			if tt.wantSoftErr {
				if !strings.Contains(result, "[ERROR]") {
					t.Errorf("InvokableRun() expected soft error containing [ERROR], got: %v", result)
				}
				return
			}

			if result == "" {
				t.Error("InvokableRun() returned empty result")
			}
		})
	}
}

func TestNewGetProjectTreeTool(t *testing.T) {
	proxy := &mockClientOperationsProxy{}
	tool := NewGetProjectTreeTool(proxy, "session-1", "project-1")

	if tool == nil {
		t.Fatal("NewGetProjectTreeTool() returned nil")
	}

	treeTool, ok := tool.(*GetProjectTreeTool)
	if !ok {
		t.Error("NewGetProjectTreeTool() did not return *GetProjectTreeTool")
	}

	if treeTool.sessionID != "session-1" {
		t.Errorf("NewGetProjectTreeTool() sessionID = %v, want session-1", treeTool.sessionID)
	}

	if treeTool.projectKey != "project-1" {
		t.Errorf("NewGetProjectTreeTool() projectKey = %v, want project-1", treeTool.projectKey)
	}
}

func TestGetProjectTreeTool_Info(t *testing.T) {
	proxy := &mockClientOperationsProxy{}
	tool := NewGetProjectTreeTool(proxy, "session-1", "project-1").(*GetProjectTreeTool)

	ctx := context.Background()
	info, err := tool.Info(ctx)

	if err != nil {
		t.Errorf("Info() unexpected error: %v", err)
	}

	if info == nil {
		t.Fatal("Info() returned nil")
	}

	if info.Name != "get_project_tree" {
		t.Errorf("Info() Name = %v, want get_project_tree", info.Name)
	}
}

func TestGetProjectTreeTool_InvokableRun(t *testing.T) {
	tests := []struct {
		name        string
		args        string
		proxy       *mockClientOperationsProxy
		wantSoftErr bool // soft error = result contains "[ERROR]", err is nil
	}{
		{
			name: "successful get tree",
			args: `{"max_depth": 5}`,
			proxy: &mockClientOperationsProxy{
				getProjectTreeFunc: func(ctx context.Context, sessionID, projectKey, path string, maxDepth int) (string, error) {
					// Return valid JSON tree that can be converted to compact format
					return `{"name":"project","path":".","is_directory":true,"children":[{"name":"src","path":"src","is_directory":true,"children":[{"name":"main.go","path":"src/main.go","is_directory":false}]}]}`, nil
				},
			},
			wantSoftErr: false,
		},
		{
			name: "default max_depth",
			args: `{}`,
			proxy: &mockClientOperationsProxy{
				getProjectTreeFunc: func(ctx context.Context, sessionID, projectKey, path string, maxDepth int) (string, error) {
					return `{"name":"project","path":".","is_directory":true,"children":[]}`, nil
				},
			},
			wantSoftErr: false,
		},
		{
			name:        "invalid JSON - returns soft error so agent can retry",
			args:        `{invalid}`,
			proxy:       &mockClientOperationsProxy{},
			wantSoftErr: true,
		},
		{
			name: "proxy error - returns soft error so agent can adapt",
			args: `{}`,
			proxy: &mockClientOperationsProxy{
				getProjectTreeFunc: func(ctx context.Context, sessionID, projectKey, path string, maxDepth int) (string, error) {
					return "", errors.New("tree failed")
				},
			},
			wantSoftErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := NewGetProjectTreeTool(tt.proxy, "session-1", "project-1").(*GetProjectTreeTool)
			ctx := context.Background()

			result, err := tool.InvokableRun(ctx, tt.args)

			// Tools should return soft errors (in result string) to let agent continue
			if err != nil {
				t.Errorf("InvokableRun() unexpected hard error: %v (tools should return soft errors)", err)
				return
			}

			if tt.wantSoftErr {
				if !strings.Contains(result, "[ERROR]") {
					t.Errorf("InvokableRun() expected soft error containing [ERROR], got: %v", result)
				}
				return
			}

			if result == "" {
				t.Error("InvokableRun() returned empty result")
			}
		})
	}
}
