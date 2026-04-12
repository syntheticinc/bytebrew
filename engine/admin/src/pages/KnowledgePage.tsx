import { useState, useRef, useCallback } from 'react';
import { api } from '../api/client';
import { useApi } from '../hooks/useApi';
import DataTable from '../components/DataTable';
import type { AgentInfo, KnowledgeFile, KnowledgeStatus } from '../types';

const ACCEPTED_TYPES = '.txt,.md,.csv';

function formatStatus(status: KnowledgeStatus['status']) {
  switch (status) {
    case 'ready':
      return <span className="inline-flex items-center gap-1.5 px-2 py-0.5 text-xs font-medium rounded-full bg-emerald-500/15 text-emerald-400">Ready</span>;
    case 'indexing':
      return <span className="inline-flex items-center gap-1.5 px-2 py-0.5 text-xs font-medium rounded-full bg-amber-500/15 text-amber-400">Indexing</span>;
    case 'empty':
      return <span className="inline-flex items-center gap-1.5 px-2 py-0.5 text-xs font-medium rounded-full bg-brand-shade3/15 text-brand-shade3">Empty</span>;
    default:
      return null;
  }
}

function formatFileStatus(status: KnowledgeFile['status'], error?: string) {
  switch (status) {
    case 'ready':
      return <span className="inline-flex items-center px-2 py-0.5 text-xs font-medium rounded-full bg-emerald-500/15 text-emerald-400">Ready</span>;
    case 'indexing':
      return <span className="inline-flex items-center px-2 py-0.5 text-xs font-medium rounded-full bg-amber-500/15 text-amber-400">Indexing</span>;
    case 'uploading':
      return <span className="inline-flex items-center px-2 py-0.5 text-xs font-medium rounded-full bg-blue-500/15 text-blue-400">Uploading</span>;
    case 'error':
      return (
        <span className="inline-flex items-center px-2 py-0.5 text-xs font-medium rounded-full bg-red-500/15 text-red-400" title={error}>
          Error
        </span>
      );
    default:
      return null;
  }
}

const bookOpenIcon = (
  <svg className="w-12 h-12 mx-auto mb-4 text-brand-shade3/40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
    <path d="M2 3h6a4 4 0 014 4v14a3 3 0 00-3-3H2z" />
    <path d="M22 3h-6a4 4 0 00-4 4v14a3 3 0 013-3h7z" />
  </svg>
);

const bookOpenIconSmall = (
  <svg className="w-10 h-10 text-brand-shade3/40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
    <path d="M2 3h6a4 4 0 014 4v14a3 3 0 00-3-3H2z" />
    <path d="M22 3h-6a4 4 0 00-4 4v14a3 3 0 013-3h7z" />
  </svg>
);

const uploadIcon = (
  <svg className="w-8 h-8 text-brand-shade3/50" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
    <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4" />
    <polyline points="17 8 12 3 7 8" />
    <line x1="12" y1="3" x2="12" y2="15" />
  </svg>
);

