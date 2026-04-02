import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { useApi } from '../hooks/useApi';
import DataTable from '../components/DataTable';
import { emptyIcons } from '../components/EmptyState';
import StatusBadge from '../components/StatusBadge';
import DetailPanel, { DetailRow, DetailSection } from '../components/DetailPanel';
import ConfirmDialog from '../components/ConfirmDialog';
import type { AgentInfo, AgentDetail } from '../types';

export default function AgentsPage() {
  const { data: agents, loading, error, refetch } = useApi(() => api.listAgents());
  const navigate = useNavigate();

  const [selected, setSelected] = useState<AgentDetail | null>(null);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);

  async function handleRowClick(row: AgentInfo) {
    setLoadingDetail(true);
    try {
      const detail = await api.getAgent(row.name);
      setSelected(detail);
    } catch {
      // visible in console
    } finally {
      setLoadingDetail(false);
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await api.deleteAgent(deleteTarget);
      setDeleteTarget(null);
      setSelected(null);
      refetch();
    } catch {
      // visible in console
    } finally {
      setDeleting(false);
    }
  }

  const columns = [
    {
      key: 'name',
      header: 'Name',
      render: (row: AgentInfo) => (
        <span className="flex items-center gap-2">
          {row.name}
        </span>
      ),
    },
    {
      key: 'kit',
      header: 'Kit',
      render: (row: AgentInfo) =>
        row.kit ? (
          <span className="px-2 py-0.5 bg-brand-accent/10 text-brand-accent rounded text-xs font-medium">
            {row.kit}
          </span>
        ) : (
          <span className="text-brand-shade3">--</span>
        ),
    },
    { key: 'tools_count', header: 'Tools' },
    {
      key: 'has_knowledge',
      header: 'Knowledge',
      render: (row: AgentInfo) =>
        row.has_knowledge ? (
          <StatusBadge status="active" />
        ) : (
          <span className="text-brand-shade3">--</span>
        ),
    },
  ];

  if (loading) return <div className="text-brand-shade3">Loading agents...</div>;
  if (error) return <div className="text-red-400">Error: {error}</div>;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-brand-light">Agents</h1>
        <button
          onClick={() => navigate('/agents/new')}
          className="px-4 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover transition-colors"
        >
          Create Agent
        </button>
      </div>

      <div className="bg-brand-dark-alt rounded-card border border-brand-shade3/15">
        <DataTable
          columns={columns}
          data={agents ?? []}
          keyField="name"
          onRowClick={handleRowClick}
          activeKey={selected?.name}
          emptyMessage="No agents configured"
          emptyIcon={emptyIcons.agents}
          emptyAction={{ label: 'Create Agent', onClick: () => navigate('/agents/new') }}
        />
      </div>

      {/* Detail Panel */}
      <DetailPanel
        open={selected !== null}
        onClose={() => setSelected(null)}
        title={selected?.name ?? ''}
        actions={
          selected ? (
            <>
              <button
                onClick={() => navigate(`/agents/${selected.name}/edit`)}
                className="flex-1 px-4 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover transition-colors"
              >
                Edit
              </button>
              <button
                onClick={() => setDeleteTarget(selected.name)}
                className="px-4 py-2 text-red-400 border border-red-500/30 rounded-btn text-sm font-medium hover:bg-red-500/10 transition-colors"
              >
                Delete
              </button>
            </>
          ) : undefined
        }
      >
        {loadingDetail ? (
          <div className="text-brand-shade3 text-sm">Loading...</div>
        ) : selected ? (
          <>
            <DetailSection title="General">
              <DetailRow label="Lifecycle">
                <StatusBadge status={selected.lifecycle === 'persistent' ? 'active' : 'idle'} className="text-xs" />
                <span className="ml-2 text-xs">{selected.lifecycle}</span>
              </DetailRow>
              {selected.kit && <DetailRow label="Kit">{selected.kit}</DetailRow>}
              <DetailRow label="Tool Execution">{selected.tool_execution}</DetailRow>
              <DetailRow label="Max Steps">{selected.max_steps}</DetailRow>
              <DetailRow label="Max Context">{selected.max_context_size.toLocaleString()}</DetailRow>
            </DetailSection>

            <DetailSection title="System Prompt">
              <pre className="p-3 bg-brand-dark rounded-btn text-xs text-brand-shade2 whitespace-pre-wrap max-h-48 overflow-y-auto border border-brand-shade3/15">
                {selected.system_prompt}
              </pre>
            </DetailSection>

            {selected.tools?.length > 0 && (
              <DetailSection title="Tools">
                <div className="flex flex-wrap gap-1.5">
                  {selected.tools?.map((t) => (
                    <span key={t} className="px-2 py-0.5 bg-brand-dark border border-brand-shade3/30 rounded text-xs text-brand-light">
                      {t}
                    </span>
                  ))}
                </div>
              </DetailSection>
            )}

            {selected.mcp_servers?.length > 0 && (
              <DetailSection title="MCP Servers">
                <div className="flex flex-wrap gap-1.5">
                  {selected.mcp_servers?.map((s) => (
                    <span key={s} className="px-2 py-0.5 bg-purple-500/10 border border-purple-500/30 rounded text-xs text-purple-400">
                      {s}
                    </span>
                  ))}
                </div>
              </DetailSection>
            )}

            {selected.can_spawn?.length > 0 && (
              <DetailSection title="Can Spawn">
                <div className="flex flex-wrap gap-1.5">
                  {selected.can_spawn?.map((a) => (
                    <span key={a} className="px-2 py-0.5 bg-blue-500/10 border border-blue-500/30 rounded text-xs text-blue-400">
                      {a}
                    </span>
                  ))}
                </div>
              </DetailSection>
            )}

            {selected.confirm_before?.length > 0 && (
              <DetailSection title="Confirm Before">
                <div className="flex flex-wrap gap-1.5">
                  {selected.confirm_before?.map((t) => (
                    <span key={t} className="px-2 py-0.5 bg-yellow-500/10 border border-yellow-500/30 rounded text-xs text-yellow-400">
                      {t}
                    </span>
                  ))}
                </div>
              </DetailSection>
            )}

            {selected.escalation && (
              <DetailSection title="Escalation">
                <DetailRow label="Action">{selected.escalation.action}</DetailRow>
                {selected.escalation.webhook_url && (
                  <DetailRow label="Webhook">{selected.escalation.webhook_url}</DetailRow>
                )}
                {selected.escalation?.triggers?.length > 0 && (
                  <DetailRow label="Triggers">{selected.escalation.triggers.join(', ')}</DetailRow>
                )}
              </DetailSection>
            )}
          </>
        ) : null}
      </DetailPanel>

      <ConfirmDialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title="Delete Agent"
        message={
          <>
            Are you sure you want to delete agent <strong className="text-brand-light">{deleteTarget}</strong>?
            This action cannot be undone.
          </>
        }
        confirmLabel="Delete"
        loading={deleting}
        variant="danger"
      />
    </div>
  );
}
