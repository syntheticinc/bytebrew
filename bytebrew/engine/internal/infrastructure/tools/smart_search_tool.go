package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	pb "github.com/syntheticinc/bytebrew/bytebrew/engine/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain/search"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// GroupedSearchProxy defines the interface for grouped search operations.
// Consumer-side interface: defined here where it's used.
type GroupedSearchProxy interface {
	// ExecuteSubQueries sends sub-queries to client and waits for results
	ExecuteSubQueries(ctx context.Context, sessionID string, subQueries []*pb.SubQuery) ([]*pb.SubResult, error)
}

// SmartSearchArgs represents arguments for smart_search tool
type SmartSearchArgs struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

// SmartSearchTool implements Eino InvokableTool for hybrid code search
// It combines vector search, grep search, and symbol search for comprehensive results
type SmartSearchTool struct {
	proxy      GroupedSearchProxy
	sessionID  string
	projectKey string
}

// NewSmartSearchTool creates a new smart_search tool
func NewSmartSearchTool(proxy GroupedSearchProxy, sessionID, projectKey string) tool.InvokableTool {
	return &SmartSearchTool{
		proxy:      proxy,
		sessionID:  sessionID,
		projectKey: projectKey,
	}
}

// Info returns tool information for LLM
func (t *SmartSearchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "smart_search",
		Desc: `Search code using semantic, pattern, and symbol matching combined. The primary search tool — use this first.

Returns file paths with line numbers and code snippets. Use read_file to see full context.

When to use:
- Exploring unfamiliar code: "authentication middleware", "error handling"
- Finding definitions: "func NewUserService", "type Config struct"
- Finding usages: "UserRepository", "handleError"

EFFICIENCY: Do NOT call multiple times with similar queries. Combine related terms.
Example: "user authentication JWT" (one call), NOT "user auth" + "JWT token" + "login" (three calls).`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.String,
				Desc:     "What to search for (natural language or code pattern)",
				Required: true,
			},
			"limit": {
				Type:     schema.Integer,
				Desc:     "Max results (default: 10)",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the tool with JSON arguments
func (t *SmartSearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args SmartSearchArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		slog.ErrorContext(ctx, "[SmartSearchTool] failed to parse arguments", "error", err, "raw", argumentsInJSON)
		return fmt.Sprintf("[ERROR] Invalid arguments for smart_search: %v. Please provide query parameter.", err), nil
	}

	if args.Query == "" {
		slog.ErrorContext(ctx, "[SmartSearchTool] query is required but was empty", "raw_args", argumentsInJSON)
		return "[ERROR] smart_search requires a non-empty query parameter. Do NOT retry with empty query. Instead: use get_project_tree to explore the project structure, or read_file to read specific files.", nil
	}

	if args.Limit == 0 {
		args.Limit = 10
	}

	if t.proxy == nil {
		return "[ERROR] Search service not configured", nil
	}

	slog.InfoContext(ctx, "[SmartSearchTool] starting grouped search", "query", args.Query, "limit", args.Limit)

	// Build sub-queries for all three search strategies
	subQueries := []*pb.SubQuery{
		{Type: "symbol", Query: args.Query, Limit: int32(args.Limit)},
		{Type: "vector", Query: args.Query, Limit: int32(args.Limit * 2)}, // More results for vector to allow filtering
		{Type: "grep", Query: args.Query, Limit: int32(args.Limit * 2)},
	}

	// Execute all sub-queries via proxy (client executes in parallel)
	subResults, err := t.proxy.ExecuteSubQueries(ctx, t.sessionID, subQueries)
	if err != nil {
		slog.ErrorContext(ctx, "[SmartSearchTool] failed to execute sub-queries", "error", err)
		return fmt.Sprintf("[ERROR] Search failed: %v", err), nil
	}

	// Parse and merge results from sub-queries
	var vectorCitations, grepCitations, symbolCitations []*search.Citation
	var symbolCount, vectorCount, grepCount int

	for _, sr := range subResults {
		switch sr.Type {
		case "symbol":
			symbolCount = int(sr.Count)
			if sr.Error == "" {
				symbolCitations, _ = parseSymbolResults(sr.Result)
			}
		case "vector":
			vectorCount = int(sr.Count)
			if sr.Error == "" {
				vectorCitations, _ = parseVectorResults([]byte(sr.Result))
			}
		case "grep":
			grepCount = int(sr.Count)
			if sr.Error == "" {
				grepCitations, _ = parseGrepResults(sr.Result)
			}
		}
	}

	slog.InfoContext(ctx, "[SmartSearchTool] sub-query results",
		"symbol_count", symbolCount,
		"vector_count", vectorCount,
		"grep_count", grepCount)

	// Merge results
	merged := mergeResults(vectorCitations, grepCitations, symbolCitations, args.Limit)

	if len(merged) == 0 {
		return fmt.Sprintf("No results found for query: \"%s\". Try different search terms.", args.Query), nil
	}

	// Format output as compact citations
	output := formatCitations(merged)
	slog.InfoContext(ctx, "[SmartSearchTool] returning results", "count", len(merged))
	return output, nil
}
