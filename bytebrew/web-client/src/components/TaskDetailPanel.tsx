import { useState, useEffect } from 'react';
import { api } from '../api/client';
import { StatusBadge } from './StatusBadge';
import type { TaskDetailResponse } from '../types';

interface TaskDetailPanelProps {
  taskId: number;
  onClose: () => void;
  onCancelled: () => void;
}

function formatDate(value: string | undefined): string {
  if (!value) return '-';
  return new Date(value).toLocaleString();
}

export function TaskDetailPanel({ taskId, onClose, onCancelled }: TaskDetailPanelProps) {
  const [task, setTask] = useState<TaskDetailResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [cancelling, setCancelling] = useState(false);
  const [confirmCancel, setConfirmCancel] = useState(false);

  useEffect(() => {
    setLoading(true);
    setError('');
    api
      .getTask(taskId)
      .then((data) => {
        setTask(data);
        setLoading(false);
      })
      .catch((err: Error) => {
        setError(err.message);
        setLoading(false);
      });
  }, [taskId]);

  const handleCancel = () => {
    setCancelling(true);
    api
      .cancelTask(taskId)
      .then(() => {
        setCancelling(false);
        setConfirmCancel(false);
        onCancelled();
      })
      .catch((err: Error) => {
        setError(err.message);
        setCancelling(false);
        setConfirmCancel(false);
      });
  };

  const canCancel = task && (task.status === 'pending' || task.status === 'running');

  return (
    <>
      {/* Backdrop */}
      <div className="fixed inset-0 z-40 bg-black/50" onClick={onClose} />

      {/* Panel */}
      <div className="fixed right-0 top-0 z-50 flex h-full w-full max-w-lg flex-col border-l border-brand-shade3/15 bg-brand-dark animate-slide-in-right">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-brand-shade3/15 px-6 py-4">
          <h2 className="text-sm font-bold text-brand-light">Task #{taskId}</h2>
          <button
            onClick={onClose}
            className="text-brand-shade3 transition-colors hover:text-brand-light"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
              <path d="M4 4l8 8M12 4l-8 8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-6 py-5">
          {loading ? (
            <div className="text-sm text-brand-shade3">Loading...</div>
          ) : error ? (
            <div className="rounded-btn border border-red-500/30 bg-red-500/10 px-4 py-2 text-sm text-red-400">
              {error}
            </div>
          ) : task ? (
            <div className="space-y-5">
              {/* Title */}
              <div>
                <label className="mb-1 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                  Title
                </label>
                <p className="text-sm text-brand-light">{task.title}</p>
              </div>

              {/* Description */}
              {task.description && (
                <div>
                  <label className="mb-1 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                    Description
                  </label>
                  <p className="text-sm text-brand-shade2">{task.description}</p>
                </div>
              )}

              {/* Agent & Status row */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="mb-1 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                    Agent
                  </label>
                  <p className="text-sm text-brand-accent">{task.agent_name}</p>
                </div>
                <div>
                  <label className="mb-1 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                    Status
                  </label>
                  <StatusBadge status={task.status} />
                </div>
              </div>

              {/* Source & Mode row */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="mb-1 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                    Source
                  </label>
                  <p className="text-sm text-brand-shade2">{task.source}</p>
                </div>
                <div>
                  <label className="mb-1 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                    Mode
                  </label>
                  <p className="text-sm text-brand-shade2">{task.mode || '-'}</p>
                </div>
              </div>

              {/* Timestamps */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="mb-1 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                    Created
                  </label>
                  <p className="text-xs text-brand-shade3">{formatDate(task.created_at)}</p>
                </div>
                <div>
                  <label className="mb-1 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                    Started
                  </label>
                  <p className="text-xs text-brand-shade3">{formatDate(task.started_at)}</p>
                </div>
              </div>
              <div>
                <label className="mb-1 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                  Completed
                </label>
                <p className="text-xs text-brand-shade3">{formatDate(task.completed_at)}</p>
              </div>

              {/* Result */}
              {task.result && (
                <div>
                  <label className="mb-1 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                    Result
                  </label>
                  <pre className="max-h-48 overflow-auto rounded-btn border border-brand-shade3/15 bg-brand-dark-alt p-3 text-xs text-brand-shade1">
                    {task.result}
                  </pre>
                </div>
              )}

              {/* Error */}
              {task.error && (
                <div>
                  <label className="mb-1 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                    Error
                  </label>
                  <pre className="max-h-48 overflow-auto rounded-btn border border-red-500/20 bg-red-500/5 p-3 text-xs text-red-400">
                    {task.error}
                  </pre>
                </div>
              )}

              {/* Cancel button */}
              {canCancel && (
                <div className="pt-2">
                  {confirmCancel ? (
                    <div className="flex items-center gap-3">
                      <span className="text-xs text-brand-shade3">Cancel this task?</span>
                      <button
                        onClick={handleCancel}
                        disabled={cancelling}
                        className="rounded-btn bg-red-500 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-red-600 disabled:opacity-50"
                      >
                        {cancelling ? 'Cancelling...' : 'Yes, cancel'}
                      </button>
                      <button
                        onClick={() => setConfirmCancel(false)}
                        className="rounded-btn px-3 py-1.5 text-xs text-brand-shade3 transition-colors hover:text-brand-light"
                      >
                        No
                      </button>
                    </div>
                  ) : (
                    <button
                      onClick={() => setConfirmCancel(true)}
                      className="rounded-btn border border-red-500/30 px-4 py-2 text-xs font-medium text-red-400 transition-colors hover:border-red-500/50 hover:bg-red-500/10"
                    >
                      Cancel Task
                    </button>
                  )}
                </div>
              )}
            </div>
          ) : null}
        </div>
      </div>
    </>
  );
}
