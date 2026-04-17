import { useMemo } from 'react';
import type { V2Agent, V2AgentRelation, V2Trigger } from '../../mocks/v2';

// ─── Types ──────────────────────────────────────────────────────────────────

interface V2DelegationTreeProps {
  triggers: V2Trigger[];
  agents: V2Agent[];
  relations: V2AgentRelation[];
  entryAgentId: string;
  highlightAgentIds?: Set<string>;
  highlightTriggerIds?: Set<string>;
  onAgentOpen?: (agentId: string) => void;
  onTriggerOpen?: (triggerId: string) => void;
  onAddChild?: (parentAgentId: string) => void;
  onRemoveDelegation?: (agentId: string) => void;
}

interface TreeNode {
  agent: V2Agent;
  children: TreeNode[];
  depth: number;
}

// ─── Constants ──────────────────────────────────────────────────────────────

const STUB_PX = 24; // vertical gap between card edge and horizontal bus
const SIBLING_PAD_PX = 24; // horizontal padding around each node (half-gap between siblings)
const LINE_COLOR = 'rgba(148, 163, 184, 0.55)'; // slate-400/55 — visible on dark

// ─── Tree building ──────────────────────────────────────────────────────────

function buildTree(
  agents: V2Agent[],
  relations: V2AgentRelation[],
  entryId: string,
): TreeNode | null {
  const byId = new Map(agents.map((a) => [a.id, a]));
  const childrenByParent = new Map<string, string[]>();
  for (const r of relations) {
    const list = childrenByParent.get(r.sourceAgentId) ?? [];
    list.push(r.targetAgentId);
    childrenByParent.set(r.sourceAgentId, list);
  }

  const visited = new Set<string>();
  function build(id: string, depth: number): TreeNode | null {
    if (visited.has(id)) return null;
    visited.add(id);
    const agent = byId.get(id);
    if (!agent) return null;
    const childIds = childrenByParent.get(id) ?? [];
    const children = childIds
      .map((cid) => build(cid, depth + 1))
      .filter((n): n is TreeNode => n !== null);
    return { agent, children, depth };
  }

  return build(entryId, 0);
}

// ─── Card components ────────────────────────────────────────────────────────

function TriggerCard({
  trigger,
  highlighted,
  onClick,
}: {
  trigger: V2Trigger;
  highlighted: boolean;
  onClick?: () => void;
}) {
  const ring = highlighted
    ? 'ring-2 ring-purple-400 shadow-[0_0_14px_rgba(168,85,247,0.45)]'
    : 'ring-1 ring-brand-shade3/25 hover:ring-brand-shade3/50';
  const typeLabel =
    trigger.type === 'cron' ? 'Cron' : trigger.type === 'webhook' ? 'Webhook' : 'Chat';
  return (
    <button
      onClick={onClick}
      className={`flex items-center gap-2 bg-brand-dark-alt/80 backdrop-blur-sm rounded-btn py-2 px-3 min-w-[160px] text-left transition-all ${ring}`}
    >
      <span
        className={`w-1.5 h-1.5 rounded-full shrink-0 ${
          trigger.enabled ? 'bg-brand-accent' : 'bg-brand-shade3/40'
        }`}
      />
      <div className="min-w-0 flex-1">
        <div className="text-[12px] font-semibold text-brand-light truncate leading-tight">
          {trigger.title}
        </div>
        <div className="text-[9px] uppercase tracking-wider text-brand-shade3 mt-0.5">
          {typeLabel}
        </div>
      </div>
    </button>
  );
}

