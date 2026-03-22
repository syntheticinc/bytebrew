package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGenericSpawner implements GenericAgentSpawner for testing.
type mockGenericSpawner struct {
	spawnResult string
	spawnErr    error
	spawnParams SpawnParams

	waitResult WaitResult
	waitErr    error

	stopErr     error
	stoppedID   string
	hasBlocking bool
}

func (m *mockGenericSpawner) SpawnAgent(ctx context.Context, params SpawnParams) (string, error) {
	m.spawnParams = params
	return m.spawnResult, m.spawnErr
}

func (m *mockGenericSpawner) WaitForAllSessionAgents(ctx context.Context, sessionID string) (WaitResult, error) {
	return m.waitResult, m.waitErr
}

func (m *mockGenericSpawner) HasBlockingWait(sessionID string) bool {
	return m.hasBlocking
}

func (m *mockGenericSpawner) NotifyUserMessage(sessionID, message string) {}

func (m *mockGenericSpawner) StopAgent(agentID string) error {
	m.stoppedID = agentID
	return m.stopErr
}

// mockGenericInspector implements GenericAgentInspector for testing.
type mockGenericInspector struct {
	statusInfo *AgentInfo
	statusOK   bool
	allInfos   []AgentInfo
}

func (m *mockGenericInspector) GetStatusInfo(agentID string) (*AgentInfo, bool) {
	return m.statusInfo, m.statusOK
}

func (m *mockGenericInspector) GetAllAgentInfos() []AgentInfo {
	return m.allInfos
}

func TestSpawnTool_Info(t *testing.T) {
	spawner := &mockGenericSpawner{}
	inspector := &mockGenericInspector{}
	st := NewSpawnTool("coder", "sess-1", spawner, inspector)

	info, err := st.Info(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "spawn_coder", info.Name)
	assert.Contains(t, info.Desc, "coder")
}

