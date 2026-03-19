import { useState, useEffect } from 'react';
import { api } from '../api/client';
import type { AgentInfo } from '../types';

interface CreateTaskModalProps {
  onClose: () => void;
  onCreated: () => void;
}

export function CreateTaskModal({ onClose, onCreated }: CreateTaskModalProps) {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [agentName, setAgentName] = useState('');
  const [agents, setAgents] = useState<AgentInfo[]>([]);
  const [loadingAgents, setLoadingAgents] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    api
      .listAgents()
      .then((data) => {
        const filtered = data.filter((a) => a.name);
        setAgents(filtered);
        if (filtered.length > 0 && filtered[0]) {
          setAgentName(filtered[0].name);
        }
        setLoadingAgents(false);
      })
      .catch((err: Error) => {
        setError(err.message);
        setLoadingAgents(false);
      });
  }, []);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim() || !agentName) return;

    setSubmitting(true);
    setError('');
    api
      .createTask({
        title: title.trim(),
        description: description.trim() || undefined,
        agent_name: agentName,
      })
      .then(() => {
        setSubmitting(false);
        onCreated();
      })
      .catch((err: Error) => {
        setError(err.message);
        setSubmitting(false);
      });
  };

  return (
    <>
      {/* Backdrop */}
      <div className="fixed inset-0 z-40 bg-black/60" onClick={onClose} />

      {/* Modal */}
      <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div className="w-full max-w-md animate-fade-in rounded-card border border-brand-shade3/15 bg-brand-dark-alt shadow-2xl">
          {/* Header */}
          <div className="flex items-center justify-between border-b border-brand-shade3/15 px-6 py-4">
            <h2 className="text-sm font-bold text-brand-light">New Task</h2>
            <button
              onClick={onClose}
              className="text-brand-shade3 transition-colors hover:text-brand-light"
            >
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                <path d="M4 4l8 8M12 4l-8 8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
              </svg>
            </button>
          </div>

          {/* Form */}
          <form onSubmit={handleSubmit} className="px-6 py-5">
            {error && (
              <div className="mb-4 rounded-btn border border-red-500/30 bg-red-500/10 px-4 py-2 text-sm text-red-400">
                {error}
              </div>
            )}

            <div className="space-y-4">
              {/* Title */}
              <div>
                <label className="mb-1.5 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                  Title <span className="text-brand-accent">*</span>
                </label>
                <input
                  type="text"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder="Enter task title"
                  required
                  className="w-full rounded-btn border border-brand-shade3/20 bg-brand-dark px-3 py-2 text-sm text-brand-light placeholder-brand-shade3/50 outline-none transition-colors focus:border-brand-accent/50"
                />
              </div>

              {/* Description */}
              <div>
                <label className="mb-1.5 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                  Description
                </label>
                <textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Optional description"
                  rows={3}
                  className="w-full resize-none rounded-btn border border-brand-shade3/20 bg-brand-dark px-3 py-2 text-sm text-brand-light placeholder-brand-shade3/50 outline-none transition-colors focus:border-brand-accent/50"
                />
              </div>

              {/* Agent */}
              <div>
                <label className="mb-1.5 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                  Agent <span className="text-brand-accent">*</span>
                </label>
                {loadingAgents ? (
                  <div className="text-xs text-brand-shade3">Loading agents...</div>
                ) : (
                  <select
                    value={agentName}
                    onChange={(e) => setAgentName(e.target.value)}
                    required
                    className="w-full rounded-btn border border-brand-shade3/20 bg-brand-dark px-3 py-2 text-sm text-brand-light outline-none transition-colors focus:border-brand-accent/50"
                  >
                    {agents.map((agent) => (
                      <option key={agent.name} value={agent.name}>
                        {agent.name}
                      </option>
                    ))}
                  </select>
                )}
              </div>
            </div>

            {/* Actions */}
            <div className="mt-6 flex items-center justify-end gap-3">
              <button
                type="button"
                onClick={onClose}
                className="rounded-btn px-4 py-2 text-xs text-brand-shade3 transition-colors hover:text-brand-light"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={submitting || !title.trim() || !agentName}
                className="rounded-btn bg-brand-accent px-4 py-2 text-xs font-medium text-white transition-colors hover:bg-brand-accent-hover disabled:opacity-50"
              >
                {submitting ? 'Creating...' : 'Create Task'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </>
  );
}