function AgentCard({
  agent,
  isEntry,
  highlighted,
  onClick,
  onAddChild,
  onRemove,
}: {
  agent: V2Agent;
  isEntry: boolean;
  highlighted: boolean;
  onClick?: () => void;
  onAddChild?: () => void;
  onRemove?: () => void;
}) {
  const ring = highlighted
    ? 'ring-2 ring-purple-400 shadow-[0_0_24px_rgba(168,85,247,0.45)]'
    : 'ring-1 ring-brand-shade3/25 hover:ring-brand-shade3/50';
  const stateDot =
    agent.state === 'active'
      ? 'bg-emerald-400'
      : agent.state === 'degraded'
        ? 'bg-amber-400'
        : 'bg-brand-shade3/50';
  return (
    <div className="group relative v2tree-card-wrap">
      <button
        onClick={onClick}
        className={`relative w-[220px] rounded-card bg-brand-dark-surface cursor-pointer transition-all duration-200 ${ring}`}
      >
      <div className="flex items-center gap-3 px-3 pt-3 pb-2">
        <div className="shrink-0 w-9 h-9 rounded-full bg-gradient-to-br from-brand-shade3/30 to-brand-shade3/10 flex items-center justify-center text-[13px] font-semibold text-brand-light border border-brand-shade3/20">
          {agent.avatarInitials}
        </div>
        <div className="min-w-0 flex-1 text-left">
          <div className="flex items-center gap-1.5">
            <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${stateDot}`} />
            <span className="text-[13px] font-semibold text-brand-light truncate">
              {agent.name}
            </span>
          </div>
          <div className="text-[10px] text-brand-shade3 font-mono truncate mt-0.5">
            {agent.model}
          </div>
        </div>
        {isEntry && (
          <span
            title="Entry orchestrator"
            className="shrink-0 px-1.5 py-0.5 rounded text-[9px] font-semibold tracking-wider uppercase bg-brand-accent/15 text-brand-accent border border-brand-accent/30"
          >
            Entry
          </span>
        )}
      </div>
      <div className="px-3 pb-2 text-[10px] text-left">
        {agent.activeSessions > 0 ? (
          <span className="text-emerald-400 font-medium">
            {agent.activeSessions} active
          </span>
        ) : (
          <span className="text-brand-shade3/60 uppercase tracking-wider">idle</span>
        )}
      </div>
    </button>
      {onAddChild && (
        <button
          onClick={(e) => {
            e.stopPropagation();
            onAddChild();
          }}
          title="Add delegate under this agent"
          className="absolute left-1/2 -translate-x-1/2 -bottom-3 z-10 h-6 px-2.5 rounded-full bg-brand-dark-surface text-brand-accent text-[10px] font-semibold leading-none flex items-center gap-1 border border-brand-accent/40 shadow-sm hover:bg-brand-accent hover:text-white hover:border-brand-accent transition-colors opacity-0 group-hover:opacity-100"
        >
          <span className="text-[13px] leading-none">+</span>
          <span className="tracking-wider uppercase">Delegate</span>
        </button>
      )}
      {onRemove && !isEntry && (
        <button
          onClick={(e) => {
            e.stopPropagation();
            onRemove();
          }}
          title="Remove delegation (detaches subtree from schema)"
          className="absolute -right-2 -top-2 z-10 w-5 h-5 rounded-full bg-brand-dark-surface text-brand-shade3/70 text-[11px] leading-none flex items-center justify-center border border-brand-shade3/25 hover:bg-red-500/15 hover:text-red-400 hover:border-red-500/40 transition-colors opacity-0 group-hover:opacity-100"
        >
          ×
        </button>
      )}
    </div>
  );
}

// ─── Tree recursive render ──────────────────────────────────────────────────

function TreeBranch({
  node,
  entryId,
  highlightAgentIds,
  onAgentOpen,
  onAddChild,
  onRemoveDelegation,
}: {
  node: TreeNode;
  entryId: string;
  highlightAgentIds?: Set<string>;
  onAgentOpen?: (agentId: string) => void;
  onAddChild?: (parentAgentId: string) => void;
  onRemoveDelegation?: (agentId: string) => void;
}) {
  const hasChildren = node.children.length > 0;
  return (
    <li className="v2tree-node">
      <AgentCard
        agent={node.agent}
        isEntry={node.agent.id === entryId}
        highlighted={highlightAgentIds?.has(node.agent.id) ?? false}
        onClick={() => onAgentOpen?.(node.agent.id)}
        onAddChild={onAddChild ? () => onAddChild(node.agent.id) : undefined}
        onRemove={onRemoveDelegation ? () => onRemoveDelegation(node.agent.id) : undefined}
      />
      {hasChildren && (
        <>
          <div className="v2tree-parent-stub" />
          <ul className="v2tree-children">
            {node.children.map((child) => (
              <TreeBranch
                key={child.agent.id}
                node={child}
                entryId={entryId}
                highlightAgentIds={highlightAgentIds}
                onAgentOpen={onAgentOpen}
                onAddChild={onAddChild}
                onRemoveDelegation={onRemoveDelegation}
              />
            ))}
          </ul>
        </>
      )}
    </li>
  );
}

// ─── Main component ─────────────────────────────────────────────────────────

export default function V2DelegationTree({
  triggers,
  agents,
  relations,
  entryAgentId,
  highlightAgentIds,
  highlightTriggerIds,
  onAgentOpen,
  onTriggerOpen,
  onAddChild,
  onRemoveDelegation,
}: V2DelegationTreeProps) {
  const tree = useMemo(
    () => buildTree(agents, relations, entryAgentId),
    [agents, relations, entryAgentId],
  );

  if (!tree) {
    return (
      <div className="h-full flex items-center justify-center text-[12px] text-brand-shade3">
        Entry agent not found in this schema.
      </div>
    );
  }

  return (
    <div className="h-full w-full overflow-auto">
      <style>{TREE_CSS}</style>
      <div className="min-w-max mx-auto flex flex-col items-center py-10 px-8">
        {/* Triggers row */}
        {triggers.length > 0 && (
          <>
            <ul className="v2tree-triggers">
              {triggers.map((t) => (
                <li key={t.id} className="v2tree-trigger-node">
                  <TriggerCard
                    trigger={t}
                    highlighted={highlightTriggerIds?.has(t.id) ?? false}
                    onClick={() => onTriggerOpen?.(t.id)}
                  />
                </li>
              ))}
            </ul>
            {/* Stub from triggers bus to entry card */}
            <div className="v2tree-triggers-to-entry-stub" />
          </>
        )}

        {/* Tree */}
        <ul className="v2tree-root">
          <TreeBranch
            node={tree}
            entryId={entryAgentId}
            highlightAgentIds={highlightAgentIds}
            onAgentOpen={onAgentOpen}
            onAddChild={onAddChild}
            onRemoveDelegation={onRemoveDelegation}
          />
        </ul>
      </div>
    </div>
  );
}

// ─── CSS (scoped org-chart connectors) ──────────────────────────────────────
//
// Layout contract:
//   - Each node column contains the card, then optionally: down-stub, children ul.
//   - Children ul is a flex row. Each child li has:
//       - top:0 horizontal bus segment (clipped at center for first/last).
//       - top:0 vertical stub extending STUB_PX downward from bus to card.
//     The child card sits at top: STUB_PX inside the li.
//   - Parent down-stub is a separate <div> drawn between parent card and children ul.
//   - Single-child case hides the bus (display:none on :only-child::before)
//     and the result is a continuous 2×STUB_PX vertical line.

const TREE_CSS = `
.v2tree-node,
.v2tree-trigger-node { --v2line: ${LINE_COLOR}; }

/* Root and children containers */
.v2tree-root,
.v2tree-children {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  justify-content: center;
  align-items: flex-start;
  gap: 0;
}

.v2tree-node {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-left: ${SIBLING_PAD_PX}px;
  padding-right: ${SIBLING_PAD_PX}px;
  position: relative;
}

/* Parent down-stub: solid vertical between parent card and children ul.
   Use 1px-wide background div (not border on 0-width) for pixel-perfect
   alignment with the children bus and up-stubs. */
.v2tree-parent-stub {
  width: 1px;
  height: ${STUB_PX}px;
  background-color: ${LINE_COLOR};
}

/* Children row: no padding; each child reserves its own top space */
.v2tree-children {
  position: relative;
}

/* Each child node: padding-top reserves STUB_PX for up-stub (card pushed down) */
.v2tree-children > .v2tree-node {
  padding-top: ${STUB_PX}px;
}

/* Horizontal bus at TOP of child node (y=0 of node, above the card).
   1px-tall background div (not border-top) so sub-pixel snapping is
   consistent across segments. Clipped at center for first/last child. */
.v2tree-children > .v2tree-node::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 1px;
  background-color: ${LINE_COLOR};
}
.v2tree-children > .v2tree-node:first-child::before { left: calc(50% - 0.5px); }
.v2tree-children > .v2tree-node:last-child::before  { right: calc(50% - 0.5px); }
.v2tree-children > .v2tree-node:only-child::before  { display: none; }

/* Up-stub from bus (y=0) down to card top (y=STUB_PX). 1px-wide div centered
   at 50% minus half-pixel — pixel-perfect with parent-stub and bus. */
.v2tree-children > .v2tree-node::after {
  content: '';
  position: absolute;
  top: 0;
  left: calc(50% - 0.5px);
  width: 1px;
  height: ${STUB_PX}px;
  background-color: ${LINE_COLOR};
}

/* ─── Triggers row (mirrored layout) ────────────────────────────────────── */

.v2tree-triggers {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  justify-content: center;
  align-items: flex-end;
  gap: 0;
}

.v2tree-trigger-node {
  position: relative;
  padding: 0 ${SIBLING_PAD_PX}px ${STUB_PX}px ${SIBLING_PAD_PX}px;
}

/* Bus at bottom of each trigger column */
.v2tree-trigger-node::before {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 1px;
  background-color: ${LINE_COLOR};
}
.v2tree-triggers > .v2tree-trigger-node:first-child::before { left: calc(50% - 0.5px); }
.v2tree-triggers > .v2tree-trigger-node:last-child::before  { right: calc(50% - 0.5px); }
.v2tree-triggers > .v2tree-trigger-node:only-child::before  { display: none; }

/* Down-stub from trigger card bottom to bus */
.v2tree-trigger-node::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: calc(50% - 0.5px);
  width: 1px;
  height: ${STUB_PX}px;
  background-color: ${LINE_COLOR};
}

/* Continuous vertical from triggers bus to entry card */
.v2tree-triggers-to-entry-stub {
  width: 1px;
  height: ${STUB_PX}px;
  background-color: ${LINE_COLOR};
}
`;
