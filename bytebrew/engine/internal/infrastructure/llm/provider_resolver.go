package llm

import (
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/cloudwego/eino/components/model"
)

// flowTypeRoles maps each FlowType to the "role" string sent to the proxy.
var flowTypeRoles = map[domain.FlowType]string{
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
//   - "byok" (or empty): all flow types use the BYOK model.
//   - "proxy": each flow type gets a ProxyChatModel with its own role.
//   - "auto": each flow type gets an AutoChatModel(proxy, byok).
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

// resolveProxy creates a ModelSelector where every flow type is backed by a
// ProxyChatModel with the matching role.
func resolveProxy(cfg ProviderResolverConfig) *ModelSelector {
	// Default proxy model (used for unknown flow types).
	defaultProxy := NewProxyChatModel(cfg.CloudAPIURL, cfg.AccessToken, "default")
	selector := NewModelSelector(defaultProxy, "proxy-llm")

	for ft, role := range flowTypeRoles {
		proxy := NewProxyChatModel(cfg.CloudAPIURL, cfg.AccessToken, role)
		selector.SetModel(ft, proxy, "proxy-llm")
	}

	return selector
}

// resolveAuto creates a ModelSelector where every flow type is backed by an
// AutoChatModel that tries proxy first and falls back to byok.
func resolveAuto(cfg ProviderResolverConfig) *ModelSelector {
	defaultProxy := NewProxyChatModel(cfg.CloudAPIURL, cfg.AccessToken, "default")
	defaultAuto := NewAutoChatModel(defaultProxy, cfg.BYOKModel)
	selector := NewModelSelector(defaultAuto, "auto-llm")

	for ft, role := range flowTypeRoles {
		proxy := NewProxyChatModel(cfg.CloudAPIURL, cfg.AccessToken, role)
		auto := NewAutoChatModel(proxy, cfg.BYOKModel)
		selector.SetModel(ft, auto, "auto-llm")
	}

	return selector
}
