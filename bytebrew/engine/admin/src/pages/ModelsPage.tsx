import { useState, type FormEvent } from 'react';
import { api } from '../api/client';
import { useApi } from '../hooks/useApi';
import Modal from '../components/Modal';
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
  const [showAdd, setShowAdd] = useState(false);
  const [editTarget, setEditTarget] = useState<Model | null>(null);
  const [form, setForm] = useState<CreateModelRequest>({ ...emptyForm });
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);

  function openEdit(model: Model) {
    setForm({
      name: model.name,
      type: model.type,
      base_url: model.base_url ?? '',
      model_name: model.model_name,
      api_key: '',
    });
    setEditTarget(model);
  }

  function closeEdit() {
    setEditTarget(null);
    setForm({ ...emptyForm });
  }

  async function handleAdd(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      await api.createModel(form);
      setShowAdd(false);
      setForm({ ...emptyForm });
      refetch();
    } catch {
      // visible in console
    } finally {
      setSaving(false);
    }
  }

  async function handleEdit(e: FormEvent) {
    e.preventDefault();
    if (!editTarget) return;
    setSaving(true);
    try {
      await api.updateModel(editTarget.name, form);
      closeEdit();
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
      refetch();
    } catch {
      // visible in console
    }
  }

  function renderForm(onSubmit: (e: FormEvent) => void, submitLabel: string, onCancel: () => void, isEdit: boolean) {
    return (
      <form onSubmit={onSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-brand-dark mb-1">Display Name</label>
          <input
            type="text"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            required
            disabled={isEdit}
            placeholder="my-llama"
            className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent disabled:opacity-50 disabled:bg-brand-light"
          />
          {isEdit && (
            <p className="text-xs text-brand-shade3 mt-1">Name cannot be changed.</p>
          )}
        </div>
        <div>
          <label className="block text-sm font-medium text-brand-dark mb-1">Provider</label>
          <select
            value={form.type}
            onChange={(e) => setForm({ ...form, type: e.target.value })}
            className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent"
          >
            {PROVIDER_TYPES.map((p) => (
              <option key={p.value} value={p.value}>
                {p.label}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium text-brand-dark mb-1">Model Name</label>
          <input
            type="text"
            value={form.model_name}
            onChange={(e) => setForm({ ...form, model_name: e.target.value })}
            required
            placeholder="llama-4-scout"
            className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-brand-dark mb-1">Base URL</label>
          <input
            type="text"
            value={form.base_url}
            onChange={(e) => setForm({ ...form, base_url: e.target.value })}
            placeholder="http://localhost:11434"
            className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent"
          />
          <p className="text-xs text-brand-shade3 mt-1">Required for Ollama and OpenAI-compatible providers.</p>
        </div>
        {form.type !== 'ollama' && (
          <div>
            <label className="block text-sm font-medium text-brand-dark mb-1">API Key</label>
            <input
              type="password"
              value={form.api_key}
              onChange={(e) => setForm({ ...form, api_key: e.target.value })}
              placeholder={isEdit ? '(unchanged if empty)' : 'sk-...'}
              className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent"
            />
            {isEdit && (
              <p className="text-xs text-brand-shade3 mt-1">Leave empty to keep the existing key.</p>
            )}
          </div>
        )}
        <div className="flex justify-end gap-2 pt-2">
          <button
            type="button"
            onClick={onCancel}
            className="px-4 py-2 text-sm text-brand-dark border border-brand-shade2 rounded-btn hover:bg-brand-light"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={saving}
            className="px-4 py-2 text-sm text-brand-light bg-brand-accent rounded-btn hover:bg-brand-accent-hover disabled:opacity-50"
          >
            {saving ? 'Saving...' : submitLabel}
          </button>
        </div>
      </form>
    );
  }

  if (loading) return <div className="text-brand-shade3">Loading models...</div>;
  if (error) return <div className="text-red-600">Error: {error}</div>;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-brand-dark">Models</h1>
        <button
          onClick={() => setShowAdd(true)}
          className="px-4 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover transition-colors"
        >
          Add Model
        </button>
      </div>

      <div className="space-y-3">
        {(models ?? []).length === 0 ? (
          <div className="text-center py-12 text-brand-shade3 bg-white rounded-card border border-brand-shade1">
            No models configured. Add an LLM provider.
          </div>
        ) : (
          (models ?? []).map((m) => (
            <div key={m.id} className="bg-white rounded-card border border-brand-shade1 p-4 flex items-center justify-between">
              <div>
                <div className="flex items-center gap-2">
                  <span className="font-medium text-brand-dark">{m.name}</span>
                  <span className="text-xs text-brand-shade3 bg-brand-light px-2 py-0.5 rounded">
                    {m.type}
                  </span>
                </div>
                <div className="text-xs text-brand-shade3 mt-1">
                  Model: {m.model_name}
                  {m.base_url && ` | URL: ${m.base_url}`}
                  {m.has_api_key && ' | API key configured'}
                </div>
              </div>
              <div className="flex items-center gap-3">
                <button
                  onClick={() => openEdit(m)}
                  className="text-brand-accent hover:underline text-sm"
                >
                  Edit
                </button>
                <button
                  onClick={() => setDeleteTarget(m.name)}
                  className="text-red-600 hover:text-red-800 text-sm"
                >
                  Remove
                </button>
              </div>
            </div>
          ))
        )}
      </div>

      {/* Add modal */}
      <Modal open={showAdd} onClose={() => setShowAdd(false)} title="Add Model">
        {renderForm(handleAdd, 'Add Model', () => setShowAdd(false), false)}
      </Modal>

      {/* Edit modal */}
      <Modal open={editTarget !== null} onClose={closeEdit} title="Edit Model">
        {renderForm(handleEdit, 'Save Changes', closeEdit, true)}
      </Modal>

      {/* Delete confirmation */}
      <Modal
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        title="Remove Model"
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
              Remove
            </button>
          </>
        }
      >
        <p className="text-sm text-brand-shade3">
          Remove model <strong className="text-brand-dark">{deleteTarget}</strong>?
        </p>
      </Modal>
    </div>
  );
}
