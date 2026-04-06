import { useState, useEffect, useMemo, useRef, type FormEvent } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { useApi } from '../hooks/useApi';
import { useModelRegistry } from '../hooks/useModelRegistry';
import TierBadge from '../components/TierBadge';
import type { CreateAgentRequest, Model, MCPServer, AgentInfo, ToolMetadata, SecurityZone } from '../types';

const ZONE_ORDER: SecurityZone[] = ['safe', 'caution', 'dangerous'];

const ZONE_CONFIG: Record<SecurityZone, {
  label: string;
  subtitle: string;
  borderClass: string;
  bgClass: string;
  activeClass: string;
  labelClass: string;
  zoneBorderClass: string;
  zoneBgClass: string;
}> = {
  safe: {
    label: 'Safe',
    subtitle: 'No security risk',
    borderClass: 'border-brand-shade3/30',
    bgClass: 'bg-brand-dark',
    activeClass: 'border-status-active bg-status-active/15 text-status-active',
    labelClass: 'text-brand-shade3',
    zoneBorderClass: 'border-brand-shade3/30',
    zoneBgClass: '',
  },
  caution: {
    label: 'Caution',
    subtitle: 'Read-only access, external content',
    borderClass: 'border-amber-500/30',
    bgClass: 'bg-amber-500/10',
    activeClass: 'border-amber-500 bg-amber-500/15 text-amber-400',
    labelClass: 'text-amber-400',
    zoneBorderClass: 'border-amber-500/30',
    zoneBgClass: '',
  },
  dangerous: {
    label: 'Dangerous Tools — Filesystem & Command Access',
    subtitle: '',
    borderClass: 'border-brand-accent/50',
    bgClass: 'bg-brand-accent/10',
    activeClass: 'border-brand-accent bg-brand-accent/15 text-brand-accent',
    labelClass: 'text-brand-accent',
    zoneBorderClass: 'border-brand-accent/50',
    zoneBgClass: '',
  },
};

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
    max_turn_duration: 120,
    tools: [],
    can_spawn: [],
    mcp_servers: [],
    confirm_before: [],
  });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [confirmInput, setConfirmInput] = useState('');
  const [dangerousExpanded, setDangerousExpanded] = useState(false);

  const { data: models } = useApi<Model[]>(() => api.listModels());
  const { data: mcpServers } = useApi<MCPServer[]>(() => api.listMCPServers());
  const { data: allAgents } = useApi<AgentInfo[]>(() => api.listAgents());
  const { data: toolMetadata } = useApi<ToolMetadata[]>(() => api.listToolMetadata());
  const { registryByModelName } = useModelRegistry();

  // Compute tier warning for the selected model
  const tierWarning = useMemo(() => {
    if (!form.model_id || !models) return null;
    const selectedModel = models.find((m) => m.id === form.model_id);
    if (!selectedModel) return null;

    const registryEntry = registryByModelName.get(selectedModel.model_name);
    const hasCanSpawn = (form.can_spawn ?? []).length > 0;

    if (!registryEntry) {
      return {
        type: 'info' as const,
        message: 'Model not in registry -- not tested for agent use.',
      };
    }

    if (hasCanSpawn && registryEntry.tier >= 2) {
      return {
        type: 'warning' as const,
        message: `This model (${registryEntry.display_name}) is classified as Tier ${registryEntry.tier}. It may not reliably handle complex multi-step tool calling. Consider using a Tier 1 model for orchestrator agents.`,
        tier: registryEntry.tier,
      };
    }

    return {
      type: 'ok' as const,
      tier: registryEntry.tier,
    };
  }, [form.model_id, form.can_spawn, models, registryByModelName]);

  // Group tools by security zone
  const toolsByZone = useMemo(() => {
    const grouped: Record<SecurityZone, ToolMetadata[]> = { safe: [], caution: [], dangerous: [] };
    if (!toolMetadata) return grouped;
    for (const tool of toolMetadata) {
      const zone = tool.security_zone ?? 'caution';
      if (grouped[zone]) {
        grouped[zone].push(tool);
      } else {
        grouped.caution.push(tool);
      }
    }
    // Sort alphabetically within each zone
    for (const zone of ZONE_ORDER) {
      grouped[zone].sort((a, b) => a.name.localeCompare(b.name));
    }
    return grouped;
  }, [toolMetadata]);

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
        max_turn_duration: agent.max_turn_duration,
        tools: agent.tools,
        can_spawn: agent.can_spawn,
        mcp_servers: agent.mcp_servers,
        confirm_before: agent.confirm_before,
      });
      // Expand dangerous section if agent already has dangerous tools
      const dangerousNames = new Set(toolsByZone.dangerous.map((t) => t.name));
      if (agent.tools?.some((t) => dangerousNames.has(t))) {
        setDangerousExpanded(true);
      }
    }).catch((err: Error) => setError(err.message));
  }, [name, isNew, toolsByZone]);

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

  function ToolChip({ tool, zone }: { tool: ToolMetadata; zone: SecurityZone }) {
    const [showPopover, setShowPopover] = useState(false);
    const timeoutRef = useRef<ReturnType<typeof setTimeout>>(null);
    const isSelected = form.tools?.includes(tool.name) ?? false;
    const cfg = ZONE_CONFIG[zone];

    function handleMouseEnter() {
      timeoutRef.current = setTimeout(() => setShowPopover(true), 300);
    }
    function handleMouseLeave() {
      if (timeoutRef.current !== null) clearTimeout(timeoutRef.current);
      setShowPopover(false);
    }

    return (
      <div className="relative" onMouseEnter={handleMouseEnter} onMouseLeave={handleMouseLeave}>
        <label
          className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-btn border text-sm cursor-pointer transition-colors ${
            isSelected ? cfg.activeClass : `${cfg.borderClass} bg-brand-dark-alt text-brand-shade2 hover:${cfg.bgClass}`
          }`}
        >
          <input
            type="checkbox"
            checked={isSelected}
            onChange={() => updateField('tools', toggleInArray(form.tools ?? [], tool.name))}
            className="sr-only"
          />
          {tool.name}
        </label>

        {showPopover && (
          <div className="absolute z-50 w-80 p-3 bg-brand-dark-alt rounded-card border border-brand-shade3/30 shadow-lg left-0 top-full mt-1">
            <div className="font-semibold text-brand-light text-sm mb-1">{tool.name}</div>
            <p className="text-xs text-brand-shade3 leading-relaxed mb-1">{tool.description}</p>
            {tool.risk_warning && (
              <p className="text-xs text-brand-accent bg-brand-accent/10 p-2 rounded leading-relaxed">{tool.risk_warning}</p>
            )}
            {tool.hint && (
              <p className="text-xs text-amber-400 bg-amber-500/10 p-2 rounded leading-relaxed mt-1">{tool.hint}</p>
            )}
          </div>
        )}
        {/* Show hint inline when tool is checked and companion is not */}
        {isSelected && tool.hint && tool.companion && !(form.tools ?? []).includes(tool.companion) && (
          <div className="text-xs text-amber-400 mt-1 ml-6">{tool.hint}</div>
        )}
      </div>
    );
  }

  return (
    <div className="max-w-3xl">
      <h1 className="text-2xl font-bold text-brand-light mb-6">
        {isNew ? 'Create Agent' : `Edit: ${name}`}
      </h1>

      {error && (
        <div className="mb-4 p-3 bg-red-500/10 border border-red-500/30 rounded-btn text-sm text-red-400">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Name */}
        <div>
          <label className="block text-sm font-medium text-brand-light mb-1">Name</label>
          <input
            type="text"
            value={form.name}
            onChange={(e) => updateField('name', e.target.value)}
            required
            disabled={!isNew}
            pattern="^[a-z][a-z0-9-]*$"
            placeholder="my-agent"
            className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light placeholder-brand-shade3 focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent disabled:bg-brand-dark disabled:text-brand-shade3 disabled:opacity-60 transition-colors"
          />
          <p className="mt-1 text-xs text-brand-shade3">Lowercase letters, numbers, and hyphens. Must start with a letter.</p>
        </div>

        {/* Model */}
        <div>
          <label className="block text-sm font-medium text-brand-light mb-1">Model</label>
          <select
            value={form.model_id ?? ''}
            onChange={(e) => updateField('model_id', e.target.value ? Number(e.target.value) : undefined)}
            className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent transition-colors"
          >
            <option value="">Default model</option>
            {(models ?? []).map((m) => {
              const entry = registryByModelName.get(m.model_name);
              return (
                <option key={m.id} value={m.id}>
                  {m.name} ({m.model_name}){entry ? ` - Tier ${entry.tier}` : ''}
                </option>
              );
            })}
          </select>

          {tierWarning?.type === 'ok' && tierWarning.tier && (
            <div className="mt-2">
              <TierBadge tier={tierWarning.tier} />
            </div>
          )}

          {tierWarning?.type === 'warning' && (
            <div className="mt-2 p-3 bg-amber-500/10 border border-amber-500/30 rounded-btn text-xs text-amber-400 leading-relaxed">
              <span className="font-semibold">Warning:</span> {tierWarning.message}
            </div>
          )}

          {tierWarning?.type === 'info' && (
            <div className="mt-2 p-3 bg-blue-500/10 border border-blue-500/30 rounded-btn text-xs text-blue-400 leading-relaxed">
              {tierWarning.message}
            </div>
          )}
        </div>

        {/* System Prompt */}
        <div>
          <label className="block text-sm font-medium text-brand-light mb-1">System Prompt</label>
          <textarea
            value={form.system_prompt}
            onChange={(e) => updateField('system_prompt', e.target.value)}
            required
            rows={10}
            className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light font-mono focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent transition-colors"
          />
        </div>

        {/* Kit */}
        <div>
          <label className="block text-sm font-medium text-brand-light mb-1">Kit</label>
          <select
            value={form.kit ?? ''}
            onChange={(e) => updateField('kit', e.target.value)}
            className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent transition-colors"
          >
            <option value="">None</option>
            <option value="developer">Developer</option>
          </select>
        </div>

        {/* Lifecycle + Execution */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-brand-light mb-1">Lifecycle</label>
            <select
              value={form.lifecycle}
              onChange={(e) => updateField('lifecycle', e.target.value)}
              className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent transition-colors"
            >
              <option value="persistent">Persistent</option>
              <option value="spawn">Spawn</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-light mb-1">Tool Execution</label>
            <select
              value={form.tool_execution}
              onChange={(e) => updateField('tool_execution', e.target.value)}
              className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent transition-colors"
            >
              <option value="sequential">Sequential</option>
              <option value="parallel">Parallel</option>
            </select>
          </div>
        </div>

        {/* Max Steps + Max Context */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-brand-light mb-1">Max Steps</label>
            <input
              type="number"
              value={form.max_steps}
              onChange={(e) => updateField('max_steps', Number(e.target.value))}
              min={1}
              max={500}
              className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent transition-colors"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-light mb-1">Max Context Size</label>
            <input
              type="number"
              value={form.max_context_size}
              onChange={(e) => updateField('max_context_size', Number(e.target.value))}
              min={1000}
              max={200000}
              step={1000}
              className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent transition-colors"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-light mb-1">Max Turn Duration (s)</label>
            <input
              type="number"
              value={form.max_turn_duration}
              onChange={(e) => updateField('max_turn_duration', Number(e.target.value))}
              min={30}
              max={600}
              step={10}
              className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent transition-colors"
            />
          </div>
        </div>

        {/* Builtin Tools — grouped by security zone */}
        <div>
          <label className="block text-sm font-medium text-brand-light mb-3">Builtin Tools</label>

          {!toolMetadata ? (
            <p className="text-sm text-brand-shade3">Loading tool metadata...</p>
          ) : (
            <div className="space-y-4">
              {/* Safe Zone */}
              {toolsByZone.safe.length > 0 && (
                <div className={`border ${ZONE_CONFIG.safe.zoneBorderClass} rounded-card p-3`}>
                  <p className={`text-xs font-semibold ${ZONE_CONFIG.safe.labelClass} mb-2`}>Safe — No security risk</p>
                  <div className="flex flex-wrap gap-2">
                    {toolsByZone.safe.map((tool) => (
                      <ToolChip key={tool.name} tool={tool} zone="safe" />
                    ))}
                  </div>
                </div>
              )}

              {/* Caution Zone */}
              {toolsByZone.caution.length > 0 && (
                <div className={`border ${ZONE_CONFIG.caution.zoneBorderClass} rounded-card p-3`}>
                  <p className={`text-xs font-semibold ${ZONE_CONFIG.caution.labelClass} mb-2`}>Caution — Read-only access, external content</p>
                  <div className="flex flex-wrap gap-2">
                    {toolsByZone.caution.map((tool) => (
                      <ToolChip key={tool.name} tool={tool} zone="caution" />
                    ))}
                  </div>
                </div>
              )}

              {/* Dangerous Zone — collapsed by default */}
              <div className={`border-2 ${ZONE_CONFIG.dangerous.zoneBorderClass} rounded-card overflow-hidden`}>
                <button
                  type="button"
                  onClick={() => setDangerousExpanded(!dangerousExpanded)}
                  className="w-full flex items-center justify-between px-3 py-2 bg-brand-accent/10 text-brand-accent text-sm font-semibold hover:bg-brand-accent/15 transition-colors"
                >
                  <span>Dangerous Tools — Filesystem & Command Access</span>
                  <span className="text-brand-accent/60">{dangerousExpanded ? '▲' : '▼'}</span>
                </button>

                {dangerousExpanded && (
                  <div className="p-3 space-y-3">
                    <div className="p-3 bg-brand-accent/10 border border-brand-accent/20 rounded-btn">
                      <p className="text-xs text-brand-shade2 leading-relaxed">
                        <span className="font-semibold text-brand-accent">Warning:</span>{' '}
                        These tools give the agent direct access to the server filesystem and shell.
                        Only enable for fully trusted agents in isolated environments.
                        Misuse can lead to data loss, credential exposure, or system compromise.
                      </p>
                    </div>

                    <div className="flex flex-wrap gap-2">
                      {toolsByZone.dangerous.map((tool) => (
                        <ToolChip key={tool.name} tool={tool} zone="dangerous" />
                      ))}
                    </div>

                    {/* execute_command extra warning */}
                    {(form.tools ?? []).includes('execute_command') && (
                      <div className="mt-2 p-3 bg-red-500/10 border border-brand-accent/40 rounded-btn text-xs text-brand-shade2 leading-relaxed">
                        <strong className="text-brand-accent">CRITICAL:</strong> execute_command allows the agent to run ARBITRARY shell commands
                        with the server process permissions. This includes installing software, modifying system files,
                        accessing the network, and deleting data. Never enable for user-facing agents.
                      </div>
                    )}
                  </div>
                )}
              </div>
            </div>
          )}
        </div>

        {/* MCP Servers */}
        <div>
          <label className="block text-sm font-medium text-brand-light mb-2">MCP Servers</label>
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
                      : 'border-brand-shade3/30 bg-brand-dark-alt text-brand-shade2 hover:border-brand-shade3'
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
          <label className="block text-sm font-medium text-brand-light mb-2">Can Spawn</label>
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
                      : 'border-brand-shade3/30 bg-brand-dark-alt text-brand-shade2 hover:border-brand-shade3'
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
          <label className="block text-sm font-medium text-brand-light mb-1">Confirm Before (tools requiring user confirmation)</label>
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
              className="flex-1 px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light placeholder-brand-shade3 focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent transition-colors"
            />
            <button
              type="button"
              onClick={addConfirmBefore}
              className="px-3 py-2 bg-brand-dark-alt border border-brand-shade3/30 rounded-btn text-sm text-brand-shade2 hover:text-brand-light hover:bg-brand-dark transition-colors"
            >
              Add
            </button>
          </div>
          <div className="flex flex-wrap gap-1">
            {(form.confirm_before ?? []).map((t) => (
              <span
                key={t}
                className="inline-flex items-center gap-1 px-2 py-1 bg-yellow-500/10 border border-yellow-500/30 rounded text-xs text-yellow-400"
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
                  className="text-yellow-500 hover:text-yellow-300"
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
