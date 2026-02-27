package grpc

import (
	"context"
	"encoding/json"
	"fmt"

	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
)

// ClientOperationsAdapter adapts gRPC client to Tools proxy interface
type ClientOperationsAdapter struct {
	client *ClientOperationsClient
}

// NewClientOperationsAdapter creates a new adapter
func NewClientOperationsAdapter(client *ClientOperationsClient) *ClientOperationsAdapter {
	return &ClientOperationsAdapter{
		client: client,
	}
}

// ReadFile implements ClientOperationsProxy interface
func (a *ClientOperationsAdapter) ReadFile(ctx context.Context, sessionID, filePath string, startLine, endLine int32) (string, error) {
	return a.client.ReadFile(ctx, sessionID, filePath, startLine, endLine)
}

// SearchCode implements ClientOperationsProxy interface
func (a *ClientOperationsAdapter) SearchCode(ctx context.Context, sessionID, query, projectKey string, limit int32, minScore float32) ([]byte, error) {
	chunks, err := a.client.SearchCode(ctx, sessionID, query, projectKey, limit, minScore)
	if err != nil {
		return nil, err
	}

	// Convert chunks to JSON for LLM consumption
	return json.Marshal(chunks)
}

// GetProjectTree implements ClientOperationsProxy interface
func (a *ClientOperationsAdapter) GetProjectTree(ctx context.Context, sessionID, projectKey, path string, maxDepth int) (string, error) {
	metadata, nodes, err := a.client.GetProjectTree(ctx, sessionID, projectKey, true)
	if err != nil {
		return "", err
	}

	// Create response structure
	response := struct {
		Metadata *pb.ProjectMetadata `json:"metadata"`
		Nodes    []*pb.TreeNode      `json:"nodes"`
	}{
		Metadata: metadata,
		Nodes:    nodes,
	}

	// Convert to JSON for LLM consumption
	jsonData, err := json.Marshal(response)
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

// GrepSearch implements ClientOperationsProxy interface
func (a *ClientOperationsAdapter) GrepSearch(ctx context.Context, sessionID, pattern string, limit int32, fileTypes []string, ignoreCase bool) (string, error) {
	matches, err := a.client.GrepSearch(ctx, sessionID, pattern, limit, fileTypes, ignoreCase)
	if err != nil {
		return "", err
	}

	// Format matches as text for agent consumption
	var result string
	for _, m := range matches {
		result += fmt.Sprintf("%s:%d\n", m.FilePath, m.Line)
		result += "  " + m.Content + "\n\n"
	}

	return result, nil
}

// SymbolSearch implements ClientOperationsProxy interface
func (a *ClientOperationsAdapter) SymbolSearch(ctx context.Context, sessionID, symbolName string, limit int32, symbolTypes []string) (string, error) {
	matches, err := a.client.SymbolSearch(ctx, sessionID, symbolName, limit, symbolTypes)
	if err != nil {
		return "", err
	}

	// Format matches as text for agent consumption
	var result string
	for _, m := range matches {
		result += "[" + m.SymbolType + "] " + m.SymbolName
		if m.Signature != "" {
			result += " - " + m.Signature
		}
		result += "\n"
		result += fmt.Sprintf("  %s:%d-%d\n\n", m.FilePath, m.StartLine, m.EndLine)
	}

	return result, nil
}
