import React, { useCallback, useState } from 'react';
import type { Node, Edge, NodeMouseHandler } from '@xyflow/react';
import { api } from '../api/client';
import type { AgentDetail } from '../types';

// ─── Types ────────────────────────────────────────────────────────────────────

export interface ContextMenuState {
  x: number;
  y: number;
  canvasX: number;
  canvasY: number;
}

export interface NodeMenuState {
  x: number;
  y: number;
  nodeId: string;
  nodeType: string;
}

export interface EdgeMenuState {
  x: number;
  y: number;
  edgeId: string;
  source: string;
  target: string;
}

// ─── Hook ─────────────────────────────────────────────────────────────────────

interface UseCanvasInteractionParams {
  isPrototype: boolean;
  protoSchema: string;
  selectedSchema: string;
  navigate: (path: string) => void;
  addToast: (message: string, type: 'success' | 'error' | 'info' | 'warning') => void;
  handleSelect: (name: string) => void;
  selectedAgent: AgentDetail | null;
  setSelectedAgent: React.Dispatch<React.SetStateAction<AgentDetail | null>>;
  agentsCache: React.MutableRefObject<Map<string, AgentDetail>>;
  setNodes: React.Dispatch<React.SetStateAction<Node[]>>;
  setEdges: React.Dispatch<React.SetStateAction<Edge[]>>;
  showSavedIndicator: (state: 'saved' | 'saving') => void;
  setSavedIndicator: React.Dispatch<React.SetStateAction<'saved' | 'saving' | null>>;
  reactFlowRef: React.RefObject<HTMLDivElement | null>;
}

export function useCanvasInteraction({
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
}: UseCanvasInteractionParams) {
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null);
  const [nodeMenu, setNodeMenu] = useState<NodeMenuState | null>(null);
  const [edgeMenu, setEdgeMenu] = useState<EdgeMenuState | null>(null);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [selectedTrigger, setSelectedTrigger] = useState<Record<string, unknown> | null>(null);
  const onNodeClick: NodeMouseHandler = useCallback((_event, node) => {
    setSelectedNodeId(node.id);

    if (isPrototype) {
      // Trigger node -> show trigger config panel
      if (node.type === 'triggerNode') {
        setSelectedTrigger(node.data as Record<string, unknown>);
        return;
      }
      // Agent node -> navigate to drill-in
      setSelectedTrigger(null);
      navigate(`/builder/${encodeURIComponent(protoSchema)}/${node.id}`);
      return;
    }

    // Production mode — trigger node → open config panel
    if (node.type === 'triggerNode' || node.id.startsWith('trigger-')) {
      setSelectedTrigger(node.data as Record<string, unknown>);
      return;
    }
    // Agent node — navigate to drill-in
    setSelectedTrigger(null);
    navigate(`/builder/${encodeURIComponent(selectedSchema)}/${node.id}`);
  }, [handleSelect, addToast, isPrototype, navigate, protoSchema, selectedSchema]);

  const onPaneContextMenu = useCallback((event: MouseEvent | React.MouseEvent) => {
    event.preventDefault();
    const bounds = reactFlowRef.current?.getBoundingClientRect();
    const x = event.clientX;
    const y = event.clientY;
    const canvasX = bounds ? x - bounds.left : x;
    const canvasY = bounds ? y - bounds.top : y;
    setContextMenu({ x, y, canvasX, canvasY });
    setNodeMenu(null);
  }, [reactFlowRef]);

  const onPaneClick = useCallback(() => {
    if (contextMenu) setContextMenu(null);
    if (edgeMenu) setEdgeMenu(null);
    if (nodeMenu) setNodeMenu(null);
    if (selectedAgent) setSelectedAgent(null);
    setSelectedNodeId(null);
  }, [contextMenu, edgeMenu, nodeMenu, selectedAgent, setSelectedAgent]);

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
          const d = n.data as Record<string, unknown>;
          return { ...n, data: { ...d, spawnCount: (updated.can_spawn ?? []).length } };
        }),
      );
      if (selectedAgent?.name === source) setSelectedAgent(updated);
      showSavedIndicator('saved');
    } catch (err) {
      addToast(`Failed to remove connection: ${err instanceof Error ? err.message : 'Unknown error'}`, 'error');
      setSavedIndicator(null);
    }
  }, [edgeMenu, selectedAgent, setEdges, setNodes, addToast, agentsCache, showSavedIndicator, setSavedIndicator, setSelectedAgent]);

  return {
    contextMenu,
    setContextMenu,
    nodeMenu,
    setNodeMenu,
    edgeMenu,
    setEdgeMenu,
    selectedNodeId,
    setSelectedNodeId,
    selectedTrigger,
    setSelectedTrigger,
    onNodeClick,
    onPaneContextMenu,
    onPaneClick,
    onEdgeContextMenu,
    onNodeContextMenu,
    handleDeleteEdge,
  };
}
