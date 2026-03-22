import { useState, type FormEvent } from 'react';
import { api } from '../api/client';
import { useApi } from '../hooks/useApi';
import DataTable from '../components/DataTable';
import DetailPanel, { DetailRow, DetailSection } from '../components/DetailPanel';
import FormModal from '../components/FormModal';
import FormField from '../components/FormField';
import ConfirmDialog from '../components/ConfirmDialog';
import type { Model, CreateModelRequest } from '../types';

const PROVIDER_TYPES = [
  { value: 'ollama', label: 'Ollama (local)' },
  { value: 'openai_compatible', label: 'OpenAI Compatible (OpenRouter, vLLM, etc.)' },
  { value: 'anthropic', label: 'Anthropic' },
];

const emptyForm: CreateModelRequest = {
  name: '',
  type: 'ollama',
  base_url: '',
  model_name: '',
  api_key: '',
};

export default function ModelsPage() {
  const { data: models, loading, error, refetch } = useApi(() => api.listModels());

  const [selected, setSelected] = useState<Model | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editTarget, setEditTarget] = useState<Model | null>(null);
  const [form, setForm] = useState<CreateModelRequest>({ ...emptyForm });
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);

  function openCreate() {
    setForm({ ...emptyForm });
    setEditTarget(null);
    setShowForm(true);
  }

  function openEdit(model: Model) {
    setForm({
      name: model.name,
      type: model.type,
      base_url: model.base_url ?? '',
      model_name: model.model_name,
      api_key: '',
    });
    setEditTarget(model);
    setShowForm(true);
  }

  function closeForm() {
    setShowForm(false);
    setEditTarget(null);
    setForm({ ...emptyForm });
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      if (editTarget) {
        await api.updateModel(editTarget.name, form);
      } else {
        await api.createModel(form);
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
    if (!deleteTarget) return;
    try {
      await api.deleteModel(deleteTarget);
      setDeleteTarget(null);
      setSelected(null);
      refetch();
    } catch {
      // visible in console
    }
  }

  const isEdit = editTarget !== null;

  const columns = [
    { key: 'name', header: 'Name' },
    {
      key: 'type',
      header: 'Provider',
      render: (row: Model) => (
        <span className="px-2 py-0.5 bg-brand-light rounded text-xs text-brand-shade3 font-medium">
          {row.type}
        </span>
      ),
    },
    { key: 'model_name', header: 'Model' },
    {
      key: 'base_url',
      header: 'URL',
      render: (row: Model) => (
        <span className="font-mono text-xs text-brand-shade3">{row.base_url || '--'}</span>
      ),
    },
    {
      key: 'has_api_key',
      header: 'API Key',
      render: (row: Model) =>
        row.has_api_key ? (
          <span className="text-xs text-status-active font-medium">Configured</span>
        ) : (
          <span className="text-xs text-brand-shade3">--</span>
        ),
    },
  ];

  if (loading) return <div className="text-brand-shade3">Loading models...</div>;
  if (error) return <div className="text-red-400">Error: {error}</div>;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-brand-light">Models</h1>
        <button
          onClick={openCreate}
          className="px-4 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover transition-colors"
        >
          Add Model
        </button>
      </div>

      <div className="bg-brand-dark-alt rounded-card border border-brand-shade3/15">
        <DataTable
          columns={columns}
          data={models ?? []}
          keyField="id"
          onRowClick={setSelected}
          activeKey={selected?.id}
          emptyMessage="No models configured"
          emptyIcon="&#x1F9E0;"
          emptyAction={{ label: 'Add Model', onClick: openCreate }}
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
                onClick={() => openEdit(selected)}
                className="flex-1 px-4 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover transition-colors"
              >
                Edit
              </button>
              <button
                onClick={() => setDeleteTarget(selected.name)}
                className="px-4 py-2 text-red-400 border border-red-500/30 rounded-btn text-sm font-medium hover:bg-red-500/10 transition-colors"
              >
                Remove
              </button>
            </>
          ) : undefined
        }
      >
        {selected && (
          <>
            <DetailSection title="Provider">
              <DetailRow label="Type">
                <span className="px-2 py-0.5 bg-brand-light rounded text-xs font-medium">
                  {selected.type}
                </span>
              </DetailRow>
              <DetailRow label="Model Name">{selected.model_name}</DetailRow>
              {selected.base_url && <DetailRow label="Base URL"><code className="font-mono text-xs">{selected.base_url}</code></DetailRow>}
              <DetailRow label="API Key">
                {selected.has_api_key ? (
                  <span className="text-status-active font-medium text-xs">Configured</span>
                ) : (
                  <span className="text-brand-shade3 text-xs">Not set</span>
                )}
              </DetailRow>
            </DetailSection>

            <DetailSection title="Timestamps">
              <DetailRow label="Created">{new Date(selected.created_at).toLocaleString()}</DetailRow>
            </DetailSection>
          </>
        )}
      </DetailPanel>

      {/* Create / Edit Form Modal */}
      <FormModal
        open={showForm}
        onClose={closeForm}
        title={isEdit ? 'Edit Model' : 'Add Model'}
        onSubmit={handleSubmit}
        submitLabel={isEdit ? 'Save Changes' : 'Add Model'}
        loading={saving}
      >
        <FormField
          label="Display Name"
          value={form.name}
          onChange={(v) => setForm({ ...form, name: v })}
          required
          disabled={isEdit}
          placeholder="my-llama"
          hint={isEdit ? 'Name cannot be changed.' : undefined}
        />
        <FormField
          label="Provider"
          type="select"
          value={form.type}
          onChange={(v) => setForm({ ...form, type: v })}
          options={PROVIDER_TYPES}
        />
        <FormField
          label="Model Name"
          value={form.model_name}
          onChange={(v) => setForm({ ...form, model_name: v })}
          required
          placeholder="llama-4-scout"
        />
        <FormField
          label="Base URL"
          value={form.base_url ?? ''}
          onChange={(v) => setForm({ ...form, base_url: v })}
          placeholder="http://localhost:11434"
          hint="Required for Ollama and OpenAI-compatible providers."
        />
        {form.type !== 'ollama' && (
          <FormField
            label="API Key"
            type="password"
            value={form.api_key ?? ''}
            onChange={(v) => setForm({ ...form, api_key: v })}
            placeholder={isEdit ? '(unchanged if empty)' : 'sk-...'}
            hint={isEdit ? 'Leave empty to keep the existing key.' : undefined}
          />
        )}
      </FormModal>

      <ConfirmDialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title="Remove Model"
        message={
          <>
            Remove model <strong className="text-brand-light">{deleteTarget}</strong>?
          </>
        }
        confirmLabel="Remove"
        variant="danger"
      />
    </div>
  );
}
