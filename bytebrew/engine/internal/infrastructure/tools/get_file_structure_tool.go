package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/indexing"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// GetFileStructureArgs represents arguments for get_file_structure tool.
type GetFileStructureArgs struct {
	FilePath string `json:"file_path"`
}

// GetFileStructureTool shows the structure (classes, functions, interfaces) of a file from the index.
type GetFileStructureTool struct {
	store       *indexing.ChunkStore
	projectRoot string
}

// NewGetFileStructureTool creates a new get_file_structure tool.
func NewGetFileStructureTool(store *indexing.ChunkStore, projectRoot string) tool.InvokableTool {
	return &GetFileStructureTool{
		store:       store,
		projectRoot: projectRoot,
	}
}

// Info returns tool information for LLM.
func (t *GetFileStructureTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_file_structure",
		Desc: `Show the structure of a file: all classes, structs, interfaces, functions, and methods defined in it.
Useful for understanding a file's organization without reading the full source code.
Requires the project to be indexed.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"file_path": {
				Type:     schema.String,
				Desc:     "Relative path to the file from project root. Example: 'internal/domain/user.go', 'src/app.ts'.",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun executes the tool with JSON arguments.
func (t *GetFileStructureTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args GetFileStructureArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		slog.WarnContext(ctx, "GetFileStructureTool: failed to parse arguments", "error", err, "raw", argumentsInJSON)
		return fmt.Sprintf("[ERROR] Invalid arguments for get_file_structure: %v. Please provide file_path parameter.", err), nil
	}

	if args.FilePath == "" {
		return "[ERROR] file_path is required. Please specify the file path, e.g. {\"file_path\": \"internal/domain/user.go\"}", nil
	}

	// Strip leading slashes
	cleanPath := strings.TrimLeft(args.FilePath, "/\\")

	// Resolve to absolute path for ChunkStore lookup
	resolvedPath := filepath.Join(t.projectRoot, cleanPath)
	resolvedPath = filepath.Clean(resolvedPath)

	chunks, err := t.store.GetByFilePath(ctx, resolvedPath)
	if err != nil {
		return "", fmt.Errorf("get file structure: %w", err)
	}

	if len(chunks) == 0 {
		return fmt.Sprintf("No indexed symbols found for %q. The file may not be indexed yet.", cleanPath), nil
	}

	// Sort by start line
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].StartLine < chunks[j].StartLine
	})

	// Group by type
	groups := map[indexing.ChunkType][]indexing.CodeChunk{}
	for _, c := range chunks {
		groups[c.ChunkType] = append(groups[c.ChunkType], c)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n", cleanPath))

	typeOrder := []struct {
		ct    indexing.ChunkType
		title string
	}{
		{indexing.ChunkInterface, "Interfaces"},
		{indexing.ChunkClass, "Classes"},
		{indexing.ChunkStruct, "Structs"},
		{indexing.ChunkFunction, "Functions"},
		{indexing.ChunkMethod, "Methods"},
		{indexing.ChunkOther, "Other"},
	}

	for _, to := range typeOrder {
		items := groups[to.ct]
		if len(items) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("\n## %s\n", to.title))
		for _, c := range items {
			sig := c.Signature
			if sig == "" {
				sig = c.Name
			}
			sb.WriteString(fmt.Sprintf("- %s (line %d-%d)\n", sig, c.StartLine, c.EndLine))
		}
	}

	slog.InfoContext(ctx, "get_file_structure", "path", cleanPath, "symbols", len(chunks))
	return sb.String(), nil
}