func TestSpawnTool_Spawn(t *testing.T) {
	tests := []struct {
		name        string
		args        string
		spawnResult string
		spawnErr    error
		wantResult  string
		wantErr     bool
		wantErrMsg  string
	}{
		{
			name:        "successful spawn",
			args:        `{"action":"spawn","description":"implement feature X"}`,
			spawnResult: "agent-123",
			wantResult:  "Agent spawned with ID: agent-123",
		},
		{
			name:       "spawn without description",
			args:       `{"action":"spawn"}`,
			wantErr:    true,
			wantErrMsg: "description required",
		},
		{
			name:       "spawn error from spawner",
			args:       `{"action":"spawn","description":"do stuff"}`,
			spawnErr:   fmt.Errorf("max agents reached"),
			wantErr:    true,
			wantErrMsg: "spawn agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spawner := &mockGenericSpawner{
				spawnResult: tt.spawnResult,
				spawnErr:    tt.spawnErr,
			}
			inspector := &mockGenericInspector{}
			st := NewSpawnTool("coder", "sess-1", spawner, inspector)

			result, err := st.InvokableRun(context.Background(), tt.args)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

func TestSpawnTool_SpawnPassesCorrectParams(t *testing.T) {
	spawner := &mockGenericSpawner{spawnResult: "agent-1"}
	inspector := &mockGenericInspector{}
	st := NewSpawnTool("reviewer", "sess-42", spawner, inspector)

	_, err := st.InvokableRun(context.Background(), `{"action":"spawn","description":"review PR"}`)
	require.NoError(t, err)

	assert.Equal(t, "sess-42", spawner.spawnParams.SessionID)
	assert.Equal(t, "reviewer", spawner.spawnParams.AgentName)
	assert.Equal(t, "review PR", spawner.spawnParams.Description)
	assert.True(t, spawner.spawnParams.Blocking)
}

func TestSpawnTool_Wait(t *testing.T) {
	summaries := []AgentSummary{
		{AgentID: "a1", AgentName: "coder", Summary: "done", Status: "completed"},
	}
	spawner := &mockGenericSpawner{
		waitResult: WaitResult{Summaries: summaries},
	}
	inspector := &mockGenericInspector{}
	st := NewSpawnTool("coder", "sess-1", spawner, inspector)

	result, err := st.InvokableRun(context.Background(), `{"action":"wait"}`)
	require.NoError(t, err)

	var got []AgentSummary
	require.NoError(t, json.Unmarshal([]byte(result), &got))
	assert.Len(t, got, 1)
	assert.Equal(t, "a1", got[0].AgentID)
}

func TestSpawnTool_WaitError(t *testing.T) {
	spawner := &mockGenericSpawner{
		waitErr: fmt.Errorf("timeout"),
	}
	inspector := &mockGenericInspector{}
	st := NewSpawnTool("coder", "sess-1", spawner, inspector)

	_, err := st.InvokableRun(context.Background(), `{"action":"wait"}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wait for agents")
}

func TestSpawnTool_Status(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		inspector := &mockGenericInspector{
			statusInfo: &AgentInfo{ID: "a1", Status: "running"},
			statusOK:   true,
		}
		st := NewSpawnTool("coder", "sess-1", &mockGenericSpawner{}, inspector)

		result, err := st.InvokableRun(context.Background(), `{"action":"status","agent_id":"a1"}`)
		require.NoError(t, err)
		assert.Contains(t, result, "running")
	})

	t.Run("not found", func(t *testing.T) {
		inspector := &mockGenericInspector{statusOK: false}
		st := NewSpawnTool("coder", "sess-1", &mockGenericSpawner{}, inspector)

		result, err := st.InvokableRun(context.Background(), `{"action":"status","agent_id":"a1"}`)
		require.NoError(t, err)
		assert.Contains(t, result, "not found")
	})

	t.Run("missing agent_id", func(t *testing.T) {
		st := NewSpawnTool("coder", "sess-1", &mockGenericSpawner{}, &mockGenericInspector{})

		_, err := st.InvokableRun(context.Background(), `{"action":"status"}`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "agent_id required")
	})
}

func TestSpawnTool_List(t *testing.T) {
	inspector := &mockGenericInspector{
		allInfos: []AgentInfo{
			{ID: "a1", Status: "running"},
			{ID: "a2", Status: "completed"},
		},
	}
	st := NewSpawnTool("coder", "sess-1", &mockGenericSpawner{}, inspector)

	result, err := st.InvokableRun(context.Background(), `{"action":"list"}`)
	require.NoError(t, err)

	var got []AgentInfo
	require.NoError(t, json.Unmarshal([]byte(result), &got))
	assert.Len(t, got, 2)
}

func TestSpawnTool_Stop(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		spawner := &mockGenericSpawner{}
		st := NewSpawnTool("coder", "sess-1", spawner, &mockGenericInspector{})

		result, err := st.InvokableRun(context.Background(), `{"action":"stop","agent_id":"a1"}`)
		require.NoError(t, err)
		assert.Contains(t, result, "stopped")
		assert.Equal(t, "a1", spawner.stoppedID)
	})

	t.Run("missing agent_id", func(t *testing.T) {
		st := NewSpawnTool("coder", "sess-1", &mockGenericSpawner{}, &mockGenericInspector{})

		_, err := st.InvokableRun(context.Background(), `{"action":"stop"}`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "agent_id required")
	})

	t.Run("stop error", func(t *testing.T) {
		spawner := &mockGenericSpawner{stopErr: fmt.Errorf("not found")}
		st := NewSpawnTool("coder", "sess-1", spawner, &mockGenericInspector{})

		_, err := st.InvokableRun(context.Background(), `{"action":"stop","agent_id":"a1"}`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "stop agent")
	})
}

func TestSpawnTool_UnknownAction(t *testing.T) {
	st := NewSpawnTool("coder", "sess-1", &mockGenericSpawner{}, &mockGenericInspector{})

	_, err := st.InvokableRun(context.Background(), `{"action":"fly"}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown action "fly"`)
}

func TestSpawnTool_InvalidJSON(t *testing.T) {
	st := NewSpawnTool("coder", "sess-1", &mockGenericSpawner{}, &mockGenericInspector{})

	_, err := st.InvokableRun(context.Background(), `{invalid`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse args")
}
