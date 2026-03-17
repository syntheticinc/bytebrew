package domain

import "testing"

func TestFlow_Validate(t *testing.T) {
	flow := &Flow{
		Type:           FlowType("supervisor"),
		Name:           "main-supervisor",
		SystemPrompt:   "You are a supervisor agent",
		ToolNames:      []string{"manage_stories", "spawn_code_agent"},
		MaxSteps:       50,
		MaxContextSize: 100000,
		Lifecycle: LifecyclePolicy{
			SuspendOn: []string{"final_answer"},
			ReportTo:  "user",
		},
		Spawn: SpawnPolicy{
			AllowedFlows: []FlowType{FlowType("coder")},
		},
	}

	if err := flow.Validate(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestFlow_Validate_MissingType(t *testing.T) {
	flow := &Flow{
		Name:           "test",
		SystemPrompt:   "test",
		MaxSteps:       10,
		MaxContextSize: 1000,
	}

	err := flow.Validate()
	if err == nil {
		t.Error("expected error for missing type, got nil")
	}
	if err.Error() != "flow type is required" {
		t.Errorf("expected 'flow type is required', got: %v", err)
	}
}

func TestFlow_Validate_MissingName(t *testing.T) {
	flow := &Flow{
		Type:           FlowType("coder"),
		SystemPrompt:   "test",
		MaxSteps:       10,
		MaxContextSize: 1000,
	}

	err := flow.Validate()
	if err == nil {
		t.Error("expected error for missing name, got nil")
	}
	if err.Error() != "flow name is required" {
		t.Errorf("expected 'flow name is required', got: %v", err)
	}
}

func TestFlow_Validate_MissingPrompt(t *testing.T) {
	flow := &Flow{
		Type:           FlowType("coder"),
		Name:           "test",
		MaxSteps:       10,
		MaxContextSize: 1000,
	}

	err := flow.Validate()
	if err == nil {
		t.Error("expected error for missing prompt, got nil")
	}
	if err.Error() != "system prompt is required" {
		t.Errorf("expected 'system prompt is required', got: %v", err)
	}
}

func TestFlow_Validate_ZeroMaxSteps_IsValid(t *testing.T) {
	flow := &Flow{
		Type:           FlowType("coder"),
		Name:           "test",
		SystemPrompt:   "test",
		MaxSteps:       0, // 0 = unlimited, should be valid
		MaxContextSize: 1000,
	}

	err := flow.Validate()
	if err != nil {
		t.Errorf("expected no error for zero max_steps (unlimited), got: %v", err)
	}
}

func TestFlow_Validate_NegativeMaxSteps(t *testing.T) {
	flow := &Flow{
		Type:           FlowType("coder"),
		Name:           "test",
		SystemPrompt:   "test",
		MaxSteps:       -1,
		MaxContextSize: 1000,
	}

	err := flow.Validate()
	if err == nil {
		t.Error("expected error for negative max_steps, got nil")
	}
}

func TestFlow_Validate_ZeroMaxContextSize(t *testing.T) {
	flow := &Flow{
		Type:           FlowType("coder"),
		Name:           "test",
		SystemPrompt:   "test",
		MaxSteps:       10,
		MaxContextSize: 0,
	}

	err := flow.Validate()
	if err == nil {
		t.Error("expected error for zero max_context_size, got nil")
	}
	if err.Error() != "max_context_size must be positive" {
		t.Errorf("expected 'max_context_size must be positive', got: %v", err)
	}
}

func TestFlow_CanSpawn(t *testing.T) {
	flow := &Flow{
		Type:           FlowType("supervisor"),
		Name:           "supervisor",
		SystemPrompt:   "test",
		MaxSteps:       10,
		MaxContextSize: 1000,
		Spawn: SpawnPolicy{
			AllowedFlows: []FlowType{FlowType("coder"), FlowType("reviewer")},
		},
	}

	if !flow.CanSpawn(FlowType("coder")) {
		t.Error("expected supervisor to be able to spawn coder")
	}

	if !flow.CanSpawn(FlowType("reviewer")) {
		t.Error("expected supervisor to be able to spawn reviewer")
	}
}

func TestFlow_CanSpawn_NotAllowed(t *testing.T) {
	flow := &Flow{
		Type:           FlowType("coder"),
		Name:           "coder",
		SystemPrompt:   "test",
		MaxSteps:       10,
		MaxContextSize: 1000,
		Spawn: SpawnPolicy{
			AllowedFlows: []FlowType{},
		},
	}

	if flow.CanSpawn(FlowType("supervisor")) {
		t.Error("expected coder not to be able to spawn supervisor")
	}

	if flow.CanSpawn(FlowType("coder")) {
		t.Error("expected coder not to be able to spawn coder")
	}
}

func TestFlow_ShouldSuspendOn(t *testing.T) {
	flow := &Flow{
		Type:           FlowType("supervisor"),
		Name:           "supervisor",
		SystemPrompt:   "test",
		MaxSteps:       10,
		MaxContextSize: 1000,
		Lifecycle: LifecyclePolicy{
			SuspendOn: []string{"final_answer", "ask_user"},
			ReportTo:  "user",
		},
	}

	if !flow.ShouldSuspendOn("final_answer") {
		t.Error("expected flow to suspend on final_answer")
	}

	if !flow.ShouldSuspendOn("ask_user") {
		t.Error("expected flow to suspend on ask_user")
	}
}

func TestFlow_ShouldSuspendOn_Unknown(t *testing.T) {
	flow := &Flow{
		Type:           FlowType("supervisor"),
		Name:           "supervisor",
		SystemPrompt:   "test",
		MaxSteps:       10,
		MaxContextSize: 1000,
		Lifecycle: LifecyclePolicy{
			SuspendOn: []string{"final_answer"},
			ReportTo:  "user",
		},
	}

	if flow.ShouldSuspendOn("unknown_event") {
		t.Error("expected flow not to suspend on unknown_event")
	}
}
