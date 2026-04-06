import { MarkerType, type Node, type Edge } from '@xyflow/react';
import type { AgentNodeData } from '../components/builder/AgentNode';
import type { TriggerNodeData } from '../components/builder/TriggerNode';
import type { GateNodeData } from '../components/builder/GateNode';

export type SchemaName = string;

export function createMockSchemas(
  onSelect: (name: string) => void,
  onDelete: (name: string) => void,
): Record<SchemaName, { nodes: Node[]; edges: Edge[] }> {
  return {
    'Support Schema': {
      nodes: [
        // Triggers
        {
          id: 'trigger-1',
          type: 'triggerNode',
          position: { x: 80, y: 100 },
          data: {
            id: 1,
            title: 'user-message',
            type: 'webhook',
            webhook_path: '/webhook/support',
            enabled: true,
            agentName: 'classifier',
          } satisfies TriggerNodeData,
        },
        {
          id: 'trigger-2',
          type: 'triggerNode',
          position: { x: 80, y: 220 },
          data: {
            id: 2,
            title: 'daily-digest',
            type: 'cron',
            schedule: '0 9 * * *',
            enabled: true,
            agentName: 'classifier',
          } satisfies TriggerNodeData,
        },
        // Agents
        {
          id: 'classifier',
          type: 'agentNode',
          position: { x: 320, y: 60 },
          data: {
            name: 'classifier',
            modelName: 'claude-haiku-3',
            toolsCount: 3,
            spawnCount: 0,
            confirmCount: 0,
            lifecycle: 'persistent',
            state: 'ready',
            onSelect,
            onDelete,
          } satisfies AgentNodeData,
        },
        {
          id: 'support-agent',
          type: 'agentNode',
          position: { x: 560, y: 0 },
          data: {
            name: 'support-agent',
            modelName: 'claude-sonnet-3.7',
            toolsCount: 8,
            spawnCount: 1,
            confirmCount: 2,
            lifecycle: 'persistent',
            state: 'running',
            onSelect,
            onDelete,
          } satisfies AgentNodeData,
        },
        {
          id: 'escalation',
          type: 'agentNode',
          position: { x: 560, y: 200 },
          data: {
            name: 'escalation',
            modelName: 'claude-opus-4',
            toolsCount: 5,
            spawnCount: 0,
            confirmCount: 1,
            lifecycle: 'spawn',
            state: 'blocked',
            onSelect,
            onDelete,
          } satisfies AgentNodeData,
        },
        // Gate — positioned right of support-agent, handles: left=target, right=source
        {
          id: 'quality-gate',
          type: 'gateNode',
          position: { x: 820, y: 60 },
          data: {
            label: 'quality-check',
            conditionType: 'auto',
            conditionConfig: '{"type":"object","required":["approved"]}',
          } satisfies GateNodeData,
        },
      ],
      edges: [
        // Trigger → Entry agent: triggers (purple dashed)
        { id: 'te1', source: 'trigger-1', target: 'classifier', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 }, style: { stroke: '#A855F7', strokeWidth: 1.5, strokeDasharray: '6 3' }, label: 'triggers', labelStyle: { fill: '#A855F7', fontSize: 10 } },
        { id: 'te2', source: 'trigger-2', target: 'classifier', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 }, style: { stroke: '#A855F7', strokeWidth: 1.5, strokeDasharray: '6 3' }, label: 'triggers', labelStyle: { fill: '#A855F7', fontSize: 10 } },
        // classifier → support-agent: flow (green solid)
        { id: 'ae1', source: 'classifier', target: 'support-agent', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 }, label: 'flow', style: { stroke: '#4CAF50', strokeWidth: 1.5 }, labelStyle: { fill: '#4CAF50', fontSize: 10 } },
        // classifier → escalation: transfer (blue solid)
        { id: 'ae2', source: 'classifier', target: 'escalation', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 }, label: 'transfer', style: { stroke: '#4A9EFF', strokeWidth: 1.5 }, labelStyle: { fill: '#4A9EFF', fontSize: 10 } },
        // support-agent → quality-gate: flow (green solid) — enters gate from left
        { id: 'ae-gf1', source: 'support-agent', target: 'quality-gate', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 }, label: 'flow', style: { stroke: '#4CAF50', strokeWidth: 1.5 }, labelStyle: { fill: '#4CAF50', fontSize: 10 } },
        // support-agent → escalation: can_spawn (red solid)
        { id: 'ae-sp1', source: 'support-agent', target: 'escalation', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 }, label: 'can_spawn', style: { stroke: '#D7513E', strokeWidth: 1.5 }, labelStyle: { fill: '#D7513E', fontSize: 10 } },
        // quality-gate → support-agent: loop (orange dashed) — exits gate from right, curves back
        { id: 'ae-lp1', source: 'quality-gate', target: 'support-agent', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 }, label: 'loop', style: { stroke: '#F59E0B', strokeWidth: 1.5, strokeDasharray: '4 4' }, labelStyle: { fill: '#F59E0B', fontSize: 10 } },
      ],
    },

    'Dev Schema': {
      nodes: [
        // Triggers
        {
          id: 'trigger-4',
          type: 'triggerNode',
          position: { x: 80, y: 80 },
          data: {
            id: 4,
            title: 'code-request',
            type: 'webhook',
            webhook_path: '/webhook/code',
            enabled: true,
            agentName: 'dev-router',
          } satisfies TriggerNodeData,
        },
        {
          id: 'trigger-5',
          type: 'triggerNode',
          position: { x: 80, y: 200 },
          data: {
            id: 5,
            title: 'nightly-review',
            type: 'cron',
            schedule: '0 2 * * *',
            enabled: true,
            agentName: 'review-agent',
          } satisfies TriggerNodeData,
        },
        // Agents
        {
          id: 'dev-router',
          type: 'agentNode',
          position: { x: 320, y: 60 },
          data: {
            name: 'dev-router',
            modelName: 'claude-haiku-3',
            toolsCount: 2,
            spawnCount: 0,
            confirmCount: 0,
            lifecycle: 'persistent',
            state: 'ready',
            onSelect,
            onDelete,
          } satisfies AgentNodeData,
        },
        {
          id: 'code-agent',
          type: 'agentNode',
          position: { x: 560, y: 0 },
          data: {
            name: 'code-agent',
            modelName: 'claude-sonnet-3.7',
            toolsCount: 12,
            spawnCount: 0,
            confirmCount: 0,
            lifecycle: 'persistent',
            state: 'running',
            onSelect,
            onDelete,
          } satisfies AgentNodeData,
        },
        {
          id: 'review-agent',
          type: 'agentNode',
          position: { x: 560, y: 200 },
          data: {
            name: 'review-agent',
            modelName: 'claude-opus-4',
            toolsCount: 6,
            spawnCount: 0,
            confirmCount: 0,
            lifecycle: 'spawn',
            state: 'blocked',
            onSelect,
            onDelete,
          } satisfies AgentNodeData,
        },
        // Gate
        {
          id: 'lint-gate',
          type: 'gateNode',
          position: { x: 800, y: 40 },
          data: {
            label: 'lint-check',
            conditionType: 'auto',
            conditionConfig: '{"type":"object","required":["pass"]}',
          } satisfies GateNodeData,
        },
      ],
      edges: [
        // triggers (purple dashed)
        { id: 'te4', source: 'trigger-4', target: 'dev-router', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 }, style: { stroke: '#A855F7', strokeWidth: 1.5, strokeDasharray: '6 3' }, label: 'triggers', labelStyle: { fill: '#A855F7', fontSize: 10 } },
        { id: 'te5', source: 'trigger-5', target: 'review-agent', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 }, style: { stroke: '#A855F7', strokeWidth: 1.5, strokeDasharray: '6 3' }, label: 'triggers', labelStyle: { fill: '#A855F7', fontSize: 10 } },
        // dev-router → code-agent: flow (green solid)
        { id: 'ae3', source: 'dev-router', target: 'code-agent', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 }, label: 'flow', style: { stroke: '#4CAF50', strokeWidth: 1.5 }, labelStyle: { fill: '#4CAF50', fontSize: 10 } },
        // dev-router → review-agent: can_spawn (red solid)
        { id: 'ae4', source: 'dev-router', target: 'review-agent', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 }, label: 'can_spawn', style: { stroke: '#D7513E', strokeWidth: 1.5 }, labelStyle: { fill: '#D7513E', fontSize: 10 } },
        // code-agent → lint-gate: flow (green solid)
        { id: 'ae-gf2', source: 'code-agent', target: 'lint-gate', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 }, label: 'flow', style: { stroke: '#4CAF50', strokeWidth: 1.5 }, labelStyle: { fill: '#4CAF50', fontSize: 10 } },
        // lint-gate → code-agent: loop (orange dashed)
        { id: 'ae-lp2', source: 'lint-gate', target: 'code-agent', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 }, label: 'loop', style: { stroke: '#F59E0B', strokeWidth: 1.5, strokeDasharray: '4 4' }, labelStyle: { fill: '#F59E0B', fontSize: 10 } },
        // code-agent → review-agent: transfer (blue solid)
        { id: 'ae-tr1', source: 'code-agent', target: 'review-agent', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 }, label: 'transfer', style: { stroke: '#4A9EFF', strokeWidth: 1.5 }, labelStyle: { fill: '#4A9EFF', fontSize: 10 } },
      ],
    },

    'Sales Schema': {
      nodes: [
        // Triggers
        {
          id: 'trigger-6',
          type: 'triggerNode',
          position: { x: 80, y: 60 },
          data: {
            id: 6,
            title: 'lead-event',
            type: 'webhook',
            webhook_path: '/webhook/leads',
            enabled: true,
            agentName: 'lead-scorer',
          } satisfies TriggerNodeData,
        },
        {
          id: 'trigger-7',
          type: 'triggerNode',
          position: { x: 80, y: 180 },
          data: {
            id: 7,
            title: 'scoring-batch',
            type: 'cron',
            schedule: '0 */6 * * *',
            enabled: true,
            agentName: 'lead-scorer',
          } satisfies TriggerNodeData,
        },
        // Agents
        {
          id: 'lead-scorer',
          type: 'agentNode',
          position: { x: 320, y: 80 },
          data: {
            name: 'lead-scorer',
            modelName: 'claude-haiku-3',
            toolsCount: 4,
            spawnCount: 0,
            confirmCount: 0,
            lifecycle: 'persistent',
            state: 'ready',
            onSelect,
            onDelete,
          } satisfies AgentNodeData,
        },
        {
          id: 'outreach-agent',
          type: 'agentNode',
          position: { x: 560, y: 0 },
          data: {
            name: 'outreach-agent',
            modelName: 'claude-sonnet-3.7',
            toolsCount: 7,
            spawnCount: 0,
            confirmCount: 0,
            lifecycle: 'persistent',
            state: 'running',
            onSelect,
            onDelete,
          } satisfies AgentNodeData,
        },
        {
          id: 'closer-agent',
          type: 'agentNode',
          position: { x: 560, y: 200 },
          data: {
            name: 'closer-agent',
            modelName: 'claude-opus-4',
            toolsCount: 9,
            spawnCount: 0,
            confirmCount: 0,
            lifecycle: 'spawn',
            state: 'blocked',
            onSelect,
            onDelete,
          } satisfies AgentNodeData,
        },
        // Gate
        {
          id: 'score-gate',
          type: 'gateNode',
          position: { x: 800, y: 80 },
          data: {
            label: 'score-threshold',
            conditionType: 'auto',
            conditionConfig: '{"type":"object","required":["score","threshold"]}',
          } satisfies GateNodeData,
        },
      ],
      edges: [
        // triggers (purple dashed)
        { id: 'te6', source: 'trigger-6', target: 'lead-scorer', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 }, style: { stroke: '#A855F7', strokeWidth: 1.5, strokeDasharray: '6 3' }, label: 'triggers', labelStyle: { fill: '#A855F7', fontSize: 10 } },
        { id: 'te7', source: 'trigger-7', target: 'lead-scorer', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 }, style: { stroke: '#A855F7', strokeWidth: 1.5, strokeDasharray: '6 3' }, label: 'triggers', labelStyle: { fill: '#A855F7', fontSize: 10 } },
        // lead-scorer → outreach-agent: flow (green solid)
        { id: 'ae5', source: 'lead-scorer', target: 'outreach-agent', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 }, label: 'flow', style: { stroke: '#4CAF50', strokeWidth: 1.5 }, labelStyle: { fill: '#4CAF50', fontSize: 10 } },
        // lead-scorer → closer-agent: transfer (blue solid)
        { id: 'ae6', source: 'lead-scorer', target: 'closer-agent', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 }, label: 'transfer', style: { stroke: '#4A9EFF', strokeWidth: 1.5 }, labelStyle: { fill: '#4A9EFF', fontSize: 10 } },
        // outreach-agent → score-gate: flow (green solid)
        { id: 'ae-gf3', source: 'outreach-agent', target: 'score-gate', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 }, label: 'flow', style: { stroke: '#4CAF50', strokeWidth: 1.5 }, labelStyle: { fill: '#4CAF50', fontSize: 10 } },
        // score-gate → outreach-agent: loop (orange dashed)
        { id: 'ae-lp3', source: 'score-gate', target: 'outreach-agent', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 }, label: 'loop', style: { stroke: '#F59E0B', strokeWidth: 1.5, strokeDasharray: '4 4' }, labelStyle: { fill: '#F59E0B', fontSize: 10 } },
        // outreach-agent → closer-agent: can_spawn (red solid)
        { id: 'ae-sp2', source: 'outreach-agent', target: 'closer-agent', type: 'smoothstep', markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 }, label: 'can_spawn', style: { stroke: '#D7513E', strokeWidth: 1.5 }, labelStyle: { fill: '#D7513E', fontSize: 10 } },
      ],
    },
  };
}

export const SCHEMA_NAMES: SchemaName[] = ['Support Schema', 'Dev Schema', 'Sales Schema'];
