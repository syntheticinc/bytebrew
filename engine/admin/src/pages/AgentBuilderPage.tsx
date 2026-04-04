import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  ReactFlow,
  Background,
  Controls,

  useNodesState,
  useEdgesState,
  addEdge,
  MarkerType,
  useReactFlow,
  ReactFlowProvider,
  type Node,
  type Edge,
  type Connection,
  type NodeMouseHandler,
  type IsValidConnection,
  BackgroundVariant,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import dagre from '@dagrejs/dagre';
import { api } from '../api/client';
import type { AgentDetail, Model, CreateAgentRequest, CreateTriggerRequest, Trigger } from '../types';
import AgentNode, { type AgentNodeData } from '../components/builder/AgentNode';
import TriggerNode from '../components/builder/TriggerNode';
import BuilderSidePanel from '../components/builder/BuilderSidePanel';
import { ExportButton, ImportButton } from '../components/builder/BuilderExportImport';
import BuilderFlowTest from '../components/builder/BuilderFlowTest';
import BuilderAssistant, { AssistantToggleButton } from '../components/builder/BuilderAssistant';
import DriftNotification from '../components/builder/DriftNotification';
import ConfirmDialog from '../components/ConfirmDialog';
import { ToastProvider, useToast } from '../components/builder/Toast';

// ─── Constants ────────────────────────────────────────────────────────────────

const nodeTypes = { agentNode: AgentNode, triggerNode: TriggerNode };
const NODE_WIDTH = 210;
const NODE_HEIGHT = 135;
const TRIGGER_NODE_WIDTH = 190;
const TRIGGER_NODE_HEIGHT = 95;
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

function mergePositions(changed: Node[]) {
  const existing = loadPositions();
  changed.forEach((n) => { existing[n.id] = n.position; });
  localStorage.setItem(POSITIONS_KEY, JSON.stringify(existing));
}

function applyDagre(nodes: Node[], edges: Edge[]): Node[] {
  if (nodes.length === 0) return nodes;
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir: 'TB', nodesep: 60, ranksep: 90 });
  nodes.forEach((n) => {
    const isTrigger = n.type === 'triggerNode';
    const w = isTrigger ? TRIGGER_NODE_WIDTH : NODE_WIDTH;
    const h = isTrigger ? TRIGGER_NODE_HEIGHT : NODE_HEIGHT;
    g.setNode(n.id, { width: w, height: h });
  });
  edges.forEach((e) => g.setEdge(e.source, e.target));
  dagre.layout(g);
  return nodes.map((n) => {
    const isTrigger = n.type === 'triggerNode';
    const w = isTrigger ? TRIGGER_NODE_WIDTH : NODE_WIDTH;
    const h = isTrigger ? TRIGGER_NODE_HEIGHT : NODE_HEIGHT;
    const { x, y } = g.node(n.id);
    return { ...n, position: { x: x - w / 2, y: y - h / 2 } };
  });
}

// ─── Cron validation ──────────────────────────────────────────────────────────

