package tools

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/agent_registry"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/config_repo"
)

// stubTool is a minimal tool implementation for testing.
type stubTool struct {
	name string
}

func (t *stubTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: t.name}, nil
}

func (t *stubTool) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	return "ok:" + t.name, nil
}

func TestBuiltinToolStore_RegisterAndGet(t *testing.T) {
	store := NewBuiltinToolStore()

	factory := func(deps ToolDependencies) tool.InvokableTool {
		return &stubTool{name: "test_tool"}
	}
	store.Register("test_tool", factory)

	got, ok := store.Get("test_tool")
	require.True(t, ok)
	require.NotNil(t, got)

	instance := got(ToolDependencies{})
	result, err := instance.InvokableRun(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, "ok:test_tool", result)
}

func TestBuiltinToolStore_GetUnknown(t *testing.T) {
	store := NewBuiltinToolStore()

	_, ok := store.Get("nonexistent")
	assert.False(t, ok)
}

func TestBuiltinToolStore_Names(t *testing.T) {
	store := NewBuiltinToolStore()
	noopFactory := func(deps ToolDependencies) tool.InvokableTool { return &stubTool{} }

	store.Register("charlie", noopFactory)
	store.Register("alpha", noopFactory)
	store.Register("bravo", noopFactory)

	names := store.Names()
	assert.Equal(t, []string{"alpha", "bravo", "charlie"}, names)
}

func TestBuiltinToolStore_NamesEmpty(t *testing.T) {
	store := NewBuiltinToolStore()
	names := store.Names()
	assert.Empty(t, names)
}

func TestAgentToolResolver_ResolveForAgent_Whitelist(t *testing.T) {
	store := NewBuiltinToolStore()
	store.Register("tool_a", func(deps ToolDependencies) tool.InvokableTool {
		return &stubTool{name: "tool_a"}
	})
	store.Register("tool_b", func(deps ToolDependencies) tool.InvokableTool {
		return &stubTool{name: "tool_b"}
	})
	store.Register("tool_c", func(deps ToolDependencies) tool.InvokableTool {
		return &stubTool{name: "tool_c"}
	})

	resolver := NewAgentToolResolver(store)
	agent := &agent_registry.RegisteredAgent{
		Record: config_repo.AgentRecord{
			Name:         "test_agent",
			BuiltinTools: []string{"tool_a", "tool_c"},
		},
	}

	tools, err := resolver.ResolveForAgent(context.Background(), ResolveContext{
		Agent: agent,
		Deps:  ToolDependencies{},
	})
	require.NoError(t, err)
	require.Len(t, tools, 2)

	// Verify tools are in whitelist order
	info0, _ := tools[0].Info(context.Background())
	info1, _ := tools[1].Info(context.Background())
	assert.Equal(t, "tool_a", info0.Name)
	assert.Equal(t, "tool_c", info1.Name)
}

func TestAgentToolResolver_ResolveForAgent_UnknownTool(t *testing.T) {
	store := NewBuiltinToolStore()
	store.Register("tool_a", func(deps ToolDependencies) tool.InvokableTool {
		return &stubTool{name: "tool_a"}
	})

	resolver := NewAgentToolResolver(store)
	agent := &agent_registry.RegisteredAgent{
		Record: config_repo.AgentRecord{
			Name:         "test_agent",
			BuiltinTools: []string{"tool_a", "unknown_tool"},
		},
	}

	tools, err := resolver.ResolveForAgent(context.Background(), ResolveContext{
		Agent: agent,
		Deps:  ToolDependencies{},
	})
	require.Error(t, err)
	assert.Nil(t, tools)
	assert.Contains(t, err.Error(), "unknown builtin tool")
	assert.Contains(t, err.Error(), "unknown_tool")
	assert.Contains(t, err.Error(), "test_agent")
}

func TestAgentToolResolver_ResolveForAgent_EmptyWhitelist(t *testing.T) {
	store := NewBuiltinToolStore()
	store.Register("tool_a", func(deps ToolDependencies) tool.InvokableTool {
		return &stubTool{name: "tool_a"}
	})

	resolver := NewAgentToolResolver(store)
	agent := &agent_registry.RegisteredAgent{
		Record: config_repo.AgentRecord{
			Name:         "test_agent",
			BuiltinTools: nil,
		},
	}

	tools, err := resolver.ResolveForAgent(context.Background(), ResolveContext{
		Agent: agent,
		Deps:  ToolDependencies{},
	})
	require.NoError(t, err)
	assert.Empty(t, tools)
}

func TestAgentToolResolver_ResolveForAgent_PassesDeps(t *testing.T) {
	store := NewBuiltinToolStore()

	var capturedSessionID string
	store.Register("dep_tool", func(deps ToolDependencies) tool.InvokableTool {
		capturedSessionID = deps.SessionID
		return &stubTool{name: "dep_tool"}
	})

	resolver := NewAgentToolResolver(store)
	agent := &agent_registry.RegisteredAgent{
		Record: config_repo.AgentRecord{
			Name:         "test_agent",
			BuiltinTools: []string{"dep_tool"},
		},
	}

	_, err := resolver.ResolveForAgent(context.Background(), ResolveContext{
		Agent: agent,
		Deps:  ToolDependencies{SessionID: "session-42"},
	})
	require.NoError(t, err)
	assert.Equal(t, "session-42", capturedSessionID)
}

// --- Kit integration tests ---

type mockKit struct {
	tools []tool.InvokableTool
}