export default function KnowledgePage() {
  const [selectedAgent, setSelectedAgent] = useState('');
  const [actionError, setActionError] = useState<string | null>(null);
  const [deletingFile, setDeletingFile] = useState<string | null>(null);
  const [reindexingFile, setReindexingFile] = useState<string | null>(null);
  const [reindexingAll, setReindexingAll] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [dragOver, setDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const { data: agents, loading: agentsLoading } = useApi(
    () => api.listAgents(),
    [],
  );

  const {
    data: knowledgeStatus,
    loading: statusLoading,
    refetch: refetchStatus,
  } = useApi(
    () => (selectedAgent ? api.getKnowledgeStatus(selectedAgent) : Promise.resolve(null)),
    [selectedAgent],
  );

  const {
    data: files,
    loading: filesLoading,
    error: filesError,
    refetch: refetchFiles,
  } = useApi(
    () => (selectedAgent ? api.listKnowledgeFiles(selectedAgent) : Promise.resolve([])),
    [selectedAgent],
  );

  const refetch = useCallback(() => {
    refetchStatus();
    refetchFiles();
  }, [refetchStatus, refetchFiles]);

  function handleAgentChange(name: string) {
    setSelectedAgent(name);
    setActionError(null);
    setDeletingFile(null);
    setReindexingFile(null);
  }

  async function handleDelete(fileId: string) {
    if (deletingFile) return;
    setDeletingFile(fileId);
    setActionError(null);
    try {
      await api.deleteKnowledgeFile(selectedAgent, fileId);
      refetch();
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err.message : 'Failed to delete file');
    } finally {
      setDeletingFile(null);
    }
  }

  async function handleReindexFile(fileId: string) {
    if (reindexingFile) return;
    setReindexingFile(fileId);
    setActionError(null);
    try {
      await api.reindexKnowledgeFile(selectedAgent, fileId);
      refetch();
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err.message : 'Failed to reindex file');
    } finally {
      setReindexingFile(null);
    }
  }

  async function handleReindexAll() {
    if (reindexingAll) return;
    setReindexingAll(true);
    setActionError(null);
    try {
      await api.reindexKnowledge(selectedAgent);
      refetch();
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err.message : 'Failed to reindex');
    } finally {
      setReindexingAll(false);
    }
  }

  async function handleUpload(fileList: FileList | null) {
    if (!fileList || fileList.length === 0 || !selectedAgent || uploading) return;
    setUploading(true);
    setActionError(null);
    try {
      for (let i = 0; i < fileList.length; i++) {
        await api.uploadKnowledgeFile(selectedAgent, fileList[i]!);
      }
      refetch();
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err.message : 'Failed to upload file');
    } finally {
      setUploading(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault();
    setDragOver(false);
    handleUpload(e.dataTransfer.files);
  }

  const allFiles = files ?? [];

  const columns = [
    {
      key: 'name',
      header: 'Filename',
      render: (row: KnowledgeFile) => (
        <span className="text-sm text-brand-light font-medium">{row.name}</span>
      ),
    },
    {
      key: 'type',
      header: 'Type',
      className: 'w-24',
      render: (row: KnowledgeFile) => (
        <span className="text-xs text-brand-shade3 font-mono uppercase">{row.type}</span>
      ),
    },
    {
      key: 'size',
      header: 'Size',
      className: 'w-28',
      render: (row: KnowledgeFile) => (
        <span className="text-xs text-brand-shade3 font-mono">{row.size}</span>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      className: 'w-28',
      render: (row: KnowledgeFile) => formatFileStatus(row.status, row.error),
    },
    {
      key: 'uploaded_at',
      header: 'Uploaded',
      className: 'w-44',
      render: (row: KnowledgeFile) => (
        <span className="text-xs text-brand-shade3 font-mono">
          {new Date(row.uploaded_at).toLocaleString()}
        </span>
      ),
    },
    {
      key: 'actions',
      header: '',
      className: 'w-40',
      render: (row: KnowledgeFile) => (
        <div className="flex items-center gap-2">
          <button
            onClick={(e) => {
              e.stopPropagation();
              if (row.id) handleReindexFile(row.id);
            }}
            disabled={reindexingFile === row.id || row.status === 'indexing' || row.status === 'uploading'}
            className="px-2.5 py-1 text-xs text-brand-shade2 border border-brand-shade3/30 rounded-btn hover:bg-brand-dark-alt hover:text-brand-light transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {reindexingFile === row.id ? 'Reindexing...' : 'Reindex'}
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation();
              if (row.id) handleDelete(row.id);
            }}
            disabled={deletingFile === row.id}
            className="px-2.5 py-1 text-xs text-red-400 border border-red-500/30 rounded-btn hover:bg-red-500/10 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {deletingFile === row.id ? 'Deleting...' : 'Delete'}
          </button>
        </div>
      ),
    },
  ];

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-brand-light">Knowledge Base</h1>
        <div className="flex items-center gap-3">
          {selectedAgent && allFiles.length > 0 && (
            <button
              onClick={handleReindexAll}
              disabled={reindexingAll}
              className="px-4 py-2 text-sm text-brand-shade2 border border-brand-shade3/30 rounded-btn hover:bg-brand-dark-alt hover:text-brand-light transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {reindexingAll ? 'Reindexing...' : 'Reindex All'}
            </button>
          )}
          <button
            onClick={refetch}
            disabled={!selectedAgent}
            className="px-4 py-2 text-sm text-brand-shade2 border border-brand-shade3/30 rounded-btn hover:bg-brand-dark-alt hover:text-brand-light transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Agent selector */}
      <div className="mb-4 flex items-center gap-4">
        <select
          value={selectedAgent}
          onChange={(e) => handleAgentChange(e.target.value)}
          disabled={agentsLoading}
          className="px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-btn text-sm text-brand-light focus:outline-none focus:border-brand-accent min-w-[220px]"
        >
          <option value="">Select an agent...</option>
          {(agents ?? []).map((a: AgentInfo) => (
            <option key={a.name} value={a.name}>
              {a.name}
            </option>
          ))}
        </select>

        {/* Status badge */}
        {selectedAgent && !statusLoading && knowledgeStatus && (
          <div className="flex items-center gap-3">
            {formatStatus(knowledgeStatus.status)}
            <span className="text-xs text-brand-shade3">
              {knowledgeStatus.indexed_files}/{knowledgeStatus.total_files} files indexed
            </span>
          </div>
        )}
      </div>

      {actionError && (
        <div className="mb-4 p-3 bg-red-500/10 border border-red-500/30 rounded-btn text-sm text-red-400">
          Error: {actionError}
        </div>
      )}

      {/* Empty state: no agent selected */}
      {!selectedAgent && (
        <div className="bg-brand-dark-alt rounded-card border border-brand-shade3/15 p-12 text-center">
          {bookOpenIcon}
          <p className="text-sm text-brand-shade3">Select an agent to manage its knowledge base</p>
        </div>
      )}

      {/* Loading */}
      {selectedAgent && filesLoading && (
        <div className="text-brand-shade3 text-sm py-4">Loading knowledge files...</div>
      )}

      {/* Error */}
      {selectedAgent && filesError && (
        <div className="mb-4 p-3 bg-red-500/10 border border-red-500/30 rounded-btn text-sm text-red-400">
          Error: {filesError}
        </div>
      )}

      {/* Content */}
      {selectedAgent && !filesLoading && !filesError && (
        <>
          {/* Upload area */}
          <div
            onDragOver={(e) => {
              e.preventDefault();
              setDragOver(true);
            }}
            onDragLeave={() => setDragOver(false)}
            onDrop={handleDrop}
            onClick={() => fileInputRef.current?.click()}
            className={[
              'mb-4 rounded-card border-2 border-dashed p-6 text-center cursor-pointer transition-all duration-150',
              dragOver
                ? 'border-brand-accent bg-brand-accent/5'
                : 'border-brand-shade3/30 hover:border-brand-shade3/50 hover:bg-brand-dark-alt/50',
            ].join(' ')}
          >
            <div className="flex flex-col items-center gap-2">
              {uploadIcon}
              {uploading ? (
                <p className="text-sm text-brand-shade2">Uploading...</p>
              ) : (
                <>
                  <p className="text-sm text-brand-shade2">
                    Drop files here or <span className="text-brand-accent">browse</span>
                  </p>
                  <p className="text-xs text-brand-shade3">Supported: TXT, MD, CSV</p>
                </>
              )}
            </div>
            <input
              ref={fileInputRef}
              type="file"
              accept={ACCEPTED_TYPES}
              multiple
              className="hidden"
              onChange={(e) => handleUpload(e.target.files)}
            />
          </div>

          {/* Files table */}
          <div className="bg-brand-dark-alt rounded-card border border-brand-shade3/15">
            <DataTable
              columns={columns}
              data={allFiles}
              keyField="id"
              emptyMessage="No knowledge files uploaded"
              emptyIcon={bookOpenIconSmall}
            />
          </div>
        </>
      )}
    </div>
  );
}
