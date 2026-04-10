import { useCallback } from 'react';
import type { Node, Edge, Connection, IsValidConnection } from '@xyflow/react';
import { addEdge, MarkerType } from '@xyflow/react';
import { api } from '../api/client';
import type { AgentDetail } from '../types';
import type { AgentNodeData } from '../components/builder/AgentNode';

// ─── Edge factory ─────────────────────────────────────────────────────────────

export function makeEdge(source: string, target: string): Edge {
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

export function makeTriggerEdge(triggerNodeId: string, agentName: string): Edge {
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

// ─── Hook ─────────────────────────────────────────────────────────────────────

interface UseCanvasEdgesParams {
  setNodes: React.Dispatch<React.SetStateAction<Node[]>>;
  setEdges: React.Dispatch<React.SetStateAction<Edge[]>>;
  agentsCache: React.MutableRefObject<Map<string, AgentDetail>>;
  addToast: (message: string, type: 'success' | 'error' | 'info' | 'warning') => void;
  showSavedIndicator: (state: 'saved' | 'saving') => void;
  setSavedIndicator: React.Dispatch<React.SetStateAction<'saved' | 'saving' | null>>;
  isPrototype: boolean;
  selectedAgent: AgentDetail | null;
  setSelectedAgent: React.Dispatch<React.SetStateAction<AgentDetail | null>>;
}

export function useCanvasEdges({
  setNodes,
  setEdges,
  agentsCache,
  addToast,
  showSavedIndicator,
  setSavedIndicator,
  isPrototype,
  selectedAgent,
  setSelectedAgent,
}: UseCanvasEdgesParams) {
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

      // Trigger → agent: set routing target via API (canvas edge = routing config)
      if (source.startsWith('trigger-')) {
        const triggerId = parseInt(source.replace('trigger-', ''), 10);
        showSavedIndicator('saving');
        try {
          await api.setTriggerTarget(triggerId, target);
          setEdges((eds) => addEdge(makeTriggerEdge(source, target), eds));
          showSavedIndicator('saved');
        } catch (err) {
          addToast(`Failed to connect trigger: ${err instanceof Error ? err.message : 'Unknown error'}`, 'error');
          setSavedIndicator(null);
        }
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
    [selectedAgent, setNodes, setEdges, addToast, isPrototype, agentsCache, showSavedIndicator, setSavedIndicator, setSelectedAgent],
  );

  const isValidConnection = useCallback<IsValidConnection>((connection) => {
    if (!connection.source || !connection.target) return false;
    if (connection.source === connection.target) return false;
    return true;
  }, []);

  const onEdgesDelete = useCallback(
    async (deletedEdges: Edge[]) => {
      // Prototype mode: just remove from local state, no API calls
      if (isPrototype) return;

      for (const edge of deletedEdges) {
        // Trigger → agent edge: clear routing target via API
        if (edge.id.startsWith('trigger:')) {
          const triggerId = parseInt(edge.source.replace('trigger-', ''), 10);
          showSavedIndicator('saving');
          try {
            await api.clearTriggerTarget(triggerId);
            showSavedIndicator('saved');
          } catch (err) {
            setEdges((eds) => [...eds, edge]);
            addToast(`Failed to disconnect trigger: ${err instanceof Error ? err.message : 'Unknown error'}`, 'error');
            setSavedIndicator(null);
          }
          continue;
        }

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
    [selectedAgent, setNodes, setEdges, addToast, isPrototype, agentsCache, showSavedIndicator, setSavedIndicator, setSelectedAgent],
  );

  return { onConnect, onEdgesDelete, isValidConnection };
}
