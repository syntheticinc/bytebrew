package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/syntheticinc/bytebrew/engine/pkg/errors"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// EditFileArgs represents arguments for edit_file tool
type EditFileArgs struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

// EditFileTool implements Eino InvokableTool for editing files via gRPC
type EditFileTool struct {
	proxy     ClientOperationsProxy
	sessionID string
}

// NewEditFileTool creates a new edit_file tool
func NewEditFileTool(proxy ClientOperationsProxy, sessionID string) tool.InvokableTool {
	return &EditFileTool{
		proxy:     proxy,
		sessionID: sessionID,
	}
}

// Info returns tool information for LLM
func (t *EditFileTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "edit_file",
		Desc: `Find and replace text in a file. Whitespace-flexible matching is applied (indentation differences are tolerated).

Use for small targeted changes. For large rewrites, use write_file instead.

Rules:
- ALWAYS read_file first — you need the exact text to match.
- old_string must be unique in the file. If multiple matches, add surrounding lines for context or use replace_all.
- old_string and new_string must be different.
- Include enough context (3-5 lines) so old_string matches exactly one location.

Tip: Copy the exact text from read_file output (after the line number prefix) as old_string.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"file_path": {
				Type:     schema.String,
				Desc:     "Path to the file (relative to project root or absolute)",
				Required: true,
			},
			"old_string": {
				Type:     schema.String,
				Desc:     "The exact text to find in the file",
				Required: true,
			},
			"new_string": {
				Type:     schema.String,
				Desc:     "The text to replace old_string with (must be different)",
				Required: true,
			},
			"replace_all": {
				Type:     schema.Boolean,
				Desc:     "Replace all occurrences (default false). Useful for renaming.",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the tool with JSON arguments
func (t *EditFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	slog.InfoContext(ctx, "EditFileTool: raw arguments",
		"json_length", len(argumentsInJSON),
		"json_preview", truncateString(argumentsInJSON, 500))

	var args EditFileArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		slog.WarnContext(ctx, "EditFileTool: failed to parse arguments", "error", err, "raw", argumentsInJSON)
		return fmt.Sprintf("[ERROR] Invalid arguments for edit_file: %v. Raw input length: %d", err, len(argumentsInJSON)), nil
	}

	if args.FilePath == "" {
		return fmt.Sprintf("[ERROR] file_path is required for edit_file. You provided: file_path=%q, old_string length=%d, new_string length=%d. Make sure you pass the file_path parameter.",
			args.FilePath, len(args.OldString), len(args.NewString)), nil
	}

	if args.OldString == "" {
		return fmt.Sprintf("[ERROR] old_string is required for edit_file. You provided: file_path=%q, old_string=%q, new_string length=%d. You must provide the old_string parameter with the exact text to find and replace in the file.",
			args.FilePath, args.OldString, len(args.NewString)), nil
	}

	if args.OldString == args.NewString {
		return "[ERROR] old_string and new_string must be different", nil
	}

	if t.proxy == nil {
		return "", errors.New(errors.CodeInternal, "gRPC proxy not configured")
	}

	slog.InfoContext(ctx, "EditFileTool: editing file",
		"file_path", args.FilePath,
		"old_string_length", len(args.OldString),
		"new_string_length", len(args.NewString),
		"replace_all", args.ReplaceAll)

	result, err := t.proxy.EditFile(ctx, t.sessionID, args.FilePath, args.OldString, args.NewString, args.ReplaceAll)
	if err != nil {
		errCode := errors.GetCode(err)
		slog.WarnContext(ctx, "EditFileTool: error editing file",
			"file_path", args.FilePath,
			"error", err,
			"code", errCode)

		if errCode == errors.CodePermissionDenied {
			return fmt.Sprintf("[PERMISSION] Edit blocked: %s", args.FilePath), nil
		}

		return fmt.Sprintf("[ERROR] Failed to edit file '%s': %v", args.FilePath, err), nil
	}

	slog.InfoContext(ctx, "EditFileTool: file edited successfully",
		"file_path", args.FilePath)

	return result, nil
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
