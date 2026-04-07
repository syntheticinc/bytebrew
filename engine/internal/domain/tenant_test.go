package domain

import (
	"context"
	"testing"
)

func TestNewTenant_Valid(t *testing.T) {
	tenant, err := NewTenant("user@example.com", PlanFree)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tenant.Email != "user@example.com" {
		t.Errorf("expected email %q, got %q", "user@example.com", tenant.Email)
	}
	if tenant.Plan != PlanFree {
		t.Errorf("expected plan %q, got %q", PlanFree, tenant.Plan)
	}
}

func TestNewTenant_EmptyEmail(t *testing.T) {
	_, err := NewTenant("", PlanFree)
	if err == nil {
		t.Fatal("expected error for empty email")
	}
}

func TestNewTenant_InvalidPlan(t *testing.T) {
	_, err := NewTenant("user@example.com", CloudPlan("invalid"))
	if err == nil {
		t.Fatal("expected error for invalid plan")
	}
}

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
