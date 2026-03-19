import { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api/client';
import { useAuth } from '../hooks/useAuth';
import { StatusBadge } from '../components/StatusBadge';
import { TaskDetailPanel } from '../components/TaskDetailPanel';
import { CreateTaskModal } from '../components/CreateTaskModal';
import { TaskFilters } from '../components/TaskFilters';
import type { TaskResponse, PaginatedTaskResponse } from '../types';
import type { TaskFilterValues } from '../components/TaskFilters';

export function TasksPage() {
  const { logout } = useAuth();
  const [tasks, setTasks] = useState<TaskResponse[]>([]);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedTaskId, setSelectedTaskId] = useState<number | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [filters, setFilters] = useState<TaskFilterValues>({ status: '', agent: '', source: '' });

  const fetchTasks = useCallback((p: number, f: TaskFilterValues) => {
    setLoading(true);
    const params: Record<string, string> = { page: String(p), per_page: '20' };
    if (f.status) params['status'] = f.status;
    if (f.agent) params['agent_name'] = f.agent;
    if (f.source) params['source'] = f.source;

    api
      .listTasks(params)
      .then((res: PaginatedTaskResponse) => {
        setTasks(res.data);
        setTotalPages(res.total_pages);
        setLoading(false);
      })
      .catch((err: Error) => {
        setError(err.message);
        setLoading(false);
      });
  }, []);

  useEffect(() => {
    fetchTasks(page, filters);
  }, [page, filters, fetchTasks]);

  const handleFilterChange = (newFilters: TaskFilterValues) => {
    setFilters(newFilters);
    setPage(1);
  };

  const handleTaskCreated = () => {
    setShowCreateModal(false);
    setPage(1);
    fetchTasks(1, filters);
  };

  const handleTaskCancelled = () => {
    setSelectedTaskId(null);
    fetchTasks(page, filters);
  };

  return (
    <div className="min-h-screen bg-brand-dark">
      {/* Header */}
      <header className="border-b border-brand-shade3/15 px-6 py-4">
        <div className="mx-auto flex max-w-5xl items-center justify-between">
          <Link to="/chat">
            <img src="/logo-dark.svg" alt="ByteBrew" className="h-6" />
          </Link>
          <nav className="flex items-center gap-4">
            <Link to="/chat" className="text-xs text-brand-shade3 hover:text-brand-light">
              Chat
            </Link>
            <Link to="/agents" className="text-xs text-brand-shade3 hover:text-brand-light">
              Agents
            </Link>
            <Link to="/health" className="text-xs text-brand-shade3 hover:text-brand-light">
              Health
            </Link>
            <button onClick={logout} className="text-xs text-brand-shade3 hover:text-brand-light">
              Logout
            </button>
          </nav>
        </div>
      </header>

      <div className="mx-auto max-w-5xl px-6 py-8">
        {/* Title row with New Task button */}
        <div className="mb-6 flex items-center justify-between">
          <h1 className="text-xl font-bold text-brand-light">Tasks</h1>
          <button
            onClick={() => setShowCreateModal(true)}
            className="rounded-btn bg-brand-accent px-4 py-2 text-xs font-medium text-white transition-colors hover:bg-brand-accent-hover"
          >
            New Task
          </button>
        </div>

        {error && (
          <div className="mb-4 rounded-btn border border-red-500/30 bg-red-500/10 px-4 py-2 text-sm text-red-400">
            {error}
          </div>
        )}

        {/* Filters */}
        <TaskFilters filters={filters} onChange={handleFilterChange} />

        <div className="overflow-hidden rounded-card border border-brand-shade3/15">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-brand-shade3/15 bg-brand-dark-alt">
                <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-brand-shade3">
                  ID
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-brand-shade3">
                  Title
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-brand-shade3">
                  Agent
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-brand-shade3">
                  Status
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-brand-shade3">
                  Source
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-brand-shade3">
                  Created
                </th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-brand-shade3">
                    Loading...
                  </td>
                </tr>
              ) : tasks.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-brand-shade3">
                    No tasks found
                  </td>
                </tr>
              ) : (
                tasks.map((task) => (
                  <tr
                    key={task.id}
                    onClick={() => setSelectedTaskId(task.id)}
                    className="cursor-pointer border-b border-brand-shade3/10 transition-colors hover:bg-brand-dark-alt/50"
                  >
                    <td className="px-4 py-3 text-brand-shade3">#{task.id}</td>
                    <td className="px-4 py-3 text-brand-light">{task.title}</td>
                    <td className="px-4 py-3 text-brand-accent">{task.agent_name}</td>
                    <td className="px-4 py-3">
                      <StatusBadge status={task.status} />
                    </td>
                    <td className="px-4 py-3 text-brand-shade3">{task.source}</td>
                    <td className="px-4 py-3 text-brand-shade3">
                      {new Date(task.created_at).toLocaleString()}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="mt-4 flex items-center justify-center gap-2">
            <button
              disabled={page <= 1}
              onClick={() => setPage((p) => p - 1)}
              className="rounded-btn px-3 py-1.5 text-xs text-brand-shade2 transition-colors hover:bg-brand-shade3/10 disabled:opacity-30"
            >
              Previous
            </button>
            <span className="text-xs text-brand-shade3">
              Page {page} of {totalPages}
            </span>
            <button
              disabled={page >= totalPages}
              onClick={() => setPage((p) => p + 1)}
              className="rounded-btn px-3 py-1.5 text-xs text-brand-shade2 transition-colors hover:bg-brand-shade3/10 disabled:opacity-30"
            >
              Next
            </button>
          </div>
        )}
      </div>

      {/* Detail Panel */}
      {selectedTaskId !== null && (
        <TaskDetailPanel
          taskId={selectedTaskId}
          onClose={() => setSelectedTaskId(null)}
          onCancelled={handleTaskCancelled}
        />
      )}

      {/* Create Modal */}
      {showCreateModal && (
        <CreateTaskModal
          onClose={() => setShowCreateModal(false)}
          onCreated={handleTaskCreated}
        />
      )}
    </div>
  );
}
