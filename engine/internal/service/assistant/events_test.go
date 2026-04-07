package assistant

import (
	"testing"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

func TestNewNodeCreateEvent(t *testing.T) {
	pos := map[string]float64{"x": 100, "y": 200}
	e := NewNodeCreateEvent("agent-a", pos)

	if e.Type != domain.AgentEventType(AdminEventNodeCreate) {
		t.Errorf("expected type %q, got %q", AdminEventNodeCreate, e.Type)
	}
	if e.Content != "agent-a" {
		t.Errorf("expected content %q, got %q", "agent-a", e.Content)
	}
	if e.Metadata["animation"] != string(AnimationFadeIn) {
		t.Errorf("expected animation %q, got %v", AnimationFadeIn, e.Metadata["animation"])
	}
}

func TestNewNodeUpdateEvent(t *testing.T) {
	e := NewNodeUpdateEvent("agent-a", "system_prompt", "You are helpful")

	if e.Type != domain.AgentEventType(AdminEventNodeUpdate) {
		t.Errorf("expected type %q, got %q", AdminEventNodeUpdate, e.Type)
	}
	if e.Metadata["field"] != "system_prompt" {
		t.Errorf("expected field %q, got %v", "system_prompt", e.Metadata["field"])
	}
	if e.Metadata["animation"] != string(AnimationPulse) {
		t.Errorf("expected animation %q, got %v", AnimationPulse, e.Metadata["animation"])
	}
}

func TestNewEdgeCreateEvent(t *testing.T) {
	e := NewEdgeCreateEvent("agent-a", "agent-b", "flow")

	if e.Type != domain.AgentEventType(AdminEventEdgeCreate) {
		t.Errorf("expected type %q, got %q", AdminEventEdgeCreate, e.Type)
	}
	if e.Metadata["source"] != "agent-a" {
		t.Errorf("expected source %q, got %v", "agent-a", e.Metadata["source"])
	}
	if e.Metadata["animation"] != string(AnimationDraw) {
		t.Errorf("expected animation %q, got %v", AnimationDraw, e.Metadata["animation"])
	}
}

func TestNewPageNavigateEvent(t *testing.T) {
	e := NewPageNavigateEvent("agents", map[string]interface{}{"name": "agent-a"})

	if e.Type != domain.AgentEventType(AdminEventPageNavigate) {
		t.Errorf("expected type %q, got %q", AdminEventPageNavigate, e.Type)
	}
	if e.Metadata["page"] != "agents" {
		t.Errorf("expected page %q, got %v", "agents", e.Metadata["page"])
	}
}

func TestNewFieldUpdateEvent(t *testing.T) {
	e := NewFieldUpdateEvent("agent-edit", "prompt-section", "system_prompt", "You are helpful")

	if e.Type != domain.AgentEventType(AdminEventFieldUpdate) {
		t.Errorf("expected type %q, got %q", AdminEventFieldUpdate, e.Type)
	}
	if e.Metadata["component"] != "prompt-section" {
		t.Errorf("expected component %q, got %v", "prompt-section", e.Metadata["component"])
	}
	if e.Metadata["animation"] != string(AnimationTyping) {
		t.Errorf("expected animation %q, got %v", AnimationTyping, e.Metadata["animation"])
	}
}
