import React from 'react';
import type { Node } from '@xyflow/react';
import CronScheduler from '../CronScheduler';

interface TriggerConfigPanelProps {
  trigger: Record<string, unknown>;
  setTrigger: React.Dispatch<React.SetStateAction<Record<string, unknown> | null>>;
  setNodes: React.Dispatch<React.SetStateAction<Node[]>>;
}

export default function TriggerConfigPanel({ trigger, setTrigger, setNodes }: TriggerConfigPanelProps) {
  return (
    <div className="w-80 border-l border-brand-shade3/10 bg-brand-dark-surface flex flex-col shrink-0 overflow-y-auto">
      <div className="flex items-center justify-between px-4 py-3 border-b border-brand-shade3/10">
        <h3 className="text-sm font-semibold text-brand-light font-mono">Trigger Configuration</h3>
        <button onClick={() => setTrigger(null)} className="text-brand-shade3 hover:text-brand-light p-1"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 6L6 18M6 6l12 12" /></svg></button>
      </div>
      <div className="p-4 space-y-4">
        <div>
          <label className="block text-xs text-brand-shade3 mb-1 font-mono">Title</label>
          <input className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light font-mono opacity-60 cursor-not-allowed" value={String(trigger.title ?? '')} readOnly />
        </div>
        <div>
          <label className="block text-xs text-brand-shade3 mb-1 font-mono">Type</label>
          <select className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light font-mono opacity-60 cursor-not-allowed" value={String(trigger.type ?? 'webhook')} disabled>
            <option value="webhook">Webhook</option>
            <option value="cron">Cron Schedule</option>
          </select>
        </div>
        {trigger.type === 'cron' && (
          <div>
            <label className="block text-xs text-brand-shade3 mb-1 font-mono">Schedule</label>
            <CronScheduler
              value={String(trigger.schedule ?? '')}
              onChange={(cron) => {
                setTrigger((prev) => prev ? { ...prev, schedule: cron } : prev);
                // Update the node data in canvas
                setNodes((nds) => nds.map((n) => {
                  if (n.type !== 'triggerNode') return n;
                  const d = n.data as Record<string, unknown>;
                  if (d.id !== trigger.id) return n;
                  return { ...n, data: { ...d, schedule: cron } };
                }));
              }}
            />
          </div>
        )}
        {trigger.type === 'webhook' && (
          <div>
            <label className="block text-xs text-brand-shade3 mb-1 font-mono">Webhook Path</label>
            <input className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light font-mono opacity-60 cursor-not-allowed" value={String(trigger.webhook_path ?? '')} readOnly />
            <p className="mt-1 text-xs text-brand-shade3">POST requests to this path will trigger the agent</p>
          </div>
        )}
        <div>
          <label className="flex items-center gap-2 text-sm text-brand-shade2 cursor-pointer select-none">
            <input type="checkbox" className="accent-brand-accent" checked={trigger.enabled !== false} disabled />
            Enabled
          </label>
          <p className="mt-1 text-xs text-brand-shade3">Disabled triggers will not fire</p>
        </div>
        <div>
          <label className="block text-xs text-brand-shade3 mb-1 font-mono">Target Agent</label>
          <input className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light font-mono opacity-60 cursor-not-allowed" value={String(trigger.agentName ?? '')} readOnly />
          <p className="mt-1 text-xs text-brand-shade3">Entry agent that receives trigger events</p>
        </div>
      </div>
    </div>
  );
}
