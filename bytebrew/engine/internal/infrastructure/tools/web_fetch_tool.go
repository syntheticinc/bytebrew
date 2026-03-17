package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/websearch"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const maxFetchContentLength = 15000

// WebFetchProvider defines the interface for web page content extraction.
// Consumer-side interface: defined here where it's used.
type WebFetchProvider interface {
	Fetch(ctx context.Context, url string) (*websearch.WebFetchResult, error)
}

// webFetchArgs represents arguments for web_fetch tool
type webFetchArgs struct {
	URL string `json:"url"`
}

// WebFetchTool implements Eino InvokableTool for fetching web page content
type WebFetchTool struct {
	provider WebFetchProvider
}

// NewWebFetchTool creates a new web_fetch tool
func NewWebFetchTool(provider WebFetchProvider) tool.InvokableTool {
	return &WebFetchTool{provider: provider}
}

// Info returns tool information for LLM
func (t *WebFetchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "web_fetch",
		Desc: `Fetch and extract content from a web page URL. Returns the page text content.

Use this to read specific web pages when you have a URL:
- Documentation pages
- Blog posts, articles
- API references
- GitHub READMEs`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"url": {
				Type:     schema.String,
				Desc:     "URL to fetch (must start with http:// or https://)",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun executes the web fetch tool
func (t *WebFetchTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args webFetchArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		slog.ErrorContext(ctx, "[WebFetchTool] failed to parse arguments", "error", err, "raw", argumentsInJSON)
		return fmt.Sprintf("[ERROR] Invalid arguments: %v. Please provide a url.", err), nil
	}

	if args.URL == "" {
		return "[ERROR] url is required.", nil
	}

	if !strings.HasPrefix(args.URL, "http://") && !strings.HasPrefix(args.URL, "https://") {
		return "[ERROR] url must start with http:// or https://", nil
	}

	slog.InfoContext(ctx, "[WebFetchTool] fetching", "url", args.URL)

	result, err := t.provider.Fetch(ctx, args.URL)
	if err != nil {
		slog.ErrorContext(ctx, "[WebFetchTool] fetch failed", "error", err)
		return fmt.Sprintf("[ERROR] Failed to fetch URL: %v", err), nil
	}

	content := result.Content

	// Truncate if too long
	truncated := false
	if len(content) > maxFetchContentLength {
		content = content[:maxFetchContentLength]
		truncated = true
	}

	var sb strings.Builder
	if result.Title != "" {
		sb.WriteString(fmt.Sprintf("# %s\n", result.Title))
	}
	sb.WriteString(fmt.Sprintf("URL: %s\n\n", result.URL))
	sb.WriteString(content)
	if truncated {
		sb.WriteString("\n\n[content truncated]")
	}

	lines := strings.Count(sb.String(), "\n") + 1
	slog.InfoContext(ctx, "[WebFetchTool] returning content", "url", args.URL, "lines", lines, "truncated", truncated)

	return sb.String(), nil
}
