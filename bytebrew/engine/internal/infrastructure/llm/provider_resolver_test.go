package llm

import (
	"context"
	"testing"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
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

	// All flow types should return the BYOK model.
	for _, ft := range []domain.FlowType{
		domain.FlowType("supervisor"),
		domain.FlowType("coder"),
		domain.FlowType("reviewer"),
		domain.FlowType("researcher"),
	} {
		m := selector.Select(ft)
		resp, err := m.Generate(context.Background(), nil)
		require.NoError(t, err)
		assert.Equal(t, "byok", resp.Content, "flow type %s should use byok", ft)
	}
}

func TestResolveModelSelector_EmptyMode_DefaultsByok(t *testing.T) {
	byok := &mockChatModel{id: "byok"}
	selector := ResolveModelSelector(ProviderResolverConfig{
		Mode:          "",
		BYOKModel:     byok,
		BYOKModelName: "gpt-4",
	})

	m := selector.Select(domain.FlowType("supervisor"))
	resp, err := m.Generate(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "byok", resp.Content, "empty mode should default to byok")
	assert.Equal(t, "gpt-4", selector.ModelName(domain.FlowType("supervisor")))
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

	// Each flow type should get a ProxyChatModel.
	for _, ft := range []domain.FlowType{
		domain.FlowType("supervisor"),
		domain.FlowType("coder"),
		domain.FlowType("reviewer"),
		domain.FlowType("researcher"),
	} {
		m := selector.Select(ft)
		proxy, ok := m.(*ProxyChatModel)
		require.True(t, ok, "flow type %s should return *ProxyChatModel", ft)
		assert.Equal(t, "http://proxy.example.com", proxy.cloudAPIURL)
		assert.Equal(t, "token-123", proxy.accessToken)
		assert.Equal(t, flowTypeRoles[ft], proxy.role)
	}

	// Model name should be "proxy-llm".
	assert.Equal(t, "proxy-llm", selector.ModelName(domain.FlowType("supervisor")))
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

	// Each flow type should get an AutoChatModel.
	for _, ft := range []domain.FlowType{
		domain.FlowType("supervisor"),
		domain.FlowType("coder"),
		domain.FlowType("reviewer"),
		domain.FlowType("researcher"),
	} {
		m := selector.Select(ft)
		auto, ok := m.(*AutoChatModel)
		require.True(t, ok, "flow type %s should return *AutoChatModel", ft)

		// Type assert proxy to verify internal wiring (allowed in tests).
		proxy, ok := auto.proxy.(*ProxyChatModel)
		require.True(t, ok, "auto.proxy should be *ProxyChatModel for %s", ft)
		assert.Equal(t, flowTypeRoles[ft], proxy.role)
		assert.Equal(t, "http://proxy.example.com", proxy.cloudAPIURL)

		// BYOK should be the mock.
		resp, err := auto.byok.Generate(context.Background(), nil)
		require.NoError(t, err)
		assert.Equal(t, "byok", resp.Content)
	}

	// Model name should be "auto-llm".
	assert.Equal(t, "auto-llm", selector.ModelName(domain.FlowType("supervisor")))
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

	// An unknown flow type should fall back to the default proxy.
	unknownFlowType := domain.FlowType("unknown")
	m := selector.Select(unknownFlowType)
	proxy, ok := m.(*ProxyChatModel)
	require.True(t, ok, "unknown flow type should return the default *ProxyChatModel")
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

	unknownFlowType := domain.FlowType("unknown")
	m := selector.Select(unknownFlowType)
	auto, ok := m.(*AutoChatModel)
	require.True(t, ok, "unknown flow type should return the default *AutoChatModel")
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

	m := selector.Select(domain.FlowType("supervisor"))
	resp, err := m.Generate(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "byok", resp.Content, "unknown mode should fall back to byok")
}

// Verify that flowTypeRoles covers all defined FlowTypes.
func TestFlowTypeRoles_Coverage(t *testing.T) {
	expectedFlows := []domain.FlowType{
		domain.FlowType("supervisor"),
		domain.FlowType("coder"),
		domain.FlowType("reviewer"),
		domain.FlowType("researcher"),
	}

	for _, ft := range expectedFlows {
		_, ok := flowTypeRoles[ft]
		assert.True(t, ok, "flowTypeRoles should contain %s", ft)
	}
}

// Verify compile-time interface satisfaction for all model types.
func TestInterfaceSatisfaction(t *testing.T) {
	var _ model.ToolCallingChatModel = (*ProxyChatModel)(nil)
	var _ model.ToolCallingChatModel = (*AutoChatModel)(nil)
}
