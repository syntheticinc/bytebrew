package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/errors"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// LspToolArgs represents arguments for lsp tool
type LspToolArgs struct {
	SymbolName string `json:"symbol_name"`
	Operation  string `json:"operation"`
}

// LspTool implements Eino InvokableTool for LSP-based code navigation via gRPC proxy
type LspTool struct {
	proxy     ClientOperationsProxy
	sessionID string
}

// NewLspTool creates a new lsp tool
func NewLspTool(proxy ClientOperationsProxy, sessionID string) tool.InvokableTool {
	return &LspTool{
		proxy:     proxy,
		sessionID: sessionID,
	}
}

// Info returns tool information for LLM
func (t *LspTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "lsp",
		Desc: `Precise code navigation using Language Server Protocol.
Uses LSP servers (gopls, typescript-language-server, etc.) for accurate code analysis.

Operations:
- definition: Go to definition of a symbol (function, type, variable, constant)
- references: Find all references/usages of a symbol across the codebase
- implementation: Find all implementations of an interface or abstract type

You provide a symbol NAME — the system resolves the exact file position automatically
using the code index. No need to specify file path or line numbers.

When to use instead of grep_search:
- "Where is X defined?" → lsp(symbol_name="X", operation="definition")
- "What uses X?" → lsp(symbol_name="X", operation="references")
- "What implements interface X?" → lsp(symbol_name="X", operation="implementation")

When grep_search is better:
- Searching for text patterns (not symbol names)
- Searching across comments, strings, or config files
- Searching for partial matches or regex patterns`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"symbol_name": {
				Type:     schema.String,
				Desc:     "Name of the symbol to look up (function, type, interface, variable, constant)",
				Required: true,
			},
			"operation": {
				Type:     schema.String,
				Desc:     `Operation to perform: "definition", "references", or "implementation"`,
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun executes the tool with JSON arguments
func (t *LspTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args LspToolArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		slog.WarnContext(ctx, "LspTool: failed to parse arguments", "error", err, "raw", argumentsInJSON)
		return fmt.Sprintf("[ERROR] Invalid arguments for lsp: %v", err), nil
	}

	if args.SymbolName == "" {
		return "[ERROR] symbol_name is required.", nil
	}

	validOps := map[string]bool{"definition": true, "references": true, "implementation": true}
	if !validOps[args.Operation] {
		return fmt.Sprintf("[ERROR] Invalid operation: %q. Must be one of: definition, references, implementation.", args.Operation), nil
	}

	if t.proxy == nil {
		return "", errors.New(errors.CodeInternal, "gRPC proxy not configured")
	}

	result, err := t.proxy.LspRequest(ctx, t.sessionID, args.SymbolName, args.Operation)
	if err != nil {
		slog.WarnContext(ctx, "LspTool: proxy error", "symbol", args.SymbolName, "operation", args.Operation, "error", err)
		return fmt.Sprintf("[ERROR] LSP %s failed for symbol %q: %v", args.Operation, args.SymbolName, err), nil
	}

	return result, nil
}
