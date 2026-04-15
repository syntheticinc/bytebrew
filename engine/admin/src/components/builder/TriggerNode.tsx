import { Handle, Position } from '@xyflow/react';
import type { NodeProps } from '@xyflow/react';

export interface TriggerNodeData {
  id: string;
  title: string;
  type: 'cron' | 'webhook' | 'chat';
  // Type-specific config (collapsed per V2 §4.1). Surfaces as a flat object on
  // the node data so rendering can read config.schedule / config.webhook_path
  // without unwrapping a nested `config` prop.
  schedule?: string;
  webhook_path?: string;
  enabled: boolean;
  agentName?: string;
  [key: string]: unknown;
}

// ─── Icons ────────────────────────────────────────────────────────────────────

function CronIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="12" cy="12" r="10" />
      <polyline points="12 6 12 12 16 14" />
    </svg>
  );
}

function WebhookIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" />
    </svg>
  );
}

function ChatIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z" />
    </svg>
  );
}

// ─── Component ────────────────────────────────────────────────────────────────

export default function TriggerNode({ data, selected }: NodeProps) {
  const d = data as TriggerNodeData;
  const isCron = d.type === 'cron';
  const isChat = d.type === 'chat';

  return (
    <div
      className={`
        relative min-w-[170px] max-w-[210px] rounded-card transition-all duration-150 select-none
        ${selected
          ? 'shadow-[0_0_0_2px_rgba(168,85,247,0.35)] border border-purple-500/70'
          : 'border border-purple-500/25'
        }
      `}
      style={{ background: '#1A1A2E' }}
    >
      {/* Header */}
      <div
        className={`px-3 py-2 flex items-center gap-2 border-b ${
          selected ? 'border-purple-500/30' : 'border-purple-500/15'
        }`}
      >
        <span className={isCron ? 'text-amber-400' : isChat ? 'text-emerald-400' : 'text-purple-400'}>
          {isCron ? <CronIcon /> : isChat ? <ChatIcon /> : <WebhookIcon />}
        </span>
        <span className="text-sm font-semibold text-brand-light truncate">{d.title.replace(/<[^>]*>/g, '')}</span>
      </div>

      {/* Details — apply opacity only to body when disabled */}
      <div className={`px-3 py-2 space-y-1 ${d.enabled ? '' : 'opacity-50'}`}>
        {/* Type badge */}
        <div className="flex items-center gap-2">
          <span
            className={`px-1.5 py-0.5 rounded text-[10px] font-medium ${
              isCron
                ? 'bg-amber-500/15 text-amber-400'
                : isChat
                  ? 'bg-emerald-500/15 text-emerald-400'
                  : 'bg-purple-500/15 text-purple-400'
            }`}
          >
            {d.type}
          </span>
        </div>

        {/* Schedule or path */}
        {isCron && d.schedule && (
          <div className="text-[11px] text-brand-shade2 font-mono truncate" title={d.schedule}>
            {d.schedule}
          </div>
        )}
        {!isCron && d.webhook_path && (
          <div className="text-[11px] text-brand-shade2 font-mono truncate" title={d.webhook_path}>
            {d.webhook_path}
          </div>
        )}
      </div>

      {/* Disabled badge — full opacity, outside faded body */}
      {!d.enabled && (
        <div className="px-3 pb-1.5">
          <span className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-brand-shade3/15 text-brand-shade3">
            disabled
          </span>
        </div>
      )}

      <Handle
        type="source"
        position={Position.Bottom}
        className="!w-2.5 !h-2.5 !border-brand-dark transition-colors hover:!bg-brand-accent"
        style={{ background: isCron ? '#F59E0B' : isChat ? '#10B981' : '#A855F7' }}
      />
    </div>
  );
}
