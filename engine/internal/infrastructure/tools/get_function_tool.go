package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/indexing"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// GetFunctionArgs represents arguments for get_function tool.
type GetFunctionArgs struct {
	Name string `json:"name"`
}

// GetFunctionTool retrieves function/method source code by name from the index.
type GetFunctionTool struct {
	store    *indexing.ChunkStore
	embedder *indexing.EmbeddingsClient
}

// NewGetFunctionTool creates a new get_function tool.
func NewGetFunctionTool(store *indexing.ChunkStore, embedder *indexing.EmbeddingsClient) tool.InvokableTool {
	return &GetFunctionTool{
		store:    store,
		embedder: embedder,
	}
}

// Info returns tool information for LLM.
func (t *GetFunctionTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_function",
		Desc: `Retrieve function or method source code by name from the project index.
Returns the full source code with file path and line numbers.
Use this when you know the exact function name you want to inspect.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"name": {
				Type:     schema.String,
				Desc:     "Function or method name to look up. Example: 'handleRequest', 'NewServer', 'Process'.",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun executes the tool with JSON arguments.
func (t *GetFunctionTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args GetFunctionArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		slog.WarnContext(ctx, "GetFunctionTool: failed to parse arguments", "error", err, "raw", argumentsInJSON)
		return fmt.Sprintf("[ERROR] Invalid arguments for get_function: %v. Please provide name parameter.", err), nil
	}

	if args.Name == "" {
		return "[ERROR] name is required. Please specify the function name, e.g. {\"name\": \"handleRequest\"}", nil
	}

	// Exact match by name, filter to functions and methods
	chunks, err := t.store.GetByName(ctx, args.Name)
	if err != nil {
		return "", fmt.Errorf("get function by name: %w", err)
	}

	funcTypes := map[indexing.ChunkType]bool{
		indexing.ChunkFunction: true,
		indexing.ChunkMethod:   true,
	}
	filtered := filterByType(chunks, funcTypes)

	if len(filtered) > 0 {
		slog.InfoContext(ctx, "get_function exact match", "name", args.Name, "results", len(filtered))
		return formatChunksAsCode(filtered), nil
	}

	// Semantic fallback
	if t.embedder != nil {
		semQuery := "function " + args.Name
		embedding, err := t.embedder.Embed(ctx, semQuery)
		if err == nil {
			results, err := t.store.Search(ctx, embedding, 10)
			if err == nil {
				var semChunks []indexing.CodeChunk
				nameLower := strings.ToLower(args.Name)
				for _, r := range results {
					if r.Score < 0.3 {
						continue
					}
					if !funcTypes[r.Chunk.ChunkType] {
						continue
					}
					if strings.Contains(strings.ToLower(r.Chunk.Name), nameLower) {
						semChunks = append(semChunks, r.Chunk)
					}
				}
				if len(semChunks) > 0 {
					slog.InfoContext(ctx, "get_function semantic match", "name", args.Name, "results", len(semChunks))
					return formatChunksAsCode(semChunks), nil
				}
			}
		}
	}

	return fmt.Sprintf("No function or method named %q found in the index.", args.Name), nil
}

// formatChunksAsCode formats chunks as markdown code blocks with file info.
func formatChunksAsCode(chunks []indexing.CodeChunk) string {
	var sb strings.Builder
	for i, c := range chunks {
		if i > 0 {
			sb.WriteString("\n\n---\n\n")
		}
		sb.WriteString(fmt.Sprintf("# %s:%d-%d\n", c.FilePath, c.StartLine, c.EndLine))
		sb.WriteString(fmt.Sprintf("```%s\n%s\n```", c.Language, c.Content))
	}
	return sb.String()
}
