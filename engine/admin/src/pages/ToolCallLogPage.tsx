import { useState } from 'react';
import { api } from '../api/client';
import { useApi } from '../hooks/useApi';
import DataTable from '../components/DataTable';
import type { ToolCallEntry } from '../types';

// ToolCallLogPage — list every tool call across every session, filterable and
// drill-in-able. OSS users rely on this to answer:
//   - Why did the agent call this tool with those arguments?
//   - What did the tool return?
//   - How long did it take?
//   - Did it fail?
//
// Uses the /api/v1/audit/tool-calls endpoint (rows are derived from
// session_event_log tool_call_start/tool_call_end pairs).
const PER_PAGE = 50;
const STATUS_OPTIONS = ['', 'completed', 'failed'];

function formatDuration(ms: number): string {
  if (!ms || ms < 0) return '-';
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60_000) return `${(ms / 1000).toFixed(2)}s`;
  const minutes = Math.floor(ms / 60_000);
  const seconds = ((ms % 60_000) / 1000).toFixed(0);
  return `${minutes}m ${seconds}s`;
}

function prettyJSON(raw: string): string {
  if (!raw) return '';
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return raw;
  }
}

function preview(text: string, max = 80): string {
  if (!text) return '-';
  const collapsed = text.replace(/\s+/g, ' ').trim();
  if (collapsed.length <= max) return collapsed;
  return collapsed.slice(0, max) + '...';
}

