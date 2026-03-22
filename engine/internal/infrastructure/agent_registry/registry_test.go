package agent_registry

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/config_repo"
)

type mockAgentReader struct {
	records []config_repo.AgentRecord
	err     error
}

func (m *mockAgentReader) List(_ context.Context) ([]config_repo.AgentRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.records, nil
}

func (m *mockAgentReader) GetByName(_ context.Context, name string) (*config_repo.AgentRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, r := range m.records {
		if r.Name == name {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockAgentReader) Count(_ context.Context) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return int64(len(m.records)), nil
}

func testRecords() []config_repo.AgentRecord {
	return []config_repo.AgentRecord{
		{
			Name:           "supervisor",
			ModelName:      "gpt-4",
			SystemPrompt:   "You are a supervisor.",
			Lifecycle:      "persistent",
			ToolExecution:  "sequential",
			MaxSteps:       50,
			MaxContextSize: 16000,
			BuiltinTools:   []string{"manage_stories", "spawn_agent"},
			CanSpawn:       []string{"code_agent"},
		},
		{
			Name:           "code_agent",
			ModelName:      "gpt-4",
			SystemPrompt:   "You are a code agent.",
			Lifecycle:      "spawn",
			ToolExecution:  "parallel",
			MaxSteps:       30,
			MaxContextSize: 8000,
			BuiltinTools:   []string{"read_file", "write_file"},
			CustomTools: []config_repo.CustomToolRecord{
				{Name: "lint", Config: `{"cmd":"golangci-lint"}`},
			},
		},
	}
}

func TestLoad(t *testing.T) {
	repo := &mockAgentReader{records: testRecords()}
	reg := New(repo)

	err := reg.Load(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, reg.Count())
}

func TestLoad_RepoError(t *testing.T) {
	repo := &mockAgentReader{err: fmt.Errorf("db connection refused")}
	reg := New(repo)

	err := reg.Load(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load agents")
}

func TestGet_Existing(t *testing.T) {
	repo := &mockAgentReader{records: testRecords()}
	reg := New(repo)
	require.NoError(t, reg.Load(context.Background()))

	agent, err := reg.Get("supervisor")
	require.NoError(t, err)
	assert.Equal(t, "supervisor", agent.Flow.Name)
	assert.Equal(t, "You are a supervisor.", agent.Flow.SystemPrompt)
	assert.Equal(t, "sequential", agent.Flow.ToolExecution)
	assert.Equal(t, 50, agent.Flow.MaxSteps)
	assert.Equal(t, []string{"manage_stories", "spawn_agent"}, agent.Flow.ToolNames)
}

func TestGet_Nonexistent(t *testing.T) {
	repo := &mockAgentReader{records: testRecords()}
	reg := New(repo)
	require.NoError(t, reg.Load(context.Background()))

	_, err := reg.Get("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestList_Alphabetical(t *testing.T) {
	repo := &mockAgentReader{records: testRecords()}
	reg := New(repo)
	require.NoError(t, reg.Load(context.Background()))

	names := reg.List()
	assert.Equal(t, []string{"code_agent", "supervisor"}, names)
}

func TestGetDefault_ReturnsFirstAlphabetically(t *testing.T) {
	repo := &mockAgentReader{records: testRecords()}
	reg := New(repo)
	require.NoError(t, reg.Load(context.Background()))

	agent, err := reg.GetDefault()
	require.NoError(t, err)
	assert.Equal(t, "code_agent", agent.Flow.Name)
}

func TestGetDefault_EmptyRegistry(t *testing.T) {
	repo := &mockAgentReader{records: nil}
	reg := New(repo)
	require.NoError(t, reg.Load(context.Background()))

	_, err := reg.GetDefault()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no agents configured")
}

func TestGetAll_ReturnsCopy(t *testing.T) {
	repo := &mockAgentReader{records: testRecords()}
	reg := New(repo)
	require.NoError(t, reg.Load(context.Background()))

	all := reg.GetAll()
	assert.Len(t, all, 2)

	// Mutating the copy should not affect the registry
	delete(all, "supervisor")
	assert.Equal(t, 2, reg.Count())
}

func TestReload_UpdatesCache(t *testing.T) {
	repo := &mockAgentReader{records: testRecords()}
	reg := New(repo)
	require.NoError(t, reg.Load(context.Background()))
	assert.Equal(t, 2, reg.Count())

	// Add a third agent
	repo.records = append(repo.records, config_repo.AgentRecord{
		Name:           "analyst",
		SystemPrompt:   "You are an analyst.",
		Lifecycle:      "persistent",
		ToolExecution:  "sequential",
		MaxSteps:       20,
		MaxContextSize: 4000,
	})

	require.NoError(t, reg.Reload(context.Background()))
	assert.Equal(t, 3, reg.Count())

	_, err := reg.Get("analyst")
	require.NoError(t, err)
}

func TestToFlow_LifecyclePersistent(t *testing.T) {
	rec := config_repo.AgentRecord{
		Name:           "test",
		Lifecycle:      "persistent",
		ToolExecution:  "sequential",
		MaxSteps:       10,
		MaxContextSize: 1000,
		SystemPrompt:   "prompt",
	}

	flow := toFlow(rec)
	assert.Equal(t, []string{"final_answer", "ask_user"}, flow.Lifecycle.SuspendOn)
	assert.Equal(t, "user", flow.Lifecycle.ReportTo)
}

func TestToFlow_LifecycleSpawn(t *testing.T) {
	rec := config_repo.AgentRecord{
		Name:           "test",
		Lifecycle:      "spawn",
		ToolExecution:  "parallel",
		MaxSteps:       10,
		MaxContextSize: 1000,
		SystemPrompt:   "prompt",
	}

	flow := toFlow(rec)
	assert.Empty(t, flow.Lifecycle.SuspendOn)
	assert.Equal(t, "parent_agent", flow.Lifecycle.ReportTo)
}

func TestToFlow_ToolsMerged(t *testing.T) {
	rec := config_repo.AgentRecord{
		Name:           "test",
		SystemPrompt:   "prompt",
		Lifecycle:      "persistent",
		ToolExecution:  "sequential",
		MaxSteps:       10,
		MaxContextSize: 1000,
		BuiltinTools:   []string{"read_file"},
		CustomTools: []config_repo.CustomToolRecord{
			{Name: "lint", Config: "{}"},
		},
	}

	flow := toFlow(rec)
	assert.Equal(t, []string{"read_file", "lint"}, flow.ToolNames)
}

func TestToFlow_SpawnTargets(t *testing.T) {
	rec := config_repo.AgentRecord{
		Name:           "supervisor",
		SystemPrompt:   "prompt",
		Lifecycle:      "persistent",
		ToolExecution:  "sequential",
		MaxSteps:       10,
		MaxContextSize: 1000,
		CanSpawn:       []string{"code_agent", "analyst"},
	}

	flow := toFlow(rec)
	assert.True(t, flow.CanSpawn("code_agent"))
	assert.True(t, flow.CanSpawn("analyst"))
	assert.False(t, flow.CanSpawn("unknown"))
}

func TestEmptyDB_EmptyRegistry(t *testing.T) {
	repo := &mockAgentReader{records: []config_repo.AgentRecord{}}
	reg := New(repo)

	require.NoError(t, reg.Load(context.Background()))
	assert.Equal(t, 0, reg.Count())
	assert.Empty(t, reg.List())
	assert.Empty(t, reg.GetAll())
}
