import { type FormEvent, useEffect, useMemo, useState } from 'react';
import FormModal from './FormModal';
import type { AgentInfo, CreateTaskRequest, TaskResponse } from '../types';

interface TaskCreateFormProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: CreateTaskRequest) => Promise<void>;
  agents: AgentInfo[];
  /** If set, form creates a subtask of this parent. */
  parentTask?: TaskResponse | null;
  /** Candidate tasks to pick as blockers (only used when parentTask is not set). */
  blockerCandidates?: TaskResponse[];
  loading?: boolean;
  errorMessage?: string | null;
}

/**
 * TaskCreateForm — modal form for creating a top-level task or a subtask.
 *
 * Fields:
 * - title (required)
 * - description
 * - agent_name (select from known agents; required)
 * - priority (0|1|2)
 * - acceptance_criteria (bullet list editor)
 * - blocked_by (checkbox list of existing tasks when top-level)
 * - require_approval (starts as draft)
 */
export default function TaskCreateForm({
  open,
  onClose,
  onSubmit,
  agents,
  parentTask,
  blockerCandidates = [],
  loading,
  errorMessage,
}: TaskCreateFormProps) {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [agentName, setAgentName] = useState('');
  const [priority, setPriority] = useState(0);
  const [criteria, setCriteria] = useState<string[]>(['']);
  const [blockers, setBlockers] = useState<string[]>([]);
  const [requireApproval, setRequireApproval] = useState(false);
  const [touched, setTouched] = useState(false);

  // Default agent: first in the list when opening.
  useEffect(() => {
    if (open) {
      setTitle('');
      setDescription('');
      setAgentName(agents[0]?.name ?? '');
      setPriority(0);
      setCriteria(['']);
      setBlockers([]);
      setRequireApproval(false);
      setTouched(false);
    }
  }, [open, agents]);

  const titleMissing = title.trim() === '';
  const agentMissing = agentName.trim() === '';
  const invalid = titleMissing || agentMissing;

  // Filter candidate blockers: cannot block on self / parent / existing children — but
  // in create mode we don't have an ID yet, so just offer non-terminal tasks.
  const blockerPool = useMemo(() => {
    return blockerCandidates.filter((t) =>
      !['completed', 'failed', 'cancelled'].includes(t.status),
    );
  }, [blockerCandidates]);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setTouched(true);
    if (invalid) return;

    const ac = criteria.map((s) => s.trim()).filter(Boolean);
    const data: CreateTaskRequest = {
      title: title.trim(),
      description: description.trim() || undefined,
      agent_name: agentName,
      priority,
      acceptance_criteria: ac.length > 0 ? ac : undefined,
      require_approval: requireApproval,
    };
    if (parentTask) {
      data.parent_task_id = parentTask.id;
    } else if (blockers.length > 0) {
      data.blocked_by = blockers;
    }
    await onSubmit(data);
  }

  const showTitleError = touched && titleMissing;
  const showAgentError = touched && agentMissing;

  return (
    <FormModal
      open={open}
      onClose={onClose}
      title={parentTask ? `Add subtask to #${parentTask.id.slice(0, 8)}` : 'Create task'}
      onSubmit={handleSubmit}
      submitLabel={parentTask ? 'Create subtask' : 'Create task'}
      loading={loading}
      size="lg"
    >
      {errorMessage && (
        <div className="p-2 text-xs text-red-300 bg-red-500/10 border border-red-500/30 rounded-btn">
          {errorMessage}
        </div>
      )}

      <div>
        <label className="block text-xs font-medium text-brand-shade2 mb-1">
          Title <span className="text-red-400">*</span>
        </label>
        <input
          autoFocus
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          className={`w-full px-3 py-2 bg-brand-dark border rounded-btn text-sm text-brand-light focus:outline-none transition-colors ${
            showTitleError ? 'border-red-500' : 'border-brand-shade3/30 focus:border-brand-accent'
          }`}
          placeholder="Short descriptive title"
        />
        {showTitleError && <p className="mt-1 text-xs text-red-400">Title is required.</p>}
      </div>

      <div>
        <label className="block text-xs font-medium text-brand-shade2 mb-1">Description</label>
        <textarea
          rows={3}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          className="w-full px-3 py-2 bg-brand-dark border border-brand-shade3/30 rounded-btn text-sm text-brand-light focus:outline-none focus:border-brand-accent transition-colors"
          placeholder="Optional context for the agent"
        />
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="block text-xs font-medium text-brand-shade2 mb-1">
            Agent <span className="text-red-400">*</span>
          </label>
          <select
            value={agentName}
            onChange={(e) => setAgentName(e.target.value)}
            className={`w-full px-3 py-2 bg-brand-dark border rounded-btn text-sm text-brand-light focus:outline-none transition-colors ${
              showAgentError ? 'border-red-500' : 'border-brand-shade3/30 focus:border-brand-accent'
            }`}
          >
            <option value="">— Select —</option>
            {agents.map((a) => (
              <option key={a.name} value={a.name}>
                {a.name}
              </option>
            ))}
          </select>
          {showAgentError && <p className="mt-1 text-xs text-red-400">Agent is required.</p>}
        </div>

        <div>
          <label className="block text-xs font-medium text-brand-shade2 mb-1">Priority</label>
          <select
            value={priority}
            onChange={(e) => setPriority(Number(e.target.value))}
            className="w-full px-3 py-2 bg-brand-dark border border-brand-shade3/30 rounded-btn text-sm text-brand-light focus:outline-none focus:border-brand-accent"
          >
            <option value={0}>Normal</option>
            <option value={1}>High</option>
            <option value={2}>Critical</option>
          </select>
        </div>
      </div>

      <div>
        <label className="block text-xs font-medium text-brand-shade2 mb-1">
          Acceptance criteria
        </label>
        <div className="space-y-1">
          {criteria.map((c, i) => (
            <div key={i} className="flex gap-2">
              <input
                type="text"
                value={c}
                onChange={(e) => setCriteria((arr) => arr.map((v, j) => (j === i ? e.target.value : v)))}
                className="flex-1 px-3 py-2 bg-brand-dark border border-brand-shade3/30 rounded-btn text-xs text-brand-light focus:outline-none focus:border-brand-accent"
                placeholder={`Criterion ${i + 1}`}
              />
              {criteria.length > 1 && (
                <button
                  type="button"
                  onClick={() => setCriteria((arr) => arr.filter((_, j) => j !== i))}
                  className="px-2 py-2 text-xs text-brand-shade3 border border-brand-shade3/30 rounded-btn hover:bg-brand-dark-alt hover:text-red-400 transition-colors"
                >
                  ×
                </button>
              )}
            </div>
          ))}
          <button
            type="button"
            onClick={() => setCriteria((arr) => [...arr, ''])}
            className="text-xs text-brand-accent hover:underline"
          >
            + Add criterion
          </button>
        </div>
      </div>

      {!parentTask && blockerPool.length > 0 && (
        <div>
          <label className="block text-xs font-medium text-brand-shade2 mb-1">
            Blocked by (existing tasks)
          </label>
          <div className="max-h-28 overflow-y-auto border border-brand-shade3/30 rounded-btn p-2 space-y-1">
            {blockerPool.map((t) => (
              <label key={t.id} className="flex items-center gap-2 text-xs text-brand-shade2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={blockers.includes(t.id)}
                  onChange={(e) => {
                    if (e.target.checked) {
                      setBlockers((arr) => [...arr, t.id]);
                    } else {
                      setBlockers((arr) => arr.filter((id) => id !== t.id));
                    }
                  }}
                />
                <span className="font-mono text-brand-shade3">#{t.id.slice(0, 8)}</span>
                <span className="truncate">{t.title}</span>
                <span className="ml-auto text-[10px] text-brand-shade3 bg-brand-dark px-1.5 py-0.5 rounded">
                  {t.status}
                </span>
              </label>
            ))}
          </div>
        </div>
      )}

      <label className="flex items-center gap-2 text-xs text-brand-shade2 cursor-pointer">
        <input
          type="checkbox"
          checked={requireApproval}
          onChange={(e) => setRequireApproval(e.target.checked)}
        />
        Require approval (start as draft)
      </label>

      {parentTask && (
        <div className="text-xs text-brand-shade3 italic">
          Parent: <span className="font-mono">#{parentTask.id.slice(0, 8)}</span> — {parentTask.title}
        </div>
      )}
    </FormModal>
  );
}
