package domain

import (
	"context"
	"testing"
)

func TestWithAgentID_Roundtrip(t *testing.T) {
	ctx := context.Background()
	ctx = WithAgentID(ctx, "code-agent-abc123")

	got := AgentIDFromContext(ctx)
	if got != "code-agent-abc123" {
		t.Errorf("AgentIDFromContext() = %q, want %q", got, "code-agent-abc123")
	}
}

func TestAgentIDFromContext_Empty(t *testing.T) {
	ctx := context.Background()

	got := AgentIDFromContext(ctx)
	if got != "" {
		t.Errorf("AgentIDFromContext() = %q, want empty string", got)
	}
}

func TestAgentIDFromContext_WrongType(t *testing.T) {
	// Create context with wrong type value
	type wrongKey string
	const key wrongKey = "agentID"
	ctx := context.WithValue(context.Background(), key, 12345) // int instead of string

	got := AgentIDFromContext(ctx)
	if got != "" {
		t.Errorf("AgentIDFromContext() with wrong type = %q, want empty string", got)
	}
}

func TestWithAgentID_Nested(t *testing.T) {
	ctx := context.Background()
	ctx1 := WithAgentID(ctx, "agent-1")
	ctx2 := WithAgentID(ctx1, "agent-2")

	// ctx2 should have agent-2
	got2 := AgentIDFromContext(ctx2)
	if got2 != "agent-2" {
		t.Errorf("AgentIDFromContext(ctx2) = %q, want %q", got2, "agent-2")
	}

	// ctx1 should still have agent-1
	got1 := AgentIDFromContext(ctx1)
	if got1 != "agent-1" {
		t.Errorf("AgentIDFromContext(ctx1) = %q, want %q", got1, "agent-1")
	}

	// original ctx should be empty
	gotOriginal := AgentIDFromContext(ctx)
	if gotOriginal != "" {
		t.Errorf("AgentIDFromContext(ctx) = %q, want empty string", gotOriginal)
	}
}
