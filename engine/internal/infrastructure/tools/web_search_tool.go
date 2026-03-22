package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/websearch"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// WebSearchProvider defines the interface for web search operations.
// Consumer-side interface: defined here where it's used.
type WebSearchProvider interface {
	Search(ctx context.Context, query string, opts websearch.WebSearchOptions) ([]websearch.WebSearchResult, error)
}

// webSearchArgs represents arguments for web_search tool
type webSearchArgs struct {
	Query          string   `json:"query"`
	MaxResults     int      `json:"max_results,omitempty"`
	IncludeDomains []string `json:"include_domains,omitempty"`
	ExcludeDomains []string `json:"exclude_domains,omitempty"`
}

// WebSearchTool implements Eino InvokableTool for web search
type WebSearchTool struct {
	provider WebSearchProvider
}

// NewWebSearchTool creates a new web_search tool
func NewWebSearchTool(provider WebSearchProvider) tool.InvokableTool {
	return &WebSearchTool{provider: provider}
}

// Info returns tool information for LLM
func (t *WebSearchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "web_search",
		Desc: `Search the web for current information. Returns relevant results with titles, URLs, and snippets.

Use this when you need up-to-date information that may not be in your training data:
- Latest documentation, release notes, changelogs
- Current best practices and solutions
- Information about recent events or updates
- Package versions, API references`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.String,
				Desc:     "Search query (natural language)",
				Required: true,
			},
			"max_results": {
				Type:     schema.Integer,
				Desc:     "Maximum number of results (default: 5, max: 10)",
				Required: false,
			},
			"include_domains": {
				Type: schema.Array,
				Desc: "Only include results from these domains (e.g. [\"github.com\", \"stackoverflow.com\"])",
				ElemInfo: &schema.ParameterInfo{
					Type: schema.String,
				},
				Required: false,
			},
			"exclude_domains": {
				Type: schema.Array,
				Desc: "Exclude results from these domains",
				ElemInfo: &schema.ParameterInfo{
					Type: schema.String,
				},
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the web search tool
func (t *WebSearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args webSearchArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		slog.ErrorContext(ctx, "[WebSearchTool] failed to parse arguments", "error", err, "raw", argumentsInJSON)
		return fmt.Sprintf("[ERROR] Invalid arguments: %v. Please provide a query.", err), nil
	}

	if args.Query == "" {
		return "[ERROR] query is required.", nil
	}

	if args.MaxResults <= 0 {
		args.MaxResults = 5
	}
	if args.MaxResults > 10 {
		args.MaxResults = 10
	}

	slog.InfoContext(ctx, "[WebSearchTool] searching", "query", args.Query, "max_results", args.MaxResults)

	results, err := t.provider.Search(ctx, args.Query, websearch.WebSearchOptions{
		MaxResults:     args.MaxResults,
		IncludeDomains: args.IncludeDomains,
		ExcludeDomains: args.ExcludeDomains,
	})
	if err != nil {
		slog.ErrorContext(ctx, "[WebSearchTool] search failed", "error", err)
		return fmt.Sprintf("[ERROR] Search failed: %v", err), nil
	}

	if len(results) == 0 {
		return fmt.Sprintf("No results found for: \"%s\". Try different search terms.", args.Query), nil
	}

	// Format results as markdown
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Search results for \"%s\"\n\n", args.Query))

	for i, r := range results {
		domain := extractDomain(r.URL)
		sb.WriteString(fmt.Sprintf("%d. **%s** (%s)\n", i+1, r.Title, domain))
		sb.WriteString(fmt.Sprintf("   URL: %s\n", r.URL))
		if r.Content != "" {
			// Truncate long snippets
			snippet := r.Content
			if len(snippet) > 300 {
				snippet = snippet[:300] + "..."
			}
			sb.WriteString(fmt.Sprintf("   %s\n", snippet))
		}
		sb.WriteString("\n")
	}

	slog.InfoContext(ctx, "[WebSearchTool] returning results", "count", len(results))
	return sb.String(), nil
}

// extractDomain extracts domain from a URL
func extractDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return parsed.Host
}
