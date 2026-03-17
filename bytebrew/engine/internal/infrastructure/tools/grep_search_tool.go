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

// GrepSearchArgs represents arguments for grep_search tool
type GrepSearchArgs struct {
	Pattern    string `json:"pattern"`
	Include    string `json:"include,omitempty"`
	IgnoreCase bool   `json:"ignore_case,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

// GrepSearchTool implements Eino InvokableTool for regex-based search via gRPC
type GrepSearchTool struct {
	proxy     ClientOperationsProxy
	sessionID string
}

// NewGrepSearchTool creates a new grep_search tool
func NewGrepSearchTool(proxy ClientOperationsProxy, sessionID string) tool.InvokableTool {
	return &GrepSearchTool{
		proxy:     proxy,
		sessionID: sessionID,
	}
}

// Info returns tool information for LLM
func (t *GrepSearchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "grep_search",
		Desc: `Fast content search tool that works with any codebase size.
Searches file contents using regular expressions (ripgrep).
Supports full regex syntax (e.g., "log.*Error", "function\\s+\\w+").
Filter files by pattern with the include parameter (e.g., "*.js", "*.{ts,tsx}").
Returns matching lines with file paths and line numbers, sorted by file modification time.
Use this tool when you need to find specific code patterns, function definitions, imports, error handling, etc.

When to use:
- You know the exact pattern to search for (literal text or regex)
- You need to find all occurrences of a symbol/function/import
- You need case-insensitive search (use ignore_case: true)
- You want to search specific file types (use include parameter)

Example patterns:
- "TODO:" — literal text search
- "func.*User" — regex: functions containing "User"
- "import.*express" — regex: imports with express
- "\\berror\\b" — regex: word boundary, exact word "error"`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"pattern": {
				Type:     schema.String,
				Desc:     "Regex pattern to search for. Supports full regex syntax. Literal braces must be escaped.",
				Required: true,
			},
			"include": {
				Type:     schema.String,
				Desc:     "File pattern filter (e.g., \"*.js\", \"*.{ts,tsx}\"). Only search files matching this glob.",
				Required: false,
			},
			"ignore_case": {
				Type:     schema.Boolean,
				Desc:     "Case-insensitive search. Default: false.",
				Required: false,
			},
			"limit": {
				Type:     schema.Integer,
				Desc:     "Maximum number of results to return. Default: 100.",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the tool with JSON arguments
func (t *GrepSearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args GrepSearchArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		slog.WarnContext(ctx, "GrepSearchTool: failed to parse arguments", "error", err, "raw", argumentsInJSON)
		return fmt.Sprintf("[ERROR] Invalid arguments for grep_search: %v. Please provide pattern parameter.", err), nil
	}

	// Fix LLM JSON encoding bug: \b in JSON = backspace (0x08), but LLMs mean \b = word boundary.
	args.Pattern = fixRegexEscapes(args.Pattern)

	if args.Pattern == "" {
		return "[ERROR] pattern is required. Please specify a regex pattern to search for.", nil
	}

	if args.Limit <= 0 {
		args.Limit = 100
	}

	if t.proxy == nil {
		return "", errors.New(errors.CodeInternal, "gRPC proxy not configured")
	}

	// Parse include into fileTypes slice
	var fileTypes []string
	if args.Include != "" {
		fileTypes = strings.Split(args.Include, ",")
		for i, ft := range fileTypes {
			fileTypes[i] = strings.TrimSpace(ft)
		}
	}

	result, err := t.proxy.GrepSearch(ctx, t.sessionID, args.Pattern, int32(args.Limit), fileTypes, args.IgnoreCase)
	if err != nil {
		errMsg := err.Error()
		slog.WarnContext(ctx, "GrepSearchTool: error", "pattern", args.Pattern, "error", err)

		if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline exceeded") {
			return fmt.Sprintf("[ERROR] Search timed out for pattern '%s'. Try a simpler pattern.", args.Pattern), nil
		}

		return fmt.Sprintf("[ERROR] Grep search failed for pattern '%s': %v", args.Pattern, err), nil
	}

	return result, nil
}

// fixRegexEscapes corrects common LLM JSON encoding mistakes.
// JSON spec: \b = backspace (U+0008). But LLMs typically mean \b = regex word boundary.
// We replace backspace characters with literal \b so ripgrep interprets them correctly.
func fixRegexEscapes(pattern string) string {
	return strings.ReplaceAll(pattern, "\x08", `\b`)
}
