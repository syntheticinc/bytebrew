package agentregistry

import (
	"sort"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/configrepo"
)

// DeriveRuntimeTools returns the sorted, deduplicated list of tool names that
// the agent's runtime should have access to at execution time. It is the
// single source of truth for the "what tools does this agent have?" question.
// Deterministic: same inputs → same output order.
//
// Rules:
//   - Base: agent.BuiltinTools + agent.CustomTools (explicit opt-in tools)
//   - Add `spawn_<name>` for every name in agent.CanSpawn
//   - Add memory_recall + memory_store if any capability has type="memory" and enabled=true
//   - Add knowledge_search if any capability has type="knowledge" and enabled=true
//   - Deduplicate — a tool appearing in base AND derived from a capability is listed once.
func DeriveRuntimeTools(agent configrepo.AgentRecord, capabilities []configrepo.CapabilityRecord) []string {
	seen := make(map[string]bool)
	var tools []string

	add := func(name string) {
		if !seen[name] {
			seen[name] = true
			tools = append(tools, name)
		}
	}

	// Base: explicit builtin tools.
	for _, t := range agent.BuiltinTools {
		add(t)
	}

	// Base: explicit custom tools.
	for _, ct := range agent.CustomTools {
		add(ct.Name)
	}

	// Spawn delegation: one spawn_<name> tool per target.
	for _, name := range agent.CanSpawn {
		add("spawn_" + name)
	}

	// Capability-injected tools.
	for _, cap := range capabilities {
		if !cap.Enabled {
			continue
		}
		switch cap.Type {
		case "memory":
			add("memory_recall")
			add("memory_store")
		case "knowledge":
			add("knowledge_search")
		}
	}

	sort.Strings(tools)
	return tools
}
