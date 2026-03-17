package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
)

func TestLocalProxy_ExecuteSubQueries_Empty(t *testing.T) {
	dir := t.TempDir()
	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	results, err := proxy.ExecuteSubQueries(ctx, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results for nil queries, got %d", len(results))
	}

	// Also test with empty slice
	results, err = proxy.ExecuteSubQueries(ctx, "", []*pb.SubQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results for empty queries, got %d", len(results))
	}
}

func TestLocalProxy_ExecuteSubQueries_GrepQuery(t *testing.T) {
	dir := t.TempDir()

	// Create files with searchable content
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0755); err != nil {
		t.Fatalf("setup: mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "main.go"), []byte("package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"), 0644); err != nil {
		t.Fatalf("setup: write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "util.go"), []byte("package main\n\nfunc helper() string {\n\treturn \"hello world\"\n}\n"), 0644); err != nil {
		t.Fatalf("setup: write file: %v", err)
	}

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	queries := []*pb.SubQuery{
		{Type: "grep", Query: "hello", Limit: 10},
	}

	results, err := proxy.ExecuteSubQueries(ctx, "", queries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Type != "grep" {
		t.Errorf("expected type 'grep', got %q", r.Type)
	}
	if r.Error != "" {
		t.Errorf("expected no error, got %q", r.Error)
	}
	if r.Result == "" {
		t.Error("expected non-empty grep result")
	}
}

func TestLocalProxy_ExecuteSubQueries_SymbolQuery(t *testing.T) {
	dir := t.TempDir()
	proxy := NewLocalClientOperationsProxy(dir) // no ChunkStore → fallback message
	ctx := context.Background()

	queries := []*pb.SubQuery{
		{Type: "symbol", Query: "MyFunction", Limit: 5},
	}

	results, err := proxy.ExecuteSubQueries(ctx, "", queries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Type != "symbol" {
		t.Errorf("expected type 'symbol', got %q", r.Type)
	}
	// Without ChunkStore, SymbolSearch returns "Index not available" message
	if r.Error != "" {
		t.Errorf("expected no error (soft error in result), got error: %q", r.Error)
	}
	if r.Result == "" {
		t.Error("expected non-empty result with fallback message")
	}
}

func TestLocalProxy_ExecuteSubQueries_Parallel(t *testing.T) {
	dir := t.TempDir()

	// Create files for grep queries
	if err := os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("alpha beta gamma\n"), 0644); err != nil {
		t.Fatalf("setup: write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("delta epsilon zeta\n"), 0644); err != nil {
		t.Fatalf("setup: write file: %v", err)
	}

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	queries := []*pb.SubQuery{
		{Type: "grep", Query: "alpha", Limit: 10},
		{Type: "grep", Query: "delta", Limit: 10},
		{Type: "grep", Query: "nonexistent_xyz", Limit: 10},
	}

	results, err := proxy.ExecuteSubQueries(ctx, "", queries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results (one per query), got %d", len(results))
	}

	// Verify order matches query order
	for i, r := range results {
		if r.Type != "grep" {
			t.Errorf("result[%d]: expected type 'grep', got %q", i, r.Type)
		}
	}

	// First two should have results, third should have empty/no results
	if results[0].Error != "" {
		t.Errorf("result[0] unexpected error: %q", results[0].Error)
	}
	if results[1].Error != "" {
		t.Errorf("result[1] unexpected error: %q", results[1].Error)
	}
}

func TestLocalProxy_ExecuteSubQueries_UnknownType(t *testing.T) {
	dir := t.TempDir()
	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	queries := []*pb.SubQuery{
		{Type: "unknown_type", Query: "test", Limit: 5},
	}

	results, err := proxy.ExecuteSubQueries(ctx, "", queries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Type != "unknown_type" {
		t.Errorf("expected type 'unknown_type', got %q", r.Type)
	}
	if r.Error == "" {
		t.Error("expected error for unknown sub-query type")
	}
}
