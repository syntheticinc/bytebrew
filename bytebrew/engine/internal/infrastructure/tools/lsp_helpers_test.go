package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/lsp"
)

func TestResolveSymbolCharacter(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		symbolName string
		want       int
	}{
		{
			name:       "function at start",
			content:    "func Process() {}",
			symbolName: "Process",
			want:       5,
		},
		{
			name:       "type definition",
			content:    "type Client struct {",
			symbolName: "Client",
			want:       5,
		},
		{
			name:       "not found",
			content:    "package main",
			symbolName: "NotHere",
			want:       0,
		},
		{
			name:       "multiline content uses first line",
			content:    "func Foo() {\n\treturn\n}",
			symbolName: "Foo",
			want:       5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, resolveSymbolCharacter(tt.content, tt.symbolName))
		})
	}
}

func TestCapitalizeFirst(t *testing.T) {
	assert.Equal(t, "Definition", capitalizeFirst("definition"))
	assert.Equal(t, "References", capitalizeFirst("references"))
	assert.Equal(t, "", capitalizeFirst(""))
	assert.Equal(t, "A", capitalizeFirst("a"))
}

func TestFormatLspLocations(t *testing.T) {
	locations := []lsp.Location{
		{URI: "file:///project/src/main.go", Range: lsp.Range{Start: lsp.Position{Line: 10, Character: 0}}},
		{URI: "file:///project/src/util.go", Range: lsp.Range{Start: lsp.Position{Line: 25, Character: 5}}},
	}

	result := formatLspLocations("definition", "Process", "/project/src/handler.go", 5, locations, "/project")
	assert.Contains(t, result, "Definition")
	assert.Contains(t, result, "Process")
	assert.Contains(t, result, ":6") // sourceLine+1 (5+1=6)
}

func TestUriToRelativePath(t *testing.T) {
	// On different OSes the exact result varies, just test it doesn't panic
	// and returns something reasonable
	result := uriToRelativePath("file:///some/path/file.go", "/some/path")
	assert.NotEmpty(t, result)
}
