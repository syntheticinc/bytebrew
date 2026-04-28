package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFlowsConfig(t *testing.T) {
	// Create temporary flows.yaml
	tmpDir := t.TempDir()
	flowsPath := filepath.Join(tmpDir, "flows.yaml")

	flowsYAML := `flows:
  supervisor:
    name: "Supervisor Agent"
    system_prompt_ref: "supervisor_prompt"
    tools:
      - read_file
      - search_code
      - spawn_agent
    max_steps: 50
    max_context_size: 16000
    lifecycle:
      suspend_on:
        - final_answer
      report_to: user
    spawn_policy:
      allowed_flows:
        - coder
        - reviewer
  coder:
    name: "Code Agent"
    system_prompt_ref: "code_agent_prompt"
    tools:
      - read_file
      - write_file
    max_steps: 30
    max_context_size: 16000
    lifecycle:
      suspend_on:
        - final_answer
      report_to: parent_agent
    spawn_policy:
      allowed_flows: []
`

	err := os.WriteFile(flowsPath, []byte(flowsYAML), 0644)
	require.NoError(t, err)

	cfg, err := LoadFlowsConfig(flowsPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Len(t, cfg.Flows, 2)
	assert.Contains(t, cfg.Flows, "supervisor")
	assert.Contains(t, cfg.Flows, "coder")
}

func TestLoadFlowsConfig_AllFlows(t *testing.T) {
	// Create temporary flows.yaml with all 4 flows
	tmpDir := t.TempDir()
	flowsPath := filepath.Join(tmpDir, "flows.yaml")

	flowsYAML := `flows:
  supervisor:
    name: "Supervisor Agent"
    system_prompt_ref: "supervisor_prompt"
    tools:
      - read_file
    max_steps: 50
    max_context_size: 16000
    lifecycle:
      suspend_on: [final_answer]
      report_to: user
    spawn_policy:
      allowed_flows: [coder, reviewer, researcher]
  coder:
    name: "Code Agent"
    system_prompt_ref: "code_agent_prompt"
    tools:
      - write_file
    max_steps: 30
    max_context_size: 16000
    lifecycle:
      suspend_on: [final_answer]
      report_to: parent_agent
    spawn_policy:
      allowed_flows: []
  reviewer:
    name: "Code Reviewer"
    system_prompt_ref: "reviewer_prompt"
    tools:
      - read_file
    max_steps: 20
    max_context_size: 16000
    lifecycle:
      suspend_on: [final_answer]
      report_to: parent_agent
    spawn_policy:
      allowed_flows: []
  researcher:
    name: "Researcher"
    system_prompt_ref: "researcher_prompt"
    tools:
      - web_search
    max_steps: 25
    max_context_size: 16000
    lifecycle:
      suspend_on: [final_answer]
      report_to: parent_agent
    spawn_policy:
      allowed_flows: []
`

	err := os.WriteFile(flowsPath, []byte(flowsYAML), 0644)
	require.NoError(t, err)

	cfg, err := LoadFlowsConfig(flowsPath)
	require.NoError(t, err)

	assert.Len(t, cfg.Flows, 4)
	assert.Contains(t, cfg.Flows, "supervisor")
	assert.Contains(t, cfg.Flows, "coder")
	assert.Contains(t, cfg.Flows, "reviewer")
	assert.Contains(t, cfg.Flows, "researcher")
}

func TestLoadFlowsConfig_FileNotFound(t *testing.T) {
	cfg, err := LoadFlowsConfig("/nonexistent/flows.yaml")
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "read flows config")
}

func TestLoadFlowsConfig_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	flowsPath := filepath.Join(tmpDir, "flows.yaml")

	err := os.WriteFile(flowsPath, []byte("flows: {}\n"), 0644)
	require.NoError(t, err)

	cfg, err := LoadFlowsConfig(flowsPath)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "no flows defined")
}

func TestToDomainFlow_Supervisor(t *testing.T) {
	cfg := &FlowsConfig{
		Flows: map[string]FlowDefinition{
			"supervisor": {
				Name:            "Supervisor Agent",
				SystemPromptRef: "supervisor_prompt",
				Tools:           []string{"read_file", "spawn_agent"},
				MaxSteps:        50,
				MaxContextSize:  16000,
				Lifecycle: LifecycleConfig{
					SuspendOn: []string{"final_answer"},
					ReportTo:  "user",
				},
				SpawnPolicy: SpawnConfig{
					AllowedFlows: []string{"coder", "reviewer"},
				},
			},
		},
	}

	prompts := &PromptsConfig{
		SystemPrompt:     "Default system prompt",
		SupervisorPrompt: "You are a supervisor agent",
	}

	flow, err := cfg.ToDomainFlow("supervisor", prompts)
	require.NoError(t, err)
	require.NotNil(t, flow)

	assert.Equal(t, "supervisor", flow.Type)
	assert.Equal(t, "Supervisor Agent", flow.Name)
	assert.Equal(t, "You are a supervisor agent", flow.SystemPrompt)
	assert.Equal(t, []string{"read_file", "spawn_agent"}, flow.ToolNames)
	assert.Equal(t, 50, flow.MaxSteps)
	assert.Equal(t, 16000, flow.MaxContextSize)
	assert.Equal(t, []string{"final_answer"}, flow.Lifecycle.SuspendOn)
	assert.Equal(t, "user", flow.Lifecycle.ReportTo)
	assert.Equal(t, []string{"coder", "reviewer"}, flow.Spawn.AllowedFlows)
}

