import { Handle, Position } from '@xyflow/react';
import type { NodeProps } from '@xyflow/react';

export interface AgentNodeData {
  name: string;
  modelName: string;
  toolsCount: number;
  spawnCount: number;
  confirmCount: number;
  lifecycle: string;
  isSystem?: boolean;
  state?: 'ready' | 'running' | 'blocked' | 'degraded' | 'finished';
  isNew?: boolean;
  isRunning?: boolean;
  onSelect: (name: string) => void;
  onDelete: (name: string) => void;
  [key: string]: unknown;
}

const stateClasses: Record<NonNullable<AgentNodeData['state']>, string> = {
  ready: 'bg-status-active',
  running: 'bg-status-active animate-pulse',
  blocked: 'bg-brand-accent',
  degraded: 'bg-amber-400',
  finished: 'bg-brand-shade3',
};

export default function AgentNode({ data, selected }: NodeProps) {
  const d = data as AgentNodeData;

  const isSpawn = d.lifecycle === 'spawn';

  return (
    <div
      className={`
        relative border rounded-card min-w-[180px] max-w-[220px]
        transition-all duration-150 cursor-pointer select-none
        ${isSpawn ? 'border-dashed' : ''}
        ${selected
          ? 'border-brand-accent shadow-[0_0_0_2px_rgba(215,81,62,0.3)]'
          : isSpawn
            ? 'border-brand-shade3/55 hover:border-brand-shade3/80'
            : 'border-brand-shade3/30 hover:border-brand-shade3/60'
        }
        ${d.isNew ? 'animate-fade-in' : ''}
        ${d.isRunning ? 'animate-pulse-glow' : ''}
      `}
      style={{ background: isSpawn ? '#1A1A1A' : '#1F1F1F' }}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!w-3 !h-3 !bg-transparent !border-2 !border-brand-shade2 transition-colors"
      />

      {/* Header */}
      <div className={`px-3 py-2 border-b flex items-center gap-2 ${isSpawn ? 'border-dashed' : ''} ${selected ? 'border-brand-accent/30' : 'border-brand-shade3/20'}`}>
        <span className="text-brand-shade3 flex-shrink-0">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
            <rect x="4" y="4" width="16" height="16" rx="2" />
            <rect x="9" y="9" width="6" height="6" rx="1" />
            <path d="M9 1v3M15 1v3M9 20v3M15 20v3M20 9h3M20 15h3M1 9h3M1 15h3" />
          </svg>
        </span>
        {d.state && (
          <span className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${stateClasses[d.state]}`} />
        )}
        <span className="text-sm font-semibold text-brand-light truncate">{d.name}</span>
        {d.isSystem && (
          <span className="ml-auto px-1.5 py-0.5 rounded text-[10px] font-medium bg-brand-shade3/15 text-brand-shade3 whitespace-nowrap">
            system
          </span>
        )}
        {isSpawn && !d.isSystem && (
          <span className="ml-auto px-1.5 py-0.5 rounded text-[10px] font-medium bg-blue-500/15 text-blue-400 whitespace-nowrap">
            sub-agent
          </span>
        )}
      </div>

      {/* Details */}
      <div className="px-3 py-2 space-y-1">
        {d.modelName && (
          <div className="flex items-center gap-1.5">
            <span className="text-brand-shade3 flex-shrink-0">
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z" />
              </svg>
            </span>
            <span className="text-xs text-brand-shade2 truncate">{d.modelName}</span>
          </div>
        )}

        <div className="flex items-center gap-3 flex-wrap">
          {d.toolsCount > 0 && (
            <span className="text-xs text-brand-shade3">
              <span className="text-brand-shade2 font-medium">{d.toolsCount}</span> tools
            </span>
          )}
          {d.spawnCount > 0 && (
            <span className="text-xs text-brand-shade3">
              <span className="text-blue-400 font-medium">{d.spawnCount}</span> spawns
            </span>
          )}
        </div>

        {d.confirmCount > 0 && (
          <div className="flex items-center gap-1 text-xs text-amber-400">
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
              <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
              <line x1="12" y1="9" x2="12" y2="13" />
              <line x1="12" y1="17" x2="12.01" y2="17" />
            </svg>
            {d.confirmCount} confirm_before
          </div>
        )}
      </div>

      {/* Actions */}
      <div
        className={`px-3 py-1.5 border-t flex items-center gap-1 ${selected ? 'border-brand-accent/30' : 'border-brand-shade3/20'}`}
        onClick={(e) => e.stopPropagation()}
      >
        <button
          onClick={() => d.onSelect(d.name)}
          className="flex-1 py-0.5 text-[11px] text-brand-shade2 hover:text-brand-accent hover:bg-brand-shade3/10 rounded transition-colors"
          title="Open side panel"
        >
          Details
        </button>
        <div className="w-px h-3 bg-brand-shade3/20" />
        <button
          onClick={() => d.onDelete(d.name)}
          className="flex-1 py-0.5 text-[11px] text-brand-shade3 hover:text-red-400 hover:bg-brand-shade3/10 rounded transition-colors"
          title="Delete agent"
        >
          Delete
        </button>
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        className="!w-3 !h-3 !bg-[#A0A090] !border-brand-dark hover:!bg-brand-accent transition-colors"
      />
    </div>
  );
}
