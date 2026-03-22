package agent

import (
	"context"
	"strings"
	"testing"
)

func TestParseTestingStrategy_FullConfig(t *testing.T) {
	yaml := `
testing:
  build:
    command: "go build ./..."
    description: "Build all packages"
  unit:
    command: "go test ./..."
    description: "Run unit tests"
    pattern: "*_test.go"
    framework: "testing"
  integration:
    command: "go test -tags=e2e ./tests/..."
    description: "Run integration tests"
  lint:
    command: "golangci-lint run ./..."
    description: "Lint all packages"
  notes: "Run tests from project root"
`

	strategy, err := ParseTestingStrategy(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strategy.Testing.Build == nil {
		t.Fatal("expected build to be set")
	}
	if strategy.Testing.Build.Command != "go build ./..." {
		t.Errorf("build command = %q, want %q", strategy.Testing.Build.Command, "go build ./...")
	}
	if strategy.Testing.Build.Description != "Build all packages" {
		t.Errorf("build description = %q, want %q", strategy.Testing.Build.Description, "Build all packages")
	}

	if strategy.Testing.Unit == nil {
		t.Fatal("expected unit to be set")
	}
	if strategy.Testing.Unit.Command != "go test ./..." {
		t.Errorf("unit command = %q, want %q", strategy.Testing.Unit.Command, "go test ./...")
	}
	if strategy.Testing.Unit.Pattern != "*_test.go" {
		t.Errorf("unit pattern = %q, want %q", strategy.Testing.Unit.Pattern, "*_test.go")
	}
	if strategy.Testing.Unit.Framework != "testing" {
		t.Errorf("unit framework = %q, want %q", strategy.Testing.Unit.Framework, "testing")
	}

	if strategy.Testing.Integration == nil {
		t.Fatal("expected integration to be set")
	}
	if strategy.Testing.Integration.Command != "go test -tags=e2e ./tests/..." {
		t.Errorf("integration command = %q, want %q", strategy.Testing.Integration.Command, "go test -tags=e2e ./tests/...")
	}

	if strategy.Testing.Lint == nil {
		t.Fatal("expected lint to be set")
	}
	if strategy.Testing.Lint.Command != "golangci-lint run ./..." {
		t.Errorf("lint command = %q, want %q", strategy.Testing.Lint.Command, "golangci-lint run ./...")
	}

	if strategy.Testing.Notes != "Run tests from project root" {
		t.Errorf("notes = %q, want %q", strategy.Testing.Notes, "Run tests from project root")
	}
}

func TestParseTestingStrategy_MinimalConfig(t *testing.T) {
	yaml := `
testing:
  unit:
    command: "npm test"
`

	strategy, err := ParseTestingStrategy(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strategy.Testing.Unit == nil {
		t.Fatal("expected unit to be set")
	}
	if strategy.Testing.Unit.Command != "npm test" {
		t.Errorf("unit command = %q, want %q", strategy.Testing.Unit.Command, "npm test")
	}

	if strategy.Testing.Build != nil {
		t.Errorf("expected build to be nil, got %+v", strategy.Testing.Build)
	}
	if strategy.Testing.Integration != nil {
		t.Errorf("expected integration to be nil, got %+v", strategy.Testing.Integration)
	}
	if strategy.Testing.Lint != nil {
		t.Errorf("expected lint to be nil, got %+v", strategy.Testing.Lint)
	}
	if strategy.Testing.Notes != "" {
		t.Errorf("expected notes to be empty, got %q", strategy.Testing.Notes)
	}
}

func TestParseTestingStrategy_EmptyContent(t *testing.T) {
	strategy, err := ParseTestingStrategy("")
	if err != nil {
		t.Fatalf("unexpected error for empty content: %v", err)
	}

	// Empty YAML results in zero-value struct (all nil / empty)
	if strategy.Testing.Build != nil {
		t.Errorf("expected build to be nil")
	}
	if strategy.Testing.Unit != nil {
		t.Errorf("expected unit to be nil")
	}
	if strategy.Testing.Integration != nil {
		t.Errorf("expected integration to be nil")
	}
	if strategy.Testing.Lint != nil {
		t.Errorf("expected lint to be nil")
	}
	if strategy.Testing.Notes != "" {
		t.Errorf("expected notes to be empty")
	}
}

func TestParseTestingStrategy_InvalidYAML(t *testing.T) {
	_, err := ParseTestingStrategy("{{invalid yaml")
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "parse testing strategy") {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

func TestParseTestingStrategy_NotesOnly(t *testing.T) {
	yaml := `
testing:
  notes: |
    Always run linter before pushing.
    Tests require Docker to be running.
`

	strategy, err := ParseTestingStrategy(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strategy.Testing.Build != nil {
		t.Errorf("expected build to be nil")
	}
	if strategy.Testing.Notes == "" {
		t.Fatal("expected notes to be set")
	}
	if !strings.Contains(strategy.Testing.Notes, "Always run linter") {
		t.Errorf("notes missing expected content: %q", strategy.Testing.Notes)
	}
	if !strings.Contains(strategy.Testing.Notes, "Docker") {
		t.Errorf("notes missing expected content: %q", strategy.Testing.Notes)
	}
}

func TestTestingStrategyReminder_FormatsFull(t *testing.T) {
	strategy := &TestingStrategy{
		Testing: TestingConfig{
			Build: &CommandEntry{
				Command:     "go build ./...",
				Description: "Build all",
			},
			Unit: &CommandEntry{
				Command:     "go test ./...",
				Description: "Unit tests",
				Pattern:     "*_test.go",
				Framework:   "testing",
			},
			Integration: &CommandEntry{
				Command:     "go test -tags=e2e ./tests/...",
				Description: "E2E tests",
			},
			Lint: &CommandEntry{
				Command:     "golangci-lint run",
				Description: "Lint check",
			},
			Notes: "Run from project root",
		},
	}

	reminder := NewTestingStrategyReminder(strategy)
	content, priority, ok := reminder.GetContextReminder(context.Background(), "session-1")

	if !ok {
		t.Fatal("expected reminder to be enabled")
	}
	if priority != 85 {
		t.Errorf("priority = %d, want 85", priority)
	}
	if !strings.Contains(content, "PROJECT TESTING STRATEGY") {
		t.Errorf("missing header in content: %s", content)
	}
	if !strings.Contains(content, "go build ./...") {
		t.Errorf("missing build command: %s", content)
	}
	if !strings.Contains(content, "Build all") {
		t.Errorf("missing build description: %s", content)
	}
	if !strings.Contains(content, "go test ./...") {
		t.Errorf("missing unit command: %s", content)
	}
	if !strings.Contains(content, "pattern: *_test.go") {
		t.Errorf("missing unit pattern: %s", content)
	}
	if !strings.Contains(content, "framework: testing") {
		t.Errorf("missing unit framework: %s", content)
	}
	if !strings.Contains(content, "Unit tests") {
		t.Errorf("missing unit description: %s", content)
	}
	if !strings.Contains(content, "go test -tags=e2e") {
		t.Errorf("missing integration command: %s", content)
	}
	if !strings.Contains(content, "golangci-lint run") {
		t.Errorf("missing lint command: %s", content)
	}
	if !strings.Contains(content, "Run from project root") {
		t.Errorf("missing notes: %s", content)
	}
	if !strings.Contains(content, "Do not invent test commands") {
		t.Errorf("missing instruction footer: %s", content)
	}
}

func TestTestingStrategyReminder_EmptyStrategy(t *testing.T) {
	strategy := &TestingStrategy{
		Testing: TestingConfig{},
	}

	reminder := NewTestingStrategyReminder(strategy)
	_, _, ok := reminder.GetContextReminder(context.Background(), "session-1")

	if ok {
		t.Error("expected reminder to be disabled for empty strategy")
	}
}

func TestTestingStrategyReminder_Priority(t *testing.T) {
	strategy := &TestingStrategy{
		Testing: TestingConfig{
			Unit: &CommandEntry{Command: "go test ./..."},
		},
	}

	reminder := NewTestingStrategyReminder(strategy)
	_, priority, ok := reminder.GetContextReminder(context.Background(), "session-1")

	if !ok {
		t.Fatal("expected reminder to be enabled")
	}
	if priority != 85 {
		t.Errorf("priority = %d, want 85", priority)
	}
}
