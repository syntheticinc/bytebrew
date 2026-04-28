package domain

import (
	"testing"
)

func TestNewCodeChunk(t *testing.T) {
	tests := []struct {
		name       string
		chunkID    string
		projectKey string
		userID     string
		filePath   string
		content    string
		startLine  int
		endLine    int
		language   string
		chunkType  string
		chunkName  string
		wantErr    bool
	}{
		{
			name:       "valid code chunk",
			chunkID:    "chunk-1",
			projectKey: "project-1",
			userID:     "user-1",
			filePath:   "main.go",
			content:    "package main",
			startLine:  1,
			endLine:    10,
			language:   "go",
			chunkType:  "function",
			chunkName:  "main",
			wantErr:    false,
		},
		{
			name:       "missing chunk_id",
			chunkID:    "",
			projectKey: "project-1",
			userID:     "user-1",
			filePath:   "main.go",
			content:    "package main",
			startLine:  1,
			endLine:    10,
			language:   "go",
			chunkType:  "function",
			chunkName:  "main",
			wantErr:    true,
		},
		{
			name:       "invalid line range",
			chunkID:    "chunk-1",
			projectKey: "project-1",
			userID:     "user-1",
			filePath:   "main.go",
			content:    "package main",
			startLine:  10,
			endLine:    5,
			language:   "go",
			chunkType:  "function",
			chunkName:  "main",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk, err := NewCodeChunk(
				tt.chunkID, tt.projectKey, tt.userID, tt.filePath, tt.content,
				tt.startLine, tt.endLine, tt.language, tt.chunkType, tt.chunkName,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewCodeChunk() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && chunk == nil {
				t.Error("NewCodeChunk() returned nil chunk without error")
			}
		})
	}
}

func TestCodeChunk_LineCount(t *testing.T) {
	chunk, _ := NewCodeChunk(
		"chunk-1", "project-1", "user-1", "main.go", "content",
		1, 10, "go", "function", "main",
	)

	if got := chunk.LineCount(); got != 10 {
		t.Errorf("LineCount() = %v, want 10", got)
	}
}

func TestCodeChunk_SetEmbedding(t *testing.T) {
	chunk, _ := NewCodeChunk(
		"chunk-1", "project-1", "user-1", "main.go", "content",
		1, 10, "go", "function", "main",
	)

	tests := []struct {
		name      string
		embedding []float32
		wantErr   bool
	}{
		{
			name:      "valid embedding",
			embedding: []float32{0.1, 0.2, 0.3},
			wantErr:   false,
		},
		{
			name:      "empty embedding",
			embedding: []float32{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := chunk.SetEmbedding(tt.embedding)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetEmbedding() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCodeChunk_Validate_MaxSize(t *testing.T) {
	largeContent := make([]byte, 11000) // Exceeds max size of 10KB
	for i := range largeContent {
		largeContent[i] = 'a'
	}

	_, err := NewCodeChunk(
		"chunk-1", "project-1", "user-1", "main.go", string(largeContent),
		1, 10, "go", "function", "main",
	)

	if err == nil {
		t.Error("Expected error for chunk exceeding max size, got nil")
	}
}
