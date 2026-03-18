import { useState } from 'react';
import { api } from '../api/client';
import { useApi } from '../hooks/useApi';
import DataTable from '../components/DataTable';
import StatusBadge from '../components/StatusBadge';
import Modal from '../components/Modal';
import type { TaskResponse, TaskDetailResponse } from '../types';

const STATUS_OPTIONS = ['', 'pending', 'in_progress', 'completed', 'failed', 'cancelled', 'needs_input', 'escalated'];
const SOURCE_OPTIONS = ['', 'agent', 'cron', 'webhook', 'api', 'dashboard'];

export default function TasksPage() {
  const [filters, setFilters] = useState<Record<string, string>>({});
  const { data: tasks, loading, error, refetch } = useApi(
    () => {
      const params: Record<string, string> = {};
      for (const [k, v] of Object.entries(filters)) {
        if (v) params[k] = v;
      }
      return api.listTasks(Object.keys(params).length > 0 ? params : undefined);
    },
    [JSON.stringify(filters)],
  );

  const [selectedTask, setSelectedTask] = useState<TaskDetailResponse | null>(null);
  const [loadingDetail, setLoadingDetail] = useState(false);

  async function handleRowClick(row: TaskResponse) {
    setLoadingDetail(true);
    try {
      const detail = await api.getTask(row.id);
      setSelectedTask(detail);
    } catch {
      // visible in console
    } finally {
      setLoadingDetail(false);
    }
  }

  async function handleCancel(id: number) {
    try {
      await api.cancelTask(id);
      setSelectedTask(null);
      refetch();
    } catch {
      // visible in console
    }
  }

  const columns = [
    { key: 'id', header: 'ID' },
    { key: 'title', header: 'Title' },
    { key: 'agent_name', header: 'Agent' },
    {
      key: 'status',
      header: 'Status',
      render: (row: TaskResponse) => <StatusBadge status={row.status} />,
    },
    {
      key: 'source',
      header: 'Source',
      render: (row: TaskResponse) => (
        <span className="text-xs text-brand-shade3 bg-brand-light px-2 py-0.5 rounded">{row.source}</span>
      ),
    },
    {
      key: 'created_at',
      header: 'Created',
      render: (row: TaskResponse) => (
        <span className="text-xs text-brand-shade3">
          {new Date(row.created_at).toLocaleString()}
        </span>
      ),
    },
  ];

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-brand-dark">Tasks</h1>
        <button
          onClick={refetch}
          className="px-4 py-2 text-sm text-brand-dark border border-brand-shade2 rounded-btn hover:bg-brand-light"
        >
          Refresh
        </button>
      </div>

      {/* Filters */}
      <div className="flex gap-3 mb-4">
        <select
          value={filters['status'] ?? ''}
          onChange={(e) => setFilters({ ...filters, status: e.target.value })}
          className="px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent"
        >
          <option value="">All statuses</option>
          {STATUS_OPTIONS.filter(Boolean).map((s) => (
            <option key={s} value={s}>
              {s.replace(/_/g, ' ')}
            </option>
          ))}
        </select>
        <select
          value={filters['source'] ?? ''}
          onChange={(e) => setFilters({ ...filters, source: e.target.value })}
          className="px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent"
        >
          <option value="">All sources</option>
          {SOURCE_OPTIONS.filter(Boolean).map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </select>
        <input
          type="text"
          placeholder="Agent name..."
          value={filters['agent_name'] ?? ''}
          onChange={(e) => setFilters({ ...filters, agent_name: e.target.value })}
          className="px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent"
        />
      </div>

      {loading && <div className="text-brand-shade3">Loading tasks...</div>}
      {error && <div className="text-red-600">Error: {error}</div>}

      {!loading && !error && (
        <div className="bg-white rounded-card border border-brand-shade1">
          <DataTable
            columns={columns}
            data={tasks ?? []}
            keyField="id"
            onRowClick={handleRowClick}
            emptyMessage="No tasks found."
          />
        </div>
      )}

      {/* Task detail modal */}
      <Modal
        open={selectedTask !== null}
        onClose={() => setSelectedTask(null)}
        title={selectedTask ? `Task #${selectedTask.id}: ${selectedTask.title}` : 'Task Detail'}
        footer={
          selectedTask && ['pending', 'in_progress', 'needs_input'].includes(selectedTask.status) ? (
            <button
              onClick={() => handleCancel(selectedTask.id)}
              className="px-4 py-2 text-sm text-white bg-red-600 rounded-btn hover:bg-red-700"
            >
              Cancel Task
            </button>
          ) : undefined
        }
      >
        {loadingDetail ? (
          <div className="text-brand-shade3">Loading...</div>
        ) : selectedTask ? (
          <div className="space-y-3 text-sm">
            <div className="flex justify-between">
              <span className="text-brand-shade3">Status</span>
              <StatusBadge status={selectedTask.status} />
            </div>
            <div className="flex justify-between">
              <span className="text-brand-shade3">Agent</span>
              <span className="text-brand-dark">{selectedTask.agent_name}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-brand-shade3">Source</span>
              <span className="text-brand-dark">{selectedTask.source}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-brand-shade3">Mode</span>
              <span className="text-brand-dark">{selectedTask.mode}</span>
            </div>
            {selectedTask.description && (
              <div>
                <span className="text-brand-shade3">Description</span>
                <p className="mt-1 text-brand-dark">{selectedTask.description}</p>
              </div>
            )}
            {selectedTask.result && (
              <div>
                <span className="text-brand-shade3">Result</span>
                <pre className="mt-1 p-2 bg-brand-light rounded-btn text-xs whitespace-pre-wrap">
                  {selectedTask.result}
                </pre>
              </div>
            )}
            {selectedTask.error && (
              <div>
                <span className="text-brand-shade3">Error</span>
                <pre className="mt-1 p-2 bg-red-50 rounded-btn text-xs text-red-700 whitespace-pre-wrap">
                  {selectedTask.error}
                </pre>
              </div>
            )}
            <div className="flex justify-between text-xs text-brand-shade3">
              <span>Created: {new Date(selectedTask.created_at).toLocaleString()}</span>
              {selectedTask.completed_at && (
                <span>Completed: {new Date(selectedTask.completed_at).toLocaleString()}</span>
              )}
            </div>
          </div>
        ) : null}
      </Modal>
    </div>
  );
}
