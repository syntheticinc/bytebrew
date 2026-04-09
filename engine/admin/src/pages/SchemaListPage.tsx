import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { usePrototype } from '../hooks/usePrototype';
import { useApi } from '../hooks/useApi';
import { useAdminRefresh } from '../hooks/useAdminRefresh';
import { api } from '../api/client';
import ConfirmDialog from '../components/ConfirmDialog';
import type { Schema } from '../types';

function SchemaRow({ schema, onClick }: { schema: Schema; onClick: () => void }) {
  return (
    <tr
      onClick={onClick}
      className="border-b border-brand-shade3/5 hover:bg-brand-dark-alt/50 cursor-pointer transition-colors"
    >
      <td className="px-4 py-3">
        <span className="text-brand-light font-medium font-mono">{schema.name}</span>
        {schema.is_system && (
          <span className="ml-2 text-[10px] text-brand-accent/70 bg-brand-accent/10 px-1.5 py-0.5 rounded border border-brand-accent/20">
            System
          </span>
        )}
      </td>
      <td className="px-4 py-3 text-brand-shade2 text-xs font-mono">{schema.agents_count}</td>
      <td className="px-4 py-3 text-brand-shade3 text-xs font-mono">
        {new Date(schema.created_at).toLocaleDateString()}
      </td>
    </tr>
  );
}

const TABLE_HEADERS = ['Name', 'Agents', 'Created'];

interface FormState {
  open: boolean;
  name: string;
  description: string;
  error: string;
  saving: boolean;
}

