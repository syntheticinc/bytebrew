package domain

import "fmt"

// KnowledgeFileStatus represents the indexing status of a knowledge document.
type KnowledgeFileStatus string

const (
	KnowledgeStatusUploading KnowledgeFileStatus = "uploading" // file received, not yet indexed
	KnowledgeStatusIndexing  KnowledgeFileStatus = "indexing"  // chunking + embedding in progress
	KnowledgeStatusReady     KnowledgeFileStatus = "ready"     // indexed and searchable
	KnowledgeStatusError     KnowledgeFileStatus = "error"     // indexing failed
)

// KnowledgeFileType represents supported file formats (AC-KB-FMT-01..05).
type KnowledgeFileType string

const (
	KnowledgeTypePDF  KnowledgeFileType = "pdf"
	KnowledgeTypeDOCX KnowledgeFileType = "docx"
	KnowledgeTypeDOC  KnowledgeFileType = "doc"
	KnowledgeTypeTXT  KnowledgeFileType = "txt"
	KnowledgeTypeMD   KnowledgeFileType = "md"
	KnowledgeTypeCSV  KnowledgeFileType = "csv"
)

// SupportedKnowledgeTypes returns all supported file types.
func SupportedKnowledgeTypes() []KnowledgeFileType {
	return []KnowledgeFileType{
		KnowledgeTypePDF,
		KnowledgeTypeDOCX,
		KnowledgeTypeDOC,
		KnowledgeTypeTXT,
		KnowledgeTypeMD,
		KnowledgeTypeCSV,
	}
}

// IsKnowledgeTypeSupported returns true if the file type is supported (AC-KB-FMT-05).
func IsKnowledgeTypeSupported(fileType string) bool {
	switch KnowledgeFileType(fileType) {
	case KnowledgeTypePDF, KnowledgeTypeDOCX, KnowledgeTypeDOC,
		KnowledgeTypeTXT, KnowledgeTypeMD, KnowledgeTypeCSV:
		return true
	}
	return false
}

// KnowledgeConfig holds per-agent knowledge search parameters.
// Configured via the Knowledge capability config (AC-KB-PARAM-01..03).
type KnowledgeConfig struct {
	TopK                int     // number of results to return (default 5, AC-KB-PARAM-01)
	SimilarityThreshold float64 // minimum similarity score (default 0.75, AC-KB-PARAM-02)
}

// DefaultKnowledgeConfig returns default knowledge search configuration.
func DefaultKnowledgeConfig() KnowledgeConfig {
	return KnowledgeConfig{
		TopK:                5,
		SimilarityThreshold: 0.75,
	}
}

// Validate validates the KnowledgeConfig.
func (c *KnowledgeConfig) Validate() error {
	if c.TopK < 1 {
		return fmt.Errorf("top_k must be >= 1")
	}
	if c.TopK > 50 {
		return fmt.Errorf("top_k must be <= 50")
	}
	if c.SimilarityThreshold < 0 || c.SimilarityThreshold > 1 {
		return fmt.Errorf("similarity_threshold must be between 0 and 1")
	}
	return nil
}

// KnowledgeFileInfo represents file info for the listing API (AC-KB-LIST-02).
type KnowledgeFileInfo struct {
	ID         string
	AgentName  string
	FileName   string
	FileType   string
	FileSize   int64
	Status     KnowledgeFileStatus
	StatusMsg  string
	ChunkCount int
	CreatedAt  string
	IndexedAt  string
}
