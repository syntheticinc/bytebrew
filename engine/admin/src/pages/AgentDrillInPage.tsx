import { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import FormField from '../components/FormField';
import CapabilityBlock from '../components/builder/CapabilityBlock';
import ConfirmDialog from '../components/ConfirmDialog';
import { ToastProvider, useToast } from '../components/builder/Toast';
import type { AgentDetail, CapabilityConfig, CapabilityType, Model } from '../types';
import { CAPABILITY_META } from '../types';
import { usePrototype } from '../hooks/usePrototype';
import { MOCK_AGENTS, MOCK_MODELS } from '../mocks/agents';
import { MOCK_SESSIONS } from '../mocks/inspect';

const ALL_CAPABILITY_TYPES = Object.keys(CAPABILITY_META) as CapabilityType[];

const ZONE_ORDER = ['safe', 'caution', 'dangerous'] as const;
type Zone = (typeof ZONE_ORDER)[number];

const ZONE_CONFIG: Record<Zone, { label: string; labelClass: string; borderClass: string }> = {
  safe:      { label: 'Safe',      labelClass: 'text-brand-shade3',  borderClass: 'border-brand-shade3/30' },
  caution:   { label: 'Caution',   labelClass: 'text-amber-400',     borderClass: 'border-amber-500/30' },
  dangerous: { label: 'Dangerous', labelClass: 'text-brand-accent',  borderClass: 'border-brand-accent/50' },
};

// Mock tools grouped by zone for display when metadata is unavailable
const MOCK_TOOLS: Record<Zone, string[]> = {
  safe:      ['search_web', 'read_url', 'get_time'],
  caution:   ['read_file', 'list_files', 'send_http'],
  dangerous: ['write_file', 'execute_command', 'delete_file'],
};

function AgentDrillInInner() {
  const { addToast } = useToast();
  const { schema, agent: agentName } = useParams<{ schema: string; agent: string }>();
  const navigate = useNavigate();
  const { isPrototype } = usePrototype();

  const [agent, setAgent] = useState<AgentDetail | null>(null);
  const [capabilities, setCapabilities] = useState<CapabilityConfig[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [showCapDropdown, setShowCapDropdown] = useState(false);
  const [dangerousExpanded, setDangerousExpanded] = useState(false);
  const [enabledTools, setEnabledTools] = useState<string[]>([]);
  const [canSpawn, setCanSpawn] = useState<string[]>([]);
  const [allAgentNames, setAllAgentNames] = useState<string[]>([]);
  const [models, setModels] = useState<Model[]>([]);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!agentName) { setLoading(false); return; }

    if (isPrototype) {
      const mockAgent = MOCK_AGENTS[agentName] ?? MOCK_AGENTS['support-agent'];
      if (mockAgent) {
        setAgent(mockAgent);
        setEnabledTools(mockAgent.tools ?? []);
        setCanSpawn(mockAgent.can_spawn ?? []);
      }
      // Pre-populate capabilities for prototype (matching original prototype)
      setCapabilities([
        { type: 'memory', enabled: true, config: { unlimited_retention: false, retention_days: 30, unlimited_entries: false, max_entries: 500 } },
        { type: 'knowledge', enabled: true, config: { sources: ['support-docs.pdf'], chunks: 2341, top_k: 5, similarity_threshold: 0.75 } },
      ]);
      setModels(MOCK_MODELS as Model[]);
      setAllAgentNames(Object.keys(MOCK_AGENTS));
      setLoading(false);
      return;
    }

    api.getAgent(agentName)
      .then((data) => {
        setAgent(data);
        setEnabledTools(data.tools ?? []);
        setCanSpawn(data.can_spawn ?? []);
      })
      .catch(() => { /* fallback to empty */ })
      .finally(() => setLoading(false));
  }, [agentName, isPrototype]);

  // Fetch models and all agent names for connections (production only)
  useEffect(() => {
    if (isPrototype) return;
    api.listModels().then(setModels).catch(() => {});
    api.listAgents().then((agents: Array<{ name: string }>) => setAllAgentNames(agents.map((a) => a.name))).catch(() => {});
  }, [isPrototype]);

  // Close dropdown on outside click
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setShowCapDropdown(false);
      }
    }
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, []);

  function addCapability(type: CapabilityType) {
    if (capabilities.some((c) => c.type === type)) return;
    setCapabilities((prev) => [...prev, { type, enabled: true, config: {} }]);
    setShowCapDropdown(false);
  }

  function updateCapability(index: number, updated: CapabilityConfig) {
    setCapabilities((prev) => prev.map((c, i) => (i === index ? updated : c)));
  }

  function removeCapability(index: number) {
    setCapabilities((prev) => prev.filter((_, i) => i !== index));
  }

  function toggleTool(name: string) {
    setEnabledTools((prev) =>
      prev.includes(name) ? prev.filter((t) => t !== name) : [...prev, name],
    );
  }

  async function handleSave() {
    if (!agentName || !agent) return;
    setSaving(true);
    try {
      await api.updateAgent(agentName, {
        system_prompt: agent.system_prompt,
        model_id: agent.model_id,
        lifecycle: agent.lifecycle,
        tool_execution: agent.tool_execution,
        max_steps: agent.max_steps,
        max_context_size: agent.max_context_size,
        tools: enabledTools,
        can_spawn: canSpawn,
      });
      addToast('Agent saved successfully', 'success');
    } catch (err) {
      addToast(err instanceof Error ? err.message : 'Save failed', 'error');
    } finally {
      setSaving(false);
    }
  }

  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  async function handleDeleteConfirmed() {
    if (!agentName) return;
    try {
      await api.deleteAgent(agentName);
      navigate('/builder');
    } catch (err) {
      addToast(err instanceof Error ? err.message : 'Delete failed', 'error');
    }
  }

  function updateAgentField<K extends keyof AgentDetail>(key: K, value: AgentDetail[K]) {
    setAgent((prev) => (prev ? { ...prev, [key]: value } : prev));
  }

  const usedCapabilityTypes = new Set(capabilities.map((c) => c.type));
  const availableCapTypes = ALL_CAPABILITY_TYPES.filter((t) => !usedCapabilityTypes.has(t));

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64 text-brand-shade3 text-sm font-mono">
        Loading…
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full min-h-0">
      {/* Breadcrumb header */}
      <div className="flex items-center justify-between px-6 py-3 border-b border-brand-shade3/10 bg-brand-dark-surface flex-shrink-0">
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => navigate('/builder')}
            className="flex items-center gap-2 text-brand-shade3 hover:text-brand-light transition-colors text-sm font-mono px-2 py-1 -ml-2 rounded-btn hover:bg-brand-dark-alt"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
              <path d="M19 12H5M12 19l-7-7 7-7" />
            </svg>
            {schema}
          </button>
          <span className="text-brand-shade3/40 text-sm">/</span>
          <span className="text-brand-light text-sm font-mono font-semibold">{agentName}</span>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => navigate(`/builder/${schema}/${agentName}/inspect/${MOCK_SESSIONS[0]?.id ?? 'latest'}`)}
            className="px-4 py-1.5 border border-brand-shade3/40 text-brand-shade2 rounded-btn text-sm font-medium font-mono hover:text-brand-light hover:border-brand-shade3 transition-colors"
          >
            Inspect
          </button>
          <button
            type="button"
            onClick={handleSave}
            disabled={saving}
            className="px-4 py-1.5 bg-brand-accent text-brand-light rounded-btn text-sm font-medium font-mono hover:bg-brand-accent/90 disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving…' : 'Save'}
          </button>
          <button
            type="button"
            onClick={() => setShowDeleteConfirm(true)}
            className="px-4 py-1.5 border border-brand-accent/40 text-brand-accent rounded-btn text-sm font-medium font-mono hover:bg-brand-accent/10 transition-colors"
          >
            Delete
          </button>
        </div>
      </div>

      {/* Delete confirmation */}
      <ConfirmDialog
        open={showDeleteConfirm}
        onClose={() => setShowDeleteConfirm(false)}
        onConfirm={() => { setShowDeleteConfirm(false); handleDeleteConfirmed(); }}
        title={`Delete "${agentName}"?`}
        message="This action cannot be undone."
        confirmLabel="Delete"
        variant="danger"
      />

      {/* Scrollable body */}
      <div className="flex-1 overflow-y-auto px-6 py-6 space-y-4">

        {/* Model + Lifecycle card */}
        <div className="bg-brand-dark-surface border border-brand-shade3/10 rounded-card p-4">
          <h2 className="flex items-center gap-2 text-xs font-semibold text-brand-shade3 uppercase tracking-widest mb-3 font-mono">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5"><path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z" /><path d="M3.27 6.96L12 12.01l8.73-5.05M12 22.08V12" /></svg>
            Model & Lifecycle
          </h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-brand-light mb-1">Model</label>
              <select
                value={agent?.model_id ?? ''}
                onChange={(e) => updateAgentField('model_id', e.target.value ? Number(e.target.value) : undefined)}
                className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light font-mono focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent transition-colors"
              >
                <option value="">Select model...</option>
                {models.map((m) => (
                  <option key={m.id} value={m.id}>{m.name} ({m.model_name})</option>
                ))}
              </select>
              <p className="mt-1 text-xs text-brand-shade3">LLM model used for agent reasoning</p>
            </div>
            <FormField
              label="Lifecycle"
              type="select"
              value={agent?.lifecycle ?? 'persistent'}
              onChange={(v) => updateAgentField('lifecycle', v as AgentDetail['lifecycle'])}
              options={[
                { value: 'persistent', label: 'Persistent' },
                { value: 'spawn', label: 'Spawn' },
              ]}
              hint="Persistent: always running. Spawn: created on-demand by other agents"
            />
          </div>
        </div>

        {/* System Prompt card */}
        <div className="bg-brand-dark-surface border border-brand-shade3/10 rounded-card p-4">
          <h2 className="flex items-center gap-2 text-xs font-semibold text-brand-shade3 uppercase tracking-widest mb-3 font-mono">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5"><path d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z" /></svg>
            System Prompt
          </h2>
          <textarea
            value={agent?.system_prompt ?? ''}
            onChange={(e) => updateAgentField('system_prompt', e.target.value)}
            rows={6}
            className="w-full px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light font-mono focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent transition-colors resize-y"
            style={{ minHeight: '120px' }}
          />
          <p className="mt-1 text-xs text-brand-shade3">Instructions that define agent behavior, personality, and constraints</p>
        </div>

        {/* Parameters card */}
        <div className="bg-brand-dark-surface border border-brand-shade3/10 rounded-card p-4">
          <h2 className="flex items-center gap-2 text-xs font-semibold text-brand-shade3 uppercase tracking-widest mb-3 font-mono">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5"><line x1="4" y1="21" x2="4" y2="14" /><line x1="4" y1="10" x2="4" y2="3" /><line x1="12" y1="21" x2="12" y2="12" /><line x1="12" y1="8" x2="12" y2="3" /><line x1="20" y1="21" x2="20" y2="16" /><line x1="20" y1="12" x2="20" y2="3" /><line x1="1" y1="14" x2="7" y2="14" /><line x1="9" y1="8" x2="15" y2="8" /><line x1="17" y1="16" x2="23" y2="16" /></svg>
            Parameters
          </h2>
          <div className="grid grid-cols-3 gap-4">
            <FormField
              label="Max Turn Steps"
              type="number"
              value={agent?.max_steps ?? 50}
              onChange={(v) => updateAgentField('max_steps', Number(v))}
              min={1}
              max={500}
              hint="Max actions per turn (tool calls, reasoning, responses). Prevents infinite loops within a single interaction"
            />
            <FormField
              label="Context Size"
              type="number"
              value={agent?.max_context_size ?? 16000}
              onChange={(v) => updateAgentField('max_context_size', Number(v))}
              min={1000}
              max={200000}
              step={1000}
              hint="Token window for conversation history (larger = more memory, higher cost)"
            />
            <FormField
              label="Execution"
              type="select"
              value={agent?.tool_execution ?? 'sequential'}
              onChange={(v) => updateAgentField('tool_execution', v as AgentDetail['tool_execution'])}
              options={[
                { value: 'sequential', label: 'Sequential' },
                { value: 'parallel', label: 'Parallel' },
              ]}
              hint="Sequential: tools one by one. Parallel: multiple tools simultaneously"
            />
          </div>
        </div>

        {/* Capabilities card */}
        <div className="bg-brand-dark-surface border border-brand-shade3/10 rounded-card p-4">
          <div className="flex items-center justify-between mb-3">
            <h2 className="flex items-center gap-2 text-xs font-semibold text-brand-shade3 uppercase tracking-widest font-mono">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5"><rect x="3" y="3" width="7" height="7" rx="1" /><rect x="14" y="3" width="7" height="7" rx="1" /><rect x="3" y="14" width="7" height="7" rx="1" /><rect x="14" y="14" width="7" height="7" rx="1" /></svg>
              Capabilities
            </h2>
            <div className="relative" ref={dropdownRef}>
              <button
                type="button"
                onClick={() => setShowCapDropdown((v) => !v)}
                className="px-3 py-1 border border-brand-shade3/30 rounded-btn text-xs text-brand-shade2 font-mono hover:text-brand-light hover:border-brand-shade3/60 transition-colors"
              >
                + Add
              </button>
              {showCapDropdown && availableCapTypes.length > 0 && (
                <div className="absolute right-0 top-full mt-1 z-50 bg-brand-dark-alt border border-brand-shade3/20 rounded-card shadow-lg min-w-[180px]">
                  {availableCapTypes.map((type) => (
                    <button
                      key={type}
                      type="button"
                      onClick={() => addCapability(type)}
                      className="w-full flex items-center gap-2 px-3 py-2 text-left text-sm text-brand-shade2 hover:bg-brand-dark hover:text-brand-light font-mono transition-colors"
                    >
                      <span className="text-[10px] font-semibold text-brand-shade3 bg-brand-dark px-1.5 py-0.5 rounded-card">{CAPABILITY_META[type].abbr}</span>
                      <span>{CAPABILITY_META[type].label}</span>
                    </button>
                  ))}
                </div>
              )}
              {showCapDropdown && availableCapTypes.length === 0 && (
                <div className="absolute right-0 top-full mt-1 z-50 bg-brand-dark-alt border border-brand-shade3/20 rounded-card shadow-lg px-3 py-2 text-xs text-brand-shade3 font-mono">
                  All capabilities added
                </div>
              )}
            </div>
          </div>
          {capabilities.length === 0 ? (
            <p className="text-sm text-brand-shade3 font-mono">No capabilities configured. Click + Add to extend agent with memory, knowledge, guardrails, and more.</p>
          ) : (
            <div className="space-y-2">
              {capabilities.map((cap, i) => (
                <CapabilityBlock
                  key={cap.type}
                  capability={cap}
                  onChange={(updated) => updateCapability(i, updated)}
                  onRemove={() => removeCapability(i)}
                />
              ))}
            </div>
          )}
        </div>

        {/* Tools card */}
        <div className="bg-brand-dark-surface border border-brand-shade3/10 rounded-card p-4">
          <h2 className="flex items-center gap-2 text-xs font-semibold text-brand-shade3 uppercase tracking-widest mb-1 font-mono">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5"><path d="M14.7 6.3a1 1 0 000 1.4l1.6 1.6a1 1 0 001.4 0l3.77-3.77a6 6 0 01-7.94 7.94l-6.91 6.91a2.12 2.12 0 01-3-3l6.91-6.91a6 6 0 017.94-7.94l-3.76 3.76z" /></svg>
            Tools
          </h2>
          <p className="text-xs text-brand-shade3 mb-3">Predefined engine tools available to this agent. External integrations connect via MCP servers.</p>
          <div className="space-y-3">
            {ZONE_ORDER.map((zone) => {
              if (zone === 'dangerous') {
                return (
                  <div key={zone} className={`border-2 ${ZONE_CONFIG[zone].borderClass} rounded-card overflow-hidden`}>
                    <button
                      type="button"
                      onClick={() => setDangerousExpanded((v) => !v)}
                      className="w-full flex items-center justify-between px-3 py-2 bg-brand-accent/10 text-brand-accent text-xs font-semibold font-mono hover:bg-brand-accent/15 transition-colors"
                    >
                      <span>Dangerous — Filesystem & Command Access</span>
                      <span className="text-brand-accent/60">{dangerousExpanded ? '▲' : '▼'}</span>
                    </button>
                    {dangerousExpanded && (
                      <div className="p-3 flex flex-wrap gap-2">
                        {MOCK_TOOLS[zone].map((tool) => (
                          <ToolChip key={tool} name={tool} zone={zone} enabled={enabledTools.includes(tool)} onToggle={toggleTool} />
                        ))}
                      </div>
                    )}
                  </div>
                );
              }
              return (
                <div key={zone} className={`border ${ZONE_CONFIG[zone].borderClass} rounded-card p-3`}>
                  <p className={`text-xs font-semibold ${ZONE_CONFIG[zone].labelClass} mb-2 font-mono`}>
                    {ZONE_CONFIG[zone].label}
                  </p>
                  <div className="flex flex-wrap gap-2">
                    {MOCK_TOOLS[zone].map((tool) => (
                      <ToolChip key={tool} name={tool} zone={zone} enabled={enabledTools.includes(tool)} onToggle={toggleTool} />
                    ))}
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        {/* Connections card */}
        <div className="bg-brand-dark-surface border border-brand-shade3/10 rounded-card p-4">
          <h2 className="flex items-center gap-2 text-xs font-semibold text-brand-shade3 uppercase tracking-widest mb-3 font-mono">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5"><circle cx="5" cy="6" r="2" /><circle cx="19" cy="6" r="2" /><circle cx="12" cy="18" r="2" /><line x1="5" y1="8" x2="12" y2="16" /><line x1="19" y1="8" x2="12" y2="16" /></svg>
            Connections</h2>
          <div className="grid grid-cols-2 gap-4">
            <div className="border border-brand-shade3/10 rounded-card p-3">
              <p className="text-xs text-brand-shade3 font-mono mb-2">Receives from</p>
              <p className="text-xs text-brand-shade3/50 font-mono italic">
                Determined by other agents' can_spawn config (read-only)
              </p>
            </div>
            <div className="border border-brand-shade3/10 rounded-card p-3">
              <p className="text-xs text-brand-shade3 font-mono mb-2">Can spawn</p>
              {allAgentNames.filter((n) => n !== agentName).length === 0 ? (
                <p className="text-xs text-brand-shade3/50 font-mono">No other agents available</p>
              ) : (
                <div className="space-y-1">
                  {allAgentNames.filter((n) => n !== agentName).map((name) => (
                    <label key={name} className="flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={canSpawn.includes(name)}
                        onChange={() =>
                          setCanSpawn((prev) =>
                            prev.includes(name) ? prev.filter((n) => n !== name) : [...prev, name],
                          )
                        }
                        className="accent-brand-accent"
                      />
                      <span className="text-xs text-brand-shade2 font-mono">{name}</span>
                    </label>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>

      </div>
    </div>
  );
}

// ─── ToolChip ─────────────────────────────────────────────────────────────────

interface ToolChipProps {
  name: string;
  zone: Zone;
  enabled: boolean;
  onToggle: (name: string) => void;
}

const ZONE_ACTIVE: Record<Zone, string> = {
  safe:      'border-status-active bg-status-active/15 text-status-active',
  caution:   'border-amber-500 bg-amber-500/15 text-amber-400',
  dangerous: 'border-brand-accent bg-brand-accent/15 text-brand-accent',
};

function ToolChip({ name, zone, enabled, onToggle }: ToolChipProps) {
  return (
    <label
      className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-btn border text-sm font-mono cursor-pointer transition-colors ${
        enabled
          ? ZONE_ACTIVE[zone]
          : `${ZONE_CONFIG[zone].borderClass} bg-brand-dark-alt text-brand-shade2 hover:text-brand-light`
      }`}
    >
      <input type="checkbox" checked={enabled} onChange={() => onToggle(name)} className="sr-only" />
      {name}
    </label>
  );
}

// ─── Wrapper with ToastProvider ──────────────────────────────────────────────

export default function AgentDrillInPage() {
  return (
    <ToastProvider>
      <AgentDrillInInner />
    </ToastProvider>
  );
}
