package llm

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// ---------- conversion: schema <-> OpenAI ----------

func schemaMessageToOpenAI(msg *schema.Message) openAIMessage {
	oai := openAIMessage{
		Role:    string(msg.Role),
		Content: msg.Content,
	}

	if msg.Name != "" {
		oai.Name = msg.Name
	}
	if msg.ToolCallID != "" {
		oai.ToolCallID = msg.ToolCallID
	}
	if len(msg.ToolCalls) > 0 {
		oai.ToolCalls = make([]openAIToolCall, 0, len(msg.ToolCalls))
		for _, tc := range msg.ToolCalls {
			oai.ToolCalls = append(oai.ToolCalls, openAIToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: openAIFunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
	}

	return oai
}

func oaiMessageToSchema(oai *openAIMessage) *schema.Message {
	msg := &schema.Message{
		Role:    schema.RoleType(oai.Role),
		Content: oai.Content,
	}

	if oai.Name != "" {
		msg.Name = oai.Name
	}
	if oai.ToolCallID != "" {
		msg.ToolCallID = oai.ToolCallID
	}
	if len(oai.ToolCalls) > 0 {
		msg.ToolCalls = make([]schema.ToolCall, 0, len(oai.ToolCalls))
		for _, tc := range oai.ToolCalls {
			name, args := sanitizeToolCall(tc.Function.Name, tc.Function.Arguments)
			msg.ToolCalls = append(msg.ToolCalls, schema.ToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: schema.FunctionCall{
					Name:      name,
					Arguments: args,
				},
			})
		}
	}

	return msg
}

// sanitizeToolCall fixes malformed tool calls where the model generates Python-style
// function calls: "manage_tasks(action=create, title=\"...\")" as the Name with empty Arguments.
// Returns cleaned name and arguments.
func sanitizeToolCall(name, arguments string) (string, string) {
	parenIdx := strings.Index(name, "(")
	if parenIdx < 0 {
		return name, arguments
	}

	// Arguments already present and valid — name just has garbage suffix, strip it
	if arguments != "" && arguments != "{}" {
		if json.Valid([]byte(arguments)) {
			return name[:parenIdx], arguments
		}
	}

	// Extract tool name and parse Python-style key=value args into JSON
	toolName := strings.TrimSpace(name[:parenIdx])
	argsStr := name[parenIdx+1:]

	// Remove trailing ")" and any garbage after it (e.g. "\n</function")
	if closeIdx := strings.LastIndex(argsStr, ")"); closeIdx >= 0 {
		argsStr = argsStr[:closeIdx]
	}

	argsMap := parsePythonStyleArgs(argsStr)
	if len(argsMap) == 0 {
		return toolName, arguments
	}

	jsonBytes, err := json.Marshal(argsMap)
	if err != nil {
		slog.WarnContext(context.Background(), "failed to marshal sanitized tool args", "tool", toolName, "error", err)
		return toolName, arguments
	}

	slog.InfoContext(context.Background(), "sanitized malformed tool call", "original_name", name[:min(len(name), 80)], "tool", toolName)
	return toolName, string(jsonBytes)
}

// parsePythonStyleArgs parses "action=create, title=\"Some title\", items=[\"a\",\"b\"]"
// into map[string]interface{}.
func parsePythonStyleArgs(s string) map[string]interface{} {
	result := map[string]interface{}{}
	parts := splitRespectingQuotesAndBrackets(s)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		eqIdx := strings.Index(part, "=")
		if eqIdx < 0 {
			continue
		}
		key := strings.TrimSpace(part[:eqIdx])
		val := strings.TrimSpace(part[eqIdx+1:])

		// Try JSON parse first (handles arrays, objects, numbers, booleans)
		var jsonVal interface{}
		if err := json.Unmarshal([]byte(val), &jsonVal); err == nil {
			result[key] = jsonVal
			continue
		}

		// Strip quotes for string values
		val = strings.Trim(val, "\"'")
		result[key] = val
	}

	return result
}

// splitRespectingQuotesAndBrackets splits on commas but respects quoted strings and brackets.
func splitRespectingQuotesAndBrackets(s string) []string {
	var parts []string
	var current strings.Builder
	depth := 0     // bracket/brace depth
	inQuote := false
	escaped := false

	for _, ch := range s {
		if escaped {
			current.WriteRune(ch)
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			current.WriteRune(ch)
			continue
		}
		if ch == '"' {
			inQuote = !inQuote
			current.WriteRune(ch)
			continue
		}
		if !inQuote {
			if ch == '[' || ch == '{' {
				depth++
			} else if ch == ']' || ch == '}' {
				depth--
			}
			if ch == ',' && depth == 0 {
				parts = append(parts, current.String())
				current.Reset()
				continue
			}
		}
		current.WriteRune(ch)
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func schemaToolsToOpenAI(tools []*schema.ToolInfo) []openAIToolDef {
	defs := make([]openAIToolDef, 0, len(tools))
	for _, t := range tools {
		def := openAIToolDef{
			Type: "function",
			Function: openAIToolFunc{
				Name:        t.Name,
				Description: t.Desc,
			},
		}

		if t.ParamsOneOf == nil {
			defs = append(defs, def)
			continue
		}

		jsonSchema, err := t.ParamsOneOf.ToJSONSchema()
		if err != nil {
			slog.WarnContext(context.Background(), "skip tool params schema", "tool", t.Name, "error", err)
			defs = append(defs, def)
			continue
		}
		if jsonSchema == nil {
			defs = append(defs, def)
			continue
		}

		raw, err := json.Marshal(jsonSchema)
		if err != nil {
			slog.WarnContext(context.Background(), "skip tool params marshal", "tool", t.Name, "error", err)
			defs = append(defs, def)
			continue
		}
		def.Function.Parameters = raw
		defs = append(defs, def)
	}
	return defs
}
