package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/syntheticinc/bytebrew/engine/pkg/errors"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// WriteFileArgs represents arguments for write_file tool
type WriteFileArgs struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

// WriteFileTool implements Eino InvokableTool for writing files via gRPC
type WriteFileTool struct {
	proxy     ClientOperationsProxy
	sessionID string
}

// NewWriteFileTool creates a new write_file tool
func NewWriteFileTool(proxy ClientOperationsProxy, sessionID string) tool.InvokableTool {
	return &WriteFileTool{
		proxy:     proxy,
		sessionID: sessionID,
	}
}

// Info returns tool information for LLM
func (t *WriteFileTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "write_file",
		Desc: `Create or OVERWRITE a file with the provided content. The ENTIRE file is replaced — include ALL content, not just changed parts.

When to use write_file vs edit_file:
- write_file: new files, rewriting/refactoring most of a file, multiple changes in one file
- edit_file: small targeted changes (1-3 lines) in a large file

CRITICAL: content must be PLAIN TEXT source code, not JSON. Use \n for newlines.
NEVER output code as text to the user — always write it to a file.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"file_path": {
				Type:     schema.String,
				Desc:     "Path to the file (relative to project root or absolute)",
				Required: true,
			},
			"content": {
				Type:     schema.String,
				Desc:     "The full content to write to the file as a PLAIN TEXT string. Write the actual source code directly, NOT as JSON array or structured data. Use \\n for newlines within the string.",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun executes the tool with JSON arguments
func (t *WriteFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	slog.InfoContext(ctx, "WriteFileTool: raw arguments",
		"json_length", len(argumentsInJSON),
		"json_preview", truncateString(argumentsInJSON, 500))

	var args WriteFileArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		slog.WarnContext(ctx, "WriteFileTool: failed to parse arguments", "error", err, "raw", argumentsInJSON)
		return fmt.Sprintf("[ERROR] Invalid arguments for write_file: %v. Raw input length: %d", err, len(argumentsInJSON)), nil
	}

	if args.FilePath == "" {
		return fmt.Sprintf("[ERROR] file_path is required for write_file. You provided: file_path=%q, content length=%d. Make sure you pass the file_path parameter.",
			args.FilePath, len(args.Content)), nil
	}

	// Defensive validation: reject JSON content for source code files
	if !isJSONExpectedFile(args.FilePath) && looksLikeJSON(args.Content) {
		slog.WarnContext(ctx, "WriteFileTool: content is JSON for non-JSON file",
			"file_path", args.FilePath,
			"content_preview", truncateString(args.Content, 200))
		return fmt.Sprintf("[ERROR] Content appears to be JSON/structured data instead of source code. "+
			"Write actual source code as plain text for '%s'.", args.FilePath), nil
	}

	if t.proxy == nil {
		return "", errors.New(errors.CodeInternal, "gRPC proxy not configured")
	}

	slog.InfoContext(ctx, "WriteFileTool: writing file",
		"file_path", args.FilePath,
		"content_length", len(args.Content),
		"content_preview", truncateString(args.Content, 200))

	result, err := t.proxy.WriteFile(ctx, t.sessionID, args.FilePath, args.Content)
	if err != nil {
		errCode := errors.GetCode(err)
		slog.WarnContext(ctx, "WriteFileTool: error writing file",
			"file_path", args.FilePath,
			"error", err,
			"code", errCode)

		if errCode == errors.CodePermissionDenied {
			return fmt.Sprintf("[PERMISSION] Write blocked: %s", args.FilePath), nil
		}

		return fmt.Sprintf("[ERROR] Failed to write file '%s': %v", args.FilePath, err), nil
	}

	slog.InfoContext(ctx, "WriteFileTool: file written successfully",
		"file_path", args.FilePath,
		"result_length", len(result))

	return result, nil
}

// isJSONExpectedFile returns true if the file extension normally contains JSON.
func isJSONExpectedFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".json", ".jsonl", ".geojson", ".ipynb", ".tfstate":
		return true
	}
	return false
}

// looksLikeJSON returns true if content appears to be valid JSON object or array.
func looksLikeJSON(content string) bool {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) < 2 {
		return false
	}
	if trimmed[0] != '{' && trimmed[0] != '[' {
		return false
	}
	return json.Valid([]byte(trimmed))
}
