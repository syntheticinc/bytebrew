import React, { useState, useEffect, useCallback, useRef } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  ReactFlow,
  Background,
  Controls,

  useNodesState,
  useEdgesState,
  useReactFlow,
  ReactFlowProvider,
  type Node,
  type Edge,
  BackgroundVariant,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import dagre from '@dagrejs/dagre';
import { api } from '../api/client';
import { usePrototype } from '../hooks/usePrototype';
import { useBottomPanel } from '../hooks/useBottomPanel';
import { useAdminRefresh } from '../hooks/useAdminRefresh';
import { createMockSchemas, type SchemaName } from '../mocks/canvas';
import type { AgentDetail, Model, Trigger, Schema } from '../types';
import AgentNode from '../components/builder/AgentNode';
import TriggerNode from '../components/builder/TriggerNode';
import GateNode from '../components/builder/GateNode';
import EdgeConfigPanel from '../components/builder/EdgeConfigPanel';
import GateConfigPanel from '../components/builder/GateConfigPanel';
// BuilderSidePanel removed — Details navigates to AgentDrillInPage
import DriftNotification from '../components/builder/DriftNotification';
import ConfirmDialog from '../components/ConfirmDialog';
import { ToastProvider, useToast } from '../components/builder/Toast';
import CanvasToolbar from '../components/builder/CanvasToolbar';

import { CanvasContextMenu, NodeContextMenu, EdgeContextMenu } from '../components/builder/CanvasContextMenus';
import TriggerConfigPanel from '../components/builder/TriggerConfigPanel';
import { useCanvasNodes, makeNode, makeTriggerNode, NODE_WIDTH, NODE_HEIGHT, TRIGGER_NODE_WIDTH, TRIGGER_NODE_HEIGHT } from '../hooks/useCanvasNodes';
import { useCanvasEdges, makeEdge, makeTriggerEdge } from '../hooks/useCanvasEdges';
import { useCanvasInteraction } from '../hooks/useCanvasInteraction';

// ─── Constants ────────────────────────────────────────────────────────────────

const nodeTypes = { agentNode: AgentNode, triggerNode: TriggerNode, gateNode: GateNode };
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

// ─── Inner component (needs ReactFlow context for useReactFlow) ───────────────

