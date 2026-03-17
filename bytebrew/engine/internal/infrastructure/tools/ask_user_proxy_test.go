package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestLocalProxy_AskUserQuestionnaire_SingleQuestion(t *testing.T) {
	dir := t.TempDir()
	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	questions := []map[string]interface{}{
		{"text": "Choose a framework", "options": []string{"React", "Vue", "Angular"}},
	}
	questionsJSON, err := json.Marshal(questions)
	if err != nil {
		t.Fatalf("marshal questions: %v", err)
	}

	result, err := proxy.AskUserQuestionnaire(ctx, "", string(questionsJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var answers []struct {
		Question string `json:"question"`
		Answer   string `json:"answer"`
	}
	if err := json.Unmarshal([]byte(result), &answers); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if len(answers) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(answers))
	}
	if answers[0].Question != "Choose a framework" {
		t.Errorf("expected question text 'Choose a framework', got %q", answers[0].Question)
	}
	if answers[0].Answer != "React" {
		t.Errorf("expected auto-selected first option 'React', got %q", answers[0].Answer)
	}
}

func TestLocalProxy_AskUserQuestionnaire_MultipleQuestions(t *testing.T) {
	dir := t.TempDir()
	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	questions := []map[string]interface{}{
		{"text": "Language?", "options": []string{"Go", "Rust", "Python"}},
		{"text": "Database?", "options": []string{"PostgreSQL", "SQLite"}},
		{"text": "Deploy?", "options": []string{"Docker", "Bare metal"}},
	}
	questionsJSON, err := json.Marshal(questions)
	if err != nil {
		t.Fatalf("marshal questions: %v", err)
	}

	result, err := proxy.AskUserQuestionnaire(ctx, "", string(questionsJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var answers []struct {
		Question string `json:"question"`
		Answer   string `json:"answer"`
	}
	if err := json.Unmarshal([]byte(result), &answers); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if len(answers) != 3 {
		t.Fatalf("expected 3 answers, got %d", len(answers))
	}

	expected := []struct {
		question string
		answer   string
	}{
		{"Language?", "Go"},
		{"Database?", "PostgreSQL"},
		{"Deploy?", "Docker"},
	}
	for i, exp := range expected {
		if answers[i].Question != exp.question {
			t.Errorf("answer[%d] question: expected %q, got %q", i, exp.question, answers[i].Question)
		}
		if answers[i].Answer != exp.answer {
			t.Errorf("answer[%d] answer: expected %q, got %q", i, exp.answer, answers[i].Answer)
		}
	}
}

func TestLocalProxy_AskUserQuestionnaire_FreeText(t *testing.T) {
	dir := t.TempDir()
	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	questions := []map[string]interface{}{
		{"text": "What is your name?"},
	}
	questionsJSON, err := json.Marshal(questions)
	if err != nil {
		t.Fatalf("marshal questions: %v", err)
	}

	result, err := proxy.AskUserQuestionnaire(ctx, "", string(questionsJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var answers []struct {
		Question string `json:"question"`
		Answer   string `json:"answer"`
	}
	if err := json.Unmarshal([]byte(result), &answers); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if len(answers) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(answers))
	}
	if answers[0].Question != "What is your name?" {
		t.Errorf("expected question 'What is your name?', got %q", answers[0].Question)
	}
	// Free text question (no options) → answer should be empty string
	if answers[0].Answer != "" {
		t.Errorf("expected empty answer for free text question, got %q", answers[0].Answer)
	}
}

func TestLocalProxy_AskUserQuestionnaire_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	_, err := proxy.AskUserQuestionnaire(ctx, "", "not valid json")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestLocalProxy_AskUserQuestionnaire_EmptyQuestions(t *testing.T) {
	dir := t.TempDir()
	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.AskUserQuestionnaire(ctx, "", "[]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var answers []struct {
		Question string `json:"question"`
		Answer   string `json:"answer"`
	}
	if err := json.Unmarshal([]byte(result), &answers); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if len(answers) != 0 {
		t.Errorf("expected 0 answers for empty questions, got %d", len(answers))
	}
}
