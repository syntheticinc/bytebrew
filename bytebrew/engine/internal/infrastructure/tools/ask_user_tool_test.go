package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	pkgerrors "github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/errors"
)

// simpleUserAsker implements UserAsker for testing
type simpleUserAsker struct {
	response      string
	err           error
	questionsJSON string // records last questionsJSON received
}

func (m *simpleUserAsker) AskUserQuestionnaire(_ context.Context, _, questionsJSON string) (string, error) {
	m.questionsJSON = questionsJSON
	return m.response, m.err
}

func TestAskUser_Timeout_ReturnsBlocked(t *testing.T) {
	asker := &simpleUserAsker{
		err: pkgerrors.New(pkgerrors.CodeTimeout, "tool call ask_user timed out after 5m0s"),
	}
	tool := NewAskUserTool(asker, "sess-1")

	questionsJSON := `[{"text":"Approve?"}]`
	args := fmt.Sprintf(`{"questions":%s}`, questionsJSON)

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(result, "[BLOCKED]") {
		t.Errorf("expected [BLOCKED] prefix, got: %s", result)
	}
	if !strings.Contains(result, "MUST NOT proceed") {
		t.Errorf("expected instruction not to proceed, got: %s", result)
	}
}

func TestAskUser_NonTimeoutError_ReturnsError(t *testing.T) {
	asker := &simpleUserAsker{
		err: fmt.Errorf("connection reset"),
	}
	tool := NewAskUserTool(asker, "sess-1")

	args := `{"questions":[{"text":"Approve?"}]}`
	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(result, "[ERROR]") {
		t.Errorf("expected [ERROR] prefix, got: %s", result)
	}
}

func TestAskUser_NormalResponse(t *testing.T) {
	answers := []QuestionAnswer{{Question: "Approve?", Answer: "approved"}}
	answersJSON, _ := json.Marshal(answers)
	asker := &simpleUserAsker{response: string(answersJSON)}
	tool := NewAskUserTool(asker, "sess-1")

	args := `{"questions":[{"text":"Approve?"}]}`
	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Approve?") || !strings.Contains(result, "approved") {
		t.Errorf("expected formatted response with question and answer, got: %s", result)
	}
}

func TestAskUser_MultipleQuestions(t *testing.T) {
	answers := []QuestionAnswer{
		{Question: "Platform?", Answer: "iOS"},
		{Question: "Auth?", Answer: "OAuth"},
	}
	answersJSON, _ := json.Marshal(answers)
	asker := &simpleUserAsker{response: string(answersJSON)}
	tool := NewAskUserTool(asker, "sess-1")

	questions := `[{"text":"Platform?","options":[{"label":"iOS"},{"label":"Android"}]},{"text":"Auth?","options":[{"label":"Email"},{"label":"OAuth"}]}]`
	args := fmt.Sprintf(`{"questions":%s}`, questions)

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "1. Platform?: iOS") {
		t.Errorf("expected formatted first answer, got: %s", result)
	}
	if !strings.Contains(result, "2. Auth?: OAuth") {
		t.Errorf("expected formatted second answer, got: %s", result)
	}
}

func TestAskUser_EmptyQuestionsReturnsError(t *testing.T) {
	asker := &simpleUserAsker{}
	tool := NewAskUserTool(asker, "sess-1")

	result, err := tool.InvokableRun(context.Background(), `{"questions":[]}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "[ERROR]") {
		t.Errorf("expected [ERROR] for empty questions, got: %s", result)
	}
}

func TestAskUser_TooManyQuestionsReturnsError(t *testing.T) {
	asker := &simpleUserAsker{}
	tool := NewAskUserTool(asker, "sess-1")

	// 6 questions — over limit
	args := `{"questions":[{"text":"Q1"},{"text":"Q2"},{"text":"Q3"},{"text":"Q4"},{"text":"Q5"},{"text":"Q6"}]}`
	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "[ERROR]") {
		t.Errorf("expected [ERROR] for too many questions, got: %s", result)
	}
}

func TestAskUser_EmptyQuestionTextReturnsError(t *testing.T) {
	asker := &simpleUserAsker{}
	tool := NewAskUserTool(asker, "sess-1")

	args := `{"questions":[{"text":""}]}`
	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "[ERROR]") {
		t.Errorf("expected [ERROR] for empty question text, got: %s", result)
	}
}

func TestAskUser_TooManyOptionsReturnsError(t *testing.T) {
	asker := &simpleUserAsker{}
	tool := NewAskUserTool(asker, "sess-1")

	args := `{"questions":[{"text":"Q1","options":[{"label":"A"},{"label":"B"},{"label":"C"},{"label":"D"},{"label":"E"},{"label":"F"}]}]}`
	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "[ERROR]") {
		t.Errorf("expected [ERROR] for too many options, got: %s", result)
	}
}

func TestAskUser_QuestionsPassedToProxy(t *testing.T) {
	answers := []QuestionAnswer{{Question: "Platform?", Answer: "iOS"}}
	answersJSON, _ := json.Marshal(answers)
	asker := &simpleUserAsker{response: string(answersJSON)}
	tool := NewAskUserTool(asker, "sess-1")

	args := `{"questions":[{"text":"Platform?","options":[{"label":"iOS"},{"label":"Android"}],"default":"iOS"}]}`
	_, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the proxy received valid questions JSON
	var parsedQuestions []Question
	if err := json.Unmarshal([]byte(asker.questionsJSON), &parsedQuestions); err != nil {
		t.Fatalf("proxy received invalid questions JSON: %v", err)
	}
	if len(parsedQuestions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(parsedQuestions))
	}
	if parsedQuestions[0].Text != "Platform?" {
		t.Errorf("question text = %v, want Platform?", parsedQuestions[0].Text)
	}
	if parsedQuestions[0].Default != "iOS" {
		t.Errorf("question default = %v, want iOS", parsedQuestions[0].Default)
	}
	if len(parsedQuestions[0].Options) != 2 {
		t.Errorf("question options count = %v, want 2", len(parsedQuestions[0].Options))
	}
}

func TestAskUser_RawResponseFallback(t *testing.T) {
	// If proxy returns non-JSON, should still return it as raw
	asker := &simpleUserAsker{response: "plain text answer"}
	tool := NewAskUserTool(asker, "sess-1")

	args := `{"questions":[{"text":"What?"}]}`
	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "plain text answer") {
		t.Errorf("expected raw response fallback, got: %s", result)
	}
}
