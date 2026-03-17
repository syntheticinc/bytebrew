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
        <span className="text-xs text-gray-600 bg-gray-100 px-2 py-0.5 rounded">{row.source}</span>
      ),
    },
    {
      key: 'created_at',
      header: 'Created',
      render: (row: TaskResponse) => (
        <span className="text-xs text-gray-500">
          {new Date(row.created_at).toLocaleString()}
        </span>
      ),
    },
  ];

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Tasks</h1>
        <button
          onClick={refetch}
          className="px-4 py-2 text-sm text-gray-700 border border-gray-300 rounded-md hover:bg-gray-50"
        >
          Refresh
        </button>
      </div>

      {/* Filters */}
      <div className="flex gap-3 mb-4">
        <select
          value={filters['status'] ?? ''}
          onChange={(e) => setFilters({ ...filters, status: e.target.value })}
          className="px-3 py-2 border border-gray-300 rounded-md text-sm"
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
          className="px-3 py-2 border border-gray-300 rounded-md text-sm"
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
          className="px-3 py-2 border border-gray-300 rounded-md text-sm"
        />
      </div>

      {loading && <div className="text-gray-500">Loading tasks...</div>}
      {error && <div className="text-red-600">Error: {error}</div>}

      {!loading && !error && (
        <div className="bg-white rounded-lg shadow">
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
              className="px-4 py-2 text-sm text-white bg-red-600 rounded-md hover:bg-red-700"
            >
              Cancel Task
            </button>
          ) : undefined
        }
      >
        {loadingDetail ? (
          <div className="text-gray-500">Loading...</div>
        ) : selectedTask ? (
          <div className="space-y-3 text-sm">
            <div className="flex justify-between">
              <span className="text-gray-500">Status</span>
              <StatusBadge status={selectedTask.status} />
            </div>
            <div className="flex justify-between">
              <span className="text-gray-500">Agent</span>
              <span>{selectedTask.agent_name}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-500">Source</span>
              <span>{selectedTask.source}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-500">Mode</span>
              <span>{selectedTask.mode}</span>
            </div>
            {selectedTask.description && (
              <div>
                <span className="text-gray-500">Description</span>
                <p className="mt-1 text-gray-700">{selectedTask.description}</p>
              </div>
            )}
            {selectedTask.result && (
              <div>
                <span className="text-gray-500">Result</span>
                <pre className="mt-1 p-2 bg-gray-50 rounded text-xs whitespace-pre-wrap">
                  {selectedTask.result}
                </pre>
              </div>
            )}
            {selectedTask.error && (
              <div>
                <span className="text-gray-500">Error</span>
                <pre className="mt-1 p-2 bg-red-50 rounded text-xs text-red-700 whitespace-pre-wrap">
                  {selectedTask.error}
                </pre>
              </div>
            )}
            <div className="flex justify-between text-xs text-gray-400">
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