func (m *mockKit) Tools(_ domain.KitSession) []tool.InvokableTool {
	return m.tools
}

func (m *mockKit) PostToolCall(_ context.Context, _ domain.KitSession, _ string, _ string) *domain.Enrichment {
	return nil
}

type mockKitProvider struct {
	kits map[string]Kit
}

func (m *mockKitProvider) Get(name string) (Kit, error) {
	k, ok := m.kits[name]
	if !ok {
		return nil, fmt.Errorf("kit %q not registered", name)
	}
	return k, nil
}

func TestAgentToolResolver_KitTools_Appended(t *testing.T) {
	store := NewBuiltinToolStore()
	store.Register("tool_a", func(deps ToolDependencies) tool.InvokableTool {
		return &stubTool{name: "tool_a"}
	})

	kitTool := &stubTool{name: "kit_tool"}
	kp := &mockKitProvider{
		kits: map[string]Kit{
			"developer": &mockKit{tools: []tool.InvokableTool{kitTool}},
		},
	}

	resolver := NewAgentToolResolver(store)
	resolver.SetKitProvider(kp)

	agent := &agent_registry.RegisteredAgent{
		Record: config_repo.AgentRecord{
			Name:         "code_agent",
			Kit:          "developer",
			BuiltinTools: []string{"tool_a"},
		},
	}

	session := &domain.KitSession{SessionID: "sess-1", ProjectRoot: "/tmp"}
	tools, err := resolver.ResolveForAgent(context.Background(), ResolveContext{
		Agent:      agent,
		Deps:       ToolDependencies{},
		KitSession: session,
	})
	require.NoError(t, err)
	require.Len(t, tools, 2)

	info0, _ := tools[0].Info(context.Background())
	info1, _ := tools[1].Info(context.Background())
	assert.Equal(t, "tool_a", info0.Name)
	assert.Equal(t, "kit_tool", info1.Name)
}

func TestAgentToolResolver_NoKit_NoExtraTools(t *testing.T) {
	store := NewBuiltinToolStore()
	store.Register("tool_a", func(deps ToolDependencies) tool.InvokableTool {
		return &stubTool{name: "tool_a"}
	})

	resolver := NewAgentToolResolver(store)
	// No kit provider set

	agent := &agent_registry.RegisteredAgent{
		Record: config_repo.AgentRecord{
			Name:         "basic_agent",
			Kit:          "",
			BuiltinTools: []string{"tool_a"},
		},
	}

	tools, err := resolver.ResolveForAgent(context.Background(), ResolveContext{
		Agent: agent,
		Deps:  ToolDependencies{},
	})
	require.NoError(t, err)
	assert.Len(t, tools, 1)
}

func TestAgentToolResolver_KitName_NoSession(t *testing.T) {
	store := NewBuiltinToolStore()
	store.Register("tool_a", func(deps ToolDependencies) tool.InvokableTool {
		return &stubTool{name: "tool_a"}
	})

	kp := &mockKitProvider{
		kits: map[string]Kit{
			"developer": &mockKit{tools: []tool.InvokableTool{&stubTool{name: "kit_tool"}}},
		},
	}

	resolver := NewAgentToolResolver(store)
	resolver.SetKitProvider(kp)

	agent := &agent_registry.RegisteredAgent{
		Record: config_repo.AgentRecord{
			Name:         "code_agent",
			Kit:          "developer",
			BuiltinTools: []string{"tool_a"},
		},
	}

	// KitSession is nil — kit tools should NOT be added
	tools, err := resolver.ResolveForAgent(context.Background(), ResolveContext{
		Agent:      agent,
		Deps:       ToolDependencies{},
		KitSession: nil,
	})
	require.NoError(t, err)
	assert.Len(t, tools, 1)
}

func TestAgentToolResolver_KitNotFound_Error(t *testing.T) {
	store := NewBuiltinToolStore()

	kp := &mockKitProvider{
		kits: map[string]Kit{}, // empty
	}

	resolver := NewAgentToolResolver(store)
	resolver.SetKitProvider(kp)

	agent := &agent_registry.RegisteredAgent{
		Record: config_repo.AgentRecord{
			Name: "code_agent",
			Kit:  "nonexistent",
		},
	}

	session := &domain.KitSession{SessionID: "sess-1"}
	_, err := resolver.ResolveForAgent(context.Background(), ResolveContext{
		Agent:      agent,
		Deps:       ToolDependencies{},
		KitSession: session,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestAgentToolResolver_KitReturnsNilTools(t *testing.T) {
	store := NewBuiltinToolStore()
	store.Register("tool_a", func(deps ToolDependencies) tool.InvokableTool {
		return &stubTool{name: "tool_a"}
	})

	kp := &mockKitProvider{
		kits: map[string]Kit{
			"developer": &mockKit{tools: nil}, // kit returns nil tools (skeleton)
		},
	}

	resolver := NewAgentToolResolver(store)
	resolver.SetKitProvider(kp)

	agent := &agent_registry.RegisteredAgent{
		Record: config_repo.AgentRecord{
			Name:         "code_agent",
			Kit:          "developer",
			BuiltinTools: []string{"tool_a"},
		},
	}

	session := &domain.KitSession{SessionID: "sess-1"}
	tools, err := resolver.ResolveForAgent(context.Background(), ResolveContext{
		Agent:      agent,
		Deps:       ToolDependencies{},
		KitSession: session,
	})
	require.NoError(t, err)
	assert.Len(t, tools, 1) // only builtin
}