function AgentBuilderInner() {
  const { fitView } = useReactFlow();
  const { isPrototype } = usePrototype();
  const navigate = useNavigate();
  const { schemaName } = useParams<{ schemaName: string }>();
  const { selectedSchema, setSelectedSchema } = useBottomPanel();
  const [currentSchema, setCurrentSchema] = useState<Schema | null>(null);
  const [protoSchemas, setProtoSchemas] = useState<string[]>(['Support Schema', 'Dev Schema', 'Sales Schema']);
  const [protoSchema, setProtoSchema] = useState<SchemaName>('Support Schema');

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  // Side panel removed — Details navigates to full editor (AgentDrillInPage).
  // selectedAgent kept as null stub so hooks don't break (they check selectedAgent?.name).
  const [selectedAgent, setSelectedAgent] = useState<AgentDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [refreshKey, setRefreshKey] = useState(0);
  const [savedIndicator, setSavedIndicator] = useState<'saved' | 'saving' | null>(null);

  const { addToast } = useToast();

  const agentsCache = useRef<Map<string, AgentDetail>>(new Map());
  const modelsRef = useRef<Model[]>([]);
  const agentsListRef = useRef<AgentDetail[]>([]);
  const savedIndicatorTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reactFlowRef = useRef<HTMLDivElement>(null);

  // Track known agent names so we can detect newly created ones after refresh
  const knownAgentNamesRef = useRef<Set<string>>(new Set());
  const newNodeTimersRef = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());

  const refetchCanvas = useCallback(() => setRefreshKey((k) => k + 1), []);
  useAdminRefresh(refetchCanvas);

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
        interaction.setContextMenu(null);
        interaction.setNodeMenu(null);
        interaction.setEdgeMenu(null);
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

  // Details → navigate to full editor page (AgentDrillInPage)
  const handleSelect = useCallback((name: string) => {
    const schema = schemaName ?? '';
    navigate(`/builder/${encodeURIComponent(schema)}/${encodeURIComponent(name)}`);
  }, [schemaName, navigate]);

  const handleDeleteRequest = useCallback((name: string) => {
    nodeOps.setDeleteTarget(name);
    nodeOps.setDeleteError('');
  }, []);

  // ── Extracted hooks ──────────────────────────────────────────────────────────

  const nodeOps = useCanvasNodes({
    nodes,
    setNodes,
    setEdges,
    agentsCache,
    agentsListRef,
    modelsRef,
    addToast,
    selectedAgent,
    setSelectedAgent,
    isPrototype,
    handleSelect,
    handleDeleteRequest,
    currentSchemaId: currentSchema?.id ?? null,
  });

  const edgeOps = useCanvasEdges({
    setNodes,
    setEdges,
    agentsCache,
    addToast,
    showSavedIndicator,
    setSavedIndicator,
    isPrototype,
    selectedAgent,
    setSelectedAgent,
  });

  const interaction = useCanvasInteraction({
    isPrototype,
    protoSchema,
    selectedSchema,
    navigate,
    addToast,
    handleSelect,
    selectedAgent,
    setSelectedAgent,
    agentsCache,
    setNodes,
    setEdges,
    showSavedIndicator,
    setSavedIndicator,
    reactFlowRef,
  });

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

    // Resolve current schema first, then load its agents
    const loadData = async (): Promise<{ details: AgentDetail[]; models: Model[]; triggers: Trigger[]; schema: Schema | null }> => {
      const [allSchemas, models] = await Promise.all([
        schemaName && !isPrototype ? api.listSchemas() : Promise.resolve([] as Schema[]),
        api.listModels(),
      ]);

      let schema: Schema | null = null;
      if (schemaName && !isPrototype) {
        schema = allSchemas.find((s) => s.name === schemaName) ?? null;
        if (!schema) {
          throw new Error(`Schema "${schemaName}" not found`);
        }
      }

      // Load triggers scoped to current schema (or all if no schema).
      const triggers = await api.listTriggers(schema?.id).catch(() => [] as Trigger[]);

      // Load agents: schema-scoped in production, all agents as fallback
      let agentNames: string[];
      if (schema) {
        agentNames = (await api.listSchemaAgents(schema.id)) ?? [];
      } else {
        const agentList = await api.listAgents();
        agentNames = agentList.map((a) => a.name);
      }

      const details = await Promise.all(agentNames.map((name) => api.getAgent(name)));
      return { details, models, triggers, schema };
    };

    loadData()
      .then(({ details, models, triggers, schema }) => {
        if (cancelled) return;
        modelsRef.current = models;
        setCurrentSchema(schema);
        if (schema) {
          setSelectedSchema(schema.name);
        }
        const modelMap = new Map(models.map((m) => [m.id, m.name]));

        const agentNames = new Set(details.map((a) => a.name));
        details.forEach((a) => agentsCache.current.set(a.name, a));
        agentsListRef.current = details;

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

        // Build trigger nodes + edges (scoped to current schema).
        // The canvas edge is the routing config: drawing it sets target, deleting it clears it.
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
        const newNodes = rawNodes.filter((n) => savedPositions[n.id] === undefined);
        if (newNodes.length > 0) {
          const laid = applyDagre([...rawNodes], rawEdges);
          const laidMap = new Map(laid.map((n) => [n.id, n.position]));
          rawNodes = rawNodes.map((n) =>
            savedPositions[n.id] !== undefined ? n : { ...n, position: laidMap.get(n.id) ?? n.position },
          );
          savePositions(rawNodes);
        }

        // Detect newly created agents and mark them with isNew for fade-in animation
        const currentNames = new Set(details.map((a) => a.name));
        if (knownAgentNamesRef.current.size > 0) {
          for (const name of currentNames) {
            if (!knownAgentNamesRef.current.has(name)) {
              // Mark new node with isNew flag
              rawNodes = rawNodes.map((n) =>
                n.id === name
                  ? { ...n, data: { ...n.data, isNew: true } }
                  : n,
              );
              // Clear isNew after animation completes (1s)
              const timer = setTimeout(() => {
                setNodes((nds) =>
                  nds.map((n) =>
                    n.id === name
                      ? { ...n, data: { ...n.data, isNew: false } }
                      : n,
                  ),
                );
                newNodeTimersRef.current.delete(name);
              }, 1000);
              newNodeTimersRef.current.set(name, timer);
            }
          }
        }
        knownAgentNamesRef.current = currentNames;

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

    return () => {
      cancelled = true;
      // Cleanup new-node timers on unmount
      for (const timer of newNodeTimersRef.current.values()) {
        clearTimeout(timer);
      }
    };
  }, [handleSelect, handleDeleteRequest, refreshKey, isPrototype, protoSchema, schemaName, setSelectedSchema]);

  // Side panel removed — agents are edited via AgentDrillInPage (full editor).
  // Canvas auto-refreshes on return via useAdminRefresh.

  // ── Auto-layout ─────────────────────────────────────────────────────────────

  function runAutoLayout() {
    const laid = applyDagre([...nodes], [...edges]);
    savePositions(laid);
    setNodes(laid);
    setTimeout(() => fitView({ padding: 0.2 }), 50);
  }

  // ── Node drag stop (save positions) ─────────────────────────────────────────

  const onNodeDragStop = useCallback(
    (_event: React.MouseEvent, node: Node) => {
      mergePositions([node]);
    },
    [],
  );

  // ── Hint text ───────────────────────────────────────────────────────────────

  const hintText = interaction.selectedNodeId
    ? 'Press Delete to remove selected \u00b7 Right-click for node actions'
    : 'Right-click canvas for menu \u00b7 Drag between handles to connect agents \u00b7 Click edge + Delete to remove \u00b7 Drag nodes to reposition';

  // ─── Render ──────────────────────────────────────────────────────────────────

  return (
    <div
      className="flex flex-col overflow-hidden"
      style={{ margin: '-24px', height: 'calc(100% + 48px)' }}
    >
      {/* Toolbar */}
      <CanvasToolbar
        isPrototype={isPrototype}
        savedIndicator={savedIndicator}
        onAutoLayout={runAutoLayout}
        onRefetch={refetchCanvas}
        onAddAgent={() => nodeOps.handleInstantAgentCreate()}
        onAddTrigger={(type) => nodeOps.handleInstantTriggerCreate(undefined, type)}
        isSystemSchema={currentSchema?.is_system === true}
        onRestoreDefaults={async () => {
          await api.restoreBuilderAssistant();
          addToast('Builder schema restored to factory defaults', 'success');
          refetchCanvas();
        }}
        schemaName={schemaName}
        onBack={() => navigate('/builder')}
        protoSchema={protoSchema}
        protoSchemas={protoSchemas}
        setProtoSchema={setProtoSchema}
        setProtoSchemas={setProtoSchemas}
      />

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
            onConnect={edgeOps.onConnect}
            onEdgesDelete={edgeOps.onEdgesDelete}
            onNodeClick={interaction.onNodeClick}
            onNodeContextMenu={interaction.onNodeContextMenu}
            onNodeDragStop={onNodeDragStop}
            onPaneContextMenu={interaction.onPaneContextMenu}
            onPaneClick={interaction.onPaneClick}
            onEdgeClick={interaction.onEdgeClick}
            onEdgeContextMenu={interaction.onEdgeContextMenu}
            nodeTypes={nodeTypes}
            fitView
            fitViewOptions={{ padding: 0.2 }}
            deleteKeyCode="Delete"
            style={{ background: '#111111' }}
            proOptions={{ hideAttribution: true }}
            connectionLineStyle={{ stroke: '#D7513E', strokeWidth: 2, strokeDasharray: '5 5' }}
            isValidConnection={edgeOps.isValidConnection}
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
                onClick={() => nodeOps.handleInstantAgentCreate({ x: 200, y: 100 })}
                className="px-4 py-2 bg-brand-accent text-brand-light rounded-btn text-xs hover:bg-brand-accent-hover transition-colors"
              >
                Create your first agent
              </button>
            </div>
          )}
        </div>

        {/* Side panel removed — Details navigates to AgentDrillInPage */}

        {interaction.selectedEdge && (
          <EdgeConfigPanel
            edge={interaction.selectedEdge}
            onClose={() => interaction.setSelectedEdge(null)}
            onSave={(_edge, _config) => {
              // TODO: persist edge config to API when backend is ready
              interaction.setSelectedEdge(null);
            }}
            onDelete={(edgeId) => {
              setEdges((eds) => eds.filter((e) => e.id !== edgeId));
              interaction.setSelectedEdge(null);
            }}
          />
        )}

        {/* Gate config panel */}
        {interaction.selectedGate && (
          <GateConfigPanel
            gate={interaction.selectedGate}
            onClose={() => interaction.setSelectedGate(null)}
            onSave={(_gateId, _config) => {
              // TODO: persist gate config to API when backend is ready
              interaction.setSelectedGate(null);
            }}
          />
        )}

        {/* Trigger config panel */}
        {interaction.selectedTrigger && (
          <TriggerConfigPanel
            trigger={interaction.selectedTrigger}
            setTrigger={interaction.setSelectedTrigger}
            setNodes={setNodes}
            setEdges={setEdges}
            isPrototype={isPrototype}
            addToast={addToast}
          />
        )}
      </div>

      {/* AI Assistant moved to BottomPanel (Layout.tsx) — floating overlay removed */}

      {/* Canvas context menu */}
      {interaction.contextMenu && (
        <CanvasContextMenu
          menu={interaction.contextMenu}
          onClose={() => interaction.setContextMenu(null)}
          onAddAgent={(pos) => nodeOps.handleInstantAgentCreate(pos)}
          onAddTrigger={(pos) => nodeOps.handleInstantTriggerCreate(pos)}
          onAutoLayout={runAutoLayout}
        />
      )}

      {/* Node context menu */}
      {interaction.nodeMenu && (
        <NodeContextMenu
          menu={interaction.nodeMenu}
          onClose={() => interaction.setNodeMenu(null)}
          onDetails={handleSelect}
          onDelete={handleDeleteRequest}
        />
      )}

      {/* Edge context menu */}
      {interaction.edgeMenu && (
        <EdgeContextMenu
          menu={interaction.edgeMenu}
          onDeleteEdge={interaction.handleDeleteEdge}
        />
      )}

      {/* Hint bar */}
      <div className="flex items-center px-4 h-7 border-t border-brand-shade3/10 bg-brand-dark flex-shrink-0">
        <span className="text-[11px] text-brand-shade3/70">
          {hintText}
        </span>
      </div>

      {/* Delete confirm */}
      <ConfirmDialog
        open={nodeOps.deleteTarget !== null}
        onClose={() => { nodeOps.setDeleteTarget(null); nodeOps.setDeleteError(''); }}
        onConfirm={nodeOps.confirmDelete}
        title={currentSchema ? 'Remove from Schema' : 'Delete Agent'}
        message={
          <>
            {currentSchema ? (
              <>Remove agent <strong className="text-brand-light">{nodeOps.deleteTarget}</strong> from schema <strong className="text-brand-light">{currentSchema.name}</strong>? The agent will remain in the system but will no longer appear on this canvas.</>
            ) : (
              <>Delete agent <strong className="text-brand-light">{nodeOps.deleteTarget}</strong>? This will also remove all spawn connections to/from it.</>
            )}
            {nodeOps.deleteError && <p className="mt-2 text-red-400 text-xs">{nodeOps.deleteError}</p>}
          </>
        }
        confirmLabel={currentSchema ? 'Remove' : 'Delete'}
        loading={nodeOps.deleting}
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
