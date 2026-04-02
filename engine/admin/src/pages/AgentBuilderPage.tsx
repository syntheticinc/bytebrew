import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  useNodesState,
  useEdgesState,
  addEdge,
  MarkerType,
  type Node,
  type Edge,
  type Connection,
  type NodeMouseHandler,
  BackgroundVariant,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import dagre from '@dagrejs/dagre';
import { api } from '../api/client';
import type { AgentDetail, Model, CreateAgentRequest } from '../types';
import AgentNode, { type AgentNodeData } from '../components/builder/AgentNode';
import BuilderSidePanel from '../components/builder/BuilderSidePanel';
import ConfirmDialog from '../components/ConfirmDialog';

// ─── Constants ────────────────────────────────────────────────────────────────

const nodeTypes = { agentNode: AgentNode };
const NODE_WIDTH = 210;
const NODE_HEIGHT = 135;
const POSITIONS_KEY = 'bytebrew_builder_positions';

// ─── Layout helpers ───────────────────────────────────────────────────────────

function loadPositions(): Record<string, { x: number; y: number }> {
  try {
    return JSON.parse(localStorage.getItem(POSITIONS_KEY) ?? '{}') as Record<string, { x: number; y: number }>;
  } catch {
    return {};
  }
}

function savePositions(nodes: Node[]) {
  const pos: Record<string, { x: number; y: number }> = {};
  nodes.forEach((n) => { pos[n.id] = n.position; });
  localStorage.setItem(POSITIONS_KEY, JSON.stringify(pos));
}

function applyDagre(nodes: Node[], edges: Edge[]): Node[] {
  if (nodes.length === 0) return nodes;
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir: 'TB', nodesep: 60, ranksep: 90 });
  nodes.forEach((n) => g.setNode(n.id, { width: NODE_WIDTH, height: NODE_HEIGHT }));
  edges.forEach((e) => g.setEdge(e.source, e.target));
  dagre.layout(g);
  return nodes.map((n) => {
    const { x, y } = g.node(n.id);
    return { ...n, position: { x: x - NODE_WIDTH / 2, y: y - NODE_HEIGHT / 2 } };
  });
}

// ─── Edge factory ─────────────────────────────────────────────────────────────

function makeEdge(source: string, target: string): Edge {
  return {
    id: `${source}->${target}`,
    source,
    target,
    type: 'smoothstep',
    markerEnd: { type: MarkerType.ArrowClosed, color: '#D7513E' },
    style: { stroke: '#D7513E', strokeWidth: 1.5, opacity: 0.7 },
    label: 'spawns',
    labelStyle: { fill: '#87867F', fontSize: 10 },
    labelBgStyle: { fill: '#1F1F1F', fillOpacity: 0.9 },
  };
}

// ─── Node factory ─────────────────────────────────────────────────────────────

function makeNode(
  agent: AgentDetail,
  modelMap: Map<number, string>,
  position: { x: number; y: number },
  onSelect: (name: string) => void,
  onDelete: (name: string) => void,
): Node {
  return {
    id: agent.name,
    type: 'agentNode',
    position,
    data: {
      name: agent.name,
      modelName: agent.model_id != null ? (modelMap.get(agent.model_id) ?? '') : '',
      toolsCount: agent.tools?.length ?? 0,
      spawnCount: agent.can_spawn?.length ?? 0,
      confirmCount: agent.confirm_before?.length ?? 0,
      lifecycle: agent.lifecycle,
      onSelect,
      onDelete,
    } satisfies AgentNodeData,
  };
}

// ─── Default new-agent form ────────────────────────────────────────────────────

const DEFAULT_NEW_FORM: Partial<CreateAgentRequest> = {
  name: '',
  system_prompt: '',
  lifecycle: 'persistent',
  tool_execution: 'sequential',
  max_steps: 50,
  max_context_size: 16000,
  tools: [],
  can_spawn: [],
  mcp_servers: [],
  confirm_before: [],
};

// ─── Component ────────────────────────────────────────────────────────────────

