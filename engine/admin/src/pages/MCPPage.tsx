import { useState, useMemo, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
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
import type { MCPServer, WellKnownMCP, CreateMCPServerRequest, MCPCatalogCategory, CircuitBreakerState } from '../types';

// ─── Category meta ──────────────────────────────────────────────────────────

const CATEGORY_META: Record<MCPCatalogCategory | 'all', { label: string; color: string }> = {
  all:            { label: 'All',            color: 'bg-brand-shade3/15 text-brand-shade2' },
  search:         { label: 'Search',         color: 'bg-blue-500/15 text-blue-400' },
  data:           { label: 'Data',           color: 'bg-emerald-500/15 text-emerald-400' },
  communication:  { label: 'Communication',  color: 'bg-purple-500/15 text-purple-400' },
  dev_tools:      { label: 'Dev Tools',      color: 'bg-amber-500/15 text-amber-400' },
  productivity:   { label: 'Productivity',   color: 'bg-pink-500/15 text-pink-400' },
  generic:        { label: 'Generic',        color: 'bg-brand-shade3/15 text-brand-shade2' },
};

const emptyForm: CreateMCPServerRequest = {
  name: '',
  type: 'stdio',
  command: '',
  args: [],
  url: '',
};

export default function MCPPage() {
  const navigate = useNavigate();
  const { data: servers, loading, error, refetch } = useApi(() => api.listMCPServers());
  const { data: wellKnown } = useApi(() => api.getWellKnownMCP());
  const { data: circuitBreakers } = useApi(() => api.listCircuitBreakers());

  const [selected, setSelected] = useState<MCPServer | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [showWellKnown, setShowWellKnown] = useState(false);
  const [editTarget, setEditTarget] = useState<MCPServer | null>(null);
  const [customForm, setCustomForm] = useState<CreateMCPServerRequest>({ ...emptyForm });
  const [envInput, setEnvInput] = useState<Record<string, string>>({});
  const [argsInput, setArgsInput] = useState('');
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [authType, setAuthType] = useState<string>('none');
  const [authEnvVar, setAuthEnvVar] = useState('');
  const [authClientId, setAuthClientId] = useState('');

  // Catalog search/filter state
  const [catalogSearch, setCatalogSearch] = useState('');
  const [catalogCategory, setCatalogCategory] = useState<MCPCatalogCategory | 'all'>('all');
  const [catalogDetail, setCatalogDetail] = useState<WellKnownMCP | null>(null);

  const circuitStateMap = useMemo(() => {
    const map: Record<string, CircuitBreakerState> = {};
    if (circuitBreakers) {
      for (const cb of circuitBreakers) {
        map[cb.name] = cb;
      }
    }
    return map;
  }, [circuitBreakers]);

  const filteredCatalog = useMemo(() => {
    if (!wellKnown) return [];
    return wellKnown.filter((wk) => {
      if (catalogCategory !== 'all' && wk.category !== catalogCategory) return false;
      if (catalogSearch) {
        const q = catalogSearch.toLowerCase();
        return wk.display.toLowerCase().includes(q) || wk.name.toLowerCase().includes(q);
      }
      return true;
    });
  }, [wellKnown, catalogSearch, catalogCategory]);

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
    setAuthType('none');
    setAuthEnvVar('');
    setAuthClientId('');
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
      key: 'circuit',
      header: 'Circuit',
      render: (row: MCPServer) => {
        const cb = circuitStateMap[row.name];
        if (!cb || cb.state === 'closed') {
          return <span className="inline-flex items-center gap-1 text-xs text-emerald-400">
            <span className="w-2 h-2 rounded-full bg-emerald-400 inline-block" />
            closed
          </span>;
        }
        if (cb.state === 'half_open') {
          return <span className="inline-flex items-center gap-1 text-xs text-amber-400">
            <span className="w-2 h-2 rounded-full bg-amber-400 inline-block" />
            half-open
          </span>;
        }
        return <span className="inline-flex items-center gap-1 text-xs text-red-400">
          <span className="w-2 h-2 rounded-full bg-red-400 inline-block" />
          open ({cb.failure_count})
        </span>;
      },
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
                {(() => {
                  const cb = circuitStateMap[selected.name];
                  if (!cb) return null;
                  return (
                    <DetailRow label="Circuit State">
                      {cb.state === 'closed'
                        ? <span className="text-emerald-400 text-sm">● Closed</span>
                        : cb.state === 'half_open'
                          ? <span className="text-amber-400 text-sm">● Half-Open</span>
                          : <span className="text-red-400 text-sm">● Open ({cb.failure_count} failures)</span>}
                    </DetailRow>
                  );
                })()}
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
                    <button
                      key={a}
                      onClick={() => navigate(`/builder/default/${encodeURIComponent(a)}`)}
                      className="px-2 py-0.5 bg-blue-500/10 border border-blue-500/25 rounded text-xs text-blue-400 hover:bg-blue-500/20 hover:border-blue-500/40 transition-colors cursor-pointer"
                      title={`Go to ${a} detail`}
                    >
                      {a}
                    </button>
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

      {/* MCP Catalog modal — with search, category filter, detail view */}
      <Modal
        open={showWellKnown}
        onClose={() => { setShowWellKnown(false); setEnvInput({}); setCatalogSearch(''); setCatalogCategory('all'); setCatalogDetail(null); }}
        title="MCP Catalog"
      >
        <div className="space-y-3">
          {/* Search */}
          <input
            type="text"
            value={catalogSearch}
            onChange={(e) => setCatalogSearch(e.target.value)}
            placeholder="Search servers..."
            className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/30 rounded-card text-sm text-brand-light placeholder-brand-shade3 focus:outline-none focus:border-brand-accent transition-colors"
          />

          {/* Category filter */}
          <div className="flex gap-1.5 flex-wrap">
            {(Object.entries(CATEGORY_META) as [MCPCatalogCategory | 'all', { label: string; color: string }][]).map(([key, meta]) => (
              <button
                key={key}
                onClick={() => setCatalogCategory(key)}
                className={`px-2.5 py-1 rounded-btn text-[11px] font-medium transition-colors ${
                  catalogCategory === key
                    ? 'bg-brand-accent text-brand-light'
                    : `${meta.color} hover:opacity-80`
                }`}
              >
                {meta.label}
              </button>
            ))}
          </div>

          {/* Server list or detail */}
          {catalogDetail ? (
            <div className="space-y-3">
              <button
                onClick={() => setCatalogDetail(null)}
                className="text-xs text-brand-accent hover:underline flex items-center gap-1"
              >
                <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <polyline points="15 18 9 12 15 6" />
                </svg>
                Back to list
              </button>
              <div className="border border-brand-shade3/30 rounded-card p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="font-semibold text-brand-light text-sm">{catalogDetail.display}</h3>
                    <span className="text-xs text-brand-shade3 font-mono">{catalogDetail.name}</span>
                  </div>
                  {catalogDetail.category && (
                    <span className={`px-2 py-0.5 rounded text-[10px] font-medium ${CATEGORY_META[catalogDetail.category]?.color ?? CATEGORY_META.generic.color}`}>
                      {CATEGORY_META[catalogDetail.category]?.label ?? catalogDetail.category}
                    </span>
                  )}
                </div>

                <div>
                  <p className="text-xs text-brand-shade3 mb-1">Command</p>
                  <code className="text-xs text-brand-light font-mono">{catalogDetail.command} {catalogDetail.args.join(' ')}</code>
                </div>

                {catalogDetail.env.length > 0 && (
                  <div>
                    <p className="text-xs text-brand-shade3 mb-1">Required Environment Variables</p>
                    <div className="space-y-1.5">
                      {catalogDetail.env.map((envKey) => (
                        <input
                          key={envKey}
                          type="text"
                          placeholder={envKey}
                          value={envInput[envKey] ?? ''}
                          onChange={(e) => setEnvInput((prev) => ({ ...prev, [envKey]: e.target.value }))}
                          className="w-full px-2.5 py-1.5 bg-brand-dark border border-brand-shade3/30 rounded-card text-xs text-brand-light placeholder-brand-shade3 font-mono focus:outline-none focus:border-brand-accent transition-colors"
                        />
                      ))}
                    </div>
                  </div>
                )}

                {catalogDetail.auth_types && catalogDetail.auth_types.length > 0 && (
                  <div>
                    <p className="text-xs text-brand-shade3 mb-1">Auth Types</p>
                    <div className="flex gap-1.5">
                      {catalogDetail.auth_types.map((at) => (
                        <span key={at} className="px-2 py-0.5 bg-brand-shade3/10 rounded text-[10px] text-brand-shade2">{at}</span>
                      ))}
                    </div>
                  </div>
                )}

                <button
                  onClick={() => handleAddWellKnown(catalogDetail)}
                  disabled={alreadyAdded.has(catalogDetail.name) || saving}
                  className="w-full py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover disabled:opacity-50 transition-colors"
                >
                  {alreadyAdded.has(catalogDetail.name) ? 'Already Added' : saving ? 'Installing...' : 'Install'}
                </button>
              </div>
            </div>
          ) : (
            <div className="space-y-2 max-h-80 overflow-y-auto">
              {filteredCatalog.map((wk) => {
                const added = alreadyAdded.has(wk.name);
                return (
                  <div
                    key={wk.name}
                    className={`border border-brand-shade3/20 rounded-card p-3 hover:border-brand-shade3/40 transition-colors cursor-pointer ${added ? 'opacity-50' : ''}`}
                    onClick={() => setCatalogDetail(wk)}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-brand-light text-sm">{wk.display}</span>
                        {wk.category && (
                          <span className={`px-1.5 py-0.5 rounded text-[9px] font-medium ${CATEGORY_META[wk.category]?.color ?? CATEGORY_META.generic.color}`}>
                            {CATEGORY_META[wk.category]?.label ?? wk.category}
                          </span>
                        )}
                      </div>
                      {added && (
                        <span className="text-[10px] text-brand-shade3">Added</span>
                      )}
                    </div>
                    <div className="text-[11px] text-brand-shade3 mt-1 font-mono truncate">
                      {wk.command} {wk.args.join(' ')}
                    </div>
                    {wk.env.length > 0 && (
                      <div className="text-[10px] text-brand-shade3/60 mt-1">
                        Requires: {wk.env.join(', ')}
                      </div>
                    )}
                  </div>
                );
              })}
              {filteredCatalog.length === 0 && (
                <p className="text-sm text-brand-shade3 text-center py-4">
                  {catalogSearch ? 'No servers match your search.' : 'No catalog servers available.'}
                </p>
              )}
            </div>
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
          label="Transport"
          type="select"
          value={customForm.type}
          onChange={(v) => setCustomForm({ ...customForm, type: v })}
          options={[
            { value: 'stdio', label: 'stdio — Local process' },
            { value: 'streamable-http', label: 'streamable-http — HTTP streaming' },
            { value: 'sse', label: 'sse — Server-Sent Events' },
            { value: 'http', label: 'http — HTTP' },
            { value: 'websocket', label: 'websocket — WebSocket' },
            { value: 'docker', label: 'docker — Docker container' },
          ]}
        />
        {customForm.type === 'stdio' && (
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
        )}
        {customForm.type === 'docker' && (
          <>
            <FormField
              label="Docker Image"
              value={customForm.command ?? ''}
              onChange={(v) => setCustomForm({ ...customForm, command: v })}
              placeholder="mcp/google-sheets:latest"
              hint="Docker image name (will be pulled if not present)"
            />
            <FormField
              label="Container Args (one per line)"
              type="textarea"
              value={argsInput}
              onChange={setArgsInput}
              rows={2}
              placeholder="--port=3000"
            />
          </>
        )}
        {(customForm.type === 'http' || customForm.type === 'sse' || customForm.type === 'streamable-http' || customForm.type === 'websocket') && (
          <FormField
            label="URL"
            value={customForm.url ?? ''}
            onChange={(v) => setCustomForm({ ...customForm, url: v })}
            placeholder={customForm.type === 'websocket' ? 'ws://localhost:3000/mcp' : 'http://localhost:3000/mcp'}
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
        <div>
          <label className="block text-sm font-medium text-brand-light mb-1">Authentication</label>
          <FormField
            label="Auth Type"
            type="select"
            value={authType}
            onChange={setAuthType}
            options={[
              { value: 'none', label: 'None' },
              { value: 'forward_headers', label: 'Forward Headers' },
              { value: 'api_key', label: 'API Key' },
              { value: 'oauth2', label: 'OAuth 2.0' },
              { value: 'service_account', label: 'Service Account' },
            ]}
          />
          {authType === 'api_key' && (
            <FormField
              label="API Key Env Variable"
              value={authEnvVar}
              onChange={setAuthEnvVar}
              placeholder="e.g. SHEETS_API_KEY"
              hint="Name of the environment variable containing the API key"
              className="mt-2"
            />
          )}
          {authType === 'oauth2' && (
            <>
              <FormField
                label="Client ID"
                value={authClientId}
                onChange={setAuthClientId}
                placeholder="OAuth client ID"
                className="mt-2"
              />
              <p className="mt-1 text-xs text-brand-shade3">OAuth flow configured via admin. Tokens are managed automatically.</p>
            </>
          )}
          {authType === 'forward_headers' && (
            <p className="mt-2 text-xs text-brand-shade3">Headers from the calling system will be forwarded to this MCP server.</p>
          )}
          {authType === 'service_account' && (
            <FormField
              label="Credentials Env Variable"
              value={authEnvVar}
              onChange={setAuthEnvVar}
              placeholder="e.g. GCP_SERVICE_ACCOUNT_JSON"
              hint="Name of the environment variable containing service account credentials"
              className="mt-2"
            />
          )}
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
