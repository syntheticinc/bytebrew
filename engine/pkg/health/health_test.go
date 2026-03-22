package health

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestManager_Register(t *testing.T) {
	manager := NewManager()

	checker := &mockChecker{name: "test"}
	manager.Register(checker)

	if len(manager.checkers) != 1 {
		t.Errorf("Expected 1 checker, got %d", len(manager.checkers))
	}
}

func TestManager_Unregister(t *testing.T) {
	manager := NewManager()

	checker := &mockChecker{name: "test"}
	manager.Register(checker)
	manager.Unregister("test")

	if len(manager.checkers) != 0 {
		t.Errorf("Expected 0 checkers, got %d", len(manager.checkers))
	}
}

func TestManager_CheckAll(t *testing.T) {
	manager := NewManager()

	manager.Register(&mockChecker{name: "test1", status: StatusHealthy})
	manager.Register(&mockChecker{name: "test2", status: StatusUnhealthy})

	checks := manager.CheckAll(context.Background())

	if len(checks) != 2 {
		t.Errorf("Expected 2 checks, got %d", len(checks))
	}
}

func TestManager_IsHealthy(t *testing.T) {
	tests := []struct {
		name     string
		checkers []Checker
		want     bool
	}{
		{
			name: "all healthy",
			checkers: []Checker{
				&mockChecker{name: "test1", status: StatusHealthy},
				&mockChecker{name: "test2", status: StatusHealthy},
			},
			want: true,
		},
		{
			name: "one unhealthy",
			checkers: []Checker{
				&mockChecker{name: "test1", status: StatusHealthy},
				&mockChecker{name: "test2", status: StatusUnhealthy},
			},
			want: false,
		},
		{
			name: "one degraded",
			checkers: []Checker{
				&mockChecker{name: "test1", status: StatusHealthy},
				&mockChecker{name: "test2", status: StatusDegraded},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager()
			for _, checker := range tt.checkers {
				manager.Register(checker)
			}

			if got := manager.IsHealthy(context.Background()); got != tt.want {
				t.Errorf("IsHealthy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDatabaseChecker(t *testing.T) {
	tests := []struct {
		name       string
		pingFn     func(ctx context.Context) error
		wantStatus Status
	}{
		{
			name: "healthy database",
			pingFn: func(ctx context.Context) error {
				return nil
			},
			wantStatus: StatusHealthy,
		},
		{
			name: "unhealthy database",
			pingFn: func(ctx context.Context) error {
				return errors.New("connection failed")
			},
			wantStatus: StatusUnhealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewDatabaseChecker("database", tt.pingFn)
			check := checker.Check(context.Background())

			if check.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", check.Status, tt.wantStatus)
			}
			if check.Name != "database" {
				t.Errorf("Name = %v, want database", check.Name)
			}
		})
	}
}

func TestLLMChecker(t *testing.T) {
	tests := []struct {
		name       string
		pingFn     func(ctx context.Context) error
		wantStatus Status
	}{
		{
			name: "healthy LLM",
			pingFn: func(ctx context.Context) error {
				return nil
			},
			wantStatus: StatusHealthy,
		},
		{
			name: "unhealthy LLM",
			pingFn: func(ctx context.Context) error {
				return errors.New("connection failed")
			},
			wantStatus: StatusDegraded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewLLMChecker("llm", tt.pingFn)
			check := checker.Check(context.Background())

			if check.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", check.Status, tt.wantStatus)
			}
			if check.Name != "llm" {
				t.Errorf("Name = %v, want llm", check.Name)
			}
		})
	}
}

// mockChecker is a mock implementation of Checker for testing
type mockChecker struct {
	name   string
	status Status
}

func (m *mockChecker) Name() string {
	return m.name
}

func (m *mockChecker) Check(ctx context.Context) Check {
	return Check{
		Name:      m.name,
		Status:    m.status,
		Timestamp: time.Now(),
	}
}
