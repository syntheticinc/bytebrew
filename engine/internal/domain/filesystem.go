package domain

import (
	"context"
	"io"
	"time"
)

// FileInfo represents information about a file
type FileInfo struct {
	Path         string
	Size         int64
	ModifiedTime time.Time
	IsDirectory  bool
}

// FileSystem defines interface for file system operations
type FileSystem interface {
	// ReadFile reads the entire file
	ReadFile(ctx context.Context, path string) ([]byte, error)

	// WriteFile writes data to a file
	WriteFile(ctx context.Context, path string, data []byte) error

	// ReadFileRange reads a range of lines from a file
	ReadFileRange(ctx context.Context, path string, startLine, endLine int) (string, error)

	// ListFiles lists files in a directory
	ListFiles(ctx context.Context, path string) ([]FileInfo, error)

	// FileExists checks if a file exists
	FileExists(ctx context.Context, path string) (bool, error)

	// GetFileInfo gets information about a file
	GetFileInfo(ctx context.Context, path string) (*FileInfo, error)
}

// FileReader defines interface for reading files
type FileReader interface {
	// Open opens a file for reading
	Open(ctx context.Context, path string) (io.ReadCloser, error)

	// ReadLines reads specific lines from a file
	ReadLines(ctx context.Context, path string, startLine, endLine int) ([]string, error)
}

// FileWriter defines interface for writing files
type FileWriter interface {
	// Create creates a new file for writing
	Create(ctx context.Context, path string) (io.WriteCloser, error)

	// Append appends data to an existing file
	Append(ctx context.Context, path string, data []byte) error
}
