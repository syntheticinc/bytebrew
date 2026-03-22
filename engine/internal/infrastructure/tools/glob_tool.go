package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/syntheticinc/bytebrew/engine/pkg/errors"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// GlobArgs represents arguments for glob tool
type GlobArgs struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

// GlobTool implements Eino InvokableTool for file name search via gRPC
type GlobTool struct {
	proxy     ClientOperationsProxy
	sessionID string
}

// NewGlobTool creates a new glob tool
func NewGlobTool(proxy ClientOperationsProxy, sessionID string) tool.InvokableTool {
	return &GlobTool{proxy: proxy, sessionID: sessionID}
}

// Info returns tool information for LLM
func (t *GlobTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "glob",
		Desc: `Fast file pattern matching tool that works with any codebase size.
Supports glob patterns like "**/*.js" or "src/**/*.ts".
Returns matching file paths sorted by modification time (newest first).
Use this tool when you need to find files by name patterns.

Examples:
- "**/*.go" — all Go files
- "src/**/*.ts" — TypeScript files in src/
- "**/test*" — files starting with "test"
- "*.json" — JSON files in root directory`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"pattern": {
				Type:     schema.String,
				Desc:     "Glob pattern to match files. Use ** for recursive directory matching, * for filename matching.",
				Required: true,
			},
			"path": {
				Type:     schema.String,
				Desc:     "Subdirectory to search in (relative to project root). Default: search entire project.",
				Required: false,
			},
			"limit": {
				Type:     schema.Integer,
				Desc:     "Maximum number of files to return. Default: 100.",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the tool with JSON arguments
func (t *GlobTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args GlobArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		slog.WarnContext(ctx, "GlobTool: failed to parse arguments", "error", err, "raw", argumentsInJSON)
		return fmt.Sprintf("[ERROR] Invalid arguments for glob: %v. Please provide pattern parameter.", err), nil
	}

	if args.Pattern == "" {
		return "[ERROR] pattern is required. Please specify a glob pattern, e.g. \"**/*.ts\".", nil
	}

	if args.Limit <= 0 {
		args.Limit = 100
	}

	if t.proxy == nil {
		return "", errors.New(errors.CodeInternal, "gRPC proxy not configured")
	}

	// If path specified, prepend to pattern
	searchPattern := args.Pattern
	if args.Path != "" {
		path := strings.TrimRight(args.Path, "/\\")
		// If pattern doesn't start with the path, prepend it.
		// Check for exact match OR path/ prefix to avoid false positives
		// where path="src" would match pattern "src_utils/**/*.ts".
		if !strings.HasPrefix(args.Pattern, path+"/") && args.Pattern != path {
			searchPattern = path + "/" + args.Pattern
		}
	}

	result, err := t.proxy.GlobSearch(ctx, t.sessionID, searchPattern, int32(args.Limit))
	if err != nil {
		errMsg := err.Error()
		slog.WarnContext(ctx, "GlobTool: error", "pattern", searchPattern, "error", err)

		if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline exceeded") {
			return fmt.Sprintf("[ERROR] Glob search timed out for pattern '%s'. Try a more specific pattern.", searchPattern), nil
		}

		return fmt.Sprintf("[ERROR] Glob search failed for pattern '%s': %v", searchPattern, err), nil
	}

	return result, nil
}
