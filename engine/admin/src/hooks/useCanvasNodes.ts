import { useCallback, useState } from 'react';
import type { Node, Edge } from '@xyflow/react';
import { api } from '../api/client';
import type { AgentDetail, Model, CreateAgentRequest, CreateTriggerRequest, Trigger } from '../types';
import type { AgentNodeData } from '../components/builder/AgentNode';

// ─── Constants ────────────────────────────────────────────────────────────────

const NODE_WIDTH = 210;
const NODE_HEIGHT = 135;
const TRIGGER_NODE_WIDTH = 190;
const TRIGGER_NODE_HEIGHT = 95;

// ─── Helpers ──────────────────────────────────────────────────────────────────

export function generateAgentName(existing: string[]): string {
  let n = 1;
  while (existing.includes(`new-agent-${n}`)) n++;
  return `new-agent-${n}`;
}

export function generateTriggerName(existing: string[]): string {
  let n = 1;
  while (existing.includes(`new-trigger-${n}`)) n++;
  return `new-trigger-${n}`;
}

function generateShortId(): string {
  return Math.random().toString(36).substring(2, 10);
}

// ─── Node factory ─────────────────────────────────────────────────────────────

export function makeNode(
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

export function makeTriggerNode(
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

// ─── Export constants for dagre ────────────────────────────────────────────────

export { NODE_WIDTH, NODE_HEIGHT, TRIGGER_NODE_WIDTH, TRIGGER_NODE_HEIGHT };

// ─── Hook ─────────────────────────────────────────────────────────────────────

interface UseCanvasNodesParams {
  nodes: Node[];
  setNodes: React.Dispatch<React.SetStateAction<Node[]>>;
  setEdges: React.Dispatch<React.SetStateAction<Edge[]>>;
  agentsCache: React.MutableRefObject<Map<string, AgentDetail>>;
  agentsListRef: React.MutableRefObject<AgentDetail[]>;
  modelsRef: React.MutableRefObject<Model[]>;
  addToast: (message: string, type: 'success' | 'error' | 'info' | 'warning') => void;
  selectedAgent: AgentDetail | null;
  setSelectedAgent: React.Dispatch<React.SetStateAction<AgentDetail | null>>;
  isPrototype: boolean;
  handleSelect: (name: string) => void;
  handleDeleteRequest: (name: string) => void;
  /** When set, delete = remove from schema (not delete agent). */
  currentSchemaId?: number | null;
}

export function useCanvasNodes({
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
  currentSchemaId,
}: UseCanvasNodesParams) {
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [deleteError, setDeleteError] = useState('');
  const [deleting, setDeleting] = useState(false);

  const confirmDelete = useCallback(async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    setDeleteError('');
    try {
      if (currentSchemaId != null) {
        // Schema-scoped: remove agent from schema only (agent stays in DB)
        await api.removeAgentFromSchema(currentSchemaId, deleteTarget);
      } else {
        // No schema context: delete agent globally
        await api.deleteAgent(deleteTarget);
        agentsCache.current.delete(deleteTarget);
      }

      setNodes((nds) => nds.filter((n) => n.id !== deleteTarget));
      setEdges((eds) => eds.filter((e) => e.source !== deleteTarget && e.target !== deleteTarget));

      if (selectedAgent?.name === deleteTarget) setSelectedAgent(null);
      setDeleteTarget(null);
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Delete failed');
    } finally {
      setDeleting(false);
    }
  }, [deleteTarget, selectedAgent, setNodes, setEdges, agentsCache, setSelectedAgent, currentSchemaId]);

  const handleInstantAgentCreate = useCallback(async (canvasPosition?: { x: number; y: number }) => {
    // Use both canvas names and cached DB names for better uniqueness
    const canvasNames = nodes.filter((n) => n.type === 'agentNode').map((n) => n.id);
    const dbNames = agentsListRef.current.map((a) => a.name);
    const allKnownNames = [...new Set([...canvasNames, ...dbNames])];
    let name = generateAgentName(allKnownNames);
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
          isNew: true,
          onSelect: handleSelect,
          onDelete: handleDeleteRequest,
        } satisfies AgentNodeData,
      };
      setNodes((nds) => [...nds, newNode]);
      setTimeout(() => {
        setNodes((nds) => nds.map((n) => n.id === name ? { ...n, data: { ...n.data, isNew: false } } : n));
      }, 1000);
      addToast(`Agent "${name}" created — click to configure`, 'success');
      return;
    }

    // Production mode: create via API with retry on name collision
    const maxAttempts = 10;
    for (let attempt = 0; attempt < maxAttempts; attempt++) {
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

        if (currentSchemaId != null) {
          await api.addAgentToSchema(currentSchemaId, created.name);
        }

        agentsCache.current.set(created.name, created);
        agentsListRef.current = [...agentsListRef.current, created];

        const modelMap = new Map(modelsRef.current.map((m) => [m.id, m.name]));
        const newNode = makeNode(created, modelMap, pos, handleSelect, handleDeleteRequest);
        newNode.data = { ...newNode.data, isNew: true };
        setNodes((nds) => [...nds, newNode]);
        setTimeout(() => {
          setNodes((nds) => nds.map((n) => n.id === created.name ? { ...n, data: { ...n.data, isNew: false } } : n));
        }, 1000);
        addToast(`Agent "${created.name}" created — click to configure`, 'success');
        return; // success — exit loop
      } catch (err) {
        const msg = err instanceof Error ? err.message : '';
        if (msg.toLowerCase().includes('already_exists') || msg.toLowerCase().includes('already exists')) {
          // Name collision — increment and retry
          allKnownNames.push(name);
          name = generateAgentName(allKnownNames);
          continue;
        }
        // Non-collision error — stop immediately
        addToast(`Failed to create agent: ${msg || 'Unknown error'}`, 'error');
        return;
      }
    }
    addToast('Failed to create agent: too many name collisions', 'error');
  }, [nodes, isPrototype, setNodes, agentsCache, agentsListRef, modelsRef, addToast, handleSelect, handleDeleteRequest, currentSchemaId]);

  const handleInstantTriggerCreate = useCallback(async (canvasPosition?: { x: number; y: number }, triggerType: 'webhook' | 'cron' | 'chat' = 'webhook') => {
    const existingNames = nodes.filter((n) => n.type === 'triggerNode').map((n) => (n.data as Record<string, unknown>).title as string);
    const defaultTitles: Record<string, string> = { webhook: 'Webhook Trigger', cron: 'Cron Trigger', chat: 'Chat Trigger' };
    const baseTitle = defaultTitles[triggerType] || 'Trigger';
    const title = existingNames.includes(baseTitle) ? generateTriggerName(existingNames) : baseTitle;
    const webhookPath = triggerType === 'webhook' ? `/webhook/${generateShortId()}` : '';
    const pos = canvasPosition ?? { x: Math.random() * 200 + 50, y: Math.random() * 200 + 50 };

    if (isPrototype) {
      const mockId = Date.now();
      const newNode = makeTriggerNode(
        { id: mockId, type: triggerType, title, webhook_path: webhookPath, enabled: true, agent_name: '', schedule: '', description: '', created_at: new Date().toISOString() } as Trigger,
        pos,
      );
      setNodes((nds) => [...nds, newNode]);
      addToast(`Trigger "${title}" created — click to configure`, 'success');
      return;
    }

    // Production mode: create via API
    try {
      const data: CreateTriggerRequest = {
        type: triggerType,
        title,
        enabled: true,
        ...(triggerType === 'webhook' ? { webhook_path: webhookPath } : {}),
        ...(currentSchemaId ? { schema_id: currentSchemaId } : {}),
      };
      const created = await api.createTrigger(data);
      const newNode = makeTriggerNode(created, pos);
      setNodes((nds) => [...nds, newNode]);
      addToast(`Trigger "${created.title}" created — click to configure`, 'success');
    } catch (err) {
      addToast(`Failed to create trigger: ${err instanceof Error ? err.message : 'Unknown error'}`, 'error');
    }
  }, [nodes, isPrototype, setNodes, addToast, currentSchemaId]);

  return {
    handleInstantAgentCreate,
    handleInstantTriggerCreate,
    confirmDelete,
    deleteTarget,
    deleteError,
    deleting,
    setDeleteTarget,
    setDeleteError,
  };
}
