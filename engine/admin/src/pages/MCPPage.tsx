import { useState, type FormEvent } from 'react';
import { api } from '../api/client';
import { useApi } from '../hooks/useApi';
import DataTable from '../components/DataTable';
import { emptyIcons } from '../components/EmptyState';
import StatusBadge from '../components/StatusBadge';
import DetailPanel, { DetailRow, DetailSection } from '../components/DetailPanel';
import FormModal from '../components/FormModal';
import FormField from '../components/FormField';
import ConfirmDialog from '../components/ConfirmDialog';
import Modal from '../components/Modal';
import type { MCPServer, WellKnownMCP, CreateMCPServerRequest } from '../types';

const emptyForm: CreateMCPServerRequest = {
  name: '',
  type: 'stdio',
  command: '',
  args: [],
  url: '',
};

export default function MCPPage() {
  const { data: servers, loading, error, refetch } = useApi(() => api.listMCPServers());
  const { data: wellKnown } = useApi(() => api.getWellKnownMCP());

  const [selected, setSelected] = useState<MCPServer | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [showWellKnown, setShowWellKnown] = useState(false);
  const [editTarget, setEditTarget] = useState<MCPServer | null>(null);
  const [customForm, setCustomForm] = useState<CreateMCPServerRequest>({ ...emptyForm });
  const [envInput, setEnvInput] = useState<Record<string, string>>({});
  const [argsInput, setArgsInput] = useState('');
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);

  function openCreate() {
    resetCustomForm();
    setEditTarget(null);
    setShowForm(true);
  }

  function openEdit(server: MCPServer) {
    setCustomForm({
      name: server.name,
      type: server.type,
      command: server.command ?? '',
      args: server.args ?? [],
      url: server.url ?? '',
    });
    setArgsInput((server.args ?? []).join('\n'));
    setEnvInput(server.env_vars ?? {});
    setEditTarget(server);
    setShowForm(true);
  }

  function closeForm() {
    setShowForm(false);
    setEditTarget(null);
    resetCustomForm();
  }

  function resetCustomForm() {
    setCustomForm({ ...emptyForm });
    setArgsInput('');
    setEnvInput({});
  }

  function buildPayload(): CreateMCPServerRequest {
    return {
      ...customForm,
      args: argsInput ? argsInput.split('\n').map((a) => a.trim()).filter(Boolean) : [],
      env_vars: Object.keys(envInput).length > 0 ? envInput : undefined,
    };
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const payload = buildPayload();
      if (editTarget) {
        await api.updateMCPServer(editTarget.name, payload);
      } else {
        await api.createMCPServer(payload);
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
      setSelected(null);
      refetch();
    } catch {
      // visible in console
    }
  }

  const alreadyAdded = new Set((servers ?? []).map((s) => s.name));
  const isEdit = editTarget !== null;

  const columns = [
    { key: 'name', header: 'Name' },
    {
      key: 'type',
      header: 'Type',
      render: (row: MCPServer) => (
        <span className="px-2 py-0.5 bg-brand-light rounded text-xs text-brand-shade3 font-medium">
          {row.type}
        </span>
      ),
    },
    {
      key: 'command',
      header: 'Command / URL',
      render: (row: MCPServer) => (
        <span className="font-mono text-xs text-brand-shade3">
          {row.command ? `${row.command} ${(row.args ?? []).join(' ')}` : row.url ?? '--'}
        </span>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (row: MCPServer) =>
        row.status ? <StatusBadge status={row.status.status} /> : <span className="text-brand-shade3 text-xs">--</span>,
    },
    {
      key: 'tools_count',
      header: 'Tools',
      render: (row: MCPServer) => (
        <span className="text-xs">{row.status?.tools_count ?? '--'}</span>
      ),
    },
  ];

  if (loading) return <div className="text-brand-shade3">Loading MCP servers...</div>;
  if (error) return <div className="text-red-400">Error: {error}</div>;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-brand-light">MCP Servers</h1>
        <div className="flex gap-2">
          <button
            onClick={() => setShowWellKnown(true)}
            className="px-4 py-2 bg-brand-dark text-brand-light rounded-btn text-sm font-medium hover:bg-brand-dark-alt transition-colors"
          >
            Add from Catalog
          </button>
          <button
            onClick={openCreate}
            className="px-4 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover transition-colors"
          >
            Add Custom
          </button>
        </div>
      </div>

      <div className="bg-brand-dark-alt rounded-card border border-brand-shade3/15">
        <DataTable
          columns={columns}
          data={servers ?? []}
          keyField="name"
          onRowClick={setSelected}
          activeKey={selected?.name}
          emptyMessage="No MCP servers configured"
          emptyIcon={emptyIcons.mcp}
          emptyAction={{ label: 'Add from Catalog', onClick: () => setShowWellKnown(true) }}
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
            <DetailSection title="Configuration">
              <DetailRow label="Type">
                <span className="px-2 py-0.5 bg-brand-light rounded text-xs font-medium">{selected.type}</span>
              </DetailRow>
              {selected.is_well_known && (
                <DetailRow label="Source">
                  <span className="px-2 py-0.5 bg-brand-accent/10 text-brand-accent rounded text-xs font-medium">catalog</span>
                </DetailRow>
              )}
              {selected.command && (
                <DetailRow label="Command">
                  <code className="font-mono text-xs">{selected.command} {(selected.args ?? []).join(' ')}</code>
                </DetailRow>
              )}
              {selected.url && (
                <DetailRow label="URL">
                  <code className="font-mono text-xs">{selected.url}</code>
                </DetailRow>
              )}
            </DetailSection>

            {selected.status && (
              <DetailSection title="Runtime Status">
                <DetailRow label="Status"><StatusBadge status={selected.status.status} /></DetailRow>
                <DetailRow label="Tools Count">{selected.status.tools_count}</DetailRow>
                {selected.status.connected_at && (
                  <DetailRow label="Connected At">{new Date(selected.status.connected_at).toLocaleString()}</DetailRow>
                )}
                {selected.status.status_message && (
                  <DetailRow label="Message">{selected.status.status_message}</DetailRow>
                )}
              </DetailSection>
            )}

            {selected.agents.length > 0 && (
              <DetailSection title="Used by Agents">
                <div className="flex flex-wrap gap-1.5">
                  {selected.agents.map((a) => (
                    <span key={a} className="px-2 py-0.5 bg-blue-50 border border-blue-200 rounded text-xs text-blue-700">
                      {a}
                    </span>
                  ))}
                </div>
              </DetailSection>
            )}

            {selected.env_vars && Object.keys(selected.env_vars).length > 0 && (
              <DetailSection title="Environment Variables">
                {Object.entries(selected.env_vars).map(([key, value]) => (
                  <DetailRow key={key} label={key}>
                    <code className="font-mono text-xs">{value ? '***' : '(empty)'}</code>
                  </DetailRow>
                ))}
              </DetailSection>
            )}
          </>
        )}
      </DetailPanel>

      {/* Well-known catalog modal */}
      <Modal
        open={showWellKnown}
        onClose={() => { setShowWellKnown(false); setEnvInput({}); }}
        title="MCP Catalog"
      >
        <div className="space-y-3 max-h-96 overflow-y-auto">
          {(wellKnown ?? []).map((wk) => {
            const added = alreadyAdded.has(wk.name);
            return (
              <div
                key={wk.name}
                className={`border border-brand-shade3/30 rounded-card p-3 ${added ? 'opacity-50' : ''}`}
              >
                <div className="flex items-center justify-between">
                  <div>
                    <span className="font-medium text-brand-dark">{wk.display}</span>
                    <div className="text-xs text-brand-shade3 mt-0.5 font-mono">
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
                        onChange={(e) => setEnvInput((prev) => ({ ...prev, [envKey]: e.target.value }))}
                        className="w-full px-2 py-1 bg-brand-dark-alt border border-brand-shade3/50 rounded-btn text-xs text-brand-light placeholder-brand-shade3 focus:outline-none focus:border-brand-accent"
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

      {/* Custom add / edit modal */}
      <FormModal
        open={showForm}
        onClose={closeForm}
        title={isEdit ? 'Edit MCP Server' : 'Add Custom MCP Server'}
        onSubmit={handleSubmit}
        submitLabel={isEdit ? 'Save Changes' : 'Add Server'}
        loading={saving}
      >
        <FormField
          label="Name"
          value={customForm.name}
          onChange={(v) => setCustomForm({ ...customForm, name: v })}
          required
          disabled={isEdit}
          hint={isEdit ? 'Name cannot be changed.' : undefined}
        />
        <FormField
          label="Type"
          type="select"
          value={customForm.type}
          onChange={(v) => setCustomForm({ ...customForm, type: v })}
          options={[
            { value: 'stdio', label: 'stdio' },
            { value: 'http', label: 'http' },
            { value: 'sse', label: 'sse' },
            { value: 'streamable-http', label: 'streamable-http' },
          ]}
        />
        {customForm.type === 'stdio' ? (
          <>
            <FormField
              label="Command"
              value={customForm.command ?? ''}
              onChange={(v) => setCustomForm({ ...customForm, command: v })}
              placeholder="npx"
            />
            <FormField
              label="Args (one per line)"
              type="textarea"
              value={argsInput}
              onChange={setArgsInput}
              rows={3}
              placeholder="@anthropic/playwright-mcp"
            />
          </>
        ) : (
          <FormField
            label="URL"
            value={customForm.url ?? ''}
            onChange={(v) => setCustomForm({ ...customForm, url: v })}
            placeholder="http://localhost:3000/mcp"
          />
        )}
        <div>
          <label className="block text-sm font-medium text-brand-light mb-1">Environment Variables</label>
          <div className="space-y-2">
            {Object.entries(envInput).map(([key, value]) => (
              <div key={key} className="flex items-center gap-2">
                <input
                  type="text"
                  value={key}
                  readOnly
                  className="w-1/3 px-2 py-1.5 bg-brand-dark border border-brand-shade3/50 rounded-btn text-xs text-brand-shade2"
                />
                <input
                  type="text"
                  value={value}
                  onChange={(e) => setEnvInput((prev) => ({ ...prev, [key]: e.target.value }))}
                  className="flex-1 px-2 py-1.5 bg-brand-dark-alt border border-brand-shade3/50 rounded-btn text-xs text-brand-light focus:outline-none focus:border-brand-accent"
                />
                <button
                  type="button"
                  onClick={() => {
                    const next = { ...envInput };
                    delete next[key];
                    setEnvInput(next);
                  }}
                  className="text-red-500 hover:text-red-700 text-xs p-1"
                >
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            ))}
            <button
              type="button"
              onClick={() => {
                const key = prompt('Variable name:');
                if (key && key.trim()) {
                  setEnvInput((prev) => ({ ...prev, [key.trim()]: '' }));
                }
              }}
              className="text-xs text-brand-accent hover:underline"
            >
              + Add variable
            </button>
          </div>
        </div>
      </FormModal>

      <ConfirmDialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title="Remove MCP Server"
        message={
          <>
            Remove MCP server <strong className="text-brand-dark">{deleteTarget}</strong>?
          </>
        }
        confirmLabel="Remove"
        variant="danger"
      />
    </div>
  );
}