export default function SchemaListPage() {
  const { isPrototype } = usePrototype();
  const navigate = useNavigate();
  const [search, setSearch] = useState('');
  const [systemExpanded, setSystemExpanded] = useState(false);

  const { data: apiSchemas, refetch } = useApi(() => api.listSchemas(), [isPrototype]);
  useAdminRefresh(refetch);
  const schemas = apiSchemas ?? [];

  const allFiltered = schemas.filter((s) =>
    s.name.toLowerCase().includes(search.toLowerCase()),
  );
  const userSchemas = allFiltered.filter((s) => !s.is_system);
  const systemSchemas = allFiltered.filter((s) => s.is_system);

  // Detail panel
  const [selected, setSelected] = useState<Schema | null>(null);

  // Create form
  const [form, setForm] = useState<FormState>({
    open: false,
    name: '',
    description: '',
    error: '',
    saving: false,
  });

  // Delete confirm
  const [deleteTarget, setDeleteTarget] = useState<Schema | null>(null);
  const [deleteError, setDeleteError] = useState('');
  const [deleting, setDeleting] = useState(false);

  function handleSchemaClick(schema: Schema) {
    navigate(`/builder/${encodeURIComponent(schema.name)}`);
  }

  async function handleCreate() {
    const name = form.name.trim();
    if (!name) {
      setForm((f) => ({ ...f, error: 'Name is required' }));
      return;
    }
    setForm((f) => ({ ...f, saving: true, error: '' }));
    try {
      await api.createSchema({ name, description: form.description.trim() || undefined });
      setForm({ open: false, name: '', description: '', error: '', saving: false });
      refetch();
    } catch (err) {
      setForm((f) => ({
        ...f,
        saving: false,
        error: err instanceof Error ? err.message : 'Failed to create schema',
      }));
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    setDeleting(true);
    setDeleteError('');
    try {
      await api.deleteSchema(deleteTarget.id);
      setDeleteTarget(null);
      if (selected?.id === deleteTarget.id) setSelected(null);
      refetch();
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Failed to delete schema');
    } finally {
      setDeleting(false);
    }
  }

  return (
    <div className="flex gap-6 p-6 max-w-7xl">
      {/* Main content */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-xl font-semibold text-brand-light">Agent Builder</h1>
            <p className="text-sm text-brand-shade3 mt-1">
              Select a schema to open its agent canvas, or create a new one.
            </p>
          </div>
          <button
            onClick={() => setForm({ open: true, name: '', description: '', error: '', saving: false })}
            className="px-4 py-2 bg-brand-accent text-white text-sm font-medium rounded-btn hover:bg-brand-accent/80 transition-colors"
          >
            + New Schema
          </button>
        </div>

        {/* Search */}
        <div className="mb-4">
          <input
            type="text"
            placeholder="Search schemas..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full max-w-sm bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light px-3 py-2 focus:outline-none focus:border-brand-accent placeholder-brand-shade3"
          />
        </div>

        {/* User schemas table */}
        <div className="bg-brand-dark-surface rounded-card border border-brand-shade3/10 overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-brand-shade3/10">
                {TABLE_HEADERS.map((h) => (
                  <th
                    key={h}
                    className="text-left px-4 py-3 text-xs font-semibold text-brand-shade3 uppercase tracking-wider"
                  >
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {userSchemas.map((schema) => (
                <SchemaRow
                  key={schema.id}
                  schema={schema}
                  onClick={() => handleSchemaClick(schema)}
                />
              ))}
            </tbody>
          </table>
          {userSchemas.length === 0 && (
            <div className="px-4 py-8 text-center text-brand-shade3 text-sm">
              {search ? 'No schemas match your search.' : 'No schemas configured.'}
            </div>
          )}
        </div>

        {/* System schemas — collapsible */}
        {systemSchemas.length > 0 && (
          <div className="mt-4">
            <button
              onClick={() => setSystemExpanded((e) => !e)}
              className="flex items-center gap-2 text-xs text-brand-shade3 hover:text-brand-shade2 transition-colors mb-2"
            >
              <svg
                width="12"
                height="12"
                viewBox="0 0 14 14"
                fill="none"
                className={`transition-transform ${systemExpanded ? 'rotate-180' : ''}`}
              >
                <path
                  d="M3 5L7 9L11 5"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
              <span className="uppercase tracking-wider font-semibold">System Schemas</span>
              <span className="text-brand-shade3/50">({systemSchemas.length})</span>
            </button>

            {systemExpanded && (
              <div className="bg-brand-dark-surface rounded-card border border-brand-accent/15 overflow-hidden">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-brand-shade3/10">
                      {TABLE_HEADERS.map((h) => (
                        <th
                          key={h}
                          className="text-left px-4 py-3 text-xs font-semibold text-brand-shade3 uppercase tracking-wider"
                        >
                          {h}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {systemSchemas.map((schema) => (
                      <SchemaRow
                        key={schema.id}
                        schema={schema}
                        onClick={() => handleSchemaClick(schema)}
                      />
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        <p className="text-xs text-brand-shade3/50 mt-4">
          {userSchemas.length} schema{userSchemas.length !== 1 ? 's' : ''} total.
          {systemSchemas.length > 0 &&
            ` ${systemSchemas.length} system schema${systemSchemas.length !== 1 ? 's' : ''} hidden.`}
          {' '}Click to open canvas.
        </p>
      </div>

      {/* Detail panel */}
      {selected && (
        <div className="w-72 flex-shrink-0 bg-brand-dark-surface rounded-card border border-brand-shade3/10 p-4 self-start">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-brand-light">{selected.name}</h3>
            <button
              onClick={() => setSelected(null)}
              className="text-brand-shade3 hover:text-brand-light transition-colors"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M18 6L6 18M6 6l12 12" />
              </svg>
            </button>
          </div>

          {selected.is_system && (
            <span className="inline-block text-[10px] text-brand-accent/70 bg-brand-accent/10 px-1.5 py-0.5 rounded border border-brand-accent/20 mb-3">
              System
            </span>
          )}

          {selected.description && (
            <p className="text-xs text-brand-shade2 mb-3">{selected.description}</p>
          )}

          <div className="space-y-2 text-xs mb-4">
            <div className="flex justify-between">
              <span className="text-brand-shade3">Agents</span>
              <span className="text-brand-shade2 font-mono">{selected.agents_count}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-brand-shade3">Created</span>
              <span className="text-brand-shade2 font-mono">
                {new Date(selected.created_at).toLocaleDateString()}
              </span>
            </div>
          </div>

          <div className="flex flex-col gap-2">
            <button
              onClick={() => handleSchemaClick(selected)}
              className="w-full px-3 py-1.5 text-xs bg-brand-accent text-brand-light rounded-btn hover:bg-brand-accent-hover transition-colors"
            >
              Open Canvas
            </button>
            {!selected.is_system && (
              <button
                onClick={() => setDeleteTarget(selected)}
                className="w-full px-3 py-1.5 text-xs text-red-400 border border-red-400/30 rounded-btn hover:bg-red-400/10 transition-colors"
              >
                Delete
              </button>
            )}
          </div>
        </div>
      )}

      {/* Create schema modal */}
      {form.open && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="bg-brand-dark-surface border border-brand-shade3/20 rounded-card p-6 w-full max-w-md shadow-xl animate-modal-in">
            <h2 className="text-sm font-semibold text-brand-light mb-4">New Schema</h2>

            <div className="space-y-3">
              <div>
                <label className="block text-xs text-brand-shade3 mb-1">Name *</label>
                <input
                  type="text"
                  value={form.name}
                  onChange={(e) => setForm((f) => ({ ...f, name: e.target.value, error: '' }))}
                  onKeyDown={(e) => { if (e.key === 'Enter') handleCreate(); }}
                  placeholder="my-schema"
                  autoFocus
                  className="w-full bg-brand-dark-alt border border-brand-shade3/30 rounded-card text-sm text-brand-light px-3 py-2 focus:outline-none focus:border-brand-accent placeholder-brand-shade3"
                />
              </div>
              <div>
                <label className="block text-xs text-brand-shade3 mb-1">Description</label>
                <input
                  type="text"
                  value={form.description}
                  onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                  placeholder="Optional description"
                  className="w-full bg-brand-dark-alt border border-brand-shade3/30 rounded-card text-sm text-brand-light px-3 py-2 focus:outline-none focus:border-brand-accent placeholder-brand-shade3"
                />
              </div>
            </div>

            {form.error && <p className="text-red-400 text-xs mt-2">{form.error}</p>}

            <div className="flex justify-end gap-2 mt-4">
              <button
                onClick={() => setForm((f) => ({ ...f, open: false }))}
                className="px-3 py-1.5 text-xs text-brand-shade2 border border-brand-shade3/30 rounded-btn hover:text-brand-light transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleCreate}
                disabled={form.saving}
                className="px-3 py-1.5 text-xs bg-brand-accent text-brand-light rounded-btn hover:bg-brand-accent-hover transition-colors disabled:opacity-50"
              >
                {form.saving ? 'Creating...' : 'Create'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete confirm */}
      <ConfirmDialog
        open={deleteTarget !== null}
        onClose={() => { setDeleteTarget(null); setDeleteError(''); }}
        onConfirm={handleDelete}
        title="Delete Schema"
        message={
          <>
            Delete schema <strong className="text-brand-light">{deleteTarget?.name}</strong>?
            This will not delete agents, only the schema itself.
            {deleteError && <p className="mt-2 text-red-400 text-xs">{deleteError}</p>}
          </>
        }
        confirmLabel="Delete"
        loading={deleting}
        variant="danger"
      />
    </div>
  );
}
