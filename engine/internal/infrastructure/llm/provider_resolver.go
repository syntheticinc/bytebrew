package llm

import (
	"github.com/cloudwego/eino/components/model"
)

// agentRoles maps each agent name to the "role" string sent to the proxy.
var agentRoles = map[string]string{
	"supervisor": "supervisor",
	"coder":      "coder",
	"reviewer":   "reviewer",
	"researcher": "researcher",
}

// ProviderResolverConfig holds the inputs for creating a ModelSelector.
type ProviderResolverConfig struct {
	Mode          string                      // "proxy" | "byok" | "auto" (empty = "byok")
	CloudAPIURL   string                      // Cloud API base URL
	AccessToken   string                      // Bearer token for proxy auth
	BYOKModel     model.ToolCallingChatModel  // BYOK chat model (OpenRouter/Ollama/etc.)
	BYOKModelName string                      // display name for BYOK model
}

// ResolveModelSelector creates a ModelSelector based on the configured mode.
//
//   - "byok" (or empty): all agent types use the BYOK model.
//   - "proxy": each agent type gets a ProxyChatModel with its own role.
//   - "auto": each agent type gets an AutoChatModel(proxy, byok).
func ResolveModelSelector(cfg ProviderResolverConfig) *ModelSelector {
	mode := cfg.Mode
	if mode == "" {
		mode = "byok"
	}

	switch mode {
	case "proxy":
		return resolveProxy(cfg)
	case "auto":
		return resolveAuto(cfg)
	default: // "byok" and any unknown value
		return NewModelSelector(cfg.BYOKModel, cfg.BYOKModelName)
	}
}

// resolveProxy creates a ModelSelector where every agent type is backed by a
// ProxyChatModel with the matching role.
func resolveProxy(cfg ProviderResolverConfig) *ModelSelector {
	// Default proxy model (used for unknown agent types).
	defaultProxy := NewProxyChatModel(cfg.CloudAPIURL, cfg.AccessToken, "default")
	selector := NewModelSelector(defaultProxy, "proxy-llm")

	for agentName, role := range agentRoles {
		proxy := NewProxyChatModel(cfg.CloudAPIURL, cfg.AccessToken, role)
		selector.SetModel(agentName, proxy, "proxy-llm")
	}

	return selector
}

// resolveAuto creates a ModelSelector where every agent type is backed by an
// AutoChatModel that tries proxy first and falls back to byok.
func resolveAuto(cfg ProviderResolverConfig) *ModelSelector {
	defaultProxy := NewProxyChatModel(cfg.CloudAPIURL, cfg.AccessToken, "default")
	defaultAuto := NewAutoChatModel(defaultProxy, cfg.BYOKModel)
	selector := NewModelSelector(defaultAuto, "auto-llm")

	for agentName, role := range agentRoles {
		proxy := NewProxyChatModel(cfg.CloudAPIURL, cfg.AccessToken, role)
		auto := NewAutoChatModel(proxy, cfg.BYOKModel)
		selector.SetModel(agentName, auto, "auto-llm")
	}

	return selector
}
