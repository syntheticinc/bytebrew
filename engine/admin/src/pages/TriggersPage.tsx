import { useState, type FormEvent } from 'react';
import { api } from '../api/client';
import { useApi } from '../hooks/useApi';
import DataTable from '../components/DataTable';
import Modal from '../components/Modal';
import type { Trigger, CreateTriggerRequest } from '../types';

const TRIGGER_TYPES = ['webhook', 'cron', 'chat'] as const;

function TriggerTypeBadge({ type }: { type: string }) {
  const colors: Record<string, string> = {
    webhook: 'text-blue-300 bg-blue-500/15',
    cron: 'text-purple-300 bg-purple-500/15',
    chat: 'text-green-300 bg-green-500/15',
  };
  return (
    <span className={`text-xs px-2 py-0.5 rounded font-mono ${colors[type] ?? 'text-brand-shade2 bg-brand-dark'}`}>
      {type}
    </span>
  );
}

function EnabledBadge({ enabled }: { enabled: boolean }) {
  return (
    <span className={`text-xs px-2 py-0.5 rounded ${enabled ? 'text-green-300 bg-green-500/15' : 'text-brand-shade3 bg-brand-dark'}`}>
      {enabled ? 'Enabled' : 'Disabled'}
    </span>
  );
}

function formatConfig(trigger: Trigger): string {
  if (trigger.type === 'cron' && trigger.config?.schedule) {
    return trigger.config.schedule;
  }
  if (trigger.type === 'webhook' && trigger.config?.webhook_path) {
    return trigger.config.webhook_path;
  }
  return '—';
}

interface TriggerFormState {
  title: string;
  type: string;
  description: string;
  enabled: boolean;
  schedule: string;
  webhookPath: string;
}

const defaultForm: TriggerFormState = {
  title: '',
  type: 'webhook',
  description: '',
  enabled: true,
  schedule: '',
  webhookPath: '',
};

function formToRequest(form: TriggerFormState): CreateTriggerRequest {
  const config: { schedule?: string; webhook_path?: string } = {};
  if (form.type === 'cron' && form.schedule) config.schedule = form.schedule;
  if (form.type === 'webhook' && form.webhookPath) config.webhook_path = form.webhookPath;
  return {
    type: form.type,
    title: form.title,
    description: form.description || undefined,
    enabled: form.enabled,
    config: Object.keys(config).length > 0 ? config : undefined,
  };
}

function triggerToForm(trigger: Trigger): TriggerFormState {
  return {
    title: trigger.title,
    type: trigger.type,
    description: trigger.description ?? '',
    enabled: trigger.enabled,
    schedule: trigger.config?.schedule ?? '',
    webhookPath: trigger.config?.webhook_path ?? '',
  };
}

