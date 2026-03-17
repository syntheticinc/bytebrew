package config

import (
	"fmt"
	"regexp"
	"strings"
)

// agentNamePattern enforces lowercase alphanumeric names with hyphens, starting with a letter.
var agentNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ValidateAgentDefinitions checks a slice of AgentDefinition for correctness.
// It verifies naming conventions, uniqueness, cross-references, and required fields.
func ValidateAgentDefinitions(defs []AgentDefinition) error {
	if len(defs) == 0 {
		return fmt.Errorf("at least one agent definition is required")
	}

	names := make(map[string]struct{}, len(defs))

	for i, def := range defs {
		if err := validateAgentName(def.Name, i); err != nil {
			return err
		}

		if _, exists := names[def.Name]; exists {
			return fmt.Errorf("duplicate agent name: %q", def.Name)
		}
		names[def.Name] = struct{}{}
	}

	// Second pass: validate cross-references (can_spawn must point to existing agents)
	for _, def := range defs {
		for _, spawnRef := range def.CanSpawn {
			if _, exists := names[spawnRef]; !exists {
				return fmt.Errorf("agent %q references nonexistent agent %q in can_spawn", def.Name, spawnRef)
			}
		}
	}

	// Third pass: validate required fields
	for _, def := range defs {
		if err := validateAgentPrompt(def); err != nil {
			return err
		}
	}

	return nil
}

// validateAgentName checks that the agent name follows the naming convention.
func validateAgentName(name string, index int) error {
	if name == "" {
		return fmt.Errorf("agent at index %d has empty name", index)
	}
	if !agentNamePattern.MatchString(name) {
		return fmt.Errorf("agent name %q is invalid: must match %s (lowercase, start with letter, alphanumeric and hyphens only)",
			name, agentNamePattern.String())
	}
	return nil
}

// validateAgentPrompt checks that an agent has at least one prompt source.
func validateAgentPrompt(def AgentDefinition) error {
	hasPrompt := strings.TrimSpace(def.SystemPrompt) != ""
	hasFile := strings.TrimSpace(def.SystemPromptFile) != ""

	if !hasPrompt && !hasFile {
		return fmt.Errorf("agent %q must have either system_prompt or system_prompt_file", def.Name)
	}
	return nil
}
