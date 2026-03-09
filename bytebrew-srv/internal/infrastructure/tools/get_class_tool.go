package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/indexing"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// GetClassArgs represents arguments for get_class tool.
type GetClassArgs struct {
	Name string `json:"name"`
}

// GetClassTool retrieves class, struct, or interface source code by name from the index.
type GetClassTool struct {
	store    *indexing.ChunkStore
	embedder *indexing.EmbeddingsClient
}

// NewGetClassTool creates a new get_class tool.
func NewGetClassTool(store *indexing.ChunkStore, embedder *indexing.EmbeddingsClient) tool.InvokableTool {
	return &GetClassTool{
		store:    store,
		embedder: embedder,
	}
}

// Info returns tool information for LLM.
func (t *GetClassTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_class",
		Desc: `Retrieve class, struct, or interface source code by name from the project index.
Returns the full source code with file path and line numbers.
Use this when you know the exact type name you want to inspect.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"name": {
				Type:     schema.String,
				Desc:     "Class, struct, or interface name to look up. Example: 'Server', 'UserRepository', 'Handler'.",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun executes the tool with JSON arguments.
func (t *GetClassTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args GetClassArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		slog.WarnContext(ctx, "GetClassTool: failed to parse arguments", "error", err, "raw", argumentsInJSON)
		return fmt.Sprintf("[ERROR] Invalid arguments for get_class: %v. Please provide name parameter.", err), nil
	}

	if args.Name == "" {
		return "[ERROR] name is required. Please specify the type name, e.g. {\"name\": \"Server\"}", nil
	}

	classTypes := map[indexing.ChunkType]bool{
		indexing.ChunkClass:     true,
		indexing.ChunkStruct:    true,
		indexing.ChunkInterface: true,
	}

	// Exact match
	chunks, err := t.store.GetByName(ctx, args.Name)
	if err != nil {
		return "", fmt.Errorf("get class by name: %w", err)
	}

	filtered := filterByType(chunks, classTypes)
	if len(filtered) > 0 {
		slog.InfoContext(ctx, "get_class exact match", "name", args.Name, "results", len(filtered))
		return formatChunksAsCode(filtered), nil
	}

	// Semantic fallback
	if t.embedder != nil {
		semQuery := "class struct interface " + args.Name
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
					if !classTypes[r.Chunk.ChunkType] {
						continue
					}
					if strings.Contains(strings.ToLower(r.Chunk.Name), nameLower) {
						semChunks = append(semChunks, r.Chunk)
					}
				}
				if len(semChunks) > 0 {
					slog.InfoContext(ctx, "get_class semantic match", "name", args.Name, "results", len(semChunks))
					return formatChunksAsCode(semChunks), nil
				}
			}
		}
	}

	return fmt.Sprintf("No class, struct, or interface named %q found in the index.", args.Name), nil
}
