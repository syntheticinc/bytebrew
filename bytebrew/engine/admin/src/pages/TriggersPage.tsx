import { useState, type FormEvent } from 'react';
import { api } from '../api/client';
import { useApi } from '../hooks/useApi';
import DataTable from '../components/DataTable';
import StatusBadge from '../components/StatusBadge';
import Modal from '../components/Modal';
import type { Trigger, CreateTriggerRequest, AgentInfo } from '../types';

export default function TriggersPage() {
  const { data: triggers, loading, error, refetch } = useApi(() => api.listTriggers());
  const { data: agents } = useApi(() => api.listAgents());

  const [showAdd, setShowAdd] = useState(false);
  const [form, setForm] = useState<CreateTriggerRequest>({
    type: 'cron',
    title: '',
    agent_id: 0,
    schedule: '',
    webhook_path: '',
    description: '',
    enabled: true,
  });
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<number | null>(null);

  async function handleAdd(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      await api.createTrigger(form);
      setShowAdd(false);
      setForm({ type: 'cron', title: '', agent_id: 0, schedule: '', webhook_path: '', description: '', enabled: true });
      refetch();
    } catch {
      // visible in console
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (deleteTarget === null) return;
    try {
      await api.deleteTrigger(deleteTarget);
      setDeleteTarget(null);
      refetch();
    } catch {
      // visible in console
    }
  }

  const columns = [
    { key: 'title', header: 'Title' },
    {
      key: 'type',
      header: 'Type',
      render: (row: Trigger) => (
        <span className={`px-2 py-0.5 rounded text-xs font-medium ${row.type === 'cron' ? 'bg-brand-accent/10 text-brand-accent' : 'bg-purple-100 text-purple-700'}`}>
          {row.type}
        </span>
      ),
    },
    { key: 'agent_name', header: 'Agent' },
    {
      key: 'schedule',
      header: 'Schedule / Path',
      render: (row: Trigger) => (
        <span className="font-mono text-xs">{row.schedule ?? row.webhook_path ?? '-'}</span>
      ),
    },
    {
      key: 'enabled',
      header: 'Status',
      render: (row: Trigger) => <StatusBadge status={row.enabled ? 'active' : 'disconnected'} />,
    },
    {
      key: 'actions',
      header: '',
      render: (row: Trigger) => (
        <button
          onClick={() => setDeleteTarget(row.id)}
          className="text-red-600 hover:text-red-800 text-sm"
        >
          Delete
        </button>
      ),
    },
  ];

  if (loading) return <div className="text-brand-shade3">Loading triggers...</div>;
  if (error) return <div className="text-red-600">Error: {error}</div>;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-brand-dark">Triggers</h1>
        <button
          onClick={() => setShowAdd(true)}
          className="px-4 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover transition-colors"
        >
          Add Trigger
        </button>
      </div>

      <div className="bg-white rounded-card border border-brand-shade1">
        <DataTable
          columns={columns}
          data={triggers ?? []}
          keyField="id"
          emptyMessage="No triggers configured."
        />
      </div>

      {/* Add trigger modal */}
      <Modal open={showAdd} onClose={() => setShowAdd(false)} title="Add Trigger">
        <form onSubmit={handleAdd} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-brand-dark mb-1">Title</label>
            <input
              type="text"
              value={form.title}
              onChange={(e) => setForm({ ...form, title: e.target.value })}
              required
              className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-dark mb-1">Type</label>
            <select
              value={form.type}
              onChange={(e) => setForm({ ...form, type: e.target.value })}
              className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent"
            >
              <option value="cron">Cron</option>
              <option value="webhook">Webhook</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-dark mb-1">Agent</label>
            <select
              value={form.agent_id}
              onChange={(e) => setForm({ ...form, agent_id: Number(e.target.value) })}
              required
              className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent"
            >
              <option value={0} disabled>
                Select agent...
              </option>
              {(agents ?? []).map((a: AgentInfo) => (
                <option key={a.name} value={a.name}>
                  {a.name}
                </option>
              ))}
            </select>
          </div>
          {form.type === 'cron' ? (
            <div>
              <label className="block text-sm font-medium text-brand-dark mb-1">Schedule (cron expression)</label>
              <input
                type="text"
                value={form.schedule}
                onChange={(e) => setForm({ ...form, schedule: e.target.value })}
                placeholder="*/5 * * * *"
                className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm font-mono focus:outline-none focus:border-brand-accent"
              />
            </div>
          ) : (
            <div>
              <label className="block text-sm font-medium text-brand-dark mb-1">Webhook Path</label>
              <input
                type="text"
                value={form.webhook_path}
                onChange={(e) => setForm({ ...form, webhook_path: e.target.value })}
                placeholder="/hooks/deploy"
                className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm font-mono focus:outline-none focus:border-brand-accent"
              />
            </div>
          )}
          <div>
            <label className="block text-sm font-medium text-brand-dark mb-1">Description</label>
            <textarea
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
              rows={2}
              className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent"
            />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={() => setShowAdd(false)}
              className="px-4 py-2 text-sm text-brand-dark border border-brand-shade2 rounded-btn hover:bg-brand-light"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={saving}
              className="px-4 py-2 text-sm text-brand-light bg-brand-accent rounded-btn hover:bg-brand-accent-hover disabled:opacity-50"
            >
              {saving ? 'Adding...' : 'Add Trigger'}
            </button>
          </div>
        </form>
      </Modal>

      {/* Delete confirmation */}
      <Modal
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        title="Delete Trigger"
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
              className="px-4 py-2 text-sm text-white bg-red-600 rounded-btn hover:bg-red-700"
            >
              Delete
            </button>
          </>
        }
      >
        <p className="text-sm text-brand-shade3">Delete this trigger? This action cannot be undone.</p>
      </Modal>
    </div>
  );
}
