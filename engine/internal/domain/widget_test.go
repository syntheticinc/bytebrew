package domain

import (
	"testing"
)

func TestNewWidget_Valid(t *testing.T) {
	w, err := NewWidget("My Widget", "schema-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Name != "My Widget" {
		t.Errorf("expected name %q, got %q", "My Widget", w.Name)
	}
	if w.PrimaryColor != "#6366f1" {
		t.Errorf("expected default color %q, got %q", "#6366f1", w.PrimaryColor)
	}
	if w.Position != WidgetPositionBottomRight {
		t.Errorf("expected default position bottom-right, got %s", w.Position)
	}
	if !w.Enabled {
		t.Error("expected enabled by default")
	}
}

func TestNewWidget_EmptyName(t *testing.T) {
	_, err := NewWidget("", "schema-1")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestNewWidget_EmptySchema(t *testing.T) {
	_, err := NewWidget("Widget", "")
	if err == nil {
		t.Fatal("expected error for empty schema_id")
	}
}

func TestWidget_IsOriginAllowed(t *testing.T) {
	tests := []struct {
		name      string
		whitelist []string
		origin    string
		allowed   bool
	}{
		{"wildcard", []string{"*"}, "https://example.com", true},
		{"empty whitelist", []string{}, "https://example.com", true},
		{"exact match", []string{"https://example.com"}, "https://example.com", true},
		{"no match", []string{"https://other.com"}, "https://example.com", false},
		{"subdomain", []string{"example.com"}, "https://sub.example.com", true},
		{"case insensitive", []string{"EXAMPLE.COM"}, "https://example.com", true},
		{"multiple domains", []string{"a.com", "b.com"}, "https://b.com", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Widget{DomainWhitelist: tt.whitelist}
			if got := w.IsOriginAllowed(tt.origin); got != tt.allowed {
				t.Errorf("IsOriginAllowed(%q) = %v, want %v", tt.origin, got, tt.allowed)
			}
		})
	}
}
