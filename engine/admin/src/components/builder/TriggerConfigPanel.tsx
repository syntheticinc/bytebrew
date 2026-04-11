import React, { useState } from 'react';
import type { Node, Edge } from '@xyflow/react';
import CronScheduler from '../CronScheduler';
import ConfirmDialog from '../ConfirmDialog';
import { api } from '../../api/client';

interface TriggerConfigPanelProps {
  trigger: Record<string, unknown>;
  setTrigger: React.Dispatch<React.SetStateAction<Record<string, unknown> | null>>;
  setNodes: React.Dispatch<React.SetStateAction<Node[]>>;
  setEdges: React.Dispatch<React.SetStateAction<Edge[]>>;
  isPrototype: boolean;
  addToast: (message: string, type: 'success' | 'error' | 'info' | 'warning') => void;
}

export default function TriggerConfigPanel({ trigger, setTrigger, setNodes, setEdges, isPrototype, addToast }: TriggerConfigPanelProps) {
  const [saving, setSaving] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const triggerId = trigger.id as number;
  const triggerNodeId = `trigger-${triggerId}`;

  function updateField(field: string, value: unknown) {
    setTrigger((prev) => prev ? { ...prev, [field]: value } : prev);
    // Update canvas node data in real-time
    setNodes((nds) => nds.map((n) => {
      if (n.id !== triggerNodeId) return n;
      return { ...n, data: { ...n.data, [field]: value } };
    }));
  }

  async function handleSave() {
    if (isPrototype) return;
    setSaving(true);
    try {
      await api.updateTrigger(triggerId, {
        type: trigger.type as string,
        title: trigger.title as string,
        schedule: (trigger.schedule as string) || '',
        webhook_path: (trigger.webhook_path as string) || '',
        description: (trigger.description as string) || '',
        enabled: trigger.enabled !== false,
      });
      addToast('Trigger saved', 'success');
    } catch (err) {
      addToast(`Failed to save: ${err instanceof Error ? err.message : 'Unknown error'}`, 'error');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (isPrototype) return;
    setDeleting(true);
    try {
      await api.deleteTrigger(triggerId);
      // Remove node and its edges from canvas
      setNodes((nds) => nds.filter((n) => n.id !== triggerNodeId));
      setEdges((eds) => eds.filter((e) => e.source !== triggerNodeId));
      setTrigger(null);
      addToast('Trigger deleted', 'success');
    } catch (err) {
      addToast(`Failed to delete: ${err instanceof Error ? err.message : 'Unknown error'}`, 'error');
    } finally {
      setDeleting(false);
      setShowDeleteConfirm(false);
    }
  }

  const triggerType = (trigger.type as string) || 'webhook';

  return (
    <div className="w-80 border-l border-brand-shade3/10 bg-brand-dark-surface flex flex-col shrink-0 overflow-y-auto">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-brand-shade3/10">
        <h3 className="text-sm font-semibold text-brand-light font-mono">Trigger Configuration</h3>
        <button onClick={() => setTrigger(null)} className="text-brand-shade3 hover:text-brand-light p-1">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 6L6 18M6 6l12 12" /></svg>
        </button>
      </div>

      {/* Fields */}
      <div className="p-4 space-y-4 flex-1">
        <div>
          <label className="block text-xs text-brand-shade3 mb-1 font-mono">Title</label>
          <input
            className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light font-mono focus:outline-none focus:border-brand-accent transition-colors"
            value={String(trigger.title ?? '')}
            onChange={(e) => updateField('title', e.target.value)}
          />
        </div>

        <div>
          <label className="block text-xs text-brand-shade3 mb-1 font-mono">Type</label>
          <select
            className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light font-mono focus:outline-none focus:border-brand-accent transition-colors"
            value={triggerType}
            onChange={(e) => updateField('type', e.target.value)}
          >
            <option value="webhook">Webhook</option>
            <option value="cron">Cron Schedule</option>
            <option value="chat">Chat</option>
          </select>
        </div>

        {triggerType === 'cron' && (
          <div>
            <label className="block text-xs text-brand-shade3 mb-1 font-mono">Schedule</label>
            <CronScheduler
              value={String(trigger.schedule ?? '')}
              onChange={(cron) => updateField('schedule', cron)}
            />
          </div>
        )}

        {triggerType === 'webhook' && (
          <div>
            <label className="block text-xs text-brand-shade3 mb-1 font-mono">Webhook Path</label>
            <input
              className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light font-mono focus:outline-none focus:border-brand-accent transition-colors"
              value={String(trigger.webhook_path ?? '')}
              onChange={(e) => updateField('webhook_path', e.target.value)}
            />
            <p className="mt-1 text-xs text-brand-shade3">POST requests to this path will trigger the agent</p>
          </div>
        )}

        <div>
          <label className="block text-xs text-brand-shade3 mb-1 font-mono">Description</label>
          <textarea
            className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light font-mono focus:outline-none focus:border-brand-accent transition-colors resize-none"
            rows={3}
            value={String(trigger.description ?? '')}
            onChange={(e) => updateField('description', e.target.value)}
            placeholder="Optional description..."
          />
        </div>

        <div>
          <label className="flex items-center gap-2 text-sm text-brand-shade2 cursor-pointer select-none">
            <input
              type="checkbox"
              className="accent-brand-accent"
              checked={trigger.enabled !== false}
              onChange={(e) => updateField('enabled', e.target.checked)}
            />
            Enabled
          </label>
          <p className="mt-1 text-xs text-brand-shade3">Disabled triggers will not fire</p>
        </div>

        <div>
          <label className="block text-xs text-brand-shade3 mb-1 font-mono">Target Agent</label>
          <input
            className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light font-mono opacity-60 cursor-not-allowed"
            value={String(trigger.agentName ?? 'Not connected')}
            readOnly
          />
          <p className="mt-1 text-xs text-brand-shade3">Drag edge from trigger to agent on canvas</p>
        </div>
      </div>

      {/* Footer — Save + Delete */}
      {!isPrototype && (
        <div className="px-4 py-3 border-t border-brand-shade3/10 flex items-center gap-2">
          <button
            onClick={handleSave}
            disabled={saving}
            className="flex-1 px-4 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving...' : 'Save'}
          </button>
          <button
            onClick={() => setShowDeleteConfirm(true)}
            className="px-4 py-2 text-red-400 border border-red-500/30 rounded-btn text-sm font-medium hover:bg-red-500/10 transition-colors"
          >
            Delete
          </button>
        </div>
      )}

      <ConfirmDialog
        open={showDeleteConfirm}
        onClose={() => setShowDeleteConfirm(false)}
        onConfirm={handleDelete}
        title="Delete Trigger"
        message={`Delete "${trigger.title}"? This action cannot be undone.`}
        confirmLabel="Delete"
        loading={deleting}
        variant="danger"
      />
    </div>
  );
}
