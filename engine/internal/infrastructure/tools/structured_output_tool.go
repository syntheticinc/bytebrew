package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// StructuredOutputTool displays structured data blocks (summary tables, action buttons) to the user.
type StructuredOutputTool struct {
	emitter   ToolEventEmitter
	sessionID string
}

// NewStructuredOutputTool creates a show_structured_output tool.
func NewStructuredOutputTool(emitter ToolEventEmitter, sessionID string) tool.InvokableTool {
	return &StructuredOutputTool{emitter: emitter, sessionID: sessionID}
}

type structuredOutputArgs struct {
	OutputType  string                  `json:"output_type"`
	Title       string                  `json:"title,omitempty"`
	Description string                  `json:"description,omitempty"`
	Rows        []domain.StructuredRow  `json:"rows,omitempty"`
	Actions     []domain.StructuredAction `json:"actions,omitempty"`
}

func (t *StructuredOutputTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "show_structured_output",
		Desc: `Display structured data to the user (summary tables, action buttons).

Use this to present organized information like project summaries, configuration overviews, or action choices.

Parameters:
- "output_type" (required): Type of output, e.g. "summary_table"
- "title" (optional): Title of the output block
- "description" (optional): Description text
- "rows" (optional): Array of {label, value} rows for tables
- "actions" (optional): Array of {label, type, value} action buttons (type: "primary" or "secondary")`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"output_type": {Type: schema.String, Desc: `Type of structured output, e.g. "summary_table"`, Required: true},
			"title":       {Type: schema.String, Desc: "Title of the output block"},
			"description": {Type: schema.String, Desc: "Description text"},
			"rows":        {Type: schema.String, Desc: `JSON array of rows: [{"label":"Name","value":"MyProject"}]`},
			"actions":     {Type: schema.String, Desc: `JSON array of actions: [{"label":"Deploy","type":"primary","value":"deploy"}]`},
		}),
	}, nil
}

func (t *StructuredOutputTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args structuredOutputArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return fmt.Sprintf("[ERROR] Invalid JSON: %v", err), nil
	}

	if args.OutputType == "" {
		return "[ERROR] output_type is required", nil
	}

	output := domain.StructuredOutput{
		OutputType:  args.OutputType,
		Title:       args.Title,
		Description: args.Description,
		Rows:        args.Rows,
		Actions:     args.Actions,
	}

	contentJSON, err := json.Marshal(output)
	if err != nil {
		return fmt.Sprintf("[ERROR] failed to serialize output: %v", err), nil
	}

	slog.InfoContext(ctx, "[structured_output] emitting event",
		"output_type", args.OutputType,
		"rows", len(args.Rows),
		"actions", len(args.Actions))

	if t.emitter != nil {
		_ = t.emitter.Send(&domain.AgentEvent{
			Type:      domain.EventTypeStructuredOutput,
			Timestamp: time.Now(),
			Content:   string(contentJSON),
		})
	}

	return "Structured output displayed to user.", nil
}
