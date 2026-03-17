//go:build lsp

package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TC-L-09: resolveSymbolCharacter finds the correct column of a symbol in a line of code.
func TestLSP_PositionResolution(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		symbolName string
		want       int
	}{
		{
			name:       "function keyword offset",
			content:    "func Process() {}",
			symbolName: "Process",
			want:       5,
		},
		{
			name:       "type definition offset",
			content:    "type UserRepository struct {",
			symbolName: "UserRepository",
			want:       5,
		},
		{
			name:       "method receiver",
			content:    "func (r *InMemoryRepo) GetByID(id string) string {",
			symbolName: "GetByID",
			want:       23,
		},
		{
			name:       "symbol at start of line",
			content:    "Repository interface {",
			symbolName: "Repository",
			want:       0,
		},
		{
			name:       "symbol not found returns 0",
			content:    "package main",
			symbolName: "NotHere",
			want:       0,
		},
		{
			name:       "multiline uses first line only",
			content:    "func Bar() {\n\tGetByID()\n}",
			symbolName: "Bar",
			want:       5,
		},
		{
			name:       "symbol on second line not found in first line",
			content:    "package main\nfunc Foo() {}",
			symbolName: "Foo",
			want:       0, // Foo is on second line, resolveSymbolCharacter only checks first line
		},
		{
			name:       "indented content with tabs",
			content:    "\t\tfunc Handle() {}",
			symbolName: "Handle",
			want:       7,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveSymbolCharacter(tt.content, tt.symbolName)
			assert.Equal(t, tt.want, got, "resolveSymbolCharacter(%q, %q)", tt.content, tt.symbolName)
		})
	}
}

// TC-L-10: resolveSymbolCharacter does substring matching (not whole-word).
// "Foo" WILL match inside "FooBar" because strings.Index is used internally.
// This test documents the current behavior.
func TestLSP_WholeWordMatch(t *testing.T) {
	// resolveSymbolCharacter uses strings.Index which is a substring match.
	// "Foo" matches "FooBar" at position 5.
	content := "type FooBar struct {}"
	got := resolveSymbolCharacter(content, "Foo")

	// Current behavior: substring match succeeds — "Foo" is found at position 5 inside "FooBar".
	// This is NOT whole-word matching.
	assert.Equal(t, 5, got, "resolveSymbolCharacter uses substring match, so 'Foo' matches 'FooBar'")

	// Verify that the match is indeed a substring (not whole-word):
	// If it were whole-word matching, "Foo" should NOT match "FooBar" and would return 0.
	assert.NotEqual(t, 0, got, "current implementation does NOT do whole-word matching")

	// Exact match works correctly
	gotExact := resolveSymbolCharacter(content, "FooBar")
	assert.Equal(t, 5, gotExact, "exact symbol name matches correctly")
}
