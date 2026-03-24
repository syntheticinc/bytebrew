package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	pkgerrors "github.com/syntheticinc/bytebrew/engine/pkg/errors"
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

func TestAskUser_WithInputType_SingleSelect(t *testing.T) {
	answers := []QuestionAnswer{{Question: "Platform?", Answer: "ios"}}
	answersJSON, _ := json.Marshal(answers)
	asker := &simpleUserAsker{response: string(answersJSON)}
	tool := NewAskUserTool(asker, "sess-1")

	args := `{"questions":[{"text":"Platform?","input_type":"single_select","options":[{"label":"iOS","value":"ios"},{"label":"Android","value":"android"}]}]}`
	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "ios") {
		t.Errorf("expected answer with value, got: %s", result)
	}

	// Verify input_type is passed through to proxy
	var parsedQuestions []Question
	if err := json.Unmarshal([]byte(asker.questionsJSON), &parsedQuestions); err != nil {
		t.Fatalf("proxy received invalid questions JSON: %v", err)
	}
	if parsedQuestions[0].InputType != "single_select" {
		t.Errorf("input_type = %q, want single_select", parsedQuestions[0].InputType)
	}
}

func TestAskUser_WithInputType_MultiSelect(t *testing.T) {
	answers := []QuestionAnswer{{Question: "Features?", Answer: "auth,logging"}}
	answersJSON, _ := json.Marshal(answers)
	asker := &simpleUserAsker{response: string(answersJSON)}
	tool := NewAskUserTool(asker, "sess-1")

	args := `{"questions":[{"text":"Features?","input_type":"multi_select","options":[{"label":"Auth","value":"auth"},{"label":"Logging","value":"logging"},{"label":"Metrics","value":"metrics"}]}]}`
	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "auth,logging") {
		t.Errorf("expected multi_select answer, got: %s", result)
	}

	var parsedQuestions []Question
	if err := json.Unmarshal([]byte(asker.questionsJSON), &parsedQuestions); err != nil {
		t.Fatalf("proxy received invalid questions JSON: %v", err)
	}
	if parsedQuestions[0].InputType != "multi_select" {
		t.Errorf("input_type = %q, want multi_select", parsedQuestions[0].InputType)
	}
}

func TestAskUser_WithInputType_Confirm(t *testing.T) {
	answers := []QuestionAnswer{{Question: "Deploy?", Answer: "yes"}}
	answersJSON, _ := json.Marshal(answers)
	asker := &simpleUserAsker{response: string(answersJSON)}
	tool := NewAskUserTool(asker, "sess-1")

	args := `{"questions":[{"text":"Deploy?","input_type":"confirm"}]}`
	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "yes") {
		t.Errorf("expected confirm answer, got: %s", result)
	}

	var parsedQuestions []Question
	if err := json.Unmarshal([]byte(asker.questionsJSON), &parsedQuestions); err != nil {
		t.Fatalf("proxy received invalid questions JSON: %v", err)
	}
	if parsedQuestions[0].InputType != "confirm" {
		t.Errorf("input_type = %q, want confirm", parsedQuestions[0].InputType)
	}
}

func TestAskUser_WithoutInputType(t *testing.T) {
	// Backwards compat: no input_type should default to empty (treated as "text" by clients)
	answers := []QuestionAnswer{{Question: "Name?", Answer: "MyProject"}}
	answersJSON, _ := json.Marshal(answers)
	asker := &simpleUserAsker{response: string(answersJSON)}
	tool := NewAskUserTool(asker, "sess-1")

	args := `{"questions":[{"text":"Name?"}]}`
	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "MyProject") {
		t.Errorf("expected answer, got: %s", result)
	}

	var parsedQuestions []Question
	if err := json.Unmarshal([]byte(asker.questionsJSON), &parsedQuestions); err != nil {
		t.Fatalf("proxy received invalid questions JSON: %v", err)
	}
	if parsedQuestions[0].InputType != "" {
		t.Errorf("input_type should be empty for backwards compat, got: %q", parsedQuestions[0].InputType)
	}
}

