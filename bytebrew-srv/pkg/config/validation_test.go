package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAgentDefinitions(t *testing.T) {
	validAgent := func(name string) AgentDefinition {
		return AgentDefinition{
			Name:         name,
			SystemPrompt: "You are a helpful assistant.",
		}
	}

	tests := []struct {
		name    string
		defs    []AgentDefinition
		wantErr string
	}{
		{
			name:    "empty slice",
			defs:    []AgentDefinition{},
			wantErr: "at least one agent definition is required",
		},
		{
			name: "single valid agent",
			defs: []AgentDefinition{validAgent("support")},
		},
		{
			name: "multiple valid agents",
			defs: []AgentDefinition{
				validAgent("support"),
				validAgent("code-review"),
				validAgent("devops-agent"),
			},
		},
		{
			name: "empty name",
			defs: []AgentDefinition{{
				Name:         "",
				SystemPrompt: "prompt",
			}},
			wantErr: "has empty name",
		},
		{
			name: "name starts with digit",
			defs: []AgentDefinition{{
				Name:         "1agent",
				SystemPrompt: "prompt",
			}},
			wantErr: "is invalid",
		},
		{
			name: "name with uppercase",
			defs: []AgentDefinition{{
				Name:         "MyAgent",
				SystemPrompt: "prompt",
			}},
			wantErr: "is invalid",
		},
		{
			name: "name with underscore",
			defs: []AgentDefinition{{
				Name:         "my_agent",
				SystemPrompt: "prompt",
			}},
			wantErr: "is invalid",
		},
		{
			name: "name with spaces",
			defs: []AgentDefinition{{
				Name:         "my agent",
				SystemPrompt: "prompt",
			}},
			wantErr: "is invalid",
		},
		{
			name: "valid name with hyphens and digits",
			defs: []AgentDefinition{validAgent("code-review-v2")},
		},
		{
			name: "duplicate names",
			defs: []AgentDefinition{
				validAgent("support"),
				validAgent("support"),
			},
			wantErr: "duplicate agent name",
		},
		{
			name: "can_spawn references nonexistent agent",
			defs: []AgentDefinition{{
				Name:         "supervisor",
				SystemPrompt: "prompt",
				CanSpawn:     []string{"nonexistent"},
			}},
			wantErr: "references nonexistent agent",
		},
		{
			name: "valid can_spawn reference",
			defs: []AgentDefinition{
				{
					Name:         "supervisor",
					SystemPrompt: "prompt",
					CanSpawn:     []string{"worker"},
				},
				validAgent("worker"),
			},
		},
		{
			name: "missing both system_prompt and system_prompt_file",
			defs: []AgentDefinition{{
				Name: "no-prompt",
			}},
			wantErr: "must have either system_prompt or system_prompt_file",
		},
		{
			name: "has system_prompt_file instead of system_prompt",
			defs: []AgentDefinition{{
				Name:             "file-prompt",
				SystemPromptFile: "prompts/agent.md",
			}},
		},
		{
			name: "whitespace-only system_prompt is treated as empty",
			defs: []AgentDefinition{{
				Name:         "blank-prompt",
				SystemPrompt: "   \n  ",
			}},
			wantErr: "must have either system_prompt or system_prompt_file",
		},
		{
			name: "can_spawn self-reference is allowed if agent exists",
			defs: []AgentDefinition{{
				Name:         "recursive",
				SystemPrompt: "prompt",
				CanSpawn:     []string{"recursive"},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgentDefinitions(tt.defs)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestValidateAgentName_Pattern(t *testing.T) {
	valid := []string{"a", "agent", "my-agent", "agent1", "a1b2c3", "support-v2"}
	invalid := []string{"", "1abc", "Agent", "my_agent", "agent!", "my agent", "-start"}

	for _, name := range valid {
		t.Run("valid_"+name, func(t *testing.T) {
			err := validateAgentName(name, 0)
			assert.NoError(t, err)
		})
	}

	for _, name := range invalid {
		t.Run("invalid_"+name, func(t *testing.T) {
			err := validateAgentName(name, 0)
			assert.Error(t, err)
		})
	}
}