function isValidCron(expr: string): boolean {
  const parts = expr.trim().split(/\s+/);
  if (parts.length !== 5) return false;
  const ranges: [number, number][] = [
    [0, 59],  // minute
    [0, 23],  // hour
    [1, 31],  // day
    [1, 12],  // month
    [0, 7],   // weekday
  ];
  return parts.every((part, i) => {
    if (part === '*') return true;
    const range = ranges[i];
    if (!range) return false;
    const [min, max] = range;
    const num = Number(part);
    return Number.isInteger(num) && num >= min && num <= max;
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
    style: { stroke: '#D7513E', strokeWidth: 1.5, opacity: 0.85 },
    label: 'spawns',
    labelStyle: { fill: '#CBC9BC', fontSize: 11 },
    labelBgStyle: { fill: '#1F1F1F', fillOpacity: 0.9 },
  };
}

function makeTriggerEdge(triggerNodeId: string, agentName: string): Edge {
  return {
    id: `trigger:${triggerNodeId}->${agentName}`,
    source: triggerNodeId,
    target: agentName,
    type: 'smoothstep',
    markerEnd: { type: MarkerType.ArrowClosed, color: '#A855F7' },
    style: { stroke: '#A855F7', strokeWidth: 1.5, strokeDasharray: '6 3', opacity: 0.85 },
    label: 'triggers',
    labelStyle: { fill: '#CBC9BC', fontSize: 11 },
    labelBgStyle: { fill: '#1F1F1F', fillOpacity: 0.9 },
  };
}

function makeTriggerNode(
  trigger: Trigger,
  position: { x: number; y: number },
): Node {
  const nodeId = `trigger-${trigger.id}`;
  return {
    id: nodeId,
    type: 'triggerNode',
    position,
    data: {
      id: trigger.id,
      title: trigger.title,
      type: trigger.type,
      schedule: trigger.schedule,
      webhook_path: trigger.webhook_path,
      enabled: trigger.enabled,
      agentName: trigger.agent_name,
    },
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

const DEFAULT_TRIGGER_FORM: Partial<CreateTriggerRequest> = {
  type: 'cron',
  title: '',
  enabled: true,
};

// ─── Cron presets ─────────────────────────────────────────────────────────────

const CRON_PRESETS = [
  { label: 'Every hour', value: '0 * * * *' },
  { label: 'Daily at 9:00', value: '0 9 * * *' },
  { label: 'Weekly Monday', value: '0 9 * * 1' },
];

// ─── Inner component (needs ReactFlow context for useReactFlow) ───────────────

function AgentBuilderInner() {
  const { fitView } = useReactFlow();

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const [agents, setAgents] = useState<AgentDetail[]>([]);
  const [selectedAgent, setSelectedAgent] = useState<AgentDetail | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [deleteError, setDeleteError] = useState('');
  const [deleting, setDeleting] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showNewForm, setShowNewForm] = useState(false);
  const [newForm, setNewForm] = useState<Partial<CreateAgentRequest>>(DEFAULT_NEW_FORM);
  const [nameError, setNameError] = useState('');
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState('');
  const [showFlowTest, setShowFlowTest] = useState(false);
  const [showAssistant, setShowAssistant] = useState(false);
  const [refreshKey, setRefreshKey] = useState(0);
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number; canvasX: number; canvasY: number } | null>(null);
  const [nodeMenu, setNodeMenu] = useState<{ x: number; y: number; nodeId: string; nodeType: string } | null>(null);
  const [edgeMenu, setEdgeMenu] = useState<{ x: number; y: number; edgeId: string; source: string; target: string } | null>(null);
  const [showTriggerForm, setShowTriggerForm] = useState(false);
  const [triggerForm, setTriggerForm] = useState<Partial<CreateTriggerRequest>>(DEFAULT_TRIGGER_FORM);
  const [triggerCreating, setTriggerCreating] = useState(false);
  const [triggerError, setTriggerError] = useState('');
  const [cronError, setCronError] = useState('');
  const [savedIndicator, setSavedIndicator] = useState<'saved' | 'saving' | null>(null);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

  const { addToast } = useToast();

  const agentsCache = useRef<Map<string, AgentDetail>>(new Map());
  const modelsRef = useRef<Model[]>([]);
  const agentsListRef = useRef<AgentDetail[]>([]);
  const savedIndicatorTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const refetchCanvas = useCallback(() => setRefreshKey((k) => k + 1), []);

  function showSavedIndicator(state: 'saved' | 'saving') {
    setSavedIndicator(state);
    if (savedIndicatorTimer.current) clearTimeout(savedIndicatorTimer.current);
    if (state === 'saved') {
      savedIndicatorTimer.current = setTimeout(() => setSavedIndicator(null), 2500);
    }
  }

  // Close side panel on Escape
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        setSelectedAgent(null);
        setContextMenu(null);
        setNodeMenu(null);
        setEdgeMenu(null);
        setShowNewForm(false);
        setShowTriggerForm(false);
      }
    }
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, []);

  // Cleanup saved indicator timer on unmount
  useEffect(() => {
    return () => {
      if (savedIndicatorTimer.current) clearTimeout(savedIndicatorTimer.current);
    };
  }, []);

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
    setDeleteError('');
  }, []);

  // ── Initial load ────────────────────────────────────────────────────────────

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError('');

    Promise.all([api.listAgents(), api.listModels(), api.listTriggers().catch(() => [] as Trigger[])])
      .then(async ([agentList, models, triggers]) => {
        if (cancelled) return;
        modelsRef.current = models;
        const modelMap = new Map(models.map((m) => [m.id, m.name]));

        // Fetch full details for all agents (need can_spawn for edges)
        const details = await Promise.all(agentList.map((a) => api.getAgent(a.name)));
        if (cancelled) return;

        const agentNames = new Set(details.map((a) => a.name));
        details.forEach((a) => agentsCache.current.set(a.name, a));
        agentsListRef.current = details;
        setAgents(details);

        const savedPositions = loadPositions();

        // Build agent nodes
        let rawNodes: Node[] = details.map((agent, i) =>
          makeNode(
            agent,
            modelMap,
            savedPositions[agent.name] ?? { x: i * 250, y: 0 },
            handleSelect,
            handleDeleteRequest,
          ),
        );

        // Build spawn edges
        const rawEdges: Edge[] = [];
        for (const agent of details) {
          for (const target of agent.can_spawn ?? []) {
            if (agentNames.has(target)) {
              rawEdges.push(makeEdge(agent.name, target));
            }
          }
        }

        // Build trigger nodes + edges
        for (const trigger of triggers) {
          const nodeId = `trigger-${trigger.id}`;
          const triggerNode = makeTriggerNode(
            trigger,
            savedPositions[nodeId] ?? { x: 0, y: 0 },
          );
          rawNodes.push(triggerNode);

          if (trigger.agent_name && agentNames.has(trigger.agent_name)) {
            rawEdges.push(makeTriggerEdge(nodeId, trigger.agent_name));
          }
        }

        // Apply dagre only to nodes without saved positions (new nodes).
        // Nodes with saved positions keep their user-arranged locations.
        const newNodes = rawNodes.filter((n) => savedPositions[n.id] === undefined);
        if (newNodes.length > 0) {
          const laid = applyDagre([...rawNodes], rawEdges);
          const laidMap = new Map(laid.map((n) => [n.id, n.position]));
          rawNodes = rawNodes.map((n) =>
            savedPositions[n.id] !== undefined ? n : { ...n, position: laidMap.get(n.id) ?? n.position },
          );
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
  }, [handleSelect, handleDeleteRequest, refreshKey]);

  // ── Connect handler (draw edge = add can_spawn) ──────────────────────────────

  const onConnect = useCallback(
    async (connection: Connection) => {
      const { source, target } = connection;
      if (!source || !target) return;

      // Self-connection guard
      if (source === target) {
        addToast('Cannot connect an agent to itself', 'warning');
        return;
      }

      // Trigger nodes are read-only — cannot create edges from them manually
      if (source.startsWith('trigger-')) {
        addToast('Trigger connections are managed on the Triggers page', 'info');
        return;
      }

      const sourceAgent = agentsCache.current.get(source);
      if (!sourceAgent) return;

      const currentSpawn = sourceAgent.can_spawn ?? [];
      if (currentSpawn.includes(target)) {
        addToast('This connection already exists', 'warning');
        return;
      }

      showSavedIndicator('saving');
      try {
        const updated = await api.updateAgent(source, { can_spawn: [...currentSpawn, target] });
        agentsCache.current.set(source, updated);

        setNodes((nds) =>
          nds.map((n) => {
            if (n.id !== source) return n;
            const d = n.data as AgentNodeData;
            return { ...n, data: { ...d, spawnCount: (updated.can_spawn ?? []).length } };
          }),
        );

        setEdges((eds) => addEdge(makeEdge(source, target), eds));

        // Refresh side panel if this agent is open
        if (selectedAgent?.name === source) setSelectedAgent(updated);
        showSavedIndicator('saved');
      } catch (err) {
        addToast(`Failed to connect agents: ${err instanceof Error ? err.message : 'Unknown error'}`, 'error');
        setSavedIndicator(null);
      }
    },
    [selectedAgent, setNodes, setEdges, addToast],
  );

  // ── isValidConnection — prevent self-loops and trigger→agent drags ──────────

  const isValidConnection = useCallback<IsValidConnection>((connection) => {
    if (!connection.source || !connection.target) return false;
    if (connection.source === connection.target) return false;
    return true;
  }, []);

  // ── Delete edge = remove can_spawn ──────────────────────────────────────────

  const onEdgesDelete = useCallback(
    async (deletedEdges: Edge[]) => {
      for (const edge of deletedEdges) {
        // Trigger edges are read-only — managed via Triggers page
        if (edge.id.startsWith('trigger:')) continue;

        const sourceAgent = agentsCache.current.get(edge.source);
        if (!sourceAgent) continue;
        const updatedSpawn = (sourceAgent.can_spawn ?? []).filter((a) => a !== edge.target);

        // Optimistic: edges already removed by ReactFlow — restore on failure
        showSavedIndicator('saving');
        try {
          const updated = await api.updateAgent(edge.source, { can_spawn: updatedSpawn });
          agentsCache.current.set(edge.source, updated);

          setNodes((nds) =>
            nds.map((n) => {
              if (n.id !== edge.source) return n;
              const d = n.data as AgentNodeData;
              return { ...n, data: { ...d, spawnCount: (updated.can_spawn ?? []).length } };
            }),
          );

          if (selectedAgent?.name === edge.source) setSelectedAgent(updated);
          showSavedIndicator('saved');
        } catch (err) {
          // Restore the deleted edge on failure
          setEdges((eds) => [...eds, edge]);
          addToast(`Failed to remove connection: ${err instanceof Error ? err.message : 'Unknown error'}`, 'error');
          setSavedIndicator(null);
        }
      }
    },
    [selectedAgent, setNodes, setEdges, addToast],
  );

  // ── Save positions on drag stop ─────────────────────────────────────────────

  const onNodeDragStop = useCallback(
    (_event: React.MouseEvent, node: Node) => {
      mergePositions([node]);
    },
    [],
  );

  // ── Auto-layout ─────────────────────────────────────────────────────────────

  function runAutoLayout() {
    const laid = applyDagre([...nodes], [...edges]);
    savePositions(laid);
    setNodes(laid);
    // fitView after layout so all nodes are visible
    setTimeout(() => fitView({ padding: 0.2 }), 50);
  }

  // ── Delete agent ────────────────────────────────────────────────────────────

  async function confirmDelete() {
    if (!deleteTarget) return;
    setDeleting(true);
    setDeleteError('');
    try {
      await api.deleteAgent(deleteTarget);
      agentsCache.current.delete(deleteTarget);

      setNodes((nds) => nds.filter((n) => n.id !== deleteTarget));
      setEdges((eds) => eds.filter((e) => e.source !== deleteTarget && e.target !== deleteTarget));

      if (selectedAgent?.name === deleteTarget) setSelectedAgent(null);
      setDeleteTarget(null);
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Delete failed');
    } finally {
      setDeleting(false);
    }
  }

  // ── Create new agent ────────────────────────────────────────────────────────

  async function handleCreate() {
    const trimmedName = (newForm.name ?? '').trim();
    if (!trimmedName || !newForm.system_prompt) return;
    if (nameError) return;
    setCreateError('');
    setCreating(true);
    try {
      const created = await api.createAgent({ ...newForm, name: trimmedName } as CreateAgentRequest);
      agentsCache.current.set(created.name, created);
      agentsListRef.current = [...agentsListRef.current, created];
      setAgents((prev) => [...prev, created]);

      const modelMap = new Map(modelsRef.current.map((m) => [m.id, m.name]));
      const pos = { x: Math.random() * 400 + 100, y: Math.random() * 200 + 50 };
      const newNode = makeNode(created, modelMap, pos, handleSelect, handleDeleteRequest);

      setNodes((nds) => [...nds, newNode]);
      setShowNewForm(false);
      setNewForm(DEFAULT_NEW_FORM);
      setNameError('');
      setSelectedAgent(created);
      addToast(`Agent "${created.name}" created`, 'success');
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : 'Create failed');
    } finally {
      setCreating(false);
    }
  }

  // ── Create new trigger ─────────────────────────────────────────────────────

  async function handleCreateTrigger() {
    if (!triggerForm.title || !triggerForm.agent_name) return;
    if (triggerForm.type === 'cron' && triggerForm.schedule && !isValidCron(triggerForm.schedule)) {
      setCronError('Invalid cron expression');
      return;
    }
    setTriggerError('');
    setCronError('');
    setTriggerCreating(true);
    try {
      await api.createTrigger(triggerForm as CreateTriggerRequest);
      setShowTriggerForm(false);
      setTriggerForm(DEFAULT_TRIGGER_FORM);
      refetchCanvas();
      addToast('Trigger created', 'success');
    } catch (err) {
      setTriggerError(err instanceof Error ? err.message : 'Create failed');
    } finally {
      setTriggerCreating(false);
    }
  }

  // ── Side panel save callback ────────────────────────────────────────────────

  function handleSaved(updated: AgentDetail) {
    const prev = agentsCache.current.get(updated.name);
    agentsCache.current.set(updated.name, updated);
    setSelectedAgent(updated);
    showSavedIndicator('saved');

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
            lifecycle: updated.lifecycle,
          },
        };
      }),
    );

    // Sync spawn edges with updated can_spawn
    const oldSpawn = new Set(prev?.can_spawn ?? []);
    const newSpawn = new Set(updated.can_spawn ?? []);
    const added = [...newSpawn].filter((t) => !oldSpawn.has(t));
    const removed = [...oldSpawn].filter((t) => !newSpawn.has(t));
    if (added.length > 0 || removed.length > 0) {
      setEdges((eds) => {
        let result = eds.filter(
          (e) => !(e.source === updated.name && removed.includes(e.target)),
        );
        for (const target of added) {
          if (!result.some((e) => e.source === updated.name && e.target === target)) {
            result = addEdge(makeEdge(updated.name, target), result);
          }
        }
        return result;
      });
    }
  }

  // ── Prevent node click selecting when clicking node buttons ────────────────

  const onNodeClick: NodeMouseHandler = useCallback((_event, node) => {
    setSelectedNodeId(node.id);
    // Trigger nodes: show toast with details and link to Triggers page
    if (node.id.startsWith('trigger-')) {
      const data = node.data as Record<string, unknown>;
      addToast(`Trigger "${data.title as string}" — manage on the Triggers page`, 'info');
      return;
    }
    handleSelect(node.id);
  }, [handleSelect, addToast]);

  // ── Canvas context menu (right-click) ────────────────────────────────────
  const reactFlowRef = useRef<HTMLDivElement>(null);

  const onPaneContextMenu = useCallback((event: MouseEvent | React.MouseEvent) => {
    event.preventDefault();
    const bounds = reactFlowRef.current?.getBoundingClientRect();
    const x = event.clientX;
    const y = event.clientY;
    const canvasX = bounds ? x - bounds.left : x;
    const canvasY = bounds ? y - bounds.top : y;
    setContextMenu({ x, y, canvasX, canvasY });
    setNodeMenu(null);
  }, []);

  const onPaneClick = useCallback(() => {
    if (contextMenu) setContextMenu(null);
    if (edgeMenu) setEdgeMenu(null);
    if (nodeMenu) setNodeMenu(null);
    if (selectedAgent) setSelectedAgent(null);
    setSelectedNodeId(null);
  }, [contextMenu, edgeMenu, nodeMenu, selectedAgent]);

  const onEdgeContextMenu = useCallback((event: MouseEvent | React.MouseEvent, edge: Edge) => {
    event.preventDefault();
    // Trigger edges are read-only — show info toast
    if (edge.id.startsWith('trigger:')) {
      addToast('Trigger connections are managed on the Triggers page', 'info');
      return;
    }
    setEdgeMenu({ x: event.clientX, y: event.clientY, edgeId: edge.id, source: edge.source, target: edge.target });
    setContextMenu(null);
    setNodeMenu(null);
  }, [addToast]);

  // ── Node context menu ─────────────────────────────────────────────────────

  const onNodeContextMenu = useCallback((event: MouseEvent | React.MouseEvent, node: Node) => {
    event.preventDefault();
    setNodeMenu({ x: event.clientX, y: event.clientY, nodeId: node.id, nodeType: node.type ?? 'agentNode' });
    setContextMenu(null);
    setEdgeMenu(null);
  }, []);

  const handleDeleteEdge = useCallback(async () => {
    if (!edgeMenu) return;
    const { edgeId, source, target } = edgeMenu;
    setEdgeMenu(null);

    const sourceAgent = agentsCache.current.get(source);
    if (!sourceAgent) return;
    const updatedSpawn = (sourceAgent.can_spawn ?? []).filter((a) => a !== target);
    showSavedIndicator('saving');
    try {
      const updated = await api.updateAgent(source, { can_spawn: updatedSpawn });
      agentsCache.current.set(source, updated);
      setEdges((eds) => eds.filter((e) => e.id !== edgeId));
      setNodes((nds) =>
        nds.map((n) => {
          if (n.id !== source) return n;
          const d = n.data as AgentNodeData;
          return { ...n, data: { ...d, spawnCount: (updated.can_spawn ?? []).length } };
        }),
      );
      if (selectedAgent?.name === source) setSelectedAgent(updated);
      showSavedIndicator('saved');
    } catch (err) {
      addToast(`Failed to remove connection: ${err instanceof Error ? err.message : 'Unknown error'}`, 'error');
      setSavedIndicator(null);
    }
  }, [edgeMenu, selectedAgent, setEdges, setNodes, addToast]);

  // ── Name validation ────────────────────────────────────────────────────────

  function handleNameChange(raw: string) {
    const lower = raw.toLowerCase();
    const valid = /^[a-z][a-z0-9-]*$/.test(lower);
    setNewForm((p) => ({ ...p, name: lower }));
    if (!lower) {
      setNameError('Name is required');
    } else if (!valid) {
      setNameError('Lowercase letters, numbers, hyphens. Must start with a letter.');
    } else {
      setNameError('');
    }
  }

  // ── Hint text based on selection state ────────────────────────────────────

  const hintText = selectedNodeId
    ? 'Press Delete to remove selected · Right-click for node actions'
    : 'Right-click canvas for menu · Drag between handles to connect agents · Click edge + Delete to remove · Drag nodes to reposition';

  // ─── Render ──────────────────────────────────────────────────────────────────

  return (
    <div
      className="flex flex-col overflow-hidden"
      style={{ margin: '-24px', height: '100vh' }}
    >
      {/* Toolbar */}
      <div className="flex items-center gap-3 px-4 h-12 border-b border-brand-shade3/15 bg-brand-dark-alt flex-shrink-0 flex-wrap">
        <span className="text-sm font-semibold text-brand-light">Agent Builder</span>
        <div className="flex-1" />

        {/* Saved indicator */}
        {savedIndicator && (
          <span className={`text-[10px] transition-opacity ${savedIndicator === 'saving' ? 'text-brand-shade3' : 'text-green-400'}`}>
            {savedIndicator === 'saving' ? 'Saving…' : 'All changes saved'}
          </span>
        )}

        <button
          onClick={runAutoLayout}
          className="px-3 py-1.5 text-xs text-brand-shade2 border border-brand-shade3/30 rounded-btn hover:text-brand-light hover:border-brand-shade3 transition-colors"
        >
          Auto Layout
        </button>
        <ExportButton />
        <ImportButton onImported={refetchCanvas} />
        <button
          onClick={() => { setShowFlowTest((v) => !v); }}
          className={`px-3 py-1.5 text-xs border rounded-btn transition-colors inline-flex items-center gap-1.5 ${
            showFlowTest
              ? 'text-brand-accent border-brand-accent/50 bg-brand-accent/10'
              : 'text-brand-shade2 border-brand-shade3/30 hover:text-brand-light hover:border-brand-shade3'
          }`}
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <polygon points="5 3 19 12 5 21 5 3" />
          </svg>
          Test Flow
        </button>
        <button
          onClick={() => {
            setShowTriggerForm(true);
            setTriggerForm(DEFAULT_TRIGGER_FORM);
            setTriggerError('');
            setCronError('');
          }}
          className="px-3 py-1.5 text-xs text-brand-shade2 border border-brand-shade3/30 rounded-btn hover:text-brand-light hover:border-brand-shade3 transition-colors"
        >
          + Add Trigger
        </button>
        <button
          onClick={() => {
            setShowNewForm(true);
            setNewForm(DEFAULT_NEW_FORM);
            setNameError('');
            setCreateError('');
          }}
          className="px-3 py-1.5 text-xs bg-brand-accent text-brand-light rounded-btn hover:bg-brand-accent-hover transition-colors"
        >
          + Add Agent
        </button>
      </div>

      {/* Drift notification */}
      <DriftNotification checkTrigger={refreshKey} />

      {/* Canvas + Side Panel */}
      <div className="flex flex-1 min-h-0">
        <div className="flex-1 relative" ref={reactFlowRef}>
          {loading && (
            <div className="absolute inset-0 flex items-center justify-center bg-brand-dark z-10">
              <div className="flex flex-col items-center gap-3">
                <div className="w-6 h-6 border-2 border-brand-accent border-t-transparent rounded-full animate-spin" />
                <span className="text-brand-shade3 text-sm">Loading agents…</span>
              </div>
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
            onNodeContextMenu={onNodeContextMenu}
            onNodeDragStop={onNodeDragStop}
            onPaneContextMenu={onPaneContextMenu}
            onPaneClick={onPaneClick}
            onEdgeContextMenu={onEdgeContextMenu}
            nodeTypes={nodeTypes}
            fitView
            fitViewOptions={{ padding: 0.2 }}
            deleteKeyCode="Delete"
            style={{ background: '#111111' }}
            proOptions={{ hideAttribution: true }}
            connectionLineStyle={{ stroke: '#D7513E', strokeWidth: 2, strokeDasharray: '5 5' }}
            isValidConnection={isValidConnection}
            selectionOnDrag={true}
            multiSelectionKeyCode="Shift"
          >
            <Background
              variant={BackgroundVariant.Dots}
              gap={20}
              size={1}
              color="#333333"
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
            <div className="absolute inset-0 flex flex-col items-center justify-center">
              <p className="text-brand-shade2 text-sm mb-2">No agents yet</p>
              <p className="text-brand-shade3/60 text-xs mb-4">Create your first agent to get started</p>
              <button
                onClick={() => {
                  setShowNewForm(true);
                  setNewForm(DEFAULT_NEW_FORM);
                  setNameError('');
                  setCreateError('');
                }}
                className="px-4 py-2 bg-brand-accent text-brand-light rounded-btn text-xs hover:bg-brand-accent-hover transition-colors"
              >
                Create your first agent
              </button>
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

        {showFlowTest && (
          <BuilderFlowTest
            agents={agents.map((a) => ({ name: a.name, model_id: String(a.model_id ?? '') }))}
            onClose={() => setShowFlowTest(false)}
          />
        )}
      </div>

      {/* AI Assistant */}
      {showAssistant && (
        <BuilderAssistant
          onClose={() => setShowAssistant(false)}
          onConfigChanged={refetchCanvas}
        />
      )}
      {!showAssistant && (
        <AssistantToggleButton onClick={() => setShowAssistant(true)} />
      )}

      {/* Canvas context menu */}
      {contextMenu && (
        <div
          className="fixed z-50 bg-brand-dark-alt border border-brand-shade3/20 rounded-card shadow-xl py-1 min-w-[160px] animate-modal-in"
          style={{ left: contextMenu.x, top: contextMenu.y }}
        >
          <button
            className="w-full px-4 py-2 text-left text-xs text-brand-shade2 hover:bg-brand-accent/10 hover:text-brand-light transition-colors flex items-center gap-2"
            onClick={() => {
              setContextMenu(null);
              setShowNewForm(true);
              setNewForm(DEFAULT_NEW_FORM);
              setNameError('');
              setCreateError('');
            }}
          >
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="10" /><line x1="12" y1="8" x2="12" y2="16" /><line x1="8" y1="12" x2="16" y2="12" />
            </svg>
            Add Agent
          </button>
          <button
            className="w-full px-4 py-2 text-left text-xs text-brand-shade2 hover:bg-brand-accent/10 hover:text-brand-light transition-colors flex items-center gap-2"
            onClick={() => {
              setContextMenu(null);
              setShowTriggerForm(true);
              setTriggerForm(DEFAULT_TRIGGER_FORM);
              setTriggerError('');
              setCronError('');
            }}
          >
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="10" /><polyline points="12 6 12 12 16 14" />
            </svg>
            Add Trigger
          </button>
          <div className="border-t border-brand-shade3/10 my-1" />
          <button
            className="w-full px-4 py-2 text-left text-xs text-brand-shade2 hover:bg-brand-accent/10 hover:text-brand-light transition-colors flex items-center gap-2"
            onClick={() => {
              setContextMenu(null);
              runAutoLayout();
            }}
          >
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <rect x="3" y="3" width="7" height="7" /><rect x="14" y="3" width="7" height="7" /><rect x="14" y="14" width="7" height="7" /><rect x="3" y="14" width="7" height="7" />
            </svg>
            Auto Layout
          </button>
          <button
            className="w-full px-4 py-2 text-left text-xs text-brand-shade2 hover:bg-brand-accent/10 hover:text-brand-light transition-colors flex items-center gap-2"
            onClick={() => {
              setContextMenu(null);
              setShowAssistant(true);
            }}
          >
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z" />
            </svg>
            AI Assistant
          </button>
        </div>
      )}

      {/* Node context menu */}
      {nodeMenu && (
        <div
          className="fixed z-50 bg-brand-dark-alt border border-brand-shade3/20 rounded-card shadow-xl py-1 min-w-[180px] animate-modal-in"
          style={{ left: nodeMenu.x, top: nodeMenu.y }}
          onMouseLeave={() => setNodeMenu(null)}
        >
          {nodeMenu.nodeType === 'triggerNode' ? (
            <button
              className="w-full px-4 py-2 text-left text-xs text-brand-shade2 hover:bg-brand-accent/10 hover:text-brand-light transition-colors"
              onClick={() => {
                setNodeMenu(null);
                addToast('Trigger connections are managed on the Triggers page', 'info');
              }}
            >
              Manage on Triggers page
            </button>
          ) : (
            <>
              <button
                className="w-full px-4 py-2 text-left text-xs text-brand-shade2 hover:bg-brand-accent/10 hover:text-brand-light transition-colors"
                onClick={() => {
                  setNodeMenu(null);
                  handleSelect(nodeMenu.nodeId);
                }}
              >
                Details
              </button>
              <button
                className="w-full px-4 py-2 text-left text-xs text-red-400 hover:bg-red-500/10 hover:text-red-300 transition-colors"
                onClick={() => {
                  setNodeMenu(null);
                  handleDeleteRequest(nodeMenu.nodeId);
                }}
              >
                Delete
              </button>
            </>
          )}
        </div>
      )}

      {/* Edge context menu */}
      {edgeMenu && (
        <div
          className="fixed z-50 bg-brand-dark-alt border border-brand-shade3/20 rounded-card shadow-xl py-1 min-w-[180px] animate-modal-in"
          style={{ left: edgeMenu.x, top: edgeMenu.y }}
        >
          <div className="px-4 py-1.5 text-[10px] text-brand-shade3 uppercase tracking-wide border-b border-brand-shade3/10">
            {edgeMenu.source} → {edgeMenu.target}
          </div>
          <button
            className="w-full px-4 py-2 text-left text-xs text-red-400 hover:bg-red-500/10 hover:text-red-300 transition-colors flex items-center gap-2"
            onClick={handleDeleteEdge}
          >
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="3 6 5 6 21 6" /><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2" />
            </svg>
            Delete Connection
          </button>
        </div>
      )}

      {/* Hint bar */}
      <div className="flex items-center px-4 h-7 border-t border-brand-shade3/10 bg-brand-dark flex-shrink-0">
        <span className="text-[11px] text-brand-shade3/70">
          {hintText}
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
                  onChange={(e) => handleNameChange(e.target.value)}
                  placeholder="my-agent"
                  className={`w-full px-3 py-2 bg-brand-dark border rounded-card text-sm text-brand-light placeholder-brand-shade3 focus:outline-none focus:ring-1 transition-colors ${
                    nameError
                      ? 'border-red-500/60 focus:border-red-500 focus:ring-red-500/30'
                      : 'border-brand-shade3/30 focus:border-brand-accent focus:ring-brand-accent'
                  }`}
                />
                {nameError ? (
                  <p className="text-[10px] text-red-400 mt-0.5">{nameError}</p>
                ) : (
                  <p className="text-[10px] text-brand-shade3/60 mt-0.5">Lowercase letters, numbers, hyphens. Must start with a letter.</p>
                )}
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
                disabled={creating || !newForm.name || !newForm.system_prompt || !!nameError}
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

      {/* New trigger dialog */}
      {showTriggerForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/60" onClick={() => setShowTriggerForm(false)} />
          <div className="relative bg-brand-dark-alt border border-brand-shade3/20 rounded-card shadow-xl w-[440px] animate-modal-in">
            <div className="px-5 py-4 border-b border-brand-shade3/15 flex items-center justify-between">
              <h3 className="text-sm font-semibold text-brand-light">New Trigger</h3>
              <button onClick={() => setShowTriggerForm(false)} className="text-brand-shade3 hover:text-brand-light p-1">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            <div className="p-5 space-y-4">
              <div>
                <label className="block text-xs font-medium text-brand-shade3 mb-1 uppercase tracking-wide">Title *</label>
                <input
                  type="text"
                  value={triggerForm.title ?? ''}
                  onChange={(e) => setTriggerForm((p) => ({ ...p, title: e.target.value }))}
                  placeholder="Daily report"
                  className="w-full px-3 py-2 bg-brand-dark border border-brand-shade3/30 rounded-card text-sm text-brand-light placeholder-brand-shade3 focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent transition-colors"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-brand-shade3 mb-1 uppercase tracking-wide">Type</label>
                <select
                  value={triggerForm.type ?? 'cron'}
                  onChange={(e) => setTriggerForm((p) => ({ ...p, type: e.target.value }))}
                  className="w-full px-3 py-2 bg-brand-dark border border-brand-shade3/30 rounded-card text-sm text-brand-light focus:outline-none focus:border-brand-accent transition-colors"
                >
                  <option value="cron">Cron Schedule</option>
                  <option value="webhook">Webhook</option>
                </select>
              </div>

              {triggerForm.type === 'cron' ? (
                <div>
                  <label className="block text-xs font-medium text-brand-shade3 mb-1 uppercase tracking-wide">Schedule *</label>
                  {/* Presets */}
                  <div className="flex gap-1.5 mb-1.5 flex-wrap">
                    {CRON_PRESETS.map((preset) => (
                      <button
                        key={preset.value}
                        type="button"
                        onClick={() => {
                          setTriggerForm((p) => ({ ...p, schedule: preset.value }));
                          setCronError('');
                        }}
                        className="px-2 py-0.5 text-[10px] bg-brand-dark border border-brand-shade3/30 rounded text-brand-shade2 hover:text-brand-light hover:border-brand-shade3 transition-colors"
                      >
                        {preset.label}
                      </button>
                    ))}
                  </div>
                  <input
                    type="text"
                    value={triggerForm.schedule ?? ''}
                    onChange={(e) => {
                      setTriggerForm((p) => ({ ...p, schedule: e.target.value }));
                      if (cronError) setCronError('');
                    }}
                    placeholder="0 9 * * *"
                    className={`w-full px-3 py-2 bg-brand-dark border rounded-card text-sm text-brand-light placeholder-brand-shade3 font-mono focus:outline-none focus:ring-1 transition-colors ${
                      cronError
                        ? 'border-red-500/60 focus:border-red-500 focus:ring-red-500/30'
                        : 'border-brand-shade3/30 focus:border-brand-accent focus:ring-brand-accent'
                    }`}
                  />
                  {cronError ? (
                    <p className="text-[10px] text-red-400 mt-0.5">{cronError}</p>
                  ) : (
                    <p className="text-[10px] text-brand-shade3/60 mt-0.5">Cron expression (minute hour day month weekday)</p>
                  )}
                </div>
              ) : (
                <div>
                  <label className="block text-xs font-medium text-brand-shade3 mb-1 uppercase tracking-wide">Webhook Path *</label>
                  <input
                    type="text"
                    value={triggerForm.webhook_path ?? ''}
                    onChange={(e) => setTriggerForm((p) => ({ ...p, webhook_path: e.target.value }))}
                    placeholder="/my-webhook"
                    className="w-full px-3 py-2 bg-brand-dark border border-brand-shade3/30 rounded-card text-sm text-brand-light placeholder-brand-shade3 font-mono focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent transition-colors"
                  />
                  {triggerForm.webhook_path && (
                    <p className="text-[10px] text-brand-shade3/60 mt-0.5 font-mono">
                      POST /api/v1/webhooks{triggerForm.webhook_path}
                    </p>
                  )}
                </div>
              )}

              <div>
                <label className="block text-xs font-medium text-brand-shade3 mb-1 uppercase tracking-wide">Target Agent *</label>
                <select
                  value={triggerForm.agent_name ?? ''}
                  onChange={(e) => setTriggerForm((p) => ({ ...p, agent_name: e.target.value || undefined }))}
                  className="w-full px-3 py-2 bg-brand-dark border border-brand-shade3/30 rounded-card text-sm text-brand-light focus:outline-none focus:border-brand-accent transition-colors"
                >
                  <option value="">Select agent…</option>
                  {agents.map((a) => (
                    <option key={a.name} value={a.name}>{a.name}</option>
                  ))}
                </select>
              </div>

              {triggerError && (
                <p className="text-xs text-red-400">{triggerError}</p>
              )}
            </div>
            <div className="px-5 py-4 border-t border-brand-shade3/15 flex gap-3">
              <button
                onClick={handleCreateTrigger}
                disabled={triggerCreating || !triggerForm.title || !triggerForm.agent_name || (triggerForm.type === 'cron' ? !triggerForm.schedule : !triggerForm.webhook_path)}
                className="flex-1 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover disabled:opacity-40 transition-colors"
              >
                {triggerCreating ? 'Creating…' : 'Create Trigger'}
              </button>
              <button
                onClick={() => setShowTriggerForm(false)}
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
        onClose={() => { setDeleteTarget(null); setDeleteError(''); }}
        onConfirm={confirmDelete}
        title="Delete Agent"
        message={
          <>
            Delete agent <strong className="text-brand-light">{deleteTarget}</strong>?
            This will also remove all spawn connections to/from it.
            {deleteError && <p className="mt-2 text-red-400 text-xs">{deleteError}</p>}
          </>
        }
        confirmLabel="Delete"
        loading={deleting}
        variant="danger"
      />
    </div>
  );
}

// ─── Exported component (wraps with providers) ─────────────────────────────────

export default function AgentBuilderPage() {
  return (
    <ToastProvider>
      <ReactFlowProvider>
        <AgentBuilderInner />
      </ReactFlowProvider>
    </ToastProvider>
  );
}
