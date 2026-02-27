package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestIsJSONExpectedFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"config.json", true},
		{"data.jsonl", true},
		{"map.geojson", true},
		{"notebook.ipynb", true},
		{"state.tfstate", true},
		{"main.go", false},
		{"index.ts", false},
		{"style.css", false},
		{"README.md", false},
		{"Makefile", false},
		{"path/to/file.json", true},
		{"path/to/file.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isJSONExpectedFile(tt.path)
			if got != tt.want {
				t.Errorf("isJSONExpectedFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestLooksLikeJSON(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"json object", `{"key": "value"}`, true},
		{"json array", `[1, 2, 3]`, true},
		{"nested json", `{"WorkerTypePlanner": "return true\n\tdefault:\n\t\treturn false"}`, true},
		{"json array with objects", `[{"name": "test"}]`, true},
		{"go source", "package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}", false},
		{"empty string", "", false},
		{"single char", "{", false},
		{"invalid json starting with brace", "{not json at all", false},
		{"plain text", "hello world", false},
		{"html", "<html><body>test</body></html>", false},
		{"whitespace + json", "  {\"key\": \"value\"}  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeJSON(tt.content)
			if got != tt.want {
				t.Errorf("looksLikeJSON(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

func TestWriteFileTool_RejectsJSONContentForSourceFiles(t *testing.T) {
	proxy := &mockClientOperationsProxy{
		writeFileFunc: func(ctx context.Context, sessionID, filePath, content string) (string, error) {
			t.Error("proxy.WriteFile should not be called for JSON content in .go file")
			return "", nil
		},
	}

	tool := NewWriteFileTool(proxy, "test-session")

	args := WriteFileArgs{
		FilePath: "internal/domain/entity.go",
		Content:  `{"WorkerTypePlanner": "return true\n\tdefault:\n\t\treturn false"}`,
	}
	argsJSON, _ := json.Marshal(args)

	result, err := tool.InvokableRun(context.Background(), string(argsJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "[ERROR]") {
		t.Errorf("expected error message, got: %s", result)
	}
	if !strings.Contains(result, "JSON") {
		t.Errorf("expected mention of JSON in error, got: %s", result)
	}
}

func TestWriteFileTool_AllowsValidGoCode(t *testing.T) {
	var writtenContent string
	proxy := &mockClientOperationsProxy{
		writeFileFunc: func(ctx context.Context, sessionID, filePath, content string) (string, error) {
			writtenContent = content
			return "File written successfully", nil
		},
	}

	tool := NewWriteFileTool(proxy, "test-session")

	goCode := "package domain\n\ntype TestEntity struct {\n\tName  string\n\tValue int\n}\n"
	args := WriteFileArgs{
		FilePath: "internal/domain/entity.go",
		Content:  goCode,
	}
	argsJSON, _ := json.Marshal(args)

	result, err := tool.InvokableRun(context.Background(), string(argsJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "[ERROR]") {
		t.Errorf("expected success, got error: %s", result)
	}
	if writtenContent != goCode {
		t.Errorf("expected content to be written, got: %q", writtenContent)
	}
}

func TestWriteFileTool_AllowsJSONContentForJSONFiles(t *testing.T) {
	var writtenContent string
	proxy := &mockClientOperationsProxy{
		writeFileFunc: func(ctx context.Context, sessionID, filePath, content string) (string, error) {
			writtenContent = content
			return "File written successfully", nil
		},
	}

	tool := NewWriteFileTool(proxy, "test-session")

	jsonContent := `{"name": "test", "version": "1.0.0"}`
	args := WriteFileArgs{
		FilePath: "package.json",
		Content:  jsonContent,
	}
	argsJSON, _ := json.Marshal(args)

	result, err := tool.InvokableRun(context.Background(), string(argsJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "[ERROR]") {
		t.Errorf("expected success for .json file, got error: %s", result)
	}
	if writtenContent != jsonContent {
		t.Errorf("expected JSON content to be written, got: %q", writtenContent)
	}
}
