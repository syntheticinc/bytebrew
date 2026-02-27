package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/errors"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// GetProjectTreeArgs represents arguments for get_project_tree tool
type GetProjectTreeArgs struct {
	Path     string `json:"path,omitempty"`
	MaxDepth int    `json:"max_depth,omitempty"`
}

// UnmarshalJSON implements custom unmarshaling to handle max_depth as string or int
func (g *GetProjectTreeArgs) UnmarshalJSON(data []byte) error {
	type Alias GetProjectTreeArgs
	aux := &struct {
		MaxDepth interface{} `json:"max_depth,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(g),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Handle max_depth as string or int
	if aux.MaxDepth != nil {
		switch v := aux.MaxDepth.(type) {
		case float64:
			g.MaxDepth = int(v)
		case string:
			if v != "" {
				maxDepth, err := strconv.Atoi(v)
				if err != nil {
					return errors.Wrap(err, errors.CodeInvalidInput, "invalid max_depth value")
				}
				g.MaxDepth = maxDepth
			}
		}
	}

	return nil
}

// GetProjectTreeTool implements Eino InvokableTool for getting project structure via gRPC
type GetProjectTreeTool struct {
	proxy      ClientOperationsProxy
	sessionID  string
	projectKey string
}

// NewGetProjectTreeTool creates a new get-project-tree tool
func NewGetProjectTreeTool(proxy ClientOperationsProxy, sessionID, projectKey string) tool.InvokableTool {
	return &GetProjectTreeTool{
		proxy:      proxy,
		sessionID:  sessionID,
		projectKey: projectKey,
	}
}

// Info returns tool information for LLM
func (t *GetProjectTreeTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_project_tree",
		Desc: `List files and directories. Use to discover project structure before reading specific files.

Shows ONE level by default. Use path to drill down, max_depth to go deeper.
Auto-filters: node_modules, .git, dist, build, __pycache__, vendor, etc.

Examples:
- get_project_tree() → project root
- get_project_tree(path="src") → contents of src/
- get_project_tree(max_depth=2) → 2 levels deep

When to use:
- Starting research — understand project layout
- Checking if a file/directory exists before read_file
- Finding related files near a known path`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "Directory path relative to project root (default: root). Example: \"src/components\"",
				Required: false,
			},
			"max_depth": {
				Type:     schema.Integer,
				Desc:     "How many levels deep to show (default: 1)",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the tool with JSON arguments
func (t *GetProjectTreeTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args GetProjectTreeArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		slog.WarnContext(ctx, "GetProjectTreeTool: failed to parse arguments", "error", err, "raw", argumentsInJSON)
		return fmt.Sprintf("[ERROR] Invalid arguments for get_project_tree: %v", err), nil
	}

	if args.MaxDepth == 0 {
		args.MaxDepth = 1
	}

	if t.proxy == nil {
		return "", errors.New(errors.CodeInternal, "gRPC proxy not configured")
	}

	// Request subtree from client: extra depth level for child counts
	treeJSON, err := t.proxy.GetProjectTree(ctx, t.sessionID, t.projectKey, args.Path, args.MaxDepth+1)
	if err != nil {
		errMsg := err.Error()
		slog.WarnContext(ctx, "GetProjectTreeTool: error getting project tree", "error", err)

		if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline exceeded") || strings.Contains(errMsg, "connection") {
			return "[ERROR] Timeout or network error getting project tree. Please try using search_code to find specific files instead.", nil
		}
		if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "does not exist") {
			return "[ERROR] Project not found or not indexed. Please use search_code to explore the codebase instead.", nil
		}
		return fmt.Sprintf("[ERROR] Failed to get project tree: %v. Please use search_code to explore the codebase instead.", err), nil
	}

	// Format the subtree (client already navigated to the path, root IS the target)
	compactTree, err := formatTreeFromRoot(treeJSON, args.Path, args.MaxDepth)
	if err != nil {
		slog.WarnContext(ctx, "GetProjectTreeTool: failed to format tree", "error", err, "path", args.Path)
		return fmt.Sprintf("[ERROR] %v", err), nil
	}

	return compactTree, nil
}

// TreeNode represents a node in the project tree
type TreeNode struct {
	Path        string      `json:"path"`
	Name        string      `json:"name"`
	IsDirectory bool        `json:"is_directory"`
	Children    []*TreeNode `json:"children,omitempty"`
}

// skipPatterns lists directories/files to hide from tree output
var skipPatterns = map[string]bool{
	"node_modules":      true,
	".git":              true,
	".idea":             true,
	".vscode":           true,
	"__pycache__":       true,
	".pytest_cache":     true,
	"dist":              true,
	"build":             true,
	".next":             true,
	"coverage":          true,
	".nyc_output":       true,
	"vendor":            true,
	".gradle":           true,
	"target":            true,
	"bin":               true,
	"obj":               true,
	"packages":          true,
	"package-lock.json": true,
	"bun.lock":          true,
	"yarn.lock":         true,
	"pnpm-lock.yaml":    true,
	".DS_Store":         true,
	"Thumbs.db":         true,
}

// formatTreeFromRoot formats a tree where the JSON root IS the target directory.
// displayPath is the original path requested by the agent (for header display).
func formatTreeFromRoot(treeJSON, displayPath string, maxDepth int) (string, error) {
	var root TreeNode
	if err := json.Unmarshal([]byte(treeJSON), &root); err != nil {
		return "", err
	}

	if !root.IsDirectory {
		return fmt.Sprintf("[ERROR] Path is a file, not a directory: %s", displayPath), nil
	}

	var sb strings.Builder

	// Header
	headerPath := root.Name
	if displayPath != "" {
		headerPath = displayPath
	}
	sb.WriteString(fmt.Sprintf("Directory: %s/\n", headerPath))
	if displayPath != "" {
		sb.WriteString(fmt.Sprintf("Use read_file with path: \"%s/<filename>\"\n", displayPath))
	} else {
		sb.WriteString("Use read_file with relative paths shown below.\n")
	}
	sb.WriteString("\n")

	// Render children
	if root.Children == nil || len(root.Children) == 0 {
		sb.WriteString("(empty directory)\n")
		return sb.String(), nil
	}

	renderTreeLevel(&sb, root.Children, "", maxDepth, 0)

	return sb.String(), nil
}

// renderTreeLevel renders tree nodes with depth limit
func renderTreeLevel(sb *strings.Builder, nodes []*TreeNode, prefix string, maxDepth, currentDepth int) {
	if currentDepth >= maxDepth {
		// Show summary for truncated directories
		dirCount := 0
		for _, node := range nodes {
			if node.IsDirectory && !skipPatterns[node.Name] {
				dirCount++
			}
		}
		if dirCount > 0 {
			sb.WriteString(prefix)
			sb.WriteString(fmt.Sprintf("└── ... (%d more subdirectories)\n", dirCount))
		}
		return
	}

	// Filter out skipped items
	filtered := make([]*TreeNode, 0, len(nodes))
	for _, node := range nodes {
		if !skipPatterns[node.Name] {
			filtered = append(filtered, node)
		}
	}

	for i, node := range filtered {
		isLast := i == len(filtered)-1

		connector := "├── "
		if isLast {
			connector = "└── "
		}

		sb.WriteString(prefix)
		sb.WriteString(connector)
		sb.WriteString(node.Name)
		if node.IsDirectory {
			sb.WriteString("/")
			// Show child count for directories
			childCount := countVisibleChildren(node.Children)
			if childCount > 0 {
				sb.WriteString(fmt.Sprintf(" (%d)", childCount))
			}
		}
		sb.WriteString("\n")

		// Recurse for directories
		if node.IsDirectory && len(node.Children) > 0 {
			newPrefix := prefix
			if isLast {
				newPrefix += "    "
			} else {
				newPrefix += "│   "
			}
			renderTreeLevel(sb, node.Children, newPrefix, maxDepth, currentDepth+1)
		}
	}
}

// countVisibleChildren counts non-skipped children
func countVisibleChildren(children []*TreeNode) int {
	count := 0
	for _, child := range children {
		if !skipPatterns[child.Name] {
			count++
		}
	}
	return count
}
