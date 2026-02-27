package grpc

import (
	"context"

	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/errors"
	"google.golang.org/grpc"
)

// ClientOperationsClient wraps gRPC client for ClientOperations service
type ClientOperationsClient struct {
	client pb.ClientOperationsServiceClient
}

// NewClientOperationsClient creates a new ClientOperations client
func NewClientOperationsClient(conn *grpc.ClientConn) *ClientOperationsClient {
	return &ClientOperationsClient{
		client: pb.NewClientOperationsServiceClient(conn),
	}
}

// ReadFile reads a file from the client's filesystem
func (c *ClientOperationsClient) ReadFile(ctx context.Context, sessionID, filePath string, startLine, endLine int32) (string, error) {
	req := &pb.ReadFileRequest{
		SessionId: sessionID,
		FilePath:  filePath,
		StartLine: startLine,
		EndLine:   endLine,
	}

	resp, err := c.client.ReadFile(ctx, req)
	if err != nil {
		return "", errors.Wrap(err, errors.CodeInternal, "failed to read file via gRPC")
	}

	if resp.Error != nil {
		// Return error with original code from response (don't wrap to preserve error code)
		return "", errors.New(resp.Error.Code, resp.Error.Message)
	}

	return resp.Content, nil
}

// SearchCode performs vector search on the client's Qdrant instance
func (c *ClientOperationsClient) SearchCode(ctx context.Context, sessionID, query, projectKey string, limit int32, minScore float32) ([]*pb.CodeChunk, error) {
	req := &pb.SearchCodeRequest{
		SessionId:  sessionID,
		Query:      query,
		ProjectKey: projectKey,
		Limit:      limit,
		MinScore:   minScore,
	}

	resp, err := c.client.SearchCode(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to search code via gRPC")
	}

	if resp.Error != nil {
		return nil, errors.New(resp.Error.Code, resp.Error.Message)
	}

	return resp.Chunks, nil
}

// GetProjectTree returns the project file tree with descriptions
func (c *ClientOperationsClient) GetProjectTree(ctx context.Context, sessionID, projectKey string, includeDescriptions bool) (*pb.ProjectMetadata, []*pb.TreeNode, error) {
	req := &pb.GetProjectTreeRequest{
		SessionId:           sessionID,
		ProjectKey:          projectKey,
		IncludeDescriptions: includeDescriptions,
	}

	resp, err := c.client.GetProjectTree(ctx, req)
	if err != nil {
		return nil, nil, errors.Wrap(err, errors.CodeInternal, "failed to get project tree via gRPC")
	}

	if resp.Error != nil {
		return nil, nil, errors.New(resp.Error.Code, resp.Error.Message)
	}

	return resp.Metadata, resp.Nodes, nil
}

// GrepSearch performs pattern-based search using ripgrep on the client
func (c *ClientOperationsClient) GrepSearch(ctx context.Context, sessionID, pattern string, limit int32, fileTypes []string, ignoreCase bool) ([]*pb.GrepMatch, error) {
	req := &pb.GrepSearchRequest{
		SessionId:  sessionID,
		Pattern:    pattern,
		Limit:      limit,
		FileTypes:  fileTypes,
		IgnoreCase: ignoreCase,
	}

	resp, err := c.client.GrepSearch(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to grep search via gRPC")
	}

	if resp.Error != nil {
		return nil, errors.New(resp.Error.Code, resp.Error.Message)
	}

	return resp.Matches, nil
}

// SymbolSearch searches for code symbols by name on the client
func (c *ClientOperationsClient) SymbolSearch(ctx context.Context, sessionID, symbolName string, limit int32, symbolTypes []string) ([]*pb.SymbolMatch, error) {
	req := &pb.SymbolSearchRequest{
		SessionId:   sessionID,
		SymbolName:  symbolName,
		Limit:       limit,
		SymbolTypes: symbolTypes,
	}

	resp, err := c.client.SymbolSearch(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to symbol search via gRPC")
	}

	if resp.Error != nil {
		return nil, errors.New(resp.Error.Code, resp.Error.Message)
	}

	return resp.Matches, nil
}