func TestAskUser_CustomInputType(t *testing.T) {
	// Unknown input_type values should be passed through as-is
	answers := []QuestionAnswer{{Question: "Team?", Answer: "team-alpha"}}
	answersJSON, _ := json.Marshal(answers)
	asker := &simpleUserAsker{response: string(answersJSON)}
	tool := NewAskUserTool(asker, "sess-1")

	args := `{"questions":[{"text":"Team?","input_type":"kilo_team_form","options":[{"label":"Alpha","value":"team-alpha"}]}]}`
	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "team-alpha") {
		t.Errorf("expected answer, got: %s", result)
	}

	var parsedQuestions []Question
	if err := json.Unmarshal([]byte(asker.questionsJSON), &parsedQuestions); err != nil {
		t.Fatalf("proxy received invalid questions JSON: %v", err)
	}
	if parsedQuestions[0].InputType != "kilo_team_form" {
		t.Errorf("custom input_type not passed through, got: %q", parsedQuestions[0].InputType)
	}
}

func TestAskUser_OptionWithValue(t *testing.T) {
	answers := []QuestionAnswer{{Question: "Size?", Answer: "lg"}}
	answersJSON, _ := json.Marshal(answers)
	asker := &simpleUserAsker{response: string(answersJSON)}
	tool := NewAskUserTool(asker, "sess-1")

	args := `{"questions":[{"text":"Size?","options":[{"label":"Large","value":"lg"},{"label":"Small","value":"sm"}]}]}`
	_, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsedQuestions []Question
	if err := json.Unmarshal([]byte(asker.questionsJSON), &parsedQuestions); err != nil {
		t.Fatalf("proxy received invalid questions JSON: %v", err)
	}
	if len(parsedQuestions[0].Options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(parsedQuestions[0].Options))
	}
	if parsedQuestions[0].Options[0].Value != "lg" {
		t.Errorf("option[0].value = %q, want lg", parsedQuestions[0].Options[0].Value)
	}
	if parsedQuestions[0].Options[1].Value != "sm" {
		t.Errorf("option[1].value = %q, want sm", parsedQuestions[0].Options[1].Value)
	}
}

func TestAskUser_OptionWithoutValue(t *testing.T) {
	// When value is omitted, it should be empty in JSON (client defaults to label)
	answers := []QuestionAnswer{{Question: "Color?", Answer: "Red"}}
	answersJSON, _ := json.Marshal(answers)
	asker := &simpleUserAsker{response: string(answersJSON)}
	tool := NewAskUserTool(asker, "sess-1")

	args := `{"questions":[{"text":"Color?","options":[{"label":"Red"},{"label":"Blue"}]}]}`
	_, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsedQuestions []Question
	if err := json.Unmarshal([]byte(asker.questionsJSON), &parsedQuestions); err != nil {
		t.Fatalf("proxy received invalid questions JSON: %v", err)
	}
	if len(parsedQuestions[0].Options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(parsedQuestions[0].Options))
	}
	// Value should be empty when not provided (client defaults to label)
	if parsedQuestions[0].Options[0].Value != "" {
		t.Errorf("option[0].value should be empty when not set, got: %q", parsedQuestions[0].Options[0].Value)
	}
	if parsedQuestions[0].Options[0].Label != "Red" {
		t.Errorf("option[0].label = %q, want Red", parsedQuestions[0].Options[0].Label)
	}
}

func TestAskUser_WithColumns(t *testing.T) {
	answers := []QuestionAnswer{{Question: "Template?", Answer: "basic"}}
	answersJSON, _ := json.Marshal(answers)
	asker := &simpleUserAsker{response: string(answersJSON)}
	tool := NewAskUserTool(asker, "sess-1")

	args := `{"questions":[{"text":"Template?","input_type":"single_select","columns":3,"options":[{"label":"Basic","value":"basic"},{"label":"Pro","value":"pro"},{"label":"Enterprise","value":"enterprise"}]}]}`
	_, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsedQuestions []Question
	if err := json.Unmarshal([]byte(asker.questionsJSON), &parsedQuestions); err != nil {
		t.Fatalf("proxy received invalid questions JSON: %v", err)
	}
	if parsedQuestions[0].Columns != 3 {
		t.Errorf("columns = %d, want 3", parsedQuestions[0].Columns)
	}
}
