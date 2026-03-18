import { useState, useEffect, type FormEvent } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { useApi } from '../hooks/useApi';
import type { CreateAgentRequest, Model, MCPServer, AgentInfo } from '../types';

const BUILTIN_TOOLS = [
  'read_file',
  'write_file',
  'execute_command',
  'list_directory',
  'search_files',
  'web_search',
  'http_request',
];

export default function AgentEditPage() {
  const { name } = useParams<{ name: string }>();
  const isNew = name === 'new' || !name;
  const navigate = useNavigate();

  const [form, setForm] = useState<CreateAgentRequest>({
    name: '',
    system_prompt: '',
    kit: '',
    lifecycle: 'persistent',
    tool_execution: 'sequential',
    max_steps: 50,
    max_context_size: 16000,
    tools: [],
    can_spawn: [],
    mcp_servers: [],
    confirm_before: [],
  });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [confirmInput, setConfirmInput] = useState('');

  const { data: models } = useApi<Model[]>(() => api.listModels());
  const { data: mcpServers } = useApi<MCPServer[]>(() => api.listMCPServers());
  const { data: allAgents } = useApi<AgentInfo[]>(() => api.listAgents());

  useEffect(() => {
    if (isNew) return;
    api.getAgent(name!).then((agent) => {
      setForm({
        name: agent.name,
        model_id: agent.model_id,
        system_prompt: agent.system_prompt,
        kit: agent.kit ?? '',
        lifecycle: agent.lifecycle,
        tool_execution: agent.tool_execution,
        max_steps: agent.max_steps,
        max_context_size: agent.max_context_size,
        tools: agent.tools,
        can_spawn: agent.can_spawn,
        mcp_servers: agent.mcp_servers,
        confirm_before: agent.confirm_before,
      });
    }).catch((err: Error) => setError(err.message));
  }, [name, isNew]);

  function updateField<K extends keyof CreateAgentRequest>(key: K, value: CreateAgentRequest[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  function toggleInArray(arr: string[], item: string): string[] {
    return arr.includes(item) ? arr.filter((x) => x !== item) : [...arr, item];
  }

  function addConfirmBefore() {
    const trimmed = confirmInput.trim();
    if (!trimmed || form.confirm_before?.includes(trimmed)) return;
    updateField('confirm_before', [...(form.confirm_before ?? []), trimmed]);
    setConfirmInput('');
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    setSaving(true);

    try {
      if (isNew) {
        await api.createAgent(form);
      } else {
        await api.updateAgent(name!, form);
      }
      navigate('/agents');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Save failed');
    } finally {
      setSaving(false);
    }
  }

  const otherAgents = (allAgents ?? []).filter((a) => a.name !== form.name);

  return (
    <div className="max-w-3xl">
      <h1 className="text-2xl font-bold text-brand-dark mb-6">
        {isNew ? 'Create Agent' : `Edit: ${name}`}
      </h1>

      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-btn text-sm text-red-700">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Name */}
        <div>
          <label className="block text-sm font-medium text-brand-dark mb-1">Name</label>
          <input
            type="text"
            value={form.name}
            onChange={(e) => updateField('name', e.target.value)}
            required
            disabled={!isNew}
            pattern="^[a-z][a-z0-9-]*$"
            placeholder="my-agent"
            className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent disabled:bg-brand-light disabled:text-brand-shade3"
          />
          <p className="mt-1 text-xs text-brand-shade3">Lowercase letters, numbers, and hyphens. Must start with a letter.</p>
        </div>

        {/* Model */}
        <div>
          <label className="block text-sm font-medium text-brand-dark mb-1">Model</label>
          <select
            value={form.model_id ?? ''}
            onChange={(e) => updateField('model_id', e.target.value ? Number(e.target.value) : undefined)}
            className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent"
          >
            <option value="">Default model</option>
            {(models ?? []).map((m) => (
              <option key={m.id} value={m.id}>
                {m.name} ({m.model_name})
              </option>
            ))}
          </select>
        </div>

        {/* System Prompt */}
        <div>
          <label className="block text-sm font-medium text-brand-dark mb-1">System Prompt</label>
          <textarea
            value={form.system_prompt}
            onChange={(e) => updateField('system_prompt', e.target.value)}
            required
            rows={10}
            className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm font-mono focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent"
          />
        </div>

        {/* Kit */}
        <div>
          <label className="block text-sm font-medium text-brand-dark mb-1">Kit</label>
          <select
            value={form.kit ?? ''}
            onChange={(e) => updateField('kit', e.target.value)}
            className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent"
          >
            <option value="">None</option>
            <option value="developer">Developer</option>
          </select>
        </div>

        {/* Lifecycle + Execution */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-brand-dark mb-1">Lifecycle</label>
            <select
              value={form.lifecycle}
              onChange={(e) => updateField('lifecycle', e.target.value)}
              className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent"
            >
              <option value="persistent">Persistent</option>
              <option value="spawn">Spawn</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-dark mb-1">Tool Execution</label>
            <select
              value={form.tool_execution}
              onChange={(e) => updateField('tool_execution', e.target.value)}
              className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent"
            >
              <option value="sequential">Sequential</option>
              <option value="parallel">Parallel</option>
            </select>
          </div>
        </div>

        {/* Max Steps + Max Context */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-brand-dark mb-1">Max Steps</label>
            <input
              type="number"
              value={form.max_steps}
              onChange={(e) => updateField('max_steps', Number(e.target.value))}
              min={1}
              max={500}
              className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-dark mb-1">Max Context Size</label>
            <input
              type="number"
              value={form.max_context_size}
              onChange={(e) => updateField('max_context_size', Number(e.target.value))}
              min={1000}
              max={200000}
              step={1000}
              className="w-full px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent"
            />
          </div>
        </div>

        {/* Builtin Tools */}
        <div>
          <label className="block text-sm font-medium text-brand-dark mb-2">Builtin Tools</label>
          <div className="flex flex-wrap gap-2">
            {BUILTIN_TOOLS.map((tool) => (
              <label
                key={tool}
                className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-btn border text-sm cursor-pointer transition-colors ${
                  form.tools?.includes(tool)
                    ? 'border-brand-accent bg-brand-accent/10 text-brand-accent'
                    : 'border-brand-shade1 bg-white text-brand-shade3 hover:border-brand-shade2'
                }`}
              >
                <input
                  type="checkbox"
                  checked={form.tools?.includes(tool) ?? false}
                  onChange={() => updateField('tools', toggleInArray(form.tools ?? [], tool))}
                  className="sr-only"
                />
                {tool}
              </label>
            ))}
          </div>
        </div>

        {/* MCP Servers */}
        <div>
          <label className="block text-sm font-medium text-brand-dark mb-2">MCP Servers</label>
          {(mcpServers ?? []).length === 0 ? (
            <p className="text-sm text-brand-shade3">No MCP servers configured.</p>
          ) : (
            <div className="flex flex-wrap gap-2">
              {(mcpServers ?? []).map((s) => (
                <label
                  key={s.name}
                  className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-btn border text-sm cursor-pointer transition-colors ${
                    form.mcp_servers?.includes(s.name)
                      ? 'border-brand-accent bg-brand-accent/10 text-brand-accent'
                      : 'border-brand-shade1 bg-white text-brand-shade3 hover:border-brand-shade2'
                  }`}
                >
                  <input
                    type="checkbox"
                    checked={form.mcp_servers?.includes(s.name) ?? false}
                    onChange={() => updateField('mcp_servers', toggleInArray(form.mcp_servers ?? [], s.name))}
                    className="sr-only"
                  />
                  {s.name}
                </label>
              ))}
            </div>
          )}
        </div>

        {/* Can Spawn */}
        <div>
          <label className="block text-sm font-medium text-brand-dark mb-2">Can Spawn</label>
          {otherAgents.length === 0 ? (
            <p className="text-sm text-brand-shade3">No other agents available.</p>
          ) : (
            <div className="flex flex-wrap gap-2">
              {otherAgents.map((a) => (
                <label
                  key={a.name}
                  className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-btn border text-sm cursor-pointer transition-colors ${
                    form.can_spawn?.includes(a.name)
                      ? 'border-brand-accent bg-brand-accent/10 text-brand-accent'
                      : 'border-brand-shade1 bg-white text-brand-shade3 hover:border-brand-shade2'
                  }`}
                >
                  <input
                    type="checkbox"
                    checked={form.can_spawn?.includes(a.name) ?? false}
                    onChange={() => updateField('can_spawn', toggleInArray(form.can_spawn ?? [], a.name))}
                    className="sr-only"
                  />
                  {a.name}
                </label>
              ))}
            </div>
          )}
        </div>

        {/* Confirm Before */}
        <div>
          <label className="block text-sm font-medium text-brand-dark mb-1">Confirm Before (tools requiring user confirmation)</label>
          <div className="flex gap-2 mb-2">
            <input
              type="text"
              value={confirmInput}
              onChange={(e) => setConfirmInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault();
                  addConfirmBefore();
                }
              }}
              placeholder="Tool name..."
              className="flex-1 px-3 py-2 bg-white border border-brand-shade1 rounded-card text-sm focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent"
            />
            <button
              type="button"
              onClick={addConfirmBefore}
              className="px-3 py-2 bg-brand-light border border-brand-shade1 rounded-btn text-sm hover:bg-brand-shade1/50"
            >
              Add
            </button>
          </div>
          <div className="flex flex-wrap gap-1">
            {(form.confirm_before ?? []).map((t) => (
              <span
                key={t}
                className="inline-flex items-center gap-1 px-2 py-1 bg-yellow-50 border border-yellow-200 rounded text-xs text-yellow-800"
              >
                {t}
                <button
                  type="button"
                  onClick={() =>
                    updateField(
                      'confirm_before',
                      (form.confirm_before ?? []).filter((x) => x !== t),
                    )
                  }
                  className="text-yellow-600 hover:text-yellow-800"
                >
                  x
                </button>
              </span>
            ))}
          </div>
        </div>

        {/* Actions */}
        <div className="flex gap-3 pt-4 border-t border-brand-shade1">
          <button
            type="submit"
            disabled={saving}
            className="px-6 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving...' : isNew ? 'Create Agent' : 'Save Changes'}
          </button>
          <button
            type="button"
            onClick={() => navigate('/agents')}
            className="px-6 py-2 text-brand-dark border border-brand-shade2 rounded-btn text-sm hover:bg-brand-light transition-colors"
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  );
}