func TestToDomainFlow_UnknownType(t *testing.T) {
	cfg := &FlowsConfig{
		Flows: map[string]FlowDefinition{
			"supervisor": {
				Name:            "Supervisor Agent",
				SystemPromptRef: "supervisor_prompt",
				Tools:           []string{"read_file"},
				MaxSteps:        50,
				MaxContextSize:  16000,
			},
		},
	}

	prompts := &PromptsConfig{
		SupervisorPrompt: "You are a supervisor agent",
	}

	flow, err := cfg.ToDomainFlow("unknown_flow", prompts)
	assert.Error(t, err)
	assert.Nil(t, flow)
	assert.Contains(t, err.Error(), "unknown flow type")
}

func TestToDomainFlow_MissingPrompt(t *testing.T) {
	cfg := &FlowsConfig{
		Flows: map[string]FlowDefinition{
			"supervisor": {
				Name:            "Supervisor Agent",
				SystemPromptRef: "supervisor_prompt",
				Tools:           []string{"read_file"},
				MaxSteps:        50,
				MaxContextSize:  16000,
			},
		},
	}

	flow, err := cfg.ToDomainFlow("supervisor", nil)
	assert.Error(t, err)
	assert.Nil(t, flow)
	assert.Contains(t, err.Error(), "prompts config is nil")
}

func TestToDomainFlow_Reviewer(t *testing.T) {
	cfg := &FlowsConfig{
		Flows: map[string]FlowDefinition{
			"reviewer": {
				Name:            "Code Reviewer",
				SystemPromptRef: "reviewer_prompt",
				Tools:           []string{"read_file", "execute_command"},
				MaxSteps:        80,
				MaxContextSize:  16000,
				Lifecycle: LifecycleConfig{
					SuspendOn: []string{"final_answer"},
					ReportTo:  "parent_agent",
				},
			},
		},
	}

	prompts := &PromptsConfig{
		SystemPrompt:   "Default",
		ReviewerPrompt: "You are a code reviewer agent",
	}

	flow, err := cfg.ToDomainFlow("reviewer", prompts)
	require.NoError(t, err)
	require.NotNil(t, flow)

	assert.Equal(t, "reviewer", flow.Type)
	assert.Equal(t, "Code Reviewer", flow.Name)
	assert.Equal(t, "You are a code reviewer agent", flow.SystemPrompt)
	assert.Equal(t, []string{"read_file", "execute_command"}, flow.ToolNames)
	assert.Equal(t, 80, flow.MaxSteps)
}

func TestResolvePromptRef_AllCases(t *testing.T) {
	tests := []struct {
		name      string
		ref       string
		prompts   *PromptsConfig
		want      string
		wantErr   bool
		errSubstr string
	}{
		{
			name: "system_prompt",
			ref:  "system_prompt",
			prompts: &PromptsConfig{
				SystemPrompt: "Default system prompt",
			},
			want:    "Default system prompt",
			wantErr: false,
		},
		{
			name: "supervisor_prompt",
			ref:  "supervisor_prompt",
			prompts: &PromptsConfig{
				SystemPrompt:     "Default system prompt",
				SupervisorPrompt: "You are a supervisor agent",
			},
			want:    "You are a supervisor agent",
			wantErr: false,
		},
		{
			name: "code_agent_prompt",
			ref:  "code_agent_prompt",
			prompts: &PromptsConfig{
				SystemPrompt:    "Default system prompt",
				CodeAgentPrompt: "You are a code agent",
			},
			want:    "You are a code agent",
			wantErr: false,
		},
		{
			name: "code_agent_prompt_fallback",
			ref:  "code_agent_prompt",
			prompts: &PromptsConfig{
				SystemPrompt: "Default system prompt",
			},
			want:    "Default system prompt",
			wantErr: false,
		},
		{
			name: "reviewer_prompt",
			ref:  "reviewer_prompt",
			prompts: &PromptsConfig{
				SystemPrompt:   "Default system prompt",
				ReviewerPrompt: "You are a code reviewer",
			},
			want:    "You are a code reviewer",
			wantErr: false,
		},
		{
			name: "researcher_prompt",
			ref:  "researcher_prompt",
			prompts: &PromptsConfig{
				SystemPrompt:     "Default system prompt",
				ResearcherPrompt: "You are a researcher",
			},
			want:    "You are a researcher",
			wantErr: false,
		},
		{
			name: "reviewer_prompt_fallback",
			ref:  "reviewer_prompt",
			prompts: &PromptsConfig{
				SystemPrompt: "Default system prompt",
			},
			want:    "Default system prompt",
			wantErr: false,
		},
		{
			name: "researcher_prompt_fallback",
			ref:  "researcher_prompt",
			prompts: &PromptsConfig{
				SystemPrompt: "Default system prompt",
			},
			want:    "Default system prompt",
			wantErr: false,
		},
		{
			name:      "unknown_ref",
			ref:       "unknown_prompt",
			prompts:   &PromptsConfig{SystemPrompt: "Default system prompt"},
			wantErr:   true,
			errSubstr: "unknown prompt reference",
		},
		{
			name:      "nil_prompts",
			ref:       "system_prompt",
			prompts:   nil,
			wantErr:   true,
			errSubstr: "prompts config is nil",
		},
		{
			name: "empty_system_prompt",
			ref:  "system_prompt",
			prompts: &PromptsConfig{
				SystemPrompt: "",
			},
			wantErr:   true,
			errSubstr: "system_prompt is empty",
		},
		{
			name: "empty_supervisor_prompt",
			ref:  "supervisor_prompt",
			prompts: &PromptsConfig{
				SystemPrompt:     "Default system prompt",
				SupervisorPrompt: "",
			},
			wantErr:   true,
			errSubstr: "supervisor_prompt is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolvePromptRef(tt.ref, tt.prompts)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
