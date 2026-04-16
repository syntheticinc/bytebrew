package llm

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveModelSelector_Byok(t *testing.T) {
	byok := &mockChatModel{id: "byok"}
	selector := ResolveModelSelector(ProviderResolverConfig{
		Mode:          "byok",
		BYOKModel:     byok,
		BYOKModelName: "gpt-4",
	})

	// All agent types should return the BYOK model.
	for _, agentName := range []string{
		"supervisor",
		"coder",
		"reviewer",
		"researcher",
	} {
		m := selector.Select(agentName)
		resp, err := m.Generate(context.Background(), nil)
		require.NoError(t, err)
		assert.Equal(t, "byok", resp.Content, "agent %s should use byok", agentName)
	}
}

func TestResolveModelSelector_EmptyMode_DefaultsByok(t *testing.T) {
	byok := &mockChatModel{id: "byok"}
	selector := ResolveModelSelector(ProviderResolverConfig{
		Mode:          "",
		BYOKModel:     byok,
		BYOKModelName: "gpt-4",
	})

	m := selector.Select("supervisor")
	resp, err := m.Generate(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "byok", resp.Content, "empty mode should default to byok")
	assert.Equal(t, "gpt-4", selector.ModelName("supervisor"))
}

func TestResolveModelSelector_Proxy(t *testing.T) {
	byok := &mockChatModel{id: "byok"}
	selector := ResolveModelSelector(ProviderResolverConfig{
		Mode:          "proxy",
		CloudAPIURL:   "http://proxy.example.com",
		AccessToken:   "token-123",
		BYOKModel:     byok,
		BYOKModelName: "gpt-4",
	})

	// Each agent type should get a ProxyChatModel.
	for _, agentName := range []string{
		"supervisor",
		"coder",
		"reviewer",
		"researcher",
	} {
		m := selector.Select(agentName)
		proxy, ok := m.(*ProxyChatModel)
		require.True(t, ok, "agent %s should return *ProxyChatModel", agentName)
		assert.Equal(t, "http://proxy.example.com", proxy.cloudAPIURL)
		assert.Equal(t, "token-123", proxy.accessToken)
		assert.Equal(t, agentRoles[agentName], proxy.role)
	}

	// Model name should be "proxy-llm".
	assert.Equal(t, "proxy-llm", selector.ModelName("supervisor"))
}

func TestResolveModelSelector_Auto(t *testing.T) {
	byok := &mockChatModel{id: "byok"}
	selector := ResolveModelSelector(ProviderResolverConfig{
		Mode:          "auto",
		CloudAPIURL:   "http://proxy.example.com",
		AccessToken:   "token-456",
		BYOKModel:     byok,
		BYOKModelName: "gpt-4",
	})

	// Each agent type should get an AutoChatModel.
	for _, agentName := range []string{
		"supervisor",
		"coder",
		"reviewer",
		"researcher",
	} {
		m := selector.Select(agentName)
		auto, ok := m.(*AutoChatModel)
		require.True(t, ok, "agent %s should return *AutoChatModel", agentName)

		// Type assert proxy to verify internal wiring (allowed in tests).
		proxy, ok := auto.proxy.(*ProxyChatModel)
		require.True(t, ok, "auto.proxy should be *ProxyChatModel for %s", agentName)
		assert.Equal(t, agentRoles[agentName], proxy.role)
		assert.Equal(t, "http://proxy.example.com", proxy.cloudAPIURL)

		// BYOK should be the mock.
		resp, err := auto.byok.Generate(context.Background(), nil)
		require.NoError(t, err)
		assert.Equal(t, "byok", resp.Content)
	}

	// Model name should be "auto-llm".
	assert.Equal(t, "auto-llm", selector.ModelName("supervisor"))
}

func TestResolveModelSelector_Proxy_DefaultModel(t *testing.T) {
	byok := &mockChatModel{id: "byok"}
	selector := ResolveModelSelector(ProviderResolverConfig{
		Mode:          "proxy",
		CloudAPIURL:   "http://proxy.example.com",
		AccessToken:   "token",
		BYOKModel:     byok,
		BYOKModelName: "gpt-4",
	})

	// An unknown agent type should fall back to the default proxy.
	m := selector.Select("unknown")
	proxy, ok := m.(*ProxyChatModel)
	require.True(t, ok, "unknown agent type should return the default *ProxyChatModel")
	assert.Equal(t, "default", proxy.role)
}

func TestResolveModelSelector_Auto_DefaultModel(t *testing.T) {
	byok := &mockChatModel{id: "byok"}
	selector := ResolveModelSelector(ProviderResolverConfig{
		Mode:          "auto",
		CloudAPIURL:   "http://proxy.example.com",
		AccessToken:   "token",
		BYOKModel:     byok,
		BYOKModelName: "gpt-4",
	})

	m := selector.Select("unknown")
	auto, ok := m.(*AutoChatModel)
	require.True(t, ok, "unknown agent type should return the default *AutoChatModel")
	proxy, ok := auto.proxy.(*ProxyChatModel)
	require.True(t, ok, "auto.proxy should be *ProxyChatModel")
	assert.Equal(t, "default", proxy.role)
}

func TestResolveModelSelector_UnknownMode_DefaultsByok(t *testing.T) {
	byok := &mockChatModel{id: "byok"}
	selector := ResolveModelSelector(ProviderResolverConfig{
		Mode:          "something-invalid",
		BYOKModel:     byok,
		BYOKModelName: "gpt-4",
	})

	m := selector.Select("supervisor")
	resp, err := m.Generate(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "byok", resp.Content, "unknown mode should fall back to byok")
}

// Verify that agentRoles covers all defined agent types.
func TestAgentRoles_Coverage(t *testing.T) {
	expectedAgents := []string{
		"supervisor",
		"coder",
		"reviewer",
		"researcher",
	}

	for _, name := range expectedAgents {
		_, ok := agentRoles[name]
		assert.True(t, ok, "agentRoles should contain %s", name)
	}
}

// Verify compile-time interface satisfaction for all model types.
func TestInterfaceSatisfaction(t *testing.T) {
	var _ model.ToolCallingChatModel = (*ProxyChatModel)(nil)
	var _ model.ToolCallingChatModel = (*AutoChatModel)(nil)
}
