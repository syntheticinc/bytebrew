package app

import (
	"context"

	deliveryhttp "github.com/syntheticinc/bytebrew/engine/internal/delivery/http"
	svcknowledge "github.com/syntheticinc/bytebrew/engine/internal/service/knowledge"
)

// knowledgeUploadHTTPAdapter bridges svcknowledge.UploadService to deliveryhttp.KnowledgeFileUploader.
type knowledgeUploadHTTPAdapter struct {
	svc *svcknowledge.UploadService
}

func (a *knowledgeUploadHTTPAdapter) UploadFile(ctx context.Context, tenantID, agentName, fileName, fileType string, fileSize int64, fileHash string, content []byte) (*deliveryhttp.KnowledgeFileResponse, error) {
	resp, err := a.svc.UploadFile(ctx, tenantID, agentName, fileName, fileType, fileSize, fileHash, content)
	if err != nil {
		return nil, err
	}
	return &deliveryhttp.KnowledgeFileResponse{
		ID:         resp.ID,
		FileName:   resp.FileName,
		FileType:   resp.FileType,
		FileSize:   resp.FileSize,
		Status:     resp.Status,
		StatusMsg:  resp.StatusMsg,
		ChunkCount: resp.ChunkCount,
		CreatedAt:  resp.CreatedAt,
		IndexedAt:  resp.IndexedAt,
	}, nil
}

// knowledgeFileListerHTTPAdapter bridges svcknowledge.UploadService to deliveryhttp.KnowledgeFileLister.
type knowledgeFileListerHTTPAdapter struct {
	svc *svcknowledge.UploadService
}

func (a *knowledgeFileListerHTTPAdapter) ListFiles(ctx context.Context, agentName string) ([]deliveryhttp.KnowledgeFileResponse, error) {
	files, err := a.svc.ListFiles(ctx, agentName)
	if err != nil {
		return nil, err
	}
	result := make([]deliveryhttp.KnowledgeFileResponse, len(files))
	for i, f := range files {
		result[i] = deliveryhttp.KnowledgeFileResponse{
			ID:         f.ID,
			FileName:   f.FileName,
			FileType:   f.FileType,
			FileSize:   f.FileSize,
			Status:     f.Status,
			StatusMsg:  f.StatusMsg,
			ChunkCount: f.ChunkCount,
			CreatedAt:  f.CreatedAt,
			IndexedAt:  f.IndexedAt,
		}
	}
	return result, nil
}

func (a *knowledgeFileListerHTTPAdapter) DeleteFile(ctx context.Context, agentName, fileID string) error {
	return a.svc.DeleteFile(ctx, agentName, fileID)
}

func (a *knowledgeFileListerHTTPAdapter) ReindexFile(ctx context.Context, agentName, fileID string) error {
	return a.svc.ReindexFile(ctx, agentName, fileID)
}

