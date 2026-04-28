package lsp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_NoConfigForExtension(t *testing.T) {
	svc := NewService("/tmp/project")
	_, err := svc.Definition(context.Background(), "/tmp/project/file.xyz", 0, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no LSP server available")
}

func TestService_HasActiveClients_Empty(t *testing.T) {
	svc := NewService("/tmp/project")
	assert.False(t, svc.HasActiveClients())
}

func TestService_Dispose_Empty(t *testing.T) {
	svc := NewService("/tmp/project")
	svc.Dispose() // should not panic
}

func TestConfigForFile(t *testing.T) {
	tests := []struct {
		file   string
		wantID string
	}{
		{"main.go", "go"},
		{"app.ts", "typescript"},
		{"app.tsx", "typescript"},
		{"index.js", "typescript"},
		{"main.py", "python"},
		{"lib.rs", "rust"},
		{"Main.java", "java"},
		{"main.c", "cpp"},
		{"main.cpp", "cpp"},
		{"main.dart", "dart"},
		{"main.rb", "ruby"},
		{"index.php", "php"},
		{"Program.cs", "csharp"},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			cfg := ConfigForFile(tt.file)
			require.NotNil(t, cfg, "expected config for %s", tt.file)
			assert.Equal(t, tt.wantID, cfg.ID)
		})
	}
}

func TestConfigForFile_Unknown(t *testing.T) {
	cfg := ConfigForFile("data.csv")
	assert.Nil(t, cfg)
}

func TestFindFileUp(t *testing.T) {
	tmp := t.TempDir()
	sub := filepath.Join(tmp, "a", "b", "c")
	require.NoError(t, os.MkdirAll(sub, 0755))

	marker := filepath.Join(tmp, "a", "go.mod")
	require.NoError(t, os.WriteFile(marker, []byte("module test"), 0644))

	result, err := findFileUp("go.mod", sub, tmp)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmp, "a"), result)
}

func TestFindFileUp_NotFound(t *testing.T) {
	tmp := t.TempDir()
	_, err := findFileUp("nonexistent.txt", tmp, tmp)
	require.Error(t, err)
}

func TestFindFileUp_AtStop(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module test"), 0644))

	result, err := findFileUp("go.mod", tmp, tmp)
	require.NoError(t, err)
	assert.Equal(t, tmp, result)
}

func TestWhichBin_NotFound(t *testing.T) {
	result := whichBin("definitely-not-a-real-binary-12345")
	assert.Empty(t, result)
}

func TestAllConfigs_Count(t *testing.T) {
	configs := AllConfigs()
	assert.Equal(t, 10, len(configs))
}
