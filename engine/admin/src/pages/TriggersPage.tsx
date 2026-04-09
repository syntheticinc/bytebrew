import { useState, type FormEvent } from 'react';
import { api } from '../api/client';
import { useApi } from '../hooks/useApi';
import { useAdminRefresh } from '../hooks/useAdminRefresh';
import DataTable from '../components/DataTable';
import { emptyIcons } from '../components/EmptyState';
import StatusBadge from '../components/StatusBadge';
import DetailPanel, { DetailRow, DetailSection } from '../components/DetailPanel';
import FormModal from '../components/FormModal';
import FormField from '../components/FormField';
import ConfirmDialog from '../components/ConfirmDialog';
import type { Trigger, CreateTriggerRequest, AgentInfo } from '../types';

const emptyForm: CreateTriggerRequest = {
  type: 'cron',
  title: '',
  agent_id: 0,
  schedule: '',
  webhook_path: '',
  description: '',
  enabled: true,
  on_complete_url: '',
  on_complete_headers: undefined,
};

export default function TriggersPage() {
  const { data: triggers, loading, error, refetch } = useApi(() => api.listTriggers());
  useAdminRefresh(refetch);
  const { data: agents } = useApi(() => api.listAgents());

  const [selected, setSelected] = useState<Trigger | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editTarget, setEditTarget] = useState<Trigger | null>(null);
  const [form, setForm] = useState<CreateTriggerRequest>({ ...emptyForm });
  const [headersText, setHeadersText] = useState('');
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<number | null>(null);

  function openCreate() {
    setForm({ ...emptyForm });
    setHeadersText('');
    setEditTarget(null);
    setShowForm(true);
  }

  function openEdit(trigger: Trigger) {
    setForm({
      type: trigger.type,
      title: trigger.title,
      agent_id: trigger.agent_id,
      schedule: trigger.schedule ?? '',
      webhook_path: trigger.webhook_path ?? '',
      description: trigger.description ?? '',
      enabled: trigger.enabled,
      on_complete_url: trigger.on_complete_url ?? '',
      on_complete_headers: trigger.on_complete_headers,
    });
    setHeadersText(
      trigger.on_complete_headers
        ? JSON.stringify(trigger.on_complete_headers, null, 2)
        : ''
    );
    setEditTarget(trigger);
    setShowForm(true);
  }

  function closeForm() {
    setShowForm(false);
    setEditTarget(null);
    setForm({ ...emptyForm });
    setHeadersText('');
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      let parsedHeaders: Record<string, string> | undefined;
      if (headersText.trim()) {
        try {
          parsedHeaders = JSON.parse(headersText.trim()) as Record<string, string>;
        } catch {
          alert('On Complete Headers must be valid JSON');
          setSaving(false);
          return;
        }
      }

      const payload: CreateTriggerRequest = {
        ...form,
        on_complete_url: form.on_complete_url?.trim() || undefined,
        on_complete_headers: parsedHeaders,
      };

      if (editTarget) {
        await api.updateTrigger(editTarget.id, payload);
      } else {
        await api.createTrigger(payload);
      }
      closeForm();
      setSelected(null);
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
      setSelected(null);
      refetch();
    } catch {
      // visible in console
    }
  }

  const agentOptions = [
    { value: '0', label: 'Select agent...' },
    ...(agents ?? []).map((a: AgentInfo) => ({ value: String(a.name), label: a.name })),
  ];

  const columns = [
    { key: 'title', header: 'Title' },
    {
      key: 'type',
      header: 'Type',
      render: (row: Trigger) => (
        <span className={`px-2 py-0.5 rounded text-xs font-medium ${row.type === 'cron' ? 'bg-brand-accent/15 text-brand-accent' : 'bg-purple-500/15 text-purple-400'}`}>
          {row.type}
        </span>
      ),
    },
    { key: 'agent_name', header: 'Agent' },
    {
      key: 'schedule',
      header: 'Schedule / Path',
      render: (row: Trigger) => (
        <span className="font-mono text-xs">{row.schedule ?? row.webhook_path ?? '--'}</span>
      ),
    },
    {
      key: 'enabled',
      header: 'Status',
      render: (row: Trigger) => <StatusBadge status={row.enabled ? 'active' : 'disconnected'} />,
    },
  ];

  if (loading) return <div className="text-brand-shade3">Loading triggers...</div>;
  if (error) return <div className="text-red-600">Error: {error}</div>;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-brand-light">Triggers</h1>
        <button
          onClick={openCreate}
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
          onRowClick={setSelected}
          activeKey={selected?.id}
          emptyMessage="No triggers configured"
          emptyIcon={emptyIcons.triggers}
          emptyAction={{ label: 'Add Trigger', onClick: openCreate }}
        />
      </div>

      {/* Detail Panel */}
      <DetailPanel
        open={selected !== null}
        onClose={() => setSelected(null)}
        title={selected?.title ?? ''}
        actions={
          selected ? (
            <>
              <button
                onClick={() => openEdit(selected)}
                className="flex-1 px-4 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover transition-colors"
              >
                Edit
              </button>
              <button
                onClick={() => setDeleteTarget(selected.id)}
                className="px-4 py-2 text-red-600 border border-red-200 rounded-btn text-sm font-medium hover:bg-red-50 transition-colors"
              >
                Delete
              </button>
            </>
          ) : undefined
        }
      >
        {selected && (
          <>
            <DetailSection title="Configuration">
              <DetailRow label="Type">
                <span className={`px-2 py-0.5 rounded text-xs font-medium ${selected.type === 'cron' ? 'bg-brand-accent/10 text-brand-accent' : 'bg-purple-100 text-purple-700'}`}>
                  {selected.type}
                </span>
              </DetailRow>
              <DetailRow label="Agent">{selected.agent_name ?? String(selected.agent_id)}</DetailRow>
              <DetailRow label="Status">
                <StatusBadge status={selected.enabled ? 'active' : 'disconnected'} />
              </DetailRow>
              {selected.schedule && <DetailRow label="Schedule"><code className="font-mono text-xs">{selected.schedule}</code></DetailRow>}
              {selected.webhook_path && <DetailRow label="Webhook Path"><code className="font-mono text-xs">{selected.webhook_path}</code></DetailRow>}
            </DetailSection>

            {selected.description && (
              <DetailSection title="Description">
                <p className="text-sm text-brand-light">{selected.description}</p>
              </DetailSection>
            )}

            {selected.on_complete_url && (
              <DetailSection title="On Complete Webhook">
                <DetailRow label="URL"><code className="font-mono text-xs break-all">{selected.on_complete_url}</code></DetailRow>
                {selected.on_complete_headers && Object.keys(selected.on_complete_headers).length > 0 && (
                  <DetailRow label="Headers">
                    <pre className="font-mono text-xs whitespace-pre-wrap">{JSON.stringify(selected.on_complete_headers, null, 2)}</pre>
                  </DetailRow>
                )}
              </DetailSection>
            )}

            <DetailSection title="Timestamps">
              <DetailRow label="Created">{new Date(selected.created_at).toLocaleString()}</DetailRow>
              {selected.last_fired_at && (
                <DetailRow label="Last Fired">{new Date(selected.last_fired_at).toLocaleString()}</DetailRow>
              )}
            </DetailSection>
          </>
        )}
      </DetailPanel>

      {/* Create / Edit Form Modal */}
      <FormModal
        open={showForm}
        onClose={closeForm}
        title={editTarget ? 'Edit Trigger' : 'Add Trigger'}
        onSubmit={handleSubmit}
        submitLabel={editTarget ? 'Save Changes' : 'Add Trigger'}
        loading={saving}
      >
        <FormField label="Title" value={form.title} onChange={(v) => setForm({ ...form, title: v })} required placeholder="Daily report" />
        <FormField
          label="Type"
          type="select"
          value={form.type}
          onChange={(v) => setForm({ ...form, type: v })}
          options={[
            { value: 'cron', label: 'Cron' },
            { value: 'webhook', label: 'Webhook' },
            { value: 'chat', label: 'Chat' },
          ]}
        />
        <FormField
          label="Agent"
          type="select"
          value={String(form.agent_id)}
          onChange={(v) => setForm({ ...form, agent_id: Number(v) })}
          options={agentOptions}
          required
        />
        {form.type === 'cron' ? (
          <FormField
            label="Schedule (cron expression)"
            value={form.schedule ?? ''}
            onChange={(v) => setForm({ ...form, schedule: v })}
            placeholder="*/5 * * * *"
            hint="Standard 5-field cron expression"
          />
        ) : (
          <FormField
            label="Webhook Path"
            value={form.webhook_path ?? ''}
            onChange={(v) => setForm({ ...form, webhook_path: v })}
            placeholder="/hooks/deploy"
          />
        )}
        <FormField
          label="Description"
          type="textarea"
          value={form.description ?? ''}
          onChange={(v) => setForm({ ...form, description: v })}
          rows={2}
        />
        {/* On Complete Webhook */}
        <div className="border border-brand-shade3/20 rounded-card p-3 space-y-3">
          <p className="text-xs font-medium text-brand-shade3 uppercase tracking-wide">On Complete Webhook</p>
          <FormField
            label="Callback URL"
            value={form.on_complete_url ?? ''}
            onChange={(v) => setForm({ ...form, on_complete_url: v })}
            placeholder="https://your-app.com/callback"
            hint="Optional. Called when the triggered task completes."
          />
          <FormField
            label="Headers (JSON)"
            type="textarea"
            value={headersText}
            onChange={setHeadersText}
            placeholder='{"Authorization": "Bearer token"}'
            rows={2}
            hint="Optional. JSON object with HTTP headers for the callback."
          />
        </div>

        <div className="flex items-center gap-2">
          <input
            type="checkbox"
            id="trigger-enabled"
            checked={form.enabled ?? true}
            onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
            className="rounded border-brand-shade1 text-brand-accent focus:ring-brand-accent"
          />
          <label htmlFor="trigger-enabled" className="text-sm text-brand-light">Enabled</label>
        </div>
      </FormModal>

      {/* Delete confirmation */}
      <ConfirmDialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title="Delete Trigger"
        message="Delete this trigger? This action cannot be undone."
        confirmLabel="Delete"
        variant="danger"
      />
    </div>
  );
}