export default function TriggersPage() {
  const { data: triggers, loading, error, refetch } = useApi(() => api.listTriggers());

  const [showCreate, setShowCreate] = useState(false);
  const [editTarget, setEditTarget] = useState<Trigger | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Trigger | null>(null);
  const [form, setForm] = useState<TriggerFormState>(defaultForm);
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  function openCreate() {
    setForm(defaultForm);
    setFormError(null);
    setShowCreate(true);
  }

  function openEdit(trigger: Trigger) {
    setForm(triggerToForm(trigger));
    setFormError(null);
    setEditTarget(trigger);
  }

  function closeForm() {
    setShowCreate(false);
    setEditTarget(null);
    setForm(defaultForm);
    setFormError(null);
  }

  function setField<K extends keyof TriggerFormState>(key: K, value: TriggerFormState[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    setFormError(null);
    try {
      if (editTarget) {
        await api.updateTrigger(editTarget.id, formToRequest(form));
      } else {
        await api.createTrigger(formToRequest(form));
      }
      refetch();
      closeForm();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to save trigger');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    try {
      await api.deleteTrigger(deleteTarget.id);
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
      render: (row: Trigger) => <TriggerTypeBadge type={row.type} />,
    },
    {
      key: 'enabled',
      header: 'Status',
      render: (row: Trigger) => <EnabledBadge enabled={row.enabled} />,
    },
    {
      key: 'config',
      header: 'Config',
      render: (row: Trigger) => (
        <span className="text-xs font-mono text-brand-shade3">{formatConfig(row)}</span>
      ),
    },
    {
      key: 'last_fired_at',
      header: 'Last Fired',
      render: (row: Trigger) =>
        row.last_fired_at ? (
          <span className="text-xs text-brand-shade3">
            {new Date(row.last_fired_at).toLocaleString()}
          </span>
        ) : (
          <span className="text-xs text-brand-shade3">Never</span>
        ),
    },
    {
      key: 'created_at',
      header: 'Created',
      render: (row: Trigger) => (
        <span className="text-xs text-brand-shade3">
          {new Date(row.created_at).toLocaleDateString()}
        </span>
      ),
    },
    {
      key: 'actions',
      header: '',
      render: (row: Trigger) => (
        <div className="flex items-center gap-3">
          <button
            onClick={(e) => { e.stopPropagation(); openEdit(row); }}
            className="text-brand-shade2 hover:text-brand-light text-sm"
          >
            Edit
          </button>
          <button
            onClick={(e) => { e.stopPropagation(); setDeleteTarget(row); }}
            className="text-red-400 hover:text-red-300 text-sm"
          >
            Delete
          </button>
        </div>
      ),
    },
  ];

  if (loading) return <div className="text-brand-shade3">Loading triggers...</div>;
  if (error) return <div className="text-red-400">Error: {error}</div>;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-brand-light">Triggers</h1>
        <button
          onClick={openCreate}
          className="px-4 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover transition-colors"
        >
          New Trigger
        </button>
      </div>

      <div className="bg-brand-dark-alt rounded-card border border-brand-shade3/15">
        <DataTable
          columns={columns}
          data={triggers ?? []}
          keyField="id"
          emptyMessage="No triggers configured. Create your first trigger to automate agent execution."
        />
      </div>

      {/* Create / Edit modal */}
      <Modal
        open={showCreate || editTarget !== null}
        onClose={closeForm}
        title={editTarget ? `Edit Trigger: ${editTarget.title}` : 'New Trigger'}
      >
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-brand-light mb-1">Title</label>
            <input
              type="text"
              value={form.title}
              onChange={(e) => setField('title', e.target.value)}
              required
              placeholder="daily-report"
              className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light placeholder-brand-shade3 focus:outline-none focus:border-brand-accent"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-brand-light mb-1">Type</label>
            <select
              value={form.type}
              onChange={(e) => setField('type', e.target.value)}
              className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light focus:outline-none focus:border-brand-accent"
            >
              {TRIGGER_TYPES.map((t) => (
                <option key={t} value={t}>{t}</option>
              ))}
            </select>
          </div>

          {form.type === 'cron' && (
            <div>
              <label className="block text-sm font-medium text-brand-light mb-1">
                Schedule <span className="text-brand-shade3 font-normal">(cron expression)</span>
              </label>
              <input
                type="text"
                value={form.schedule}
                onChange={(e) => setField('schedule', e.target.value)}
                placeholder="0 9 * * *"
                className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light font-mono placeholder-brand-shade3 focus:outline-none focus:border-brand-accent"
              />
            </div>
          )}

          {form.type === 'webhook' && (
            <div>
              <label className="block text-sm font-medium text-brand-light mb-1">
                Webhook Path
              </label>
              <input
                type="text"
                value={form.webhookPath}
                onChange={(e) => setField('webhookPath', e.target.value)}
                placeholder="/webhook/support"
                className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light font-mono placeholder-brand-shade3 focus:outline-none focus:border-brand-accent"
              />
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-brand-light mb-1">
              Description <span className="text-brand-shade3 font-normal">(optional)</span>
            </label>
            <input
              type="text"
              value={form.description}
              onChange={(e) => setField('description', e.target.value)}
              placeholder="Brief description"
              className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light placeholder-brand-shade3 focus:outline-none focus:border-brand-accent"
            />
          </div>

          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={form.enabled}
              onChange={(e) => setField('enabled', e.target.checked)}
              className="rounded border-brand-shade3/30 text-brand-accent focus:ring-brand-accent bg-brand-dark"
            />
            <span className="text-sm text-brand-light">Enabled</span>
          </label>

          {formError && (
            <div className="text-sm text-red-400">{formError}</div>
          )}

          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={closeForm}
              className="px-4 py-2 text-sm text-brand-shade2 border border-brand-shade3/30 rounded-btn hover:bg-brand-dark hover:text-brand-light transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={saving || !form.title}
              className="px-4 py-2 text-sm text-brand-light bg-brand-accent rounded-btn hover:bg-brand-accent-hover disabled:opacity-50"
            >
              {saving ? 'Saving...' : editTarget ? 'Save Changes' : 'Create'}
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
              className="px-4 py-2 text-sm text-brand-shade2 border border-brand-shade3/30 rounded-btn hover:bg-brand-dark hover:text-brand-light transition-colors"
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
        <p className="text-sm text-brand-shade2">
          Delete trigger <span className="font-medium text-brand-light">{deleteTarget?.title}</span>? This cannot be undone.
        </p>
      </Modal>
    </div>
  );
}
