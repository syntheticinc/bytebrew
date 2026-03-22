package tools

import (
	"context"
	"strings"
	"testing"
)

func TestLocalProxy_LspRequest_NoService(t *testing.T) {
	dir := t.TempDir()
	proxy := NewLocalClientOperationsProxy(dir) // no LspService
	ctx := context.Background()

	result, err := proxy.LspRequest(ctx, "", "MyFunction", "definition")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "LSP not available") {
		t.Errorf("expected 'LSP not available' message, got %q", result)
	}
}

func TestLocalProxy_LspRequest_EmptySymbol(t *testing.T) {
	dir := t.TempDir()
	proxy := NewLocalClientOperationsProxy(dir) // no LspService
	ctx := context.Background()

	// Without LspService, it returns "LSP not available" before checking symbol name.
	// This verifies the early return path.
	result, err := proxy.LspRequest(ctx, "", "", "definition")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "LSP not available") {
		t.Errorf("expected 'LSP not available' message, got %q", result)
	}
}

func TestLocalProxy_LspRequest_InvalidOperation(t *testing.T) {
	dir := t.TempDir()
	proxy := NewLocalClientOperationsProxy(dir) // no LspService
	ctx := context.Background()

	// Without LspService, callLspOperation is never reached.
	// We verify that "LSP not available" is the response for any operation.
	result, err := proxy.LspRequest(ctx, "", "SomeSymbol", "invalid_op")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "LSP not available") {
		t.Errorf("expected 'LSP not available' message, got %q", result)
	}
}

func TestLocalProxy_LspRequest_AllOperationsWithoutService(t *testing.T) {
	dir := t.TempDir()
	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	operations := []string{"definition", "references", "implementation"}
	for _, op := range operations {
		t.Run(op, func(t *testing.T) {
			result, err := proxy.LspRequest(ctx, "", "TestSymbol", op)
			if err != nil {
				t.Fatalf("unexpected error for operation %q: %v", op, err)
			}
			if !strings.Contains(result, "LSP not available") {
				t.Errorf("expected 'LSP not available' for operation %q, got %q", op, result)
			}
		})
	}
}
