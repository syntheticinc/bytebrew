// Shared helpers for reasoning about agent delegation graphs.
//
// Agent relations come from two different admin-endpoints depending on how
// the relation was created — some hand us the agent's display name as the
// endpoint key, some hand us the agent's UUID. The tree renderer and the
// SchemaDetailPage both need to look up agents by whichever key the relation
// happened to use, so we normalize via two lookup maps (by id AND by name)
// and consult both when walking the graph.

export interface GraphAgent {
  id: string;
  name: string;
}

export interface GraphRelation {
  sourceAgentId: string;
  targetAgentId: string;
}

// computeEntryAgent picks the source-only agent — one that has outgoing
// relations but no incoming ones — as the entry orchestrator. Falls back to
// null if every agent is a target of some relation (e.g. cyclic graph) or
// if there are no agents/relations at all.
//
// Returns the agent's display name (TreeAgent.id === agent.name in the
// real admin), matching how SchemaDetailPage's entryAgentId is consumed.
export function computeEntryAgent(
  agents: GraphAgent[],
  relations: GraphRelation[],
): string | null {
  if (agents.length === 0) return null;

  // Build a set of all keys that appear as a relation TARGET, including both
  // the agent name and the agent UUID, so the lookup works regardless of
  // which key the underlying endpoint used.
  const byId = new Map<string, GraphAgent>();
  const byName = new Map<string, GraphAgent>();
  for (const a of agents) {
    if (a.id) byId.set(a.id, a);
    if (a.name) byName.set(a.name, a);
  }

  const incomingKeys = new Set<string>();
  for (const r of relations) {
    // Add the raw target key first — this covers the case where the relation
    // endpoint uses a UUID that doesn't match any agent name.
    incomingKeys.add(r.targetAgentId);
    // Then try to resolve it to both id and name of the same agent so a
    // relation keyed by UUID still marks the agent as "has incoming" when
    // checked by name, and vice versa.
    const a = byId.get(r.targetAgentId) ?? byName.get(r.targetAgentId);
    if (a) {
      if (a.id) incomingKeys.add(a.id);
      if (a.name) incomingKeys.add(a.name);
    }
  }

  // Also require the agent to have at least one outgoing relation — a lone
  // agent with no edges at all is not an "entry orchestrator", it's just
  // an isolated node.
  const outgoingKeys = new Set<string>();
  for (const r of relations) {
    outgoingKeys.add(r.sourceAgentId);
    const a = byId.get(r.sourceAgentId) ?? byName.get(r.sourceAgentId);
    if (a) {
      if (a.id) outgoingKeys.add(a.id);
      if (a.name) outgoingKeys.add(a.name);
    }
  }

  for (const a of agents) {
    const hasIncoming = incomingKeys.has(a.id) || incomingKeys.has(a.name);
    const hasOutgoing = outgoingKeys.has(a.id) || outgoingKeys.has(a.name);
    if (hasOutgoing && !hasIncoming) {
      return a.name;
    }
  }

  return null;
}

// resolveAgentKey walks two lookup maps (by id and by name) and returns the
// resolved agent's canonical name, or null if the key doesn't match any
// agent. Callers use this to normalize heterogeneous relation keys to a
// single axis (agent name) before tree-building.
export function resolveAgentName(
  key: string,
  byId: Map<string, GraphAgent>,
  byName: Map<string, GraphAgent>,
): string | null {
  const a = byId.get(key) ?? byName.get(key);
  return a ? a.name : null;
}
