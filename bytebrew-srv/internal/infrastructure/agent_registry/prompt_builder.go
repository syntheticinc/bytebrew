package agent_registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/config"
)

// BuildSystemPrompt composes a full system prompt from an agent definition.
// It combines: base prompt (inline or from file) + workflow steps + confirm_before instructions.
func BuildSystemPrompt(def *config.AgentDefinition, configDir string) (string, error) {
	prompt := def.SystemPrompt
	if def.SystemPromptFile != "" {
		path := filepath.Join(configDir, def.SystemPromptFile)
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("load system_prompt_file %s: %w", def.SystemPromptFile, err)
		}
		prompt = string(data)
	}

	if strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("agent %q: no system prompt content", def.Name)
	}

	var sb strings.Builder
	sb.WriteString(prompt)

	if def.Flow != nil && len(def.Flow.Steps) > 0 {
		sb.WriteString("\n\n## Workflow\n")
		for i, step := range def.Flow.Steps {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
		}
	}

	if len(def.ConfirmBefore) > 0 {
		sb.WriteString("\n\n## Confirmation required\nAsk user before calling: ")
		sb.WriteString(strings.Join(def.ConfirmBefore, ", "))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}
