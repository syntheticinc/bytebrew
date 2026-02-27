//go:build prompt

package prompt_regression

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
)

// JudgeVerdict represents the LLM judge's evaluation result
type JudgeVerdict struct {
	Pass      bool   `json:"pass"`
	Score     int    `json:"score"`
	Reasoning string `json:"reasoning"`
}

// JudgeRubric defines evaluation criteria for LLM judge
type JudgeRubric struct {
	Name      string
	Criteria  []string
	PassScore int
}

// Evaluate sends response + rubric to LLM judge, returns structured verdict
func (h *Harness) Evaluate(ctx context.Context, response string, rubric JudgeRubric) (*JudgeVerdict, error) {
	passScore := rubric.PassScore
	if passScore == 0 {
		passScore = 4
	}

	var sb strings.Builder
	sb.WriteString("You are an expert evaluator. Evaluate the following response against the given criteria.\n\n")
	sb.WriteString(fmt.Sprintf("## Evaluation: %s\n\n", rubric.Name))
	sb.WriteString("### Criteria:\n")
	for i, c := range rubric.Criteria {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, c))
	}
	sb.WriteString("\n### Response to evaluate:\n```\n")
	sb.WriteString(response)
	sb.WriteString("\n```\n\n")
	sb.WriteString("### Scoring (1-5):\n")
	sb.WriteString("1 = Fails most criteria, poor quality\n")
	sb.WriteString("2 = Fails several important criteria\n")
	sb.WriteString("3 = Meets some criteria but has significant gaps\n")
	sb.WriteString("4 = Meets most criteria with minor gaps\n")
	sb.WriteString("5 = Excellent — meets all criteria thoroughly\n\n")
	sb.WriteString(fmt.Sprintf("Pass threshold: score >= %d\n\n", passScore))
	sb.WriteString("Return ONLY a JSON object (no markdown fences, no extra text):\n")
	sb.WriteString(`{"pass": true/false, "score": N, "reasoning": "brief explanation"}`)

	msgs := []*schema.Message{
		{
			Role:    schema.System,
			Content: "You are an evaluation judge. Return ONLY valid JSON. No markdown fences, no extra text.",
		},
		{
			Role:    schema.User,
			Content: sb.String(),
		},
	}

	result, err := h.GeneratePlain(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("judge generate: %w", err)
	}

	content := strings.TrimSpace(result.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var verdict JudgeVerdict
	if err := json.Unmarshal([]byte(content), &verdict); err != nil {
		return nil, fmt.Errorf("parse judge verdict (content: %s): %w", content, err)
	}

	return &verdict, nil
}

// AssertJudgePass calls LLM judge and fails test if score < passScore
func AssertJudgePass(t *testing.T, h *Harness, response string, rubric JudgeRubric) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	verdict, err := h.Evaluate(ctx, response, rubric)
	if err != nil {
		t.Fatalf("LLM judge evaluation failed: %v", err)
	}

	passScore := rubric.PassScore
	if passScore == 0 {
		passScore = 4
	}

	t.Logf("Judge [%s]: score=%d, pass=%v, reasoning=%s", rubric.Name, verdict.Score, verdict.Pass, verdict.Reasoning)

	if verdict.Score < passScore {
		t.Errorf("Judge [%s] FAILED: score %d < threshold %d. Reasoning: %s",
			rubric.Name, verdict.Score, passScore, verdict.Reasoning)
	}
}

// extractToolCallArgs finds a tool call by name and returns its arguments as string
func extractToolCallArgs(msg *schema.Message, toolName string) string {
	if msg == nil {
		return ""
	}
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name == toolName {
			return tc.Function.Arguments
		}
	}
	return ""
}

// extractManageTasksCreateDescription extracts description from manage_tasks(action=create)
func extractManageTasksCreateDescription(msg *schema.Message) string {
	if msg == nil {
		return ""
	}
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name != "manage_tasks" {
			continue
		}
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			continue
		}
		action, ok := args["action"].(string)
		if !ok || action != "create" {
			continue
		}
		desc, ok := args["description"].(string)
		if !ok {
			continue
		}
		return desc
	}
	return ""
}

// extractManageSubtasksCreateDescription extracts description from manage_subtasks(action=create)
func extractManageSubtasksCreateDescription(msg *schema.Message) string {
	if msg == nil {
		return ""
	}
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name != "manage_subtasks" {
			continue
		}
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			continue
		}
		action, ok := args["action"].(string)
		if !ok || action != "create" {
			continue
		}
		desc, ok := args["description"].(string)
		if !ok {
			continue
		}
		return desc
	}
	return ""
}

// formatResponseForJudge converts message into readable string for LLM judge
func formatResponseForJudge(msg *schema.Message) string {
	if msg == nil {
		return ""
	}
	var sb strings.Builder
	if msg.Content != "" {
		sb.WriteString("Text response:\n")
		sb.WriteString(msg.Content)
		sb.WriteString("\n\n")
	}
	if len(msg.ToolCalls) > 0 {
		sb.WriteString("Tool calls:\n")
		for _, tc := range msg.ToolCalls {
			sb.WriteString(fmt.Sprintf("- %s(%s)\n", tc.Function.Name, tc.Function.Arguments))
		}
	}
	return sb.String()
}
