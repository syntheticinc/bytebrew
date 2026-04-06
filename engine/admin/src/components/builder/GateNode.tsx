import { Handle, Position } from '@xyflow/react';
import type { NodeProps } from '@xyflow/react';

export interface GateNodeData {
  label: string;
  conditionType: 'auto' | 'human' | 'llm' | 'all_completed';
  conditionConfig?: string;
  [key: string]: unknown;
}

const badgeLabel: Record<GateNodeData['conditionType'], string> = {
  auto: 'AUTO',
  human: 'HUMAN',
  llm: 'LLM',
  all_completed: 'JOIN',
};

export default function GateNode({ data, selected }: NodeProps) {
  const d = data as GateNodeData;
  const badge = badgeLabel[d.conditionType] ?? d.conditionType.toUpperCase();

  return (
    <div className="flex flex-col items-center">
      {/* Diamond container — handles inside same div so they align with tips */}
      <div className="relative" style={{ width: 100, height: 100 }}>
        <Handle
          type="target"
          position={Position.Top}
          className="!w-2.5 !h-2.5 !bg-transparent !border-2 !border-amber-500/50 transition-colors"
        />

        <svg
          width="100"
          height="100"
          viewBox="0 0 100 100"
          className="absolute inset-0 cursor-pointer select-none"
        >
          <polygon
            points="50,2 98,50 50,98 2,50"
            fill="#1A1A2E"
            stroke={selected ? '#F59E0B' : 'rgba(245,158,11,0.4)'}
            strokeWidth={selected ? 2 : 1.5}
            className="transition-all duration-150"
          />
          {selected && (
            <polygon
              points="50,2 98,50 50,98 2,50"
              fill="none"
              stroke="rgba(245,158,11,0.2)"
              strokeWidth="6"
            />
          )}
          <text
            x="50"
            y="52"
            textAnchor="middle"
            dominantBaseline="middle"
            fill="#F59E0B"
            fontSize="11"
            fontFamily="IBM Plex Mono, monospace"
            fontWeight="500"
          >
            {badge}
          </text>
        </svg>

        <Handle
          type="source"
          position={Position.Bottom}
          className="!w-2.5 !h-2.5 !bg-amber-500/70 !border-brand-dark transition-colors"
        />
      </div>

      {/* Label below */}
      {d.label && (
        <span className="mt-1 text-[11px] text-brand-shade2 font-mono truncate max-w-[120px] text-center">
          {d.label}
        </span>
      )}
    </div>
  );
}
