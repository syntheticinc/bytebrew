package assistant

import (
	"testing"
)

func TestRouter_FirstVisit_NoSchemas(t *testing.T) {
	router := NewRouter(false, true)
	intent := router.Classify("I need a support system")
	if intent != IntentInterview {
		t.Errorf("expected interview for first visit, got %s", intent)
	}
}

func TestRouter_VagueRequest(t *testing.T) {
	router := NewRouter(true, false)

	tests := []struct {
		msg    string
		intent RequestIntent
	}{
		{"make something for my business", IntentInterview},
		{"I need help", IntentInterview},
		{"build me a bot", IntentInterview},
	}
	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			got := router.Classify(tt.msg)
			if got != tt.intent {
				t.Errorf("Classify(%q) = %s, want %s", tt.msg, got, tt.intent)
			}
		})
	}
}

func TestRouter_ClearRequest(t *testing.T) {
	router := NewRouter(true, false)

	tests := []struct {
		msg    string
		intent RequestIntent
	}{
		{"add a new agent called classifier to the support schema", IntentDirect},
		{"connect the memory capability to the support agent", IntentDirect},
		{"delete the trigger for cron job", IntentDirect},
		{"update the system prompt of classifier agent", IntentDirect},
	}
	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			got := router.Classify(tt.msg)
			if got != tt.intent {
				t.Errorf("Classify(%q) = %s, want %s", tt.msg, got, tt.intent)
			}
		})
	}
}

func TestRouter_Question(t *testing.T) {
	router := NewRouter(true, false)

	tests := []struct {
		msg    string
		intent RequestIntent
	}{
		{"how do flows work?", IntentAnswer},
		{"what is a schema?", IntentAnswer},
		{"can I connect multiple agents?", IntentAnswer},
		{"why is my agent not responding?", IntentAnswer},
	}
	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			got := router.Classify(tt.msg)
			if got != tt.intent {
				t.Errorf("Classify(%q) = %s, want %s", tt.msg, got, tt.intent)
			}
		})
	}
}

func TestRouter_EscapePhrase(t *testing.T) {
	router := NewRouter(false, true) // would normally interview

	got := router.Classify("just do it and create something")
	if got != IntentDirect {
		t.Errorf("expected direct for escape phrase, got %s", got)
	}
}

func TestRouter_Modification(t *testing.T) {
	router := NewRouter(true, false)

	got := router.Classify("добавь новый агент в схему")
	if got != IntentDirect {
		t.Errorf("expected direct for modification, got %s", got)
	}
}
