import React, { useState, useEffect, useCallback, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
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
import { usePrototype } from '../hooks/usePrototype';
import { createMockSchemas, type SchemaName } from '../mocks/canvas';
import type { AgentDetail, Model, CreateAgentRequest, CreateTriggerRequest, Trigger } from '../types';
import AgentNode, { type AgentNodeData } from '../components/builder/AgentNode';
import TriggerNode from '../components/builder/TriggerNode';
import GateNode from '../components/builder/GateNode';
import EdgeConfigPanel from '../components/builder/EdgeConfigPanel';
import GateConfigPanel from '../components/builder/GateConfigPanel';
import BuilderSidePanel from '../components/builder/BuilderSidePanel';
import { ExportButton, ImportButton } from '../components/builder/BuilderExportImport';
import BuilderFlowTest from '../components/builder/BuilderFlowTest';
// BuilderAssistant moved to BottomPanel (Layout.tsx)
import DriftNotification from '../components/builder/DriftNotification';
import ConfirmDialog from '../components/ConfirmDialog';
import { ToastProvider, useToast } from '../components/builder/Toast';
import CronScheduler from '../components/CronScheduler';

// ─── Constants ────────────────────────────────────────────────────────────────

const nodeTypes = { agentNode: AgentNode, triggerNode: TriggerNode, gateNode: GateNode };
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

// ─── Instant creation helpers ─────────────────────────────────────────────────

function generateAgentName(existing: string[]): string {
  let n = 1;
  while (existing.includes(`new-agent-${n}`)) n++;
  return `new-agent-${n}`;
}

function generateTriggerName(existing: string[]): string {
  let n = 1;
  while (existing.includes(`new-trigger-${n}`)) n++;
  return `new-trigger-${n}`;
}

function generateShortId(): string {
  return Math.random().toString(36).substring(2, 10);
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

// ─── Inner component (needs ReactFlow context for useReactFlow) ───────────────

function AgentBuilderInner() {
  const { fitView } = useReactFlow();
  const { isPrototype } = usePrototype();
  const navigate = useNavigate();
  const [protoSchemas, setProtoSchemas] = useState<string[]>(['Support Schema', 'Dev Schema', 'Sales Schema']);
  const [protoSchema, setProtoSchema] = useState<SchemaName>('Support Schema');
  const [protoSchemaDropdown, setProtoSchemaDropdown] = useState(false);

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const [agents, setAgents] = useState<AgentDetail[]>([]);
  const [selectedAgent, setSelectedAgent] = useState<AgentDetail | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [deleteError, setDeleteError] = useState('');
  const [deleting, setDeleting] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showFlowTest, setShowFlowTest] = useState(false);
  // showAssistant removed — AI Assistant moved to BottomPanel
  const [refreshKey, setRefreshKey] = useState(0);
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number; canvasX: number; canvasY: number } | null>(null);
  const [nodeMenu, setNodeMenu] = useState<{ x: number; y: number; nodeId: string; nodeType: string } | null>(null);
  const [edgeMenu, setEdgeMenu] = useState<{ x: number; y: number; edgeId: string; source: string; target: string } | null>(null);
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
    // Prototype mode: use mock data
    if (isPrototype) {
      const mockSchemas = createMockSchemas(handleSelect, handleDeleteRequest);
      const schema = mockSchemas[protoSchema] ?? Object.values(mockSchemas)[0];
      if (schema) {
        const laid = applyDagre(schema.nodes, schema.edges);
        setNodes(laid);
        setEdges(schema.edges);
      }
      setLoading(false);
      return;
    }

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
  }, [handleSelect, handleDeleteRequest, refreshKey, isPrototype, protoSchema]);

  // ── Connect handler (draw edge = add can_spawn) ──────────────────────────────

  const onConnect = useCallback(
    async (connection: Connection) => {
      const { source, target } = connection;
      if (!source || !target) return;

      // Prototype mode: add edge visually without API call
      if (isPrototype) {
        setEdges((eds) => addEdge({ ...connection, type: 'smoothstep', style: { stroke: '#4CAF50', strokeWidth: 1.5 }, label: 'flow' }, eds));
        return;
      }

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
    [selectedAgent, setNodes, setEdges, addToast, isPrototype],
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
      // Prototype mode: just remove from local state, no API calls
      if (isPrototype) return;

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
    [selectedAgent, setNodes, setEdges, addToast, isPrototype],
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

  // ── Instant agent creation (no modal) ──────────────────────────────────────

  async function handleInstantAgentCreate(canvasPosition?: { x: number; y: number }) {
    const existingNames = nodes.filter((n) => n.type === 'agentNode').map((n) => n.id);
    const name = generateAgentName(existingNames);
    const pos = canvasPosition ?? { x: Math.random() * 400 + 100, y: Math.random() * 200 + 50 };

    if (isPrototype) {
      const newNode: Node = {
        id: name,
        type: 'agentNode',
        position: pos,
        data: {
          name,
          modelName: '',
          toolsCount: 0,
          spawnCount: 0,
          confirmCount: 0,
          lifecycle: 'spawn',
          onSelect: handleSelect,
          onDelete: handleDeleteRequest,
        } satisfies AgentNodeData,
      };
      setNodes((nds) => [...nds, newNode]);
      addToast(`Agent "${name}" created — click to configure`, 'success');
      return;
    }

    // Production mode: create via API
    try {
      const created = await api.createAgent({
        name,
        system_prompt: '',
        lifecycle: 'spawn',
        tool_execution: 'sequential',
        max_steps: 25,
        max_context_size: 16000,
        max_turn_duration: 120,
        tools: [],
        can_spawn: [],
        mcp_servers: [],
        confirm_before: [],
      } as CreateAgentRequest);
      agentsCache.current.set(created.name, created);
      agentsListRef.current = [...agentsListRef.current, created];
      setAgents((prev) => [...prev, created]);

      const modelMap = new Map(modelsRef.current.map((m) => [m.id, m.name]));
      const newNode = makeNode(created, modelMap, pos, handleSelect, handleDeleteRequest);
      setNodes((nds) => [...nds, newNode]);
      addToast(`Agent "${created.name}" created — click to configure`, 'success');
    } catch (err) {
      addToast(`Failed to create agent: ${err instanceof Error ? err.message : 'Unknown error'}`, 'error');
    }
  }

  // ── Instant trigger creation (no modal) ──────────────────────────────────

  async function handleInstantTriggerCreate(canvasPosition?: { x: number; y: number }) {
    const existingNames = nodes.filter((n) => n.type === 'triggerNode').map((n) => (n.data as Record<string, unknown>).title as string);
    const title = generateTriggerName(existingNames);
    const webhookPath = `/webhook/${generateShortId()}`;
    const pos = canvasPosition ?? { x: Math.random() * 200 + 50, y: Math.random() * 200 + 50 };

    if (isPrototype) {
      const mockId = Date.now();
      const newNode = makeTriggerNode(
        { id: mockId, type: 'webhook', title, webhook_path: webhookPath, enabled: true, agent_name: '', schedule: '', description: '', created_at: new Date().toISOString() } as Trigger,
        pos,
      );
      setNodes((nds) => [...nds, newNode]);
      addToast(`Trigger "${title}" created — click to configure`, 'success');
      return;
    }

    // Production mode: create via API
    try {
      const created = await api.createTrigger({
        type: 'webhook',
        title,
        webhook_path: webhookPath,
        enabled: true,
      } as CreateTriggerRequest);
      const newNode = makeTriggerNode(created, pos);
      setNodes((nds) => [...nds, newNode]);
      addToast(`Trigger "${created.title}" created — click to configure`, 'success');
    } catch (err) {
      addToast(`Failed to create trigger: ${err instanceof Error ? err.message : 'Unknown error'}`, 'error');
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

  // State for prototype gate/trigger config panels
  const [selectedGate, setSelectedGate] = useState<Record<string, unknown> | null>(null);
  const [selectedTrigger, setSelectedTrigger] = useState<Record<string, unknown> | null>(null);

  const onNodeClick: NodeMouseHandler = useCallback((_event, node) => {
    setSelectedNodeId(node.id);

    // Gate node → show gate config panel (both modes)
    if (node.type === 'gateNode') {
      setSelectedGate(node.data as Record<string, unknown>);
      setSelectedTrigger(null);
      setSelectedEdge(null);
      return;
    }

    if (isPrototype) {
      // Trigger node → show trigger config panel
      if (node.type === 'triggerNode') {
        setSelectedTrigger(node.data as Record<string, unknown>);
        setSelectedGate(null);
        setSelectedEdge(null);
        return;
      }
      // Agent node → navigate to drill-in
      setSelectedGate(null);
      setSelectedTrigger(null);
      navigate(`/builder/${encodeURIComponent(protoSchema)}/${node.id}`);
      return;
    }

    // Production mode
    if (node.id.startsWith('trigger-')) {
      const data = node.data as Record<string, unknown>;
      addToast(`Trigger "${data.title as string}" — manage on the Triggers page`, 'info');
      return;
    }
    handleSelect(node.id);
  }, [handleSelect, addToast, isPrototype, navigate, protoSchema]);

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

  // ── Edge click (opens edge config side panel) ────────────────────────────
  const [selectedEdge, setSelectedEdge] = useState<Edge | null>(null);

  const onEdgeClick = useCallback((_event: React.MouseEvent, edge: Edge) => {
    setSelectedEdge(edge);
    setSelectedGate(null);
    setSelectedTrigger(null);
  }, []);

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

  // ── Hint text based on selection state ────────────────────────────────────

  const hintText = selectedNodeId
    ? 'Press Delete to remove selected · Right-click for node actions'
    : 'Right-click canvas for menu · Drag between handles to connect agents · Click edge + Delete to remove · Drag nodes to reposition';

  // ─── Render ──────────────────────────────────────────────────────────────────

  return (
    <div
      className="flex flex-col overflow-hidden"
      style={{ margin: '-24px', height: 'calc(100% + 48px)' }}
    >
      {/* Toolbar */}
      <div className="flex items-center gap-3 px-4 h-12 border-b border-brand-shade3/15 bg-brand-dark-alt flex-shrink-0 flex-wrap">
        <span className="text-sm font-semibold text-brand-light">Agent Builder</span>

        {/* Prototype: schema switcher */}
        {isPrototype && (
          <>
            <div className="w-px h-4 bg-brand-shade3/20" />
            <div className="relative">
              <button
                onClick={() => setProtoSchemaDropdown(v => !v)}
                className="bg-brand-dark border border-brand-shade3/20 rounded-btn text-brand-light text-xs px-2.5 py-1 cursor-pointer flex items-center gap-1.5 font-mono"
              >
                {protoSchema}
                <span className="text-brand-shade3 text-[10px]">&#9662;</span>
              </button>
              {protoSchemaDropdown && (
                <div className="absolute top-full left-0 mt-1 bg-brand-dark-alt border border-brand-shade3/20 rounded-card z-50 min-w-[180px] shadow-lg">
                  {protoSchemas.map(s => (
                    <div
                      key={s}
                      className={`flex items-center justify-between text-xs px-3 py-[7px] font-mono transition-colors ${
                        s === protoSchema ? 'bg-brand-accent/[0.13] text-brand-accent' : 'text-brand-light hover:bg-brand-shade3/20'
                      }`}
                    >
                      <button
                        className="flex-1 text-left cursor-pointer"
                        onClick={() => { setProtoSchema(s); setProtoSchemaDropdown(false); }}
                      >
                        {s}
                      </button>
                      <div className="flex items-center gap-1 ml-2 shrink-0">
                        <button
                          title="Rename"
                          className="text-brand-shade3 hover:text-brand-light transition-colors"
                          onClick={(e) => {
                            e.stopPropagation();
                            const newName = window.prompt('Rename schema:', s);
                            if (newName && newName.trim() && newName.trim() !== s) {
                              setProtoSchemas(prev => prev.map(n => n === s ? newName.trim() : n));
                              if (protoSchema === s) setProtoSchema(newName.trim());
                            }
                          }}
                        >
                          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M17 3a2.85 2.85 0 0 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/></svg>
                        </button>
                        {s !== protoSchema && (
                          <button
                            title="Delete"
                            className="text-brand-shade3 hover:text-red-400 transition-colors"
                            onClick={(e) => {
                              e.stopPropagation();
                              setProtoSchemas(prev => prev.filter(n => n !== s));
                            }}
                          >
                            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 6L6 18M6 6l12 12" /></svg>
                          </button>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
            <button
              className="px-2.5 py-1 text-xs text-brand-shade2 border border-brand-shade3/20 rounded-btn font-mono hover:text-brand-light transition-colors"
              onClick={() => {
                const name = window.prompt('New schema name:');
                if (name && name.trim()) {
                  const trimmed = name.trim();
                  setProtoSchemas(prev => [...prev, trimmed]);
                  setProtoSchema(trimmed);
                }
              }}
            >
              + Schema
            </button>
          </>
        )}

        <div className="flex-1" />

        {/* Saved indicator — production only */}
        {!isPrototype && savedIndicator && (
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
        {!isPrototype && <ExportButton />}
        {!isPrototype && <ImportButton onImported={refetchCanvas} />}
        {!isPrototype && (
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
        )}
        <button
          onClick={() => handleInstantTriggerCreate()}
          className="px-3 py-1.5 text-xs text-brand-shade2 border border-brand-shade3/30 rounded-btn hover:text-brand-light hover:border-brand-shade3 transition-colors"
        >
          + Add Trigger
        </button>
        <button
          onClick={() => handleInstantAgentCreate()}
          className="px-3 py-1.5 text-xs bg-brand-accent text-brand-light rounded-btn hover:bg-brand-accent-hover transition-colors"
        >
          + Add Agent
        </button>
      </div>

      {/* Drift notification — production only */}
      {!isPrototype && <DriftNotification checkTrigger={refreshKey} />}

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
            onEdgeClick={onEdgeClick}
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
                onClick={() => handleInstantAgentCreate({ x: 200, y: 100 })}
                className="px-4 py-2 bg-brand-accent text-brand-light rounded-btn text-xs hover:bg-brand-accent-hover transition-colors"
              >
                Create your first agent
              </button>
            </div>
          )}
        </div>

        {!isPrototype && selectedAgent && (
          <BuilderSidePanel
            agent={selectedAgent}
            onClose={() => setSelectedAgent(null)}
            onSaved={handleSaved}
            onDelete={handleDeleteRequest}
          />
        )}

        {selectedEdge && (
          <EdgeConfigPanel
            edge={selectedEdge}
            onClose={() => setSelectedEdge(null)}
            onSave={(_edge, _config) => {
              // TODO: persist edge config to API when backend is ready
              setSelectedEdge(null);
            }}
            onDelete={(edgeId) => {
              setEdges((eds) => eds.filter((e) => e.id !== edgeId));
              setSelectedEdge(null);
            }}
          />
        )}

        {!isPrototype && showFlowTest && (
          <BuilderFlowTest
            agents={agents.map((a) => ({ name: a.name, model_id: String(a.model_id ?? '') }))}
            onClose={() => setShowFlowTest(false)}
          />
        )}

        {/* Gate config panel */}
        {selectedGate && (
          <GateConfigPanel
            gate={selectedGate}
            onClose={() => setSelectedGate(null)}
            onSave={(_gateId, _config) => {
              // TODO: persist gate config to API when backend is ready
              setSelectedGate(null);
            }}
          />
        )}

        {/* Prototype: Trigger config panel */}
        {isPrototype && selectedTrigger && (
          <div className="w-80 border-l border-brand-shade3/10 bg-brand-dark-surface flex flex-col shrink-0 overflow-y-auto">
            <div className="flex items-center justify-between px-4 py-3 border-b border-brand-shade3/10">
              <h3 className="text-sm font-semibold text-brand-light font-mono">Trigger Configuration</h3>
              <button onClick={() => setSelectedTrigger(null)} className="text-brand-shade3 hover:text-brand-light p-1"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 6L6 18M6 6l12 12" /></svg></button>
            </div>
            <div className="p-4 space-y-4">
              <div>
                <label className="block text-xs text-brand-shade3 mb-1 font-mono">Title</label>
                <input className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light font-mono opacity-60 cursor-not-allowed" value={String(selectedTrigger.title ?? '')} readOnly />
              </div>
              <div>
                <label className="block text-xs text-brand-shade3 mb-1 font-mono">Type</label>
                <select className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light font-mono opacity-60 cursor-not-allowed" value={String(selectedTrigger.type ?? 'webhook')} disabled>
                  <option value="webhook">Webhook</option>
                  <option value="cron">Cron Schedule</option>
                </select>
              </div>
              {selectedTrigger.type === 'cron' && (
                <div>
                  <label className="block text-xs text-brand-shade3 mb-1 font-mono">Schedule</label>
                  <CronScheduler
                    value={String(selectedTrigger.schedule ?? '')}
                    onChange={(cron) => {
                      setSelectedTrigger((prev) => prev ? { ...prev, schedule: cron } : prev);
                      // Update the node data in canvas
                      setNodes((nds) => nds.map((n) => {
                        if (n.type !== 'triggerNode') return n;
                        const d = n.data as Record<string, unknown>;
                        if (d.id !== selectedTrigger.id) return n;
                        return { ...n, data: { ...d, schedule: cron } };
                      }));
                    }}
                  />
                </div>
              )}
              {selectedTrigger.type === 'webhook' && (
                <div>
                  <label className="block text-xs text-brand-shade3 mb-1 font-mono">Webhook Path</label>
                  <input className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light font-mono opacity-60 cursor-not-allowed" value={String(selectedTrigger.webhook_path ?? '')} readOnly />
                  <p className="mt-1 text-xs text-brand-shade3">POST requests to this path will trigger the agent</p>
                </div>
              )}
              <div>
                <label className="flex items-center gap-2 text-sm text-brand-shade2 cursor-pointer select-none">
                  <input type="checkbox" className="accent-brand-accent" checked={selectedTrigger.enabled !== false} disabled />
                  Enabled
                </label>
                <p className="mt-1 text-xs text-brand-shade3">Disabled triggers will not fire</p>
              </div>
              <div>
                <label className="block text-xs text-brand-shade3 mb-1 font-mono">Target Agent</label>
                <input className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light font-mono opacity-60 cursor-not-allowed" value={String(selectedTrigger.agentName ?? '')} readOnly />
                <p className="mt-1 text-xs text-brand-shade3">Entry agent that receives trigger events</p>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* AI Assistant moved to BottomPanel (Layout.tsx) — floating overlay removed */}

      {/* Canvas context menu */}
      {contextMenu && (
        <div
          className="fixed z-50 bg-brand-dark-alt border border-brand-shade3/20 rounded-card shadow-xl py-1 min-w-[160px] animate-modal-in"
          style={{ left: contextMenu.x, top: contextMenu.y }}
        >
          <button
            className="w-full px-4 py-2 text-left text-xs text-brand-shade2 hover:bg-brand-accent/10 hover:text-brand-light transition-colors flex items-center gap-2"
            onClick={() => {
              const pos = { x: contextMenu.canvasX, y: contextMenu.canvasY };
              setContextMenu(null);
              handleInstantAgentCreate(pos);
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
              const pos = { x: contextMenu.canvasX, y: contextMenu.canvasY };
              setContextMenu(null);
              handleInstantTriggerCreate(pos);
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
          {/* AI Assistant moved to BottomPanel */}
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
