package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	pkgerrors "github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/errors"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// QuestionOption represents a selectable option for a question
type QuestionOption struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// Question represents a single question in the questionnaire
type Question struct {
	Text    string           `json:"text"`
	Options []QuestionOption `json:"options,omitempty"` // up to 5
	Default string           `json:"default,omitempty"`
}

// QuestionAnswer represents the user's answer to a single question
type QuestionAnswer struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// askUserRawArgs uses json.RawMessage to handle questions as either a JSON string or an array.
// LLM sends questions as a JSON-encoded string (since schema type is String),
// but we also accept a raw array for flexibility.
type askUserRawArgs struct {
	Questions json.RawMessage `json:"questions"`
}

// UserAsker defines the interface for asking the user questions via proxy (consumer-side)
type UserAsker interface {
	AskUserQuestionnaire(ctx context.Context, sessionID, questionsJSON string) (string, error)
}

// AskUserTool implements a server-side tool for asking the user structured questions
type AskUserTool struct {
	asker     UserAsker
	sessionID string
}

// NewAskUserTool creates an ask_user tool
func NewAskUserTool(asker UserAsker, sessionID string) tool.InvokableTool {
	return &AskUserTool{asker: asker, sessionID: sessionID}
}

func (t *AskUserTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "ask_user",
		Desc: `Ask the user 1-5 structured questions. Each question can have options (2-5 choices). BLOCKING — waits for response. Use SPARINGLY.

Parameter "questions" is a JSON array of question objects:
[{"text": "What platform?", "options": [{"label": "iOS"}, {"label": "Android"}, {"label": "Both"}], "default": "Both"}]

Each question object:
- "text" (required): The question text
- "options" (optional): Array of 2-5 options, each with "label" (required) and "description" (optional)
- "default" (optional): Default answer shown to user

ONLY for product/requirements:
- "What platforms? iOS, Android, both?" — user's business need
- "Auth needed? Email, OAuth, anonymous?" — product requirement
- Task approval via manage_tasks (automatic, don't use ask_user for this)

NEVER for implementation (decide yourself):
- Technology/framework choice → pick the best
- Project structure → just create it
- Any technical decision a senior dev would make without asking

Combine ALL product questions into ONE ask_user call with multiple questions.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"questions": {Type: schema.String, Desc: `JSON array of questions. Example: [{"text":"What platform?","options":[{"label":"iOS"},{"label":"Android"}],"default":"iOS"}]`, Required: true},
		}),
	}, nil
}

func (t *AskUserTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var raw askUserRawArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &raw); err != nil {
		return fmt.Sprintf("[ERROR] Invalid JSON: %v", err), nil
	}

	// Parse questions: handle both string-encoded JSON and raw array.
	// LLM sends questions as a JSON string (schema type = String), but we also accept a raw array.
	questions, parseErr := parseQuestions(raw.Questions)
	if parseErr != nil {
		return fmt.Sprintf("[ERROR] %s", parseErr), nil
	}

	if len(questions) == 0 {
		return "[ERROR] questions array is required and must contain 1-5 questions", nil
	}

	if len(questions) > 5 {
		return "[ERROR] too many questions: maximum 5 allowed", nil
	}

	// Validate each question
	for i, q := range questions {
		if q.Text == "" {
			return fmt.Sprintf("[ERROR] question %d: text is required", i+1), nil
		}
		if len(q.Options) > 5 {
			return fmt.Sprintf("[ERROR] question %d: too many options, maximum 5 allowed", i+1), nil
		}
	}

	slog.InfoContext(ctx, "[ask_user] asking user questionnaire", "question_count", len(questions))

	// Serialize questions to JSON for the proxy
	questionsJSON, err := json.Marshal(questions)
	if err != nil {
		return fmt.Sprintf("[ERROR] failed to serialize questions: %v", err), nil
	}

	answersJSON, err := t.asker.AskUserQuestionnaire(ctx, t.sessionID, string(questionsJSON))
	if err != nil {
		slog.ErrorContext(ctx, "[ask_user] failed", "error", err)
		if pkgerrors.Is(err, pkgerrors.CodeTimeout) {
			return "[BLOCKED] User has not responded yet. " +
				"You MUST NOT proceed without explicit user approval. " +
				"Do NOT approve tasks, start work, or assume user intent. " +
				"The system will notify you when the user responds.", nil
		}
		return fmt.Sprintf("[ERROR] Failed to get user response: %v", err), nil
	}

	slog.InfoContext(ctx, "[ask_user] got response", "answer_length", len(answersJSON))

	// Parse answers
	var answers []QuestionAnswer
	if err := json.Unmarshal([]byte(answersJSON), &answers); err != nil {
		// If parsing fails, return raw response
		slog.WarnContext(ctx, "[ask_user] failed to parse answers as JSON, returning raw", "error", err)
		return fmt.Sprintf("User response: %s", answersJSON), nil
	}

	// Format answers for LLM
	var sb strings.Builder
	sb.WriteString("User responses:\n")
	for i, a := range answers {
		sb.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, a.Question, a.Answer))
	}
	return sb.String(), nil
}

// parseQuestions handles both formats:
// 1. String-encoded JSON: "questions": "[{\"text\":\"...\"}]" (LLM sends this, schema type=String)
// 2. Raw JSON array: "questions": [{"text":"..."}] (direct array)
func parseQuestions(raw json.RawMessage) ([]Question, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("questions field is required")
	}

	// Try parsing as a JSON string first (most common — LLM sends string)
	var questionsStr string
	if err := json.Unmarshal(raw, &questionsStr); err == nil {
		var questions []Question
		if err := json.Unmarshal([]byte(questionsStr), &questions); err != nil {
			return nil, fmt.Errorf("failed to parse questions string: %w", err)
		}
		return questions, nil
	}

	// Try parsing as a direct JSON array
	var questions []Question
	if err := json.Unmarshal(raw, &questions); err != nil {
		return nil, fmt.Errorf("failed to parse questions: %w", err)
	}
	return questions, nil
}
