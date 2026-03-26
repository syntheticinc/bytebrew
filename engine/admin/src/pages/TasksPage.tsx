import { useState } from 'react';
import { api } from '../api/client';
import { useApi } from '../hooks/useApi';
import DataTable from '../components/DataTable';
import { emptyIcons } from '../components/EmptyState';
import StatusBadge from '../components/StatusBadge';
import DetailPanel, { DetailRow, DetailSection } from '../components/DetailPanel';
import type { TaskResponse, TaskDetailResponse } from '../types';

const STATUS_OPTIONS = ['', 'pending', 'in_progress', 'completed', 'failed', 'cancelled', 'needs_input', 'escalated'];
const SOURCE_OPTIONS = ['', 'agent', 'cron', 'webhook', 'api', 'dashboard'];
const PER_PAGE = 20;

export default function TasksPage() {
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [page, setPage] = useState(1);
  const { data: paginatedData, loading, error, refetch } = useApi(
    () => {
      const params: Record<string, string> = {
        page: String(page),
        per_page: String(PER_PAGE),
      };
      for (const [k, v] of Object.entries(filters)) {
        if (v) params[k] = v;
      }
      return api.listTasksPaginated(params);
    },
    [JSON.stringify(filters), page],
  );

  const tasks = paginatedData?.data ?? [];
  const total = paginatedData?.total ?? 0;
  const totalPages = paginatedData?.total_pages ?? 0;

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
    { key: 'id', header: 'ID', className: 'w-16' },
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

  const canCancel = selectedTask && ['pending', 'in_progress', 'needs_input'].includes(selectedTask.status);

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-brand-dark">Tasks</h1>
        <button
          onClick={refetch}
          className="px-4 py-2 text-sm text-brand-dark border border-brand-shade2 rounded-btn hover:bg-brand-light transition-colors"
        >
          Refresh
        </button>
      </div>

      {/* Filters */}
      <div className="flex gap-3 mb-4">
        <select
          value={filters['status'] ?? ''}
          onChange={(e) => { setFilters({ ...filters, status: e.target.value }); setPage(1); }}
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
          onChange={(e) => { setFilters({ ...filters, source: e.target.value }); setPage(1); }}
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
          onChange={(e) => { setFilters({ ...filters, agent_name: e.target.value }); setPage(1); }}
          className="px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent"
        />
      </div>

      {loading && <div className="text-brand-shade3">Loading tasks...</div>}
      {error && <div className="text-red-600">Error: {error}</div>}

      {!loading && !error && (
        <div className="bg-white rounded-card border border-brand-shade1">
          <DataTable
            columns={columns}
            data={tasks}
            keyField="id"
            onRowClick={handleRowClick}
            activeKey={selectedTask?.id}
            emptyMessage="No tasks found."
            emptyIcon={emptyIcons.tasks}
          />
          {totalPages > 1 && (
            <div className="flex items-center justify-between px-4 py-3 border-t border-brand-shade1">
              <span className="text-sm text-brand-shade3">
                Showing {(page - 1) * PER_PAGE + 1}–{Math.min(page * PER_PAGE, total)} of {total} tasks
              </span>
              <div className="flex gap-1">
                <button
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1}
                  className="px-3 py-1 text-sm border border-brand-shade1 rounded-btn hover:bg-brand-light transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  &lt;
                </button>
                {Array.from({ length: totalPages }, (_, i) => i + 1)
                  .filter((p) => p === 1 || p === totalPages || Math.abs(p - page) <= 2)
                  .reduce<(number | '...')[]>((acc, p, idx, arr) => {
                    if (idx > 0 && p - (arr[idx - 1] ?? 0) > 1) acc.push('...');
                    acc.push(p);
                    return acc;
                  }, [])
                  .map((item, idx) =>
                    item === '...' ? (
                      <span key={`ellipsis-${idx}`} className="px-2 py-1 text-sm text-brand-shade3">...</span>
                    ) : (
                      <button
                        key={item}
                        onClick={() => setPage(item)}
                        className={`px-3 py-1 text-sm border rounded-btn transition-colors ${
                          item === page
                            ? 'bg-brand-accent text-white border-brand-accent'
                            : 'border-brand-shade1 hover:bg-brand-light'
                        }`}
                      >
                        {item}
                      </button>
                    ),
                  )}
                <button
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page >= totalPages}
                  className="px-3 py-1 text-sm border border-brand-shade1 rounded-btn hover:bg-brand-light transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  &gt;
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Detail Panel */}
      <DetailPanel
        open={selectedTask !== null}
        onClose={() => setSelectedTask(null)}
        title={selectedTask ? `Task #${selectedTask.id}: ${selectedTask.title}` : 'Task Detail'}
        actions={
          canCancel ? (
            <button
              onClick={() => handleCancel(selectedTask!.id)}
              className="px-4 py-2 text-sm text-white bg-red-600 rounded-btn hover:bg-red-700 transition-colors font-medium"
            >
              Cancel Task
            </button>
          ) : undefined
        }
      >
        {loadingDetail ? (
          <div className="text-brand-shade3 text-sm">Loading...</div>
        ) : selectedTask ? (
          <>
            <DetailSection title="Overview">
              <DetailRow label="Status"><StatusBadge status={selectedTask.status} /></DetailRow>
              <DetailRow label="Agent">{selectedTask.agent_name}</DetailRow>
              <DetailRow label="Source">
                <span className="text-xs bg-brand-light px-2 py-0.5 rounded">{selectedTask.source}</span>
              </DetailRow>
              <DetailRow label="Mode">{selectedTask.mode}</DetailRow>
            </DetailSection>

            {selectedTask.description && (
              <DetailSection title="Description">
                <p className="text-sm text-brand-dark">{selectedTask.description}</p>
              </DetailSection>
            )}

            {selectedTask.result && (
              <DetailSection title="Result">
                <pre className="p-3 bg-brand-light rounded-btn text-xs whitespace-pre-wrap max-h-48 overflow-y-auto border border-brand-shade1/50">
                  {selectedTask.result}
                </pre>
              </DetailSection>
            )}

            {selectedTask.error && (
              <DetailSection title="Error">
                <pre className="p-3 bg-red-50 rounded-btn text-xs text-red-700 whitespace-pre-wrap max-h-48 overflow-y-auto border border-red-200">
                  {selectedTask.error}
                </pre>
              </DetailSection>
            )}

            <DetailSection title="Timestamps">
              <DetailRow label="Created">{new Date(selectedTask.created_at).toLocaleString()}</DetailRow>
              {selectedTask.started_at && (
                <DetailRow label="Started">{new Date(selectedTask.started_at).toLocaleString()}</DetailRow>
              )}
              {selectedTask.completed_at && (
                <DetailRow label="Completed">{new Date(selectedTask.completed_at).toLocaleString()}</DetailRow>
              )}
            </DetailSection>
          </>
        ) : null}
      </DetailPanel>
    </div>
  );
}
