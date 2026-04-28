package llm

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/cloudwego/eino/schema"
)

// ---------- conversion: schema <-> Gemini ----------

// schemaMessagesToGemini converts a slice of Eino messages to Gemini contents
// and extracts the system instruction (if any).
func schemaMessagesToGemini(msgs []*schema.Message) (contents []geminiContent, systemInstruction *geminiContent) {
	for _, msg := range msgs {
		switch msg.Role {
		case schema.System:
			systemInstruction = &geminiContent{
				Parts: []geminiPart{{Text: msg.Content}},
			}

		case schema.User:
			contents = append(contents, geminiContent{
				Role:  "user",
				Parts: []geminiPart{{Text: msg.Content}},
			})

		case schema.Assistant:
			parts := schemaAssistantPartsToGemini(msg)
			contents = append(contents, geminiContent{
				Role:  "model",
				Parts: parts,
			})

		case schema.Tool:
			parts := schemaToolResultPartsToGemini(msg)
			contents = append(contents, geminiContent{
				Role:  "user",
				Parts: parts,
			})
		}
	}

	return contents, systemInstruction
}

// schemaAssistantPartsToGemini converts an assistant message to Gemini parts.
// If the message has tool calls, they are converted to functionCall parts.
func schemaAssistantPartsToGemini(msg *schema.Message) []geminiPart {
	var parts []geminiPart

	if msg.Content != "" {
		parts = append(parts, geminiPart{Text: msg.Content})
	}

	for _, tc := range msg.ToolCalls {
		args := parseToolCallArgs(tc.Function.Arguments)
		parts = append(parts, geminiPart{
			FunctionCall: &geminiFunctionCall{
				Name: tc.Function.Name,
				Args: args,
			},
		})
	}

	return parts
}

// schemaToolResultPartsToGemini converts a tool result message to Gemini functionResponse parts.
func schemaToolResultPartsToGemini(msg *schema.Message) []geminiPart {
	resp := map[string]interface{}{
		"content": msg.Content,
	}
	return []geminiPart{
		{
			FunctionResponse: &geminiFunctionResponse{
				Name:     msg.Name,
				Response: resp,
			},
		},
	}
}

// parseToolCallArgs parses JSON arguments string into a map.
func parseToolCallArgs(args string) map[string]interface{} {
	if args == "" {
		return nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(args), &result); err != nil {
		slog.WarnContext(context.Background(), "gemini: failed to parse tool call args", "error", err)
		return map[string]interface{}{"raw": args}
	}
	return result
}

// geminiResponseToSchema converts a Gemini response to an Eino schema.Message.
func geminiResponseToSchema(resp *geminiResponse) *schema.Message {
	if len(resp.Candidates) == 0 {
		return &schema.Message{Role: schema.Assistant}
	}

	candidate := resp.Candidates[0]
	msg := &schema.Message{Role: schema.Assistant}

	var textParts []string
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			textParts = append(textParts, part.Text)
		}
		if part.FunctionCall != nil {
			argsJSON, err := json.Marshal(part.FunctionCall.Args)
			if err != nil {
				slog.WarnContext(context.Background(), "gemini: failed to marshal function call args", "error", err)
				argsJSON = []byte("{}")
			}
			msg.ToolCalls = append(msg.ToolCalls, schema.ToolCall{
				ID:   part.FunctionCall.Name, // Gemini doesn't use separate IDs
				Type: "function",
				Function: schema.FunctionCall{
					Name:      part.FunctionCall.Name,
					Arguments: string(argsJSON),
				},
			})
		}
	}

	if len(textParts) > 0 {
		for i, t := range textParts {
			if i == 0 {
				msg.Content = t
			} else {
				msg.Content += "\n" + t
			}
		}
	}

	return msg
}

// schemaToolsToGemini converts Eino tool definitions to Gemini function declarations.
func schemaToolsToGemini(tools []*schema.ToolInfo) []geminiTool {
	if len(tools) == 0 {
		return nil
	}

	decls := make([]geminiFunctionDeclaration, 0, len(tools))
	for _, t := range tools {
		decl := geminiFunctionDeclaration{
			Name:        t.Name,
			Description: t.Desc,
		}

		if t.ParamsOneOf != nil {
			jsonSchema, err := t.ParamsOneOf.ToJSONSchema()
			if err != nil {
				slog.WarnContext(context.Background(), "gemini: skip tool params schema", "tool", t.Name, "error", err)
				decls = append(decls, decl)
				continue
			}
			if jsonSchema != nil {
				raw, err := json.Marshal(jsonSchema)
				if err != nil {
					slog.WarnContext(context.Background(), "gemini: skip tool params marshal", "tool", t.Name, "error", err)
					decls = append(decls, decl)
					continue
				}
				decl.Parameters = raw
			}
		}

		decls = append(decls, decl)
	}

	return []geminiTool{{FunctionDeclarations: decls}}
}
