package domain

import (
	"context"
	"testing"
)

func TestWithTenantID_Roundtrip(t *testing.T) {
	ctx := WithTenantID(context.Background(), "tenant-123")
	got := TenantIDFromContext(ctx)
	if got != "tenant-123" {
		t.Errorf("expected %q, got %q", "tenant-123", got)
	}
}

func TestTenantIDFromContext_Empty(t *testing.T) {
	got := TenantIDFromContext(context.Background())
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}
