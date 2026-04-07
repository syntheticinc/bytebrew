package assistant

import (
	"testing"
)

func TestInterviewState_NextQuestion(t *testing.T) {
	state := NewInterviewState()

	q1 := state.NextQuestion()
	if q1 == "" {
		t.Fatal("expected first question")
	}

	state.ProcessAnswer(q1, "website chat, email")
	if !state.Answered["channels"] {
		t.Error("expected channels to be answered")
	}

	q2 := state.NextQuestion()
	if q2 == "" {
		t.Fatal("expected second question")
	}
	if q2 == q1 {
		t.Error("expected different question")
	}
}

func TestInterviewState_IsComplete(t *testing.T) {
	state := NewInterviewState()
	if state.IsComplete() {
		t.Error("should not be complete initially")
	}

	state.ProcessAnswer("channels?", "chat")
	if state.IsComplete() {
		t.Error("should not be complete after just channels")
	}

	state.ProcessAnswer("questions?", "delivery, returns")
	if !state.IsComplete() {
		t.Error("should be complete after channels + queries")
	}
}

func TestInterviewState_ProcessAnswer_Channels(t *testing.T) {
	state := NewInterviewState()
	state.ProcessAnswer("What channels will users use?", "chat, email, telegram")

	if len(state.Channels) != 3 {
		t.Errorf("expected 3 channels, got %d: %v", len(state.Channels), state.Channels)
	}
}

func TestInterviewState_ProcessAnswer_Integrations(t *testing.T) {
	state := NewInterviewState()
	state.ProcessAnswer("Do you need integrations?", "Google Sheets and Slack")

	if len(state.Integrations) != 2 {
		t.Errorf("expected 2 integrations, got %d: %v", len(state.Integrations), state.Integrations)
	}
}

func TestInterviewState_Summary(t *testing.T) {
	state := NewInterviewState()
	state.ProcessAnswer("channels?", "chat")
	state.ProcessAnswer("questions?", "delivery")
	state.SchemaName = "support"
	state.Answered["name"] = true

	summary := state.Summary()
	if summary == "" {
		t.Error("expected non-empty summary")
	}
}

func TestParseListAnswer(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"chat, email, telegram", 3},
		{"chat and email", 2},
		{"chat; email; telegram", 3},
		{"just chat", 1},
		{"", 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseListAnswer(tt.input)
			if len(result) != tt.expected {
				t.Errorf("parseListAnswer(%q) = %d items, want %d", tt.input, len(result), tt.expected)
			}
		})
	}
}
