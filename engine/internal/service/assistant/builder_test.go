package assistant

import (
	"context"
	"testing"
)

func TestBuilder_FirstVisit_StartsInterview(t *testing.T) {
	ops := &mockAdminOps{}
	builder := NewBuilder(ops)

	response, err := builder.HandleMessage(context.Background(), "session-1", "I need a support system",
		false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should start interview
	if response == "" {
		t.Fatal("expected non-empty response")
	}

	// Interview state should exist
	state, ok := builder.GetInterviewState("session-1")
	if !ok {
		t.Fatal("expected interview state to exist")
	}
	if state.IsComplete() {
		t.Error("interview should not be complete yet")
	}
}

func TestBuilder_FullInterview_ToAssembly(t *testing.T) {
	ops := &mockAdminOps{}
	builder := NewBuilder(ops)
	stream := &mockEventStream{}

	// First message: starts interview
	_, err := builder.HandleMessage(context.Background(), "session-1", "I need a support system",
		false, stream)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Answer channels
	_, err = builder.HandleMessage(context.Background(), "session-1", "website chat",
		false, stream)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Answer queries — this should complete the interview and trigger assembly
	response, err := builder.HandleMessage(context.Background(), "session-1", "delivery, returns, sizing",
		false, stream)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have assembled
	if len(ops.schemas) == 0 {
		t.Error("expected schema to be created")
	}
	if len(ops.agents) == 0 {
		t.Error("expected agents to be created")
	}

	// Response should mention completion
	if response == "" {
		t.Fatal("expected completion response")
	}

	// Interview state should be cleaned up
	_, ok := builder.GetInterviewState("session-1")
	if ok {
		t.Error("expected interview state to be cleaned up after assembly")
	}
}

func TestBuilder_Question_AnswersDirectly(t *testing.T) {
	ops := &mockAdminOps{}
	builder := NewBuilder(ops)

	response, err := builder.HandleMessage(context.Background(), "session-1", "how do flows work?",
		true, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response == "" {
		t.Fatal("expected non-empty answer")
	}
	// Should not create any resources
	if len(ops.schemas) > 0 || len(ops.agents) > 0 {
		t.Error("answering a question should not create resources")
	}
}

func TestBuilder_DirectModification(t *testing.T) {
	ops := &mockAdminOps{}
	builder := NewBuilder(ops)

	response, err := builder.HandleMessage(context.Background(), "session-1", "add a new agent called classifier",
		true, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response == "" {
		t.Fatal("expected non-empty response")
	}
}

func TestIsSystemAgent(t *testing.T) {
	if !IsSystemAgent(BuilderAgentName) {
		t.Error("expected BuilderAgentName to be system agent")
	}
	if IsSystemAgent("my-agent") {
		t.Error("expected my-agent to not be system agent")
	}
}

func TestBuilder_Isolation(t *testing.T) {
	// Builder agent should not appear in user agent list
	if BuilderAgentName == "" {
		t.Error("BuilderAgentName should not be empty")
	}
	// System agent names start with __
	if BuilderAgentName[0] != '_' || BuilderAgentName[1] != '_' {
		t.Error("system agent name should start with __")
	}
}
