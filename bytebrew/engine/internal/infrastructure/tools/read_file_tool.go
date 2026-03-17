package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/errors"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ReadFileArgs represents arguments for read-file tool
type ReadFileArgs struct {
	FilePath string `json:"file_path"`
}

// ReadFileTool implements Eino InvokableTool for reading files via gRPC
type ReadFileTool struct {
	proxy     ClientOperationsProxy
	sessionID string
}

// NewReadFileTool creates a new read-file tool
func NewReadFileTool(proxy ClientOperationsProxy, sessionID string) tool.InvokableTool {
	return &ReadFileTool{
		proxy:     proxy,
		sessionID: sessionID,
	}
}

// Info returns tool information for LLM
func (t *ReadFileTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "read_file",
		Desc: `Read a single file. Returns content with line numbers (useful for edit_file references).

Rules:
- Path must be RELATIVE to project root (no leading '/', no absolute paths).
- ALWAYS read a file before editing it with edit_file.
- Cannot read directories — use get_project_tree instead.
- If file not found, check path with get_project_tree first.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"file_path": {
				Type:     schema.String,
				Desc:     "Relative path to the file from project root. Examples: 'internal/domain/user.go', 'src/app/App.tsx', 'package.json'. No absolute paths, no leading slashes.",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun executes the tool with JSON arguments
func (t *ReadFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args ReadFileArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		// Return error as result so the model can see and fix it
		slog.WarnContext(ctx, "ReadFileTool: failed to parse arguments", "error", err, "raw", argumentsInJSON)
		return fmt.Sprintf("[ERROR] Invalid arguments for read_file: %v. Please provide file_path parameter.", err), nil
	}

	if args.FilePath == "" {
		// Return error as result so the model can see and fix it
		slog.WarnContext(ctx, "ReadFileTool: file_path is required but was empty", "raw_args", argumentsInJSON)
		return "[ERROR] file_path is required. Please specify the file path you want to read, e.g. {\"file_path\": \"src/app/App.tsx\"}", nil
	}

	// Strip leading slashes — LLM sometimes adds them
	cleanPath := strings.TrimLeft(args.FilePath, "/\\")
	if cleanPath != args.FilePath {
		slog.InfoContext(ctx, "ReadFileTool: stripped leading slash from path", "original", args.FilePath, "cleaned", cleanPath)
		args.FilePath = cleanPath
	}

	if t.proxy == nil {
		return "", errors.New(errors.CodeInternal, "gRPC proxy not configured")
	}

	// Call gRPC client proxy to read file (read entire file by default)
	content, err := t.proxy.ReadFile(ctx, t.sessionID, args.FilePath, 0, 0)
	if err != nil {
		// Check error code BEFORE any wrapping
		errCode := errors.GetCode(err)
		errMsg := err.Error()
		slog.WarnContext(ctx, "ReadFileTool: error reading file", "file_path", args.FilePath, "error", err, "code", errCode)

		// Return most errors as soft errors (result strings) so agent can see and retry
		// Only fatal infrastructure errors should stop the agent

		// Path is a directory
		if strings.Contains(errMsg, "is a directory") {
			slog.DebugContext(ctx, "ReadFileTool: path is a directory, returning soft error", "file_path", args.FilePath)
			return fmt.Sprintf("[ERROR] Path is a directory, not a file: %s. This tool only reads files.", args.FilePath), nil
		}

		// File not found
		if errCode == errors.CodeNotFound || strings.Contains(errMsg, "NOT_FOUND") || strings.Contains(errMsg, "file not found") {
			slog.DebugContext(ctx, "ReadFileTool: file not found, returning soft error", "file_path", args.FilePath)
			return fmt.Sprintf("[ERROR] File not found: %s. Please verify the path is correct.", args.FilePath), nil
		}

		// Permission denied - agent can try other files
		if strings.Contains(errMsg, "permission denied") || strings.Contains(errMsg, "access denied") {
			slog.DebugContext(ctx, "ReadFileTool: permission denied, returning soft error", "file_path", args.FilePath)
			return fmt.Sprintf("[ERROR] Permission denied for file: %s. This file cannot be accessed. Please try a different file.", args.FilePath), nil
		}

		// Timeout or network errors - agent can retry or try different approach
		if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline exceeded") || strings.Contains(errMsg, "connection") {
			slog.DebugContext(ctx, "ReadFileTool: network/timeout error, returning soft error", "file_path", args.FilePath)
			return fmt.Sprintf("[ERROR] Timeout or network error reading file: %s. The operation timed out. Please try again or try a different file.", args.FilePath), nil
		}

		// Invalid path - agent can fix the path
		if strings.Contains(errMsg, "invalid path") || strings.Contains(errMsg, "malformed") {
			slog.DebugContext(ctx, "ReadFileTool: invalid path, returning soft error", "file_path", args.FilePath)
			return fmt.Sprintf("[ERROR] Invalid file path: %s. Please provide a valid relative path from project root.", args.FilePath), nil
		}

		// For any other error, return as soft error so agent can see it and adapt
		// Only return hard error if proxy is not configured (fatal infrastructure issue)
		slog.DebugContext(ctx, "ReadFileTool: generic error, returning soft error", "file_path", args.FilePath, "error", err)
		return fmt.Sprintf("[ERROR] Failed to read file %s: %v. Please verify the file path is correct or try a different approach.", args.FilePath, err), nil
	}

	// Log successful read with content length
	slog.DebugContext(ctx, "ReadFileTool: successfully read file", "file_path", args.FilePath, "content_length", len(content))

	return content, nil
}
