package flow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// EdgeRouter routes output between agents according to the edge configuration.
type EdgeRouter struct{}

// NewEdgeRouter creates a new EdgeRouter.
func NewEdgeRouter() *EdgeRouter {
	return &EdgeRouter{}
}

// RouteOutput transforms the source agent's output according to the edge config mode.
// Returns the input string to pass to the target agent.
func (r *EdgeRouter) RouteOutput(output string, edgeConfig map[string]interface{}) (string, error) {
	mode := r.resolveMode(edgeConfig)

	switch mode {
	case domain.EdgeRouteFull:
		return output, nil
	case domain.EdgeRouteFieldMapping:
		return r.applyFieldMapping(output, edgeConfig)
	case domain.EdgeRouteCustomPrompt:
		return r.applyCustomPrompt(output, edgeConfig)
	default:
		return output, nil
	}
}

func (r *EdgeRouter) resolveMode(config map[string]interface{}) domain.EdgeRouteMode {
	if config == nil {
		return domain.EdgeRouteFull
	}
	modeVal, ok := config["mode"]
	if !ok {
		return domain.EdgeRouteFull
	}
	modeStr, ok := modeVal.(string)
	if !ok {
		return domain.EdgeRouteFull
	}
	switch domain.EdgeRouteMode(modeStr) {
	case domain.EdgeRouteFieldMapping:
		return domain.EdgeRouteFieldMapping
	case domain.EdgeRouteCustomPrompt:
		return domain.EdgeRouteCustomPrompt
	default:
		return domain.EdgeRouteFull
	}
}

// applyFieldMapping extracts specific fields from JSON output.
// Config: {"mode": "field_mapping", "mappings": [{"source": "output.task", "target": "input.backend_task"}, ...]}
func (r *EdgeRouter) applyFieldMapping(output string, config map[string]interface{}) (string, error) {
	mappingsRaw, ok := config["mappings"]
	if !ok {
		return output, nil
	}

	mappings, ok := mappingsRaw.([]interface{})
	if !ok {
		return output, nil
	}

	// Parse the output as JSON
	var outputData map[string]interface{}
	if err := json.Unmarshal([]byte(output), &outputData); err != nil {
		// If output is not JSON, wrap it as {"output": "..."}
		outputData = map[string]interface{}{"output": output}
	}

	result := make(map[string]interface{})
	for _, m := range mappings {
		mapping, ok := m.(map[string]interface{})
		if !ok {
			continue
		}
		sourceField, _ := mapping["source"].(string)
		targetField, _ := mapping["target"].(string)
		if sourceField == "" || targetField == "" {
			continue
		}

		// Extract value from source using dot notation
		val := extractField(outputData, sourceField)
		if val != nil {
			setField(result, targetField, val)
		}
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal field mapping result: %w", err)
	}
	return string(resultJSON), nil
}

// applyCustomPrompt applies a template with {{output}} and {{output.field}} variables.
func (r *EdgeRouter) applyCustomPrompt(output string, config map[string]interface{}) (string, error) {
	templateRaw, ok := config["template"]
	if !ok {
		return output, nil
	}
	template, ok := templateRaw.(string)
	if !ok {
		return output, nil
	}

	// Replace {{output}} with full output
	result := strings.ReplaceAll(template, "{{output}}", output)

	// Parse output as JSON for field access
	var outputData map[string]interface{}
	if err := json.Unmarshal([]byte(output), &outputData); err == nil {
		// Replace {{output.field}} patterns
		result = replaceFieldVars(result, outputData)
	}

	return result, nil
}

// extractField extracts a value from a nested map using dot notation.
// Supports paths like "output.task" or just "task".
func extractField(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")

	// Strip leading "output." prefix if the root key doesn't exist
	if len(parts) > 1 && parts[0] == "output" {
		if _, ok := data["output"]; !ok {
			parts = parts[1:]
		}
	}

	current := interface{}(data)
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current, ok = m[part]
		if !ok {
			return nil
		}
	}
	return current
}

// setField sets a value in a nested map using dot notation.
func setField(data map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")

	// Strip leading "input." prefix
	if len(parts) > 1 && parts[0] == "input" {
		parts = parts[1:]
	}

	current := data
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return
		}
		next, ok := current[part]
		if !ok {
			next = make(map[string]interface{})
			current[part] = next
		}
		nextMap, ok := next.(map[string]interface{})
		if !ok {
			return
		}
		current = nextMap
	}
}

// replaceFieldVars replaces {{output.field}} patterns in a template.
func replaceFieldVars(template string, data map[string]interface{}) string {
	result := template
	for key, val := range data {
		placeholder := fmt.Sprintf("{{output.%s}}", key)
		valStr := fmt.Sprintf("%v", val)
		result = strings.ReplaceAll(result, placeholder, valStr)
	}
	return result
}
