import { useState } from 'react';
import { api } from '../api/client';
import { useApi } from '../hooks/useApi';
import DataTable from '../components/DataTable';
import StatusBadge from '../components/StatusBadge';
import type { AuditEntry } from '../types';

const ACTOR_TYPE_OPTIONS = ['', 'admin', 'api_token'];
const ACTION_OPTIONS = ['', 'create', 'update', 'delete', 'login'];
const RESOURCE_OPTIONS = ['', 'agent', 'model', 'mcp_server', 'trigger', 'token'];
const PER_PAGE = 25;

export default function AuditPage() {
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [page, setPage] = useState(1);
  const [expandedId, setExpandedId] = useState<number | null>(null);

  const { data: paginatedData, loading, error, refetch } = useApi(
    () => {
      const params: Record<string, string> = {
        page: String(page),
        per_page: String(PER_PAGE),
      };
      for (const [k, v] of Object.entries(filters)) {
        if (v) params[k] = v;
      }
      return api.listAuditLogs(params);
    },
    [JSON.stringify(filters), page],
  );

  const entries = paginatedData?.data ?? [];
  const total = paginatedData?.total ?? 0;
  const totalPages = paginatedData?.total_pages ?? 0;

  function updateFilter(key: string, value: string) {
    setFilters({ ...filters, [key]: value });
    setPage(1);
  }

  function handleRowClick(row: AuditEntry) {
    setExpandedId(expandedId === row.id ? null : row.id);
  }

  const actionColorMap: Record<string, string> = {
    create: 'bg-status-active/15 text-status-active',
    update: 'bg-brand-accent/15 text-brand-accent',
    delete: 'bg-red-500/15 text-red-400',
    login: 'bg-purple-500/15 text-purple-400',
  };

  const columns = [
    {
      key: 'timestamp',
      header: 'Timestamp',
      className: 'w-48',
      render: (row: AuditEntry) => (
        <span className="text-xs text-brand-shade3 font-mono">
          {new Date(row.timestamp).toLocaleString()}
        </span>
      ),
    },
    {
      key: 'actor_type',
      header: 'Actor',
      className: 'w-40',
      render: (row: AuditEntry) => (
        <div className="flex flex-col gap-0.5">
          <StatusBadge status={row.actor_type} />
          <span className="text-xs text-brand-shade3 truncate max-w-[120px]" title={row.actor_id}>
            {row.actor_id}
          </span>
        </div>
      ),
    },
    {
      key: 'action',
      header: 'Action',
      className: 'w-28',
      render: (row: AuditEntry) => (
        <span
          className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
            actionColorMap[row.action] ?? 'bg-brand-shade3/15 text-brand-shade3'
          }`}
        >
          {row.action}
        </span>
      ),
    },
    {
      key: 'resource',
      header: 'Resource',
      className: 'w-32',
      render: (row: AuditEntry) => (
        <span className="text-xs text-brand-shade2 bg-brand-dark px-2 py-0.5 rounded">
          {row.resource}
        </span>
      ),
    },
    {
      key: 'details',
      header: 'Details',
      render: (row: AuditEntry) => (
        <span className="text-xs text-brand-shade3 truncate block max-w-[300px]" title={row.details}>
          {row.details && row.details.length > 80
            ? row.details.substring(0, 80) + '...'
            : row.details || '-'}
        </span>
      ),
    },
  ];

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-brand-light">Audit Log</h1>
        <button
          onClick={refetch}
          className="px-4 py-2 text-sm text-brand-shade2 border border-brand-shade3/30 rounded-btn hover:bg-brand-dark-alt hover:text-brand-light transition-colors"
        >
          Refresh
        </button>
      </div>

      {/* Filters */}
      <div className="flex gap-3 mb-4 flex-wrap">
        <select
          value={filters['actor_type'] ?? ''}
          onChange={(e) => updateFilter('actor_type', e.target.value)}
          className="px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-btn text-sm text-brand-light focus:outline-none focus:border-brand-accent"
        >
          <option value="">All actors</option>
          {ACTOR_TYPE_OPTIONS.filter(Boolean).map((s) => (
            <option key={s} value={s}>{s.replace(/_/g, ' ')}</option>
          ))}
        </select>
        <select
          value={filters['action'] ?? ''}
          onChange={(e) => updateFilter('action', e.target.value)}
          className="px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-btn text-sm text-brand-light focus:outline-none focus:border-brand-accent"
        >
          <option value="">All actions</option>
          {ACTION_OPTIONS.filter(Boolean).map((s) => (
            <option key={s} value={s}>{s}</option>
          ))}
        </select>
        <select
          value={filters['resource'] ?? ''}
          onChange={(e) => updateFilter('resource', e.target.value)}
          className="px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-btn text-sm text-brand-light focus:outline-none focus:border-brand-accent"
        >
          <option value="">All resources</option>
          {RESOURCE_OPTIONS.filter(Boolean).map((s) => (
            <option key={s} value={s}>{s.replace(/_/g, ' ')}</option>
          ))}
        </select>
        <input
          type="date"
          value={filters['from'] ?? ''}
          onChange={(e) => updateFilter('from', e.target.value)}
          placeholder="From"
          className="px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-btn text-sm text-brand-light focus:outline-none focus:border-brand-accent"
        />
        <input
          type="date"
          value={filters['to'] ?? ''}
          onChange={(e) => updateFilter('to', e.target.value)}
          placeholder="To"
          className="px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-btn text-sm text-brand-light focus:outline-none focus:border-brand-accent"
        />
        {Object.values(filters).some(Boolean) && (
          <button
            onClick={() => { setFilters({}); setPage(1); }}
            className="px-3 py-2 text-sm text-brand-shade3 hover:text-brand-light transition-colors"
          >
            Clear filters
          </button>
        )}
      </div>

      {loading && <div className="text-brand-shade3 text-sm py-4">Loading audit logs...</div>}
      {error && (
        <div className="mb-4 p-3 bg-red-500/10 border border-red-500/30 rounded-btn text-sm text-red-400">
          Error: {error}
        </div>
      )}

      {!loading && !error && (
        <div className="bg-brand-dark-alt rounded-card border border-brand-shade3/15">
          <DataTable
            columns={columns}
            data={entries}
            keyField="id"
            onRowClick={handleRowClick}
            activeKey={expandedId}
            emptyMessage="No audit entries found."
          />

          {/* Expanded detail row */}
          {expandedId && (
            <ExpandedDetail
              entry={entries.find((e) => e.id === expandedId)}
              onClose={() => setExpandedId(null)}
            />
          )}

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between px-4 py-3 border-t border-brand-shade3/15">
              <span className="text-sm text-brand-shade3">
                Showing {(page - 1) * PER_PAGE + 1}--{Math.min(page * PER_PAGE, total)} of {total}
              </span>
              <div className="flex gap-1">
                <button
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1}
                  className="px-3 py-1 text-sm border border-brand-shade3/30 text-brand-shade2 rounded-btn hover:bg-brand-dark hover:text-brand-light transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
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
                            : 'border-brand-shade3/30 text-brand-shade2 hover:bg-brand-dark hover:text-brand-light'
                        }`}
                      >
                        {item}
                      </button>
                    ),
                  )}
                <button
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page >= totalPages}
                  className="px-3 py-1 text-sm border border-brand-shade3/30 text-brand-shade2 rounded-btn hover:bg-brand-dark hover:text-brand-light transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  &gt;
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function ExpandedDetail({ entry, onClose }: { entry?: AuditEntry; onClose: () => void }) {
  if (!entry) return null;

  let parsedDetails: Record<string, unknown> | null = null;
  try {
    parsedDetails = JSON.parse(entry.details) as Record<string, unknown>;
  } catch {
    // not JSON, show as string
  }

  return (
    <div className="border-t border-brand-shade3/15 bg-brand-dark px-6 py-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-xs font-semibold text-brand-shade3 uppercase tracking-wider">
          Entry #{entry.id} Details
        </h3>
        <button
          onClick={onClose}
          className="text-brand-shade3 hover:text-brand-light transition-colors text-xs"
        >
          Close
        </button>
      </div>
      <div className="grid grid-cols-2 gap-x-6 gap-y-2 mb-3">
        <div>
          <span className="text-xs text-brand-shade3">Timestamp</span>
          <p className="text-sm text-brand-light font-mono">{new Date(entry.timestamp).toLocaleString()}</p>
        </div>
        <div>
          <span className="text-xs text-brand-shade3">Actor</span>
          <p className="text-sm text-brand-light">{entry.actor_type}: {entry.actor_id}</p>
        </div>
        <div>
          <span className="text-xs text-brand-shade3">Action</span>
          <p className="text-sm text-brand-light">{entry.action}</p>
        </div>
        <div>
          <span className="text-xs text-brand-shade3">Resource</span>
          <p className="text-sm text-brand-light">{entry.resource}</p>
        </div>
      </div>
      {entry.details && (
        <div>
          <span className="text-xs text-brand-shade3">Details</span>
          {parsedDetails ? (
            <pre className="mt-1 p-3 bg-brand-dark-alt rounded-btn text-xs text-brand-shade2 whitespace-pre-wrap max-h-60 overflow-y-auto border border-brand-shade3/15 font-mono">
              {JSON.stringify(parsedDetails, null, 2)}
            </pre>
          ) : (
            <p className="mt-1 text-sm text-brand-shade2 whitespace-pre-wrap">{entry.details}</p>
          )}
        </div>
      )}
    </div>
  );
}
