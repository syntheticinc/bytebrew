package lsp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathToURI(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "unix absolute path",
			path: "/home/user/project/main.go",
			want: "file:///home/user/project/main.go",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// pathToURI uses filepath.Abs which is OS-dependent,
			// so we only test that it produces a file:// prefix
			result := pathToURI(tt.path)
			assert.Contains(t, result, "file://")
		})
	}
}

func TestDetectLanguageID(t *testing.T) {
	tests := []struct {
		file string
		want string
	}{
		{"main.go", "go"},
		{"app.ts", "typescript"},
		{"app.tsx", "typescriptreact"},
		{"index.js", "javascript"},
		{"app.jsx", "javascriptreact"},
		{"main.py", "python"},
		{"lib.rs", "rust"},
		{"Main.java", "java"},
		{"main.c", "c"},
		{"main.cpp", "cpp"},
		{"main.h", "cpp"},
		{"main.dart", "dart"},
		{"main.rb", "ruby"},
		{"index.php", "php"},
		{"Program.cs", "csharp"},
		{"unknown.xyz", "plaintext"},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			assert.Equal(t, tt.want, detectLanguageID(tt.file))
		})
	}
}

func TestParseLocations_Array(t *testing.T) {
	raw := json.RawMessage(`[
		{"uri": "file:///a/b.go", "range": {"start": {"line": 10, "character": 5}, "end": {"line": 10, "character": 15}}},
		{"uri": "file:///c/d.go", "range": {"start": {"line": 20, "character": 0}, "end": {"line": 25, "character": 0}}}
	]`)

	locations, err := parseLocations(raw)
	require.NoError(t, err)
	require.Len(t, locations, 2)
	assert.Equal(t, "file:///a/b.go", locations[0].URI)
	assert.Equal(t, 10, locations[0].Range.Start.Line)
	assert.Equal(t, "file:///c/d.go", locations[1].URI)
}

func TestParseLocations_Single(t *testing.T) {
	raw := json.RawMessage(`{"uri": "file:///a/b.go", "range": {"start": {"line": 5, "character": 0}, "end": {"line": 5, "character": 10}}}`)

	locations, err := parseLocations(raw)
	require.NoError(t, err)
	require.Len(t, locations, 1)
	assert.Equal(t, "file:///a/b.go", locations[0].URI)
}

func TestParseLocations_Null(t *testing.T) {
	locations, err := parseLocations(json.RawMessage(`null`))
	require.NoError(t, err)
	assert.Nil(t, locations)
}

func TestParseLocations_Empty(t *testing.T) {
	locations, err := parseLocations(json.RawMessage(`[]`))
	require.NoError(t, err)
	assert.Empty(t, locations)
}
