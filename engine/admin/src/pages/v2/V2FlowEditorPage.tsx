import { useState, useMemo } from 'react';
import { Link, useParams, useNavigate } from 'react-router-dom';
import {
  getAgentById,
  getFlowsForAgent,
  v2Sessions,
  type V2Flow,
  type V2FlowCheckpoint,
} from '../../mocks/v2';

function CheckpointCard({
  cp,
  index,
  isFirst,
  isLast,
  onEdit,
  onMove,
  onRemove,
}: {
  cp: V2FlowCheckpoint;
  index: number;
  isFirst: boolean;
  isLast: boolean;
  onEdit: () => void;
  onMove: (dir: 'up' | 'down') => void;
  onRemove: () => void;
}) {
  return (
    <div className="flex gap-3 relative">
      <div className="shrink-0 w-8 h-8 rounded-full bg-brand-dark border border-brand-shade3/40 flex items-center justify-center text-[11px] text-brand-light font-mono z-10">
        {index + 1}
      </div>
      <div className="flex-1 bg-brand-dark-surface border border-brand-shade3/15 rounded-card p-3">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0 flex-1">
            <div className="text-[13px] font-semibold text-brand-light">{cp.name}</div>
            <div className="text-[11px] text-brand-shade3 italic mt-1">Goal: {cp.goal}</div>
            <div className="text-[10px] text-brand-shade3/80 mt-1.5">
              <span className="text-emerald-400/80">✓</span> Success: {cp.successCriteria}
            </div>
          </div>
          <div className="flex items-center gap-1 shrink-0">
            <button
              onClick={() => onMove('up')}
              disabled={isFirst}
              className="w-6 h-6 flex items-center justify-center rounded hover:bg-brand-shade3/15 text-brand-shade3 hover:text-brand-light disabled:opacity-30 disabled:cursor-not-allowed"
              title="Move up"
            >
              ↑
            </button>
            <button
              onClick={() => onMove('down')}
              disabled={isLast}
              className="w-6 h-6 flex items-center justify-center rounded hover:bg-brand-shade3/15 text-brand-shade3 hover:text-brand-light disabled:opacity-30 disabled:cursor-not-allowed"
              title="Move down"
            >
              ↓
            </button>
            <button
              onClick={onEdit}
              className="w-6 h-6 flex items-center justify-center rounded hover:bg-brand-shade3/15 text-brand-shade3 hover:text-brand-light"
              title="Edit"
            >
              ✎
            </button>
            <button
              onClick={onRemove}
              className="w-6 h-6 flex items-center justify-center rounded hover:bg-red-500/15 text-brand-shade3 hover:text-red-400"
              title="Remove"
            >
              ✕
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

export default function V2FlowEditorPage() {
  const { agentId = '', flowId = '' } = useParams();
  const navigate = useNavigate();
  const agent = getAgentById(agentId);
  const flows = useMemo(() => getFlowsForAgent(agentId), [agentId]);
  const isNew = flowId === 'new';
  const baseFlow = flows.find((f) => f.id === flowId) ?? null;

  const [draft, setDraft] = useState<V2Flow>(() => {
    if (baseFlow) return { ...baseFlow, checkpoints: baseFlow.checkpoints.map((c) => ({ ...c })) };
    return {
      id: 'flow-new',
      agentId,
      name: '',
      description: '',
      triggerCondition: 'Manual entry only.',
      enabled: true,
      checkpoints: [],
    };
  });

  const [editingIdx, setEditingIdx] = useState<number | null>(null);
  const [showRunDemo, setShowRunDemo] = useState(false);

  if (!agent) {
    return (
      <div className="max-w-[800px] mx-auto text-center py-12">
        <p className="text-brand-shade3">Agent not found.</p>
      </div>
    );
  }

  const updateCheckpoint = (idx: number, patch: Partial<V2FlowCheckpoint>) => {
    setDraft((d) => ({
      ...d,
      checkpoints: d.checkpoints.map((c, i) => (i === idx ? { ...c, ...patch } : c)),
    }));
  };

  const addCheckpoint = () => {
    setDraft((d) => ({
      ...d,
      checkpoints: [
        ...d.checkpoints,
        {
          id: `cp-new-${Date.now()}`,
          name: 'New checkpoint',
          goal: 'Describe the goal',
          successCriteria: 'Describe how to detect it is done',
        },
      ],
    }));
    setEditingIdx(draft.checkpoints.length);
  };

  const moveCheckpoint = (idx: number, dir: 'up' | 'down') => {
    setDraft((d) => {
      const next = [...d.checkpoints];
      const target = dir === 'up' ? idx - 1 : idx + 1;
      if (target < 0 || target >= next.length) return d;
      const a = next[idx];
      const b = next[target];
      if (!a || !b) return d;
      next[idx] = b;
      next[target] = a;
      return { ...d, checkpoints: next };
    });
  };

  const removeCheckpoint = (idx: number) => {
    setDraft((d) => ({ ...d, checkpoints: d.checkpoints.filter((_, i) => i !== idx) }));
    setEditingIdx(null);
  };

  const sampleSession = v2Sessions.find((s) => s.participantAgentIds.includes(agentId));

  return (
    <div className="max-w-[1100px] mx-auto">
      <Link to={`/agents/${agent.id}`} className="text-[11px] text-brand-shade3 hover:text-brand-accent">
        ← {agent.name} / Flows
      </Link>

      <div className="mt-3 flex items-center justify-between">
        <h1 className="text-xl font-semibold text-brand-light">
          {isNew ? 'Create Flow' : `Edit: ${baseFlow?.name}`}
        </h1>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowRunDemo((v) => !v)}
            className="px-3 py-1.5 text-[11px] text-brand-shade2 border border-brand-shade3/25 rounded-btn hover:border-brand-shade3/50 hover:text-brand-light"
          >
            {showRunDemo ? 'Hide' : 'Show'} Runtime Preview
          </button>
          <button
            disabled
            className="px-3 py-1.5 text-[11px] bg-brand-accent/20 text-brand-accent border border-brand-accent/30 rounded-btn cursor-not-allowed"
            title="Prototype — save not wired"
          >
            Save flow
          </button>
          <button
            onClick={() => navigate(`/agents/${agent.id}`)}
            className="px-3 py-1.5 text-[11px] text-brand-shade3 hover:text-brand-light rounded-btn"
          >
            Cancel
          </button>
        </div>
      </div>

      <div className="grid grid-cols-[1fr_320px] gap-6 mt-5">
        {/* Left: flow meta + checkpoints */}
        <div className="space-y-5">
          <div className="bg-brand-dark-surface border border-brand-shade3/15 rounded-card p-4 space-y-3">
            <div>
              <label className="block text-[10px] uppercase tracking-wider text-brand-shade3 mb-1">Name</label>
              <input
                value={draft.name}
                onChange={(e) => setDraft((d) => ({ ...d, name: e.target.value }))}
                placeholder="e.g. Deep Qualification Interview"
                className="w-full bg-brand-dark border border-brand-shade3/25 rounded-btn px-3 py-2 text-[13px]"
              />
            </div>
            <div>
              <label className="block text-[10px] uppercase tracking-wider text-brand-shade3 mb-1">Description</label>
              <textarea
                value={draft.description}
                onChange={(e) => setDraft((d) => ({ ...d, description: e.target.value }))}
                rows={2}
                placeholder="What this flow accomplishes"
                className="w-full bg-brand-dark border border-brand-shade3/25 rounded-btn px-3 py-2 text-[13px]"
              />
            </div>
            <div>
              <label className="block text-[10px] uppercase tracking-wider text-brand-shade3 mb-1">
                Trigger condition
              </label>
              <textarea
                value={draft.triggerCondition}
                onChange={(e) => setDraft((d) => ({ ...d, triggerCondition: e.target.value }))}
                rows={2}
                placeholder="When the agent auto-enters this flow (or 'Manual entry only')"
                className="w-full bg-brand-dark border border-brand-shade3/25 rounded-btn px-3 py-2 text-[12px] font-mono"
              />
              <p className="text-[10px] text-brand-shade3 mt-1">
                Free text or LLM classifier prompt. Agent auto-enters when this matches incoming context.
              </p>
            </div>
          </div>

          <div>
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-[13px] font-semibold text-brand-light">
                Checkpoints ({draft.checkpoints.length})
              </h2>
              <button
                onClick={addCheckpoint}
                className="text-[11px] text-brand-accent border border-brand-accent/40 hover:bg-brand-accent/10 rounded-btn px-3 py-1"
              >
                + Add checkpoint
              </button>
            </div>

            {draft.checkpoints.length === 0 && (
              <div className="bg-brand-dark-surface border border-dashed border-brand-shade3/25 rounded-card p-6 text-center">
                <p className="text-[12px] text-brand-shade3">
                  A flow is a sequence of checkpoints. The agent advances when each checkpoint's
                  success criteria is met.
                </p>
                <button
                  onClick={addCheckpoint}
                  className="mt-3 text-[11px] text-brand-accent border border-brand-accent/40 hover:bg-brand-accent/10 rounded-btn px-3 py-1"
                >
                  + Add first checkpoint
                </button>
              </div>
            )}

            {draft.checkpoints.length > 0 && (
              <div className="relative space-y-2">
                {draft.checkpoints.length > 1 && (
                  <div className="absolute left-[15px] top-8 bottom-8 w-px bg-brand-shade3/20" />
                )}
                {draft.checkpoints.map((cp, idx) => (
                  <CheckpointCard
                    key={cp.id}
                    cp={cp}
                    index={idx}
                    isFirst={idx === 0}
                    isLast={idx === draft.checkpoints.length - 1}
                    onEdit={() => setEditingIdx(idx)}
                    onMove={(dir) => moveCheckpoint(idx, dir)}
                    onRemove={() => removeCheckpoint(idx)}
                  />
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Right: inspector (checkpoint edit) + runtime preview */}
        <div className="space-y-5">
          {(() => {
            const editing = editingIdx !== null ? draft.checkpoints[editingIdx] : undefined;
            if (editing && editingIdx !== null) {
              return (
            <div className="bg-brand-dark-surface border border-purple-500/30 rounded-card p-4 space-y-3">
              <div className="flex items-center justify-between">
                <div className="text-[12px] font-semibold text-purple-300">
                  Checkpoint #{editingIdx + 1}
                </div>
                <button onClick={() => setEditingIdx(null)} className="text-[11px] text-brand-shade3 hover:text-brand-light">
                  Close
                </button>
              </div>
              <div>
                <label className="block text-[10px] uppercase tracking-wider text-brand-shade3 mb-1">Name</label>
                <input
                  value={editing.name}
                  onChange={(e) => updateCheckpoint(editingIdx, { name: e.target.value })}
                  className="w-full bg-brand-dark border border-brand-shade3/25 rounded-btn px-3 py-2 text-[13px]"
                />
              </div>
              <div>
                <label className="block text-[10px] uppercase tracking-wider text-brand-shade3 mb-1">Goal</label>
                <textarea
                  rows={2}
                  value={editing.goal}
                  onChange={(e) => updateCheckpoint(editingIdx, { goal: e.target.value })}
                  className="w-full bg-brand-dark border border-brand-shade3/25 rounded-btn px-3 py-2 text-[12px]"
                />
              </div>
              <div>
                <label className="block text-[10px] uppercase tracking-wider text-brand-shade3 mb-1">
                  Success criteria
                </label>
                <textarea
                  rows={2}
                  value={editing.successCriteria}
                  onChange={(e) => updateCheckpoint(editingIdx, { successCriteria: e.target.value })}
                  className="w-full bg-brand-dark border border-brand-shade3/25 rounded-btn px-3 py-2 text-[12px]"
                />
              </div>
            </div>
              );
            }
            return (
              <div className="bg-brand-dark-surface border border-brand-shade3/15 rounded-card p-4 text-[11px] text-brand-shade3">
                Select a checkpoint to edit it, or add a new one.
              </div>
            );
          })()}

          {showRunDemo && (
            <div className="bg-brand-dark-surface border border-brand-shade3/15 rounded-card p-4">
              <div className="text-[12px] font-semibold text-brand-light mb-1">Runtime preview</div>
              <p className="text-[10px] text-brand-shade3 mb-3">
                When live: flow_runs tracks agent progression through checkpoints with accumulated
                context.
              </p>
              {sampleSession ? (
                <div className="space-y-1.5 text-[11px]">
                  {draft.checkpoints.map((cp, idx) => (
                    <div
                      key={cp.id}
                      className={`flex items-center gap-2 px-2 py-1 rounded border ${
                        idx === 1
                          ? 'border-purple-500/40 bg-purple-500/10 text-purple-200'
                          : idx < 1
                            ? 'border-emerald-500/25 bg-emerald-500/5 text-brand-shade2'
                            : 'border-brand-shade3/20 text-brand-shade3'
                      }`}
                    >
                      <span className="font-mono w-5">{idx + 1}</span>
                      <span className="flex-1 truncate">{cp.name}</span>
                      <span className="text-[9px] uppercase tracking-wider">
                        {idx === 1 ? 'active' : idx < 1 ? 'done' : 'pending'}
                      </span>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-[11px] text-brand-shade3">No sample session available yet.</p>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
