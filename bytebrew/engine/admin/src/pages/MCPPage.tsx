import { useState, type FormEvent } from 'react';
import { api } from '../api/client';
import { useApi } from '../hooks/useApi';
import StatusBadge from '../components/StatusBadge';
import Modal from '../components/Modal';
import type { WellKnownMCP, CreateMCPServerRequest } from '../types';

export default function MCPPage() {
  const { data: servers, loading, error, refetch } = useApi(() => api.listMCPServers());
  const { data: wellKnown } = useApi(() => api.getWellKnownMCP());

  const [showAddCustom, setShowAddCustom] = useState(false);
  const [showWellKnown, setShowWellKnown] = useState(false);
  const [customForm, setCustomForm] = useState<CreateMCPServerRequest>({
    name: '',
    type: 'stdio',
    command: '',
    args: [],
    url: '',
  });
  const [envInput, setEnvInput] = useState<Record<string, string>>({});
  const [argsInput, setArgsInput] = useState('');
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);

  async function handleAddCustom(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const payload: CreateMCPServerRequest = {
        ...customForm,
        args: argsInput ? argsInput.split('\n').map((a) => a.trim()).filter(Boolean) : [],
        env_vars: Object.keys(envInput).length > 0 ? envInput : undefined,
      };
      await api.createMCPServer(payload);
      setShowAddCustom(false);
      resetCustomForm();
      refetch();
    } catch {
      // visible in console
    } finally {
      setSaving(false);
    }
  }

  async function handleAddWellKnown(wk: WellKnownMCP) {
    setSaving(true);
    try {
      const envVars: Record<string, string> = {};
      for (const key of wk.env) {
        envVars[key] = envInput[key] ?? '';
      }
      await api.createMCPServer({
        name: wk.name,
        type: 'stdio',
        command: wk.command,
        args: wk.args,
        env_vars: Object.keys(envVars).length > 0 ? envVars : undefined,
      });
      setShowWellKnown(false);
      setEnvInput({});
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
      await api.deleteMCPServer(deleteTarget);
      setDeleteTarget(null);
      refetch();
    } catch {
      // visible in console
    }
  }

  function resetCustomForm() {
    setCustomForm({ name: '', type: 'stdio', command: '', args: [], url: '' });
    setArgsInput('');
    setEnvInput({});
  }

  const alreadyAdded = new Set((servers ?? []).map((s) => s.name));

  if (loading) return <div className="text-brand-shade3">Loading MCP servers...</div>;
  if (error) return <div className="text-red-600">Error: {error}</div>;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-brand-dark">MCP Servers</h1>
        <div className="flex gap-2">
          <button
            onClick={() => setShowWellKnown(true)}
            className="px-4 py-2 bg-brand-dark text-brand-light rounded-btn text-sm font-medium hover:bg-brand-dark-alt transition-colors"
          >
            Add from Catalog
          </button>
          <button
            onClick={() => setShowAddCustom(true)}
            className="px-4 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover transition-colors"
          >
            Add Custom
          </button>
        </div>
      </div>

      {/* Server list */}
      <div className="space-y-3">
        {(servers ?? []).length === 0 ? (
          <div className="text-center py-12 text-brand-shade3 bg-white rounded-card border border-brand-shade1">
            No MCP servers configured. Add one from the catalog or create a custom server.
          </div>
        ) : (
          (servers ?? []).map((s) => (
            <div key={s.name} className="bg-white rounded-card border border-brand-shade1 p-4 flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-brand-dark">{s.name}</span>
                    <span className="text-xs text-brand-shade3 bg-brand-light px-2 py-0.5 rounded">
                      {s.type}
                    </span>
                    {s.is_well_known && (
                      <span className="text-xs text-brand-accent bg-brand-accent/10 px-2 py-0.5 rounded">
                        catalog
                      </span>
                    )}
                  </div>
                  <div className="text-xs text-brand-shade3 mt-1">
                    {s.command && `${s.command} ${(s.args ?? []).join(' ')}`}
                    {s.url && s.url}
                  </div>
                  {s.agents.length > 0 && (
                    <div className="text-xs text-brand-shade3 mt-1">
                      Agents: {s.agents.join(', ')}
                    </div>
                  )}
                </div>
              </div>
              <div className="flex items-center gap-3">
                {s.status && (
                  <div className="text-right">
                    <StatusBadge status={s.status.status} />
                    {s.status.tools_count > 0 && (
                      <div className="text-xs text-brand-shade3 mt-1">{s.status.tools_count} tools</div>
                    )}
                  </div>
                )}
                <button
                  onClick={() => setDeleteTarget(s.name)}
                  className="text-red-600 hover:text-red-800 text-sm"
                >
                  Remove
                </button>
              </div>
            </div>
          ))
        )}
      </div>

      {/* Well-known catalog modal */}
      <Modal
        open={showWellKnown}
        onClose={() => {
          setShowWellKnown(false);
          setEnvInput({});
        }}
        title="MCP Catalog"
      >
        <div className="space-y-3 max-h-96 overflow-y-auto">
          {(wellKnown ?? []).map((wk) => {
            const added = alreadyAdded.has(wk.name);
            return (
              <div
                key={wk.name}
                className={`border border-brand-shade1 rounded-card p-3 ${added ? 'opacity-50' : ''}`}
              >
                <div className="flex items-center justify-between">
                  <div>
                    <span className="font-medium text-brand-dark">{wk.display}</span>
                    <div className="text-xs text-brand-shade3 mt-0.5">
                      {wk.command} {wk.args.join(' ')}
                    </div>
                  </div>
                  <button
                    onClick={() => handleAddWellKnown(wk)}
                    disabled={added || saving}
                    className="px-3 py-1.5 bg-brand-accent text-brand-light rounded-btn text-xs font-medium hover:bg-brand-accent-hover disabled:opacity-50 transition-colors"
                  >
                    {added ? 'Added' : 'Add'}
                  </button>
                </div>
                {wk.env.length > 0 && (
                  <div className="mt-2 space-y-1">
                    {wk.env.map((envKey) => (
                      <input
                        key={envKey}
                        type="text"
                        placeholder={envKey}
                        value={envInput[envKey] ?? ''}
                        onChange={(e) =>
                          setEnvInput((prev) => ({ ...prev, [envKey]: e.target.value }))
                        }
                        className="w-full px-2 py-1 bg-white border border-brand-shade1 rounded-btn text-xs focus:outline-none focus:border-brand-accent"
                      />
                    ))}
                  </div>
                )}
              </div>
            );
          })}
          {(wellKnown ?? []).length === 0 && (
            <p className="text-sm text-brand-shade3 text-center py-4">No well-known servers available.</p>
          )}
        </div>
      </Modal>

      {/* Custom add modal */}
      <Modal
        open={showAddCustom}
        onClose={() => {
          setShowAddCustom(false);
          resetCustomForm();
        }}
        title="Add Custom MCP Server"
      >
        <form onSubmit={handleAddCustom} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-brand-dark mb-1">Name</label>
            <input
              type="text"
              value={customForm.name}
              onChange={(e) => setCustomForm({ ...customForm, name: e.target.value })}
              required
              className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-dark mb-1">Type</label>
            <select
              value={customForm.type}
              onChange={(e) => setCustomForm({ ...customForm, type: e.target.value })}
              className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent"
            >
              <option value="stdio">stdio</option>
              <option value="http">http</option>
              <option value="sse">sse</option>
            </select>
          </div>
          {customForm.type === 'stdio' ? (
            <>
              <div>
                <label className="block text-sm font-medium text-brand-dark mb-1">Command</label>
                <input
                  type="text"
                  value={customForm.command}
                  onChange={(e) => setCustomForm({ ...customForm, command: e.target.value })}
                  placeholder="npx"
                  className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-brand-dark mb-1">Args (one per line)</label>
                <textarea
                  value={argsInput}
                  onChange={(e) => setArgsInput(e.target.value)}
                  rows={3}
                  placeholder="@anthropic/playwright-mcp"
                  className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent"
                />
              </div>
            </>
          ) : (
            <div>
              <label className="block text-sm font-medium text-brand-dark mb-1">URL</label>
              <input
                type="url"
                value={customForm.url}
                onChange={(e) => setCustomForm({ ...customForm, url: e.target.value })}
                placeholder="http://localhost:3000/mcp"
                className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent"
              />
            </div>
          )}
          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={() => {
                setShowAddCustom(false);
                resetCustomForm();
              }}
              className="px-4 py-2 text-sm text-brand-dark border border-brand-shade2 rounded-btn hover:bg-brand-light"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={saving}
              className="px-4 py-2 text-sm text-brand-light bg-brand-accent rounded-btn hover:bg-brand-accent-hover disabled:opacity-50"
            >
              {saving ? 'Adding...' : 'Add Server'}
            </button>
          </div>
        </form>
      </Modal>

      {/* Delete confirmation */}
      <Modal
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        title="Remove MCP Server"
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
          Remove MCP server <strong className="text-brand-dark">{deleteTarget}</strong>?
        </p>
      </Modal>
    </div>
  );
}
