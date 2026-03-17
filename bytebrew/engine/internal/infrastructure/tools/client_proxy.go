package tools

import (
	"context"

	pb "github.com/syntheticinc/bytebrew/bytebrew/engine/api/proto/gen"
)

// ClientOperationsProxy defines interface for gRPC client operations
type ClientOperationsProxy interface {
	ReadFile(ctx context.Context, sessionID, filePath string, startLine, endLine int32) (string, error)
	WriteFile(ctx context.Context, sessionID, filePath, content string) (string, error)
	EditFile(ctx context.Context, sessionID, filePath, oldString, newString string, replaceAll bool) (string, error)
	SearchCode(ctx context.Context, sessionID, query, projectKey string, limit int32, minScore float32) ([]byte, error)
	GetProjectTree(ctx context.Context, sessionID, projectKey, path string, maxDepth int) (string, error)
	GrepSearch(ctx context.Context, sessionID, pattern string, limit int32, fileTypes []string, ignoreCase bool) (string, error)
	GlobSearch(ctx context.Context, sessionID, pattern string, limit int32) (string, error)
	SymbolSearch(ctx context.Context, sessionID, symbolName string, limit int32, symbolTypes []string) (string, error)
	ExecuteSubQueries(ctx context.Context, sessionID string, subQueries []*pb.SubQuery) ([]*pb.SubResult, error)
	ExecuteCommand(ctx context.Context, sessionID, command, cwd string, timeout int32) (string, error)
	ExecuteCommandFull(ctx context.Context, sessionID string, arguments map[string]string) (string, error)
	AskUserQuestionnaire(ctx context.Context, sessionID, questionsJSON string) (string, error)
	LspRequest(ctx context.Context, sessionID, symbolName, operation string) (string, error)
}
