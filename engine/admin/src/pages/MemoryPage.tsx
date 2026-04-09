import { useState } from 'react';
import { api } from '../api/client';
import { useApi } from '../hooks/useApi';
import DataTable from '../components/DataTable';
import type { MemoryEntry, Schema } from '../types';

const PER_PAGE = 50;

export default function MemoryPage() {
  const [selectedSchemaId, setSelectedSchemaId] = useState<string>('');
  const [page, setPage] = useState(1);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [clearing, setClearing] = useState(false);
  const [confirmClear, setConfirmClear] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  const { data: schemas, loading: schemasLoading } = useApi(
    () => api.listSchemas(),
    [],
  );

  const {
    data: memories,
    loading: memoriesLoading,
    error: memoriesError,
    refetch,
  } = useApi(
    () => (selectedSchemaId ? api.listMemories(selectedSchemaId) : Promise.resolve([])),
    [selectedSchemaId],
  );

  const allEntries = memories ?? [];
  const totalPages = Math.ceil(allEntries.length / PER_PAGE);
  const pagedEntries = allEntries.slice((page - 1) * PER_PAGE, page * PER_PAGE);

  async function handleDelete(entry: MemoryEntry) {
    if (deletingId) return;
    setDeletingId(entry.id);
    setActionError(null);
    try {
      await api.deleteMemory(selectedSchemaId, entry.id);
      refetch();
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err.message : 'Failed to delete memory');
    } finally {
      setDeletingId(null);
    }
  }

  async function handleClearAll() {
    if (clearing) return;
    setClearing(true);
    setActionError(null);
    try {
      await api.clearMemories(selectedSchemaId);
      setConfirmClear(false);
      refetch();
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err.message : 'Failed to clear memories');
    } finally {
      setClearing(false);
    }
  }

  function handleSchemaChange(id: string) {
    setSelectedSchemaId(id);
    setPage(1);
    setConfirmClear(false);
    setActionError(null);
  }

  const columns = [
    {
      key: 'content',
      header: 'Content',
      render: (row: MemoryEntry) => (
        <span
          className="text-sm text-brand-shade2 block max-w-[400px] truncate"
          title={row.content}
        >
          {row.content.length > 120 ? row.content.substring(0, 120) + '...' : row.content}
        </span>
      ),
    },
    {
      key: 'user_id',
      header: 'User',
      className: 'w-32',
      render: (row: MemoryEntry) => (
        <span className="text-xs text-brand-shade3 font-mono">
          {row.user_id || '-'}
        </span>
      ),
    },
    {
      key: 'metadata',
      header: 'Metadata',
      className: 'w-48',
      render: (row: MemoryEntry) => {
        if (!row.metadata || Object.keys(row.metadata).length === 0) {
          return <span className="text-xs text-brand-shade3">-</span>;
        }
        const text = JSON.stringify(row.metadata);
        return (
          <span
            className="text-xs text-brand-shade3 font-mono block max-w-[180px] truncate"
            title={JSON.stringify(row.metadata, null, 2)}
          >
            {text.length > 60 ? text.substring(0, 60) + '...' : text}
          </span>
        );
      },
    },
    {
      key: 'created_at',
      header: 'Created',
      className: 'w-44',
      render: (row: MemoryEntry) => (
        <span className="text-xs text-brand-shade3 font-mono">
          {new Date(row.created_at).toLocaleString()}
        </span>
      ),
    },
    {
      key: 'actions',
      header: '',
      className: 'w-20',
      render: (row: MemoryEntry) => (
        <button
          onClick={(e) => {
            e.stopPropagation();
            handleDelete(row);
          }}
          disabled={deletingId === row.id}
          className="px-2.5 py-1 text-xs text-red-400 border border-red-500/30 rounded-btn hover:bg-red-500/10 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {deletingId === row.id ? 'Deleting...' : 'Delete'}
        </button>
      ),
    },
  ];

  const schemaName = schemas?.find((s: Schema) => String(s.id) === selectedSchemaId)?.name;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-brand-light">Memory</h1>
        <div className="flex items-center gap-3">
          {selectedSchemaId && allEntries.length > 0 && (
            <>
              {confirmClear ? (
                <div className="flex items-center gap-2">
                  <span className="text-xs text-red-400">
                    Clear all {allEntries.length} memories for {schemaName ?? 'this schema'}?
                  </span>
                  <button
                    onClick={handleClearAll}
                    disabled={clearing}
                    className="px-3 py-1.5 text-xs font-medium text-white bg-red-600 rounded-btn hover:bg-red-700 transition-colors disabled:opacity-50"
                  >
                    {clearing ? 'Clearing...' : 'Confirm'}
                  </button>
                  <button
                    onClick={() => setConfirmClear(false)}
                    className="px-3 py-1.5 text-xs text-brand-shade3 hover:text-brand-light transition-colors"
                  >
                    Cancel
                  </button>
                </div>
              ) : (
                <button
                  onClick={() => setConfirmClear(true)}
                  className="px-4 py-2 text-sm text-red-400 border border-red-500/30 rounded-btn hover:bg-red-500/10 transition-colors"
                >
                  Clear All
                </button>
              )}
            </>
          )}
          <button
            onClick={refetch}
            disabled={!selectedSchemaId}
            className="px-4 py-2 text-sm text-brand-shade2 border border-brand-shade3/30 rounded-btn hover:bg-brand-dark-alt hover:text-brand-light transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Schema selector */}
      <div className="mb-4">
        <select
          value={selectedSchemaId}
          onChange={(e) => handleSchemaChange(e.target.value)}
          disabled={schemasLoading}
          className="px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-btn text-sm text-brand-light focus:outline-none focus:border-brand-accent min-w-[220px]"
        >
          <option value="">Select a schema...</option>
          {(schemas ?? []).map((s: Schema) => (
            <option key={s.id} value={String(s.id)}>
              {s.name}
            </option>
          ))}
        </select>
      </div>

      {actionError && (
        <div className="mb-4 p-3 bg-red-500/10 border border-red-500/30 rounded-btn text-sm text-red-400">
          Error: {actionError}
        </div>
      )}

      {!selectedSchemaId && (
        <div className="bg-brand-dark-alt rounded-card border border-brand-shade3/15 p-12 text-center">
          <svg className="w-12 h-12 mx-auto mb-4 text-brand-shade3/40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
            <path d="M12 2a7 7 0 017 7c0 2.38-1.19 4.47-3 5.74V17a2 2 0 01-2 2h-4a2 2 0 01-2-2v-2.26C6.19 13.47 5 11.38 5 9a7 7 0 017-7z" />
            <path d="M9 21h6M10 17v4M14 17v4" />
          </svg>
          <p className="text-sm text-brand-shade3">Select a schema to view stored memories</p>
        </div>
      )}

      {selectedSchemaId && memoriesLoading && (
        <div className="text-brand-shade3 text-sm py-4">Loading memories...</div>
      )}

      {selectedSchemaId && memoriesError && (
        <div className="mb-4 p-3 bg-red-500/10 border border-red-500/30 rounded-btn text-sm text-red-400">
          Error: {memoriesError}
        </div>
      )}

      {selectedSchemaId && !memoriesLoading && !memoriesError && (
        <div className="bg-brand-dark-alt rounded-card border border-brand-shade3/15">
          <DataTable
            columns={columns}
            data={pagedEntries}
            keyField="id"
            emptyMessage="No stored memories"
            emptyIcon={
              <svg className="w-10 h-10 text-brand-shade3/40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                <path d="M12 2a7 7 0 017 7c0 2.38-1.19 4.47-3 5.74V17a2 2 0 01-2 2h-4a2 2 0 01-2-2v-2.26C6.19 13.47 5 11.38 5 9a7 7 0 017-7z" />
                <path d="M9 21h6M10 17v4M14 17v4" />
              </svg>
            }
          />

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between px-4 py-3 border-t border-brand-shade3/15">
              <span className="text-sm text-brand-shade3">
                Showing {(page - 1) * PER_PAGE + 1}--{Math.min(page * PER_PAGE, allEntries.length)} of {allEntries.length}
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
