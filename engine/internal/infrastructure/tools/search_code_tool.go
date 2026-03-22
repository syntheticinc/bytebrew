package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/syntheticinc/bytebrew/engine/pkg/errors"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// SearchCodeArgs represents arguments for search-code tool
type SearchCodeArgs struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

// UnmarshalJSON implements custom unmarshaling to handle limit as string or int
func (s *SearchCodeArgs) UnmarshalJSON(data []byte) error {
	type Alias SearchCodeArgs
	aux := &struct {
		Limit interface{} `json:"limit,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Handle limit as string or int
	if aux.Limit != nil {
		switch v := aux.Limit.(type) {
		case float64:
			s.Limit = int(v)
		case string:
			if v != "" {
				limit, err := strconv.Atoi(v)
				if err != nil {
					return errors.Wrap(err, errors.CodeInvalidInput, "invalid limit value")
				}
				s.Limit = limit
			}
		}
	}

	return nil
}

// SearchCodeTool implements Eino InvokableTool for code search via gRPC
type SearchCodeTool struct {
	proxy      ClientOperationsProxy
	sessionID  string
	projectKey string
}

// NewSearchCodeTool creates a new search-code tool
func NewSearchCodeTool(proxy ClientOperationsProxy, sessionID, projectKey string) tool.InvokableTool {
	return &SearchCodeTool{
		proxy:      proxy,
		sessionID:  sessionID,
		projectKey: projectKey,
	}
}

// Info returns tool information for LLM
func (t *SearchCodeTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "search_code",
		Desc: `Vector-based semantic search. Use smart_search instead — it combines vector, symbol, and grep strategies for better results. Use search_code only if smart_search is unavailable.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.String,
				Desc:     "What to search for",
				Required: true,
			},
			"limit": {
				Type:     schema.Integer,
				Desc:     "Max results (default: 5)",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the tool with JSON arguments
func (t *SearchCodeTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args SearchCodeArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		slog.WarnContext(ctx, "SearchCodeTool: failed to parse arguments", "error", err, "raw", argumentsInJSON)
		return fmt.Sprintf("[ERROR] Invalid arguments for search_code: %v. Please provide query parameter.", err), nil
	}

	if args.Query == "" {
		slog.WarnContext(ctx, "SearchCodeTool: query is required but was empty", "raw_args", argumentsInJSON)
		return "[ERROR] query is required. Please specify what you want to search for, e.g. {\"query\": \"function that handles errors\"}", nil
	}

	if args.Limit == 0 {
		args.Limit = 5
	}

	if t.proxy == nil {
		return "", errors.New(errors.CodeInternal, "gRPC proxy not configured")
	}

	// Call gRPC client proxy to search code
	results, err := t.proxy.SearchCode(ctx, t.sessionID, args.Query, t.projectKey, int32(args.Limit), 0.0)
	if err != nil {
		errMsg := err.Error()
		slog.WarnContext(ctx, "SearchCodeTool: error searching code", "query", args.Query, "error", err)

		// Return errors as soft errors so agent can see them and retry with different query
		// Timeout or network errors - agent can retry
		if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline exceeded") || strings.Contains(errMsg, "connection") {
			slog.DebugContext(ctx, "SearchCodeTool: network/timeout error, returning soft error", "query", args.Query)
			return fmt.Sprintf("[ERROR] Timeout or network error during search for query '%s'. Please try again with a simpler query or different terms.", args.Query), nil
		}

		// No results or empty index - agent can try different query
		if strings.Contains(errMsg, "no results") || strings.Contains(errMsg, "empty index") || strings.Contains(errMsg, "not found") {
			slog.DebugContext(ctx, "SearchCodeTool: no results, returning soft error", "query", args.Query)
			return fmt.Sprintf("[ERROR] No results found for query '%s'. Try using different keywords or a broader search term.", args.Query), nil
		}

		// For any other error, return as soft error so agent can adapt
		slog.DebugContext(ctx, "SearchCodeTool: generic error, returning soft error", "query", args.Query, "error", err)
		return fmt.Sprintf("[ERROR] Failed to search code for query '%s': %v. Please try a different search query or approach.", args.Query, err), nil
	}

	// Log search results
	slog.DebugContext(ctx, "SearchCodeTool: search completed", "query", args.Query, "results_length", len(results))

	// results is already JSON bytes, just convert to string
	return string(results), nil
}