export default function AgentBuilderPage() {
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const [selectedAgent, setSelectedAgent] = useState<AgentDetail | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showNewForm, setShowNewForm] = useState(false);
  const [newForm, setNewForm] = useState<Partial<CreateAgentRequest>>(DEFAULT_NEW_FORM);
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState('');

  const agentsCache = useRef<Map<string, AgentDetail>>(new Map());
  const modelsRef = useRef<Model[]>([]);

  // Stable callbacks — never change identity, so nodes don't re-render on them
  const handleSelect = useCallback(async (name: string) => {
    let agent = agentsCache.current.get(name);
    if (!agent) {
      try {
        agent = await api.getAgent(name);
        agentsCache.current.set(name, agent);
      } catch {
        return;
      }
    }
    setSelectedAgent(agent);
  }, []);

  const handleDeleteRequest = useCallback((name: string) => {
    setDeleteTarget(name);
  }, []);

  // ── Initial load ────────────────────────────────────────────────────────────

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError('');

    Promise.all([api.listAgents(), api.listModels()])
      .then(async ([agentList, models]) => {
        if (cancelled) return;
        modelsRef.current = models;
        const modelMap = new Map(models.map((m) => [m.id, m.name]));

        // Fetch full details for all agents (need can_spawn for edges)
        const details = await Promise.all(agentList.map((a) => api.getAgent(a.name)));
        if (cancelled) return;

        const agentNames = new Set(details.map((a) => a.name));
        details.forEach((a) => agentsCache.current.set(a.name, a));

        const savedPositions = loadPositions();
        const hasPositions = details.every((a) => savedPositions[a.name] !== undefined);

        let rawNodes = details.map((agent, i) =>
          makeNode(
            agent,
            modelMap,
            savedPositions[agent.name] ?? { x: i * 250, y: 0 },
            handleSelect,
            handleDeleteRequest,
          ),
        );

        const rawEdges: Edge[] = [];
        for (const agent of details) {
          for (const target of agent.can_spawn ?? []) {
            if (agentNames.has(target)) {
              rawEdges.push(makeEdge(agent.name, target));
            }
          }
        }

        if (!hasPositions) {
          rawNodes = applyDagre(rawNodes, rawEdges);
          savePositions(rawNodes);
        }

        setNodes(rawNodes);
        setEdges(rawEdges);
        setLoading(false);
      })
      .catch((err: Error) => {
        if (!cancelled) {
          setError(err.message);
          setLoading(false);
        }
      });

    return () => { cancelled = true; };
  }, [handleSelect, handleDeleteRequest]);

  // ── Connect handler (draw edge = add can_spawn) ──────────────────────────────

  const onConnect = useCallback(
    async (connection: Connection) => {
      const { source, target } = connection;
      if (!source || !target || source === target) return;

      const sourceAgent = agentsCache.current.get(source);
      if (!sourceAgent) return;

      const currentSpawn = sourceAgent.can_spawn ?? [];
      if (currentSpawn.includes(target)) return;

      try {
        const updated = await api.updateAgent(source, { can_spawn: [...currentSpawn, target] });
        agentsCache.current.set(source, updated);

        setNodes((nds) =>
          nds.map((n) => {
            if (n.id !== source) return n;
            const d = n.data as AgentNodeData;
            return { ...n, data: { ...d, spawnCount: updated.can_spawn.length } };
          }),
        );

        setEdges((eds) => addEdge(makeEdge(source, target), eds));

        // Refresh side panel if this agent is open
        if (selectedAgent?.name === source) setSelectedAgent(updated);
      } catch (err) {
        console.error('Failed to update can_spawn:', err);
      }
    },
    [selectedAgent, setNodes, setEdges],
  );

  // ── Delete edge = remove can_spawn ──────────────────────────────────────────

  const onEdgesDelete = useCallback(
    async (deletedEdges: Edge[]) => {
      for (const edge of deletedEdges) {
        const sourceAgent = agentsCache.current.get(edge.source);
        if (!sourceAgent) continue;
        const updatedSpawn = (sourceAgent.can_spawn ?? []).filter((a) => a !== edge.target);
        try {
          const updated = await api.updateAgent(edge.source, { can_spawn: updatedSpawn });
          agentsCache.current.set(edge.source, updated);

          setNodes((nds) =>
            nds.map((n) => {
              if (n.id !== edge.source) return n;
              const d = n.data as AgentNodeData;
              return { ...n, data: { ...d, spawnCount: updated.can_spawn.length } };
            }),
          );

          if (selectedAgent?.name === edge.source) setSelectedAgent(updated);
        } catch (err) {
          console.error('Failed to remove can_spawn:', err);
        }
      }
    },
    [selectedAgent, setNodes],
  );

  // ── Save positions on drag stop ─────────────────────────────────────────────

  const onNodeDragStop = useCallback(
    (_event: React.MouseEvent, _node: Node, currentNodes: Node[]) => {
      savePositions(currentNodes);
    },
    [],
  );

  // ── Auto-layout ─────────────────────────────────────────────────────────────

  function runAutoLayout() {
    // nodes/edges captured from current render — always fresh since this is
    // a plain function (not a memoized callback) called directly from a button.
    const laid = applyDagre([...nodes], [...edges]);
    savePositions(laid);
    setNodes(laid);
  }

  // ── Delete agent ────────────────────────────────────────────────────────────

  async function confirmDelete() {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await api.deleteAgent(deleteTarget);
      agentsCache.current.delete(deleteTarget);

      setNodes((nds) => nds.filter((n) => n.id !== deleteTarget));
      setEdges((eds) => eds.filter((e) => e.source !== deleteTarget && e.target !== deleteTarget));

      if (selectedAgent?.name === deleteTarget) setSelectedAgent(null);
      setDeleteTarget(null);
    } catch (err) {
      console.error('Delete failed:', err);
    } finally {
      setDeleting(false);
    }
  }

  // ── Create new agent ────────────────────────────────────────────────────────

  async function handleCreate() {
    if (!newForm.name || !newForm.system_prompt) return;
    setCreateError('');
    setCreating(true);
    try {
      const created = await api.createAgent(newForm as CreateAgentRequest);
      agentsCache.current.set(created.name, created);

      const modelMap = new Map(modelsRef.current.map((m) => [m.id, m.name]));
      const pos = { x: Math.random() * 400 + 100, y: Math.random() * 200 + 50 };
      const newNode = makeNode(created, modelMap, pos, handleSelect, handleDeleteRequest);

      setNodes((nds) => [...nds, newNode]);
      setShowNewForm(false);
      setNewForm(DEFAULT_NEW_FORM);
      setSelectedAgent(created);
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : 'Create failed');
    } finally {
      setCreating(false);
    }
  }

  // ── Side panel save callback ────────────────────────────────────────────────

  function handleSaved(updated: AgentDetail) {
    agentsCache.current.set(updated.name, updated);
    setSelectedAgent(updated);

    const modelMap = new Map(modelsRef.current.map((m) => [m.id, m.name]));
    setNodes((nds) =>
      nds.map((n) => {
        if (n.id !== updated.name) return n;
        const d = n.data as AgentNodeData;
        return {
          ...n,
          data: {
            ...d,
            modelName: updated.model_id != null ? (modelMap.get(updated.model_id) ?? '') : '',
            toolsCount: updated.tools?.length ?? 0,
            spawnCount: updated.can_spawn?.length ?? 0,
            confirmCount: updated.confirm_before?.length ?? 0,
          },
        };
      }),
    );
  }

  // ── Prevent node click selecting when clicking node buttons ────────────────

  const onNodeClick: NodeMouseHandler = useCallback((_event, node) => {
    handleSelect(node.id);
  }, [handleSelect]);

  // ── Render ──────────────────────────────────────────────────────────────────

  return (
    <div
      className="flex flex-col overflow-hidden"
      style={{ margin: '-24px', height: '100vh' }}
    >
      {/* Toolbar */}
      <div className="flex items-center gap-3 px-4 h-12 border-b border-brand-shade3/15 bg-brand-dark-alt flex-shrink-0">
        <span className="text-sm font-semibold text-brand-light">Agent Builder</span>
        <div className="flex-1" />
        <button
          onClick={runAutoLayout}
          className="px-3 py-1.5 text-xs text-brand-shade2 border border-brand-shade3/30 rounded-btn hover:text-brand-light hover:border-brand-shade3 transition-colors"
        >
          Auto Layout
        </button>
        <button
          onClick={() => { setShowNewForm(true); setCreateError(''); }}
          className="px-3 py-1.5 text-xs bg-brand-accent text-brand-light rounded-btn hover:bg-brand-accent-hover transition-colors"
        >
          + Add Agent
        </button>
      </div>

      {/* Canvas + Side Panel */}
      <div className="flex flex-1 min-h-0">
        <div className="flex-1 relative">
          {loading && (
            <div className="absolute inset-0 flex items-center justify-center bg-brand-dark z-10">
              <span className="text-brand-shade3 text-sm">Loading agents…</span>
            </div>
          )}
          {error && (
            <div className="absolute inset-0 flex items-center justify-center bg-brand-dark z-10">
              <span className="text-red-400 text-sm">Error: {error}</span>
            </div>
          )}

          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onEdgesDelete={onEdgesDelete}
            onNodeClick={onNodeClick}
            onNodeDragStop={onNodeDragStop}
            nodeTypes={nodeTypes}
            fitView
            fitViewOptions={{ padding: 0.2 }}
            deleteKeyCode="Delete"
            style={{ background: '#111111' }}
            proOptions={{ hideAttribution: true }}
          >
            <Background
              variant={BackgroundVariant.Dots}
              gap={20}
              size={1}
              color="#2a2a2a"
            />
            <Controls
              style={{
                background: '#1F1F1F',
                border: '1px solid rgba(135,134,127,0.2)',
                borderRadius: '2px',
              }}
            />
          </ReactFlow>

          {nodes.length === 0 && !loading && !error && (
            <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
              <p className="text-brand-shade3 text-sm mb-2">No agents yet</p>
              <p className="text-brand-shade3/50 text-xs">Click "+ Add Agent" to create your first agent</p>
            </div>
          )}
        </div>

        {selectedAgent && (
          <BuilderSidePanel
            agent={selectedAgent}
            onClose={() => setSelectedAgent(null)}
            onSaved={handleSaved}
            onDelete={handleDeleteRequest}
          />
        )}
      </div>

      {/* Hint bar */}
      <div className="flex items-center px-4 h-7 border-t border-brand-shade3/10 bg-brand-dark flex-shrink-0">
        <span className="text-[10px] text-brand-shade3/50">
          Drag between handles to connect agents (can_spawn) · Click edge + Delete to remove · Drag nodes to reposition
        </span>
      </div>

      {/* New agent dialog */}
      {showNewForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/60" onClick={() => setShowNewForm(false)} />
          <div className="relative bg-brand-dark-alt border border-brand-shade3/20 rounded-card shadow-xl w-[440px] animate-modal-in">
            <div className="px-5 py-4 border-b border-brand-shade3/15 flex items-center justify-between">
              <h3 className="text-sm font-semibold text-brand-light">New Agent</h3>
              <button onClick={() => setShowNewForm(false)} className="text-brand-shade3 hover:text-brand-light p-1">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            <div className="p-5 space-y-4">
              <div>
                <label className="block text-xs font-medium text-brand-shade3 mb-1 uppercase tracking-wide">Name *</label>
                <input
                  type="text"
                  value={newForm.name ?? ''}
                  onChange={(e) => setNewForm((p) => ({ ...p, name: e.target.value }))}
                  pattern="^[a-z][a-z0-9-]*$"
                  placeholder="my-agent"
                  className="w-full px-3 py-2 bg-brand-dark border border-brand-shade3/30 rounded-card text-sm text-brand-light placeholder-brand-shade3 focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent transition-colors"
                />
                <p className="text-[10px] text-brand-shade3/60 mt-0.5">Lowercase letters, numbers, hyphens. Must start with a letter.</p>
              </div>

              <div>
                <label className="block text-xs font-medium text-brand-shade3 mb-1 uppercase tracking-wide">Model</label>
                <select
                  value={newForm.model_id ?? ''}
                  onChange={(e) => setNewForm((p) => ({ ...p, model_id: e.target.value ? Number(e.target.value) : undefined }))}
                  className="w-full px-3 py-2 bg-brand-dark border border-brand-shade3/30 rounded-card text-sm text-brand-light focus:outline-none focus:border-brand-accent transition-colors"
                >
                  <option value="">Default model</option>
                  {modelsRef.current.map((m) => (
                    <option key={m.id} value={m.id}>{m.name}</option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-xs font-medium text-brand-shade3 mb-1 uppercase tracking-wide">System Prompt *</label>
                <textarea
                  value={newForm.system_prompt ?? ''}
                  onChange={(e) => setNewForm((p) => ({ ...p, system_prompt: e.target.value }))}
                  rows={5}
                  placeholder="You are a helpful assistant."
                  className="w-full px-3 py-2 bg-brand-dark border border-brand-shade3/30 rounded-card text-sm text-brand-light placeholder-brand-shade3 font-mono focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent transition-colors resize-none"
                />
              </div>

              {createError && (
                <p className="text-xs text-red-400">{createError}</p>
              )}
            </div>
            <div className="px-5 py-4 border-t border-brand-shade3/15 flex gap-3">
              <button
                onClick={handleCreate}
                disabled={creating || !newForm.name || !newForm.system_prompt}
                className="flex-1 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover disabled:opacity-40 transition-colors"
              >
                {creating ? 'Creating…' : 'Create Agent'}
              </button>
              <button
                onClick={() => setShowNewForm(false)}
                className="px-4 py-2 border border-brand-shade3/30 text-brand-shade2 rounded-btn text-sm hover:text-brand-light transition-colors"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete confirm */}
      <ConfirmDialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={confirmDelete}
        title="Delete Agent"
        message={
          <>
            Delete agent <strong className="text-brand-light">{deleteTarget}</strong>?
            This will also remove all spawn connections to/from it.
          </>
        }
        confirmLabel="Delete"
        loading={deleting}
        variant="danger"
      />
    </div>
  );
}
