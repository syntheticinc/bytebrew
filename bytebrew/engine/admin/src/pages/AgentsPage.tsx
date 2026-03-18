import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { useApi } from '../hooks/useApi';
import DataTable from '../components/DataTable';
import StatusBadge from '../components/StatusBadge';
import Modal from '../components/Modal';
import type { AgentInfo } from '../types';

export default function AgentsPage() {
  const { data: agents, loading, error, refetch } = useApi(() => api.listAgents());
  const navigate = useNavigate();
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);

  async function handleDelete() {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await api.deleteAgent(deleteTarget);
      setDeleteTarget(null);
      refetch();
    } catch {
      // error is visible in console
    } finally {
      setDeleting(false);
    }
  }

  const columns = [
    { key: 'name', header: 'Name' },
    {
      key: 'kit',
      header: 'Kit',
      render: (row: AgentInfo) =>
        row.kit ? (
          <span className="px-2 py-0.5 bg-brand-accent/10 text-brand-accent rounded text-xs font-medium">
            {row.kit}
          </span>
        ) : (
          <span className="text-brand-shade3">-</span>
        ),
    },
    { key: 'tools_count', header: 'Tools' },
    {
      key: 'has_knowledge',
      header: 'Knowledge',
      render: (row: AgentInfo) => (row.has_knowledge ? <StatusBadge status="active" /> : <span className="text-brand-shade3">-</span>),
    },
    {
      key: 'actions',
      header: '',
      render: (row: AgentInfo) => (
        <div className="flex gap-2">
          <button
            onClick={(e) => {
              e.stopPropagation();
              navigate(`/agents/${row.name}/edit`);
            }}
            className="text-brand-accent hover:text-brand-accent-hover text-sm"
          >
            Edit
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation();
              setDeleteTarget(row.name);
            }}
            className="text-red-600 hover:text-red-800 text-sm"
          >
            Delete
          </button>
        </div>
      ),
    },
  ];

  if (loading) return <div className="text-brand-shade3">Loading agents...</div>;
  if (error) return <div className="text-red-600">Error: {error}</div>;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-brand-dark">Agents</h1>
        <button
          onClick={() => navigate('/agents/new')}
          className="px-4 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover transition-colors"
        >
          Create Agent
        </button>
      </div>

      <div className="bg-white rounded-card border border-brand-shade1">
        <DataTable
          columns={columns}
          data={agents ?? []}
          keyField="name"
          onRowClick={(row) => navigate(`/agents/${row.name}/edit`)}
          emptyMessage="No agents configured. Create your first agent."
        />
      </div>

      <Modal
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        title="Delete Agent"
        footer={
          <>
            <button
              onClick={() => setDeleteTarget(null)}
              className="px-4 py-2 text-sm text-brand-dark border border-brand-shade2 rounded-btn hover:bg-brand-light"
            >
              Cancel
            </button>
            <button
              onClick={handleDelete}
              disabled={deleting}
              className="px-4 py-2 text-sm text-white bg-red-600 rounded-btn hover:bg-red-700 disabled:opacity-50"
            >
              {deleting ? 'Deleting...' : 'Delete'}
            </button>
          </>
        }
      >
        <p className="text-sm text-brand-shade3">
          Are you sure you want to delete agent <strong className="text-brand-dark">{deleteTarget}</strong>? This action cannot
          be undone.
        </p>
      </Modal>
    </div>
  );
}