export default function ToolCallLogPage() {
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [page, setPage] = useState(1);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const { data: paginated, loading, error, refetch } = useApi(
    () => {
      const params: Record<string, string> = {
        page: String(page),
        per_page: String(PER_PAGE),
      };
      for (const [k, v] of Object.entries(filters)) {
        if (v) params[k] = v;
      }
      return api.listToolCalls(params);
    },
    [JSON.stringify(filters), page],
  );

  const entries: ToolCallEntry[] = paginated?.data ?? [];
  const total = paginated?.total ?? 0;
  const totalPages = paginated?.total_pages ?? 0;

  function updateFilter(key: string, value: string) {
    setFilters({ ...filters, [key]: value });
    setPage(1);
  }

  function handleRowClick(row: ToolCallEntry) {
    setExpandedId(expandedId === row.id ? null : row.id);
  }

  const columns = [
    {
      key: 'created_at',
      header: 'Timestamp',
      className: 'w-48',
      render: (row: ToolCallEntry) => (
        <span className="text-xs text-brand-shade3 font-mono">
          {new Date(row.created_at).toLocaleString()}
        </span>
      ),
    },
    {
      key: 'agent_name',
      header: 'Agent',
      className: 'w-40',
      render: (row: ToolCallEntry) => (
        <span className="text-xs text-brand-shade2 truncate block max-w-[160px]" title={row.agent_name}>
          {row.agent_name || '-'}
        </span>
      ),
    },
    {
      key: 'tool_name',
      header: 'Tool',
      className: 'w-48',
      render: (row: ToolCallEntry) => (
        <span className="text-xs text-brand-light font-mono bg-brand-dark px-2 py-0.5 rounded">
          {row.tool_name || '-'}
        </span>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      className: 'w-24',
      render: (row: ToolCallEntry) => (
        <span
          className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
            row.status === 'failed'
              ? 'bg-red-500/15 text-red-400'
              : 'bg-status-active/15 text-status-active'
          }`}
        >
          {row.status}
        </span>
      ),
    },
    {
      key: 'duration_ms',
      header: 'Duration',
      className: 'w-24',
      render: (row: ToolCallEntry) => (
        <span className="text-xs text-brand-shade3 font-mono">
          {formatDuration(row.duration_ms)}
        </span>
      ),
    },
    {
      key: 'input',
      header: 'Arguments',
      render: (row: ToolCallEntry) => (
        <span className="text-xs text-brand-shade3 truncate block max-w-[260px] font-mono" title={row.input}>
          {preview(row.input, 80)}
        </span>
      ),
    },
    {
      key: 'output',
      header: 'Result',
      render: (row: ToolCallEntry) => (
        <span className="text-xs text-brand-shade3 truncate block max-w-[260px]" title={row.output}>
          {preview(row.output, 80)}
        </span>
      ),
    },
  ];

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-brand-light">Tool Call Log</h1>
          <p className="text-sm text-brand-shade3 mt-1">
            Every tool invoked by every agent, with arguments, results, timing and status.
          </p>
        </div>
        <button
          onClick={refetch}
          className="px-4 py-2 text-sm text-brand-shade2 border border-brand-shade3/30 rounded-btn hover:bg-brand-dark-alt hover:text-brand-light transition-colors"
        >
          Refresh
        </button>
      </div>

      {/* Filters */}
      <div className="flex gap-3 mb-4 flex-wrap">
        <input
          type="text"
          value={filters['agent'] ?? ''}
          onChange={(e) => updateFilter('agent', e.target.value)}
          placeholder="Agent name"
          className="px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-btn text-sm text-brand-light focus:outline-none focus:border-brand-accent placeholder:text-brand-shade3"
        />
        <input
          type="text"
          value={filters['tool'] ?? ''}
          onChange={(e) => updateFilter('tool', e.target.value)}
          placeholder="Tool name"
          className="px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-btn text-sm text-brand-light focus:outline-none focus:border-brand-accent placeholder:text-brand-shade3"
        />
        <input
          type="text"
          value={filters['session_id'] ?? ''}
          onChange={(e) => updateFilter('session_id', e.target.value)}
          placeholder="Session ID"
          className="px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-btn text-sm text-brand-light focus:outline-none focus:border-brand-accent placeholder:text-brand-shade3"
        />
        <select
          value={filters['status'] ?? ''}
          onChange={(e) => updateFilter('status', e.target.value)}
          className="px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-btn text-sm text-brand-light focus:outline-none focus:border-brand-accent"
        >
          <option value="">All statuses</option>
          {STATUS_OPTIONS.filter(Boolean).map((s) => (
            <option key={s} value={s}>{s}</option>
          ))}
        </select>
        <input
          type="date"
          value={filters['from'] ?? ''}
          onChange={(e) => updateFilter('from', e.target.value)}
          className="px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-btn text-sm text-brand-light focus:outline-none focus:border-brand-accent"
        />
        <input
          type="date"
          value={filters['to'] ?? ''}
          onChange={(e) => updateFilter('to', e.target.value)}
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

      {loading && <div className="text-brand-shade3 text-sm py-4">Loading tool calls...</div>}
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
            emptyMessage="No tool calls logged yet. Start a chat session with an agent and return here."
          />

          {expandedId && (
            <ExpandedDetail
              entry={entries.find((e) => e.id === expandedId)}
              onClose={() => setExpandedId(null)}
            />
          )}

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
                <span className="px-3 py-1 text-sm text-brand-shade3">
                  {page} / {totalPages}
                </span>
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

function ExpandedDetail({ entry, onClose }: { entry?: ToolCallEntry; onClose: () => void }) {
  if (!entry) return null;
  return (
    <div className="border-t border-brand-shade3/15 bg-brand-dark px-6 py-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-xs font-semibold text-brand-shade3 uppercase tracking-wider">
          Tool call {entry.id}
        </h3>
        <button
          onClick={onClose}
          className="text-brand-shade3 hover:text-brand-light transition-colors text-xs"
        >
          Close
        </button>
      </div>

      <div className="grid grid-cols-2 gap-x-6 gap-y-2 mb-4">
        <div>
          <span className="text-xs text-brand-shade3">Timestamp</span>
          <p className="text-sm text-brand-light font-mono">{new Date(entry.created_at).toLocaleString()}</p>
        </div>
        <div>
          <span className="text-xs text-brand-shade3">Duration</span>
          <p className="text-sm text-brand-light font-mono">{formatDuration(entry.duration_ms)}</p>
        </div>
        <div>
          <span className="text-xs text-brand-shade3">Session</span>
          <p className="text-sm text-brand-light font-mono break-all">{entry.session_id || '-'}</p>
        </div>
        <div>
          <span className="text-xs text-brand-shade3">Agent</span>
          <p className="text-sm text-brand-light">{entry.agent_name || '-'}</p>
        </div>
        <div>
          <span className="text-xs text-brand-shade3">Tool</span>
          <p className="text-sm text-brand-light font-mono">{entry.tool_name || '-'}</p>
        </div>
        <div>
          <span className="text-xs text-brand-shade3">Status</span>
          <p className={`text-sm ${entry.status === 'failed' ? 'text-red-400' : 'text-status-active'}`}>
            {entry.status}
          </p>
        </div>
      </div>

      <div className="mb-3">
        <span className="text-xs text-brand-shade3">Arguments</span>
        <pre className="mt-1 p-3 bg-brand-dark-alt rounded-btn text-xs text-brand-shade2 whitespace-pre-wrap max-h-60 overflow-y-auto border border-brand-shade3/15 font-mono">
          {prettyJSON(entry.input) || '(no arguments)'}
        </pre>
      </div>

      <div>
        <span className="text-xs text-brand-shade3">Result</span>
        <pre className="mt-1 p-3 bg-brand-dark-alt rounded-btn text-xs text-brand-shade2 whitespace-pre-wrap max-h-96 overflow-y-auto border border-brand-shade3/15 font-mono">
          {entry.output || '(no output)'}
        </pre>
      </div>
    </div>
  );
}
