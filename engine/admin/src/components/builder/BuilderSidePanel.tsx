import { useState, useEffect, useMemo, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../../api/client';
import { useApi } from '../../hooks/useApi';
import type { AgentDetail, AgentInfo, Model, MCPServer, ToolMetadata, SecurityZone, CreateAgentRequest } from '../../types';
import BuilderChat from './BuilderChat';

// ---------------------------------------------------------------------------
// Types & constants
// ---------------------------------------------------------------------------

interface BuilderSidePanelProps {
  agent: AgentDetail;
  agents?: AgentInfo[];
  onClose: () => void;
  onSaved: (updated: AgentDetail) => void;
  onDelete: (name: string) => void;
}

type Tab = 'details' | 'chat' | 'yaml';

const ZONE_ORDER: SecurityZone[] = ['safe', 'caution', 'dangerous'];

const ZONE_STYLE: Record<SecurityZone, {
  label: string;
  border: string;
  activeBorder: string;
  activeBg: string;
  activeText: string;
  headerBg: string;
  headerText: string;
}> = {
  safe: {
    label: 'Safe',
    border: 'border-brand-shade3/20',
    activeBorder: 'border-status-active',
    activeBg: 'bg-status-active/15',
    activeText: 'text-status-active',
    headerBg: '',
    headerText: 'text-brand-shade3',
  },
  caution: {
    label: 'Caution',
    border: 'border-amber-500/20',
    activeBorder: 'border-amber-500',
    activeBg: 'bg-amber-500/15',
    activeText: 'text-amber-400',
    headerBg: '',
    headerText: 'text-amber-400',
  },
  dangerous: {
    label: 'Dangerous',
    border: 'border-brand-accent/30',
    activeBorder: 'border-brand-accent',
    activeBg: 'bg-brand-accent/15',
    activeText: 'text-brand-accent',
    headerBg: 'bg-brand-accent/10',
    headerText: 'text-brand-accent',
  },
};

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function arraysEqual(a: string[] | undefined, b: string[] | undefined): boolean {
  const aa = a ?? [];
  const bb = b ?? [];
  if (aa.length !== bb.length) return false;
  const sorted1 = [...aa].sort();
  const sorted2 = [...bb].sort();
  return sorted1.every((v, i) => v === sorted2[i]);
}

function toggleInArray(arr: string[], item: string): string[] {
  return arr.includes(item) ? arr.filter((x) => x !== item) : [...arr, item];
}

function agentToYaml(agent: AgentDetail): string {
  const lines: string[] = [];
  lines.push(`name: ${agent.name}`);
  if (agent.model_id !== undefined) lines.push(`model_id: ${agent.model_id}`);
  lines.push(`lifecycle: ${agent.lifecycle}`);
  lines.push(`tool_execution: ${agent.tool_execution}`);
  lines.push(`max_steps: ${agent.max_steps}`);
  lines.push(`max_context_size: ${agent.max_context_size}`);
  lines.push(`max_turn_duration: ${agent.max_turn_duration}`);
  lines.push(`public: ${agent.has_knowledge ?? false}`);
  if (agent.kit) lines.push(`kit: ${agent.kit}`);

  lines.push(`system_prompt: |`);
  for (const line of agent.system_prompt.split('\n')) {
    lines.push(`  ${line}`);
  }

  if (agent.tools?.length) {
    lines.push('tools:');
    for (const t of agent.tools) lines.push(`  - ${t}`);
  } else {
    lines.push('tools: []');
  }

  if (agent.can_spawn?.length) {
    lines.push('can_spawn:');
    for (const a of agent.can_spawn) lines.push(`  - ${a}`);
  } else {
    lines.push('can_spawn: []');
  }

  if (agent.mcp_servers?.length) {
    lines.push('mcp_servers:');
    for (const s of agent.mcp_servers) lines.push(`  - ${s}`);
  } else {
    lines.push('mcp_servers: []');
  }

  if (agent.confirm_before?.length) {
    lines.push('confirm_before:');
    for (const t of agent.confirm_before) lines.push(`  - ${t}`);
  } else {
    lines.push('confirm_before: []');
  }

  return lines.join('\n');
}

// ---------------------------------------------------------------------------
// Collapsible Section
// ---------------------------------------------------------------------------

function Section({
  title,
  defaultOpen = true,
  count,
  children,
}: {
  title: string;
  defaultOpen?: boolean;
  count?: number;
  children: React.ReactNode;
}) {
  const [open, setOpen] = useState(defaultOpen);

  return (
    <div>
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="w-full flex items-center justify-between py-1 group"
      >
        <span className="text-[11px] font-semibold text-brand-shade3 uppercase tracking-wide group-hover:text-brand-light transition-colors">
          {title}
          {count !== undefined && count > 0 && (
            <span className="ml-1.5 text-brand-accent font-normal">({count})</span>
          )}
        </span>
        <svg
          className={`w-3 h-3 text-brand-shade3 transition-transform ${open ? 'rotate-180' : ''}`}
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          strokeWidth={2}
        >
          <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
        </svg>
      </button>
      {open && <div className="mt-1">{children}</div>}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Compact ToolChip for side panel
// ---------------------------------------------------------------------------

function ToolChip({
  tool,
  zone,
  checked,
  onToggle,
}: {
  tool: ToolMetadata;
  zone: SecurityZone;
  checked: boolean;
  onToggle: () => void;
}) {
  const [showPopover, setShowPopover] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout>>(null);
  const style = ZONE_STYLE[zone];

  function handleMouseEnter() {
    timeoutRef.current = setTimeout(() => setShowPopover(true), 400);
  }
  function handleMouseLeave() {
    if (timeoutRef.current !== null) clearTimeout(timeoutRef.current);
    setShowPopover(false);
  }

  return (
    <div className="relative" onMouseEnter={handleMouseEnter} onMouseLeave={handleMouseLeave}>
      <label
        className={`inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full border text-[10px] cursor-pointer transition-colors ${
          checked
            ? `${style.activeBorder} ${style.activeBg} ${style.activeText}`
            : `${style.border} bg-brand-dark text-brand-shade2 hover:text-brand-light`
        }`}
      >
        <input
          type="checkbox"
          checked={checked}
          onChange={onToggle}
          className="sr-only"
        />
        {tool.name}
      </label>

      {showPopover && (
        <div className="absolute z-50 w-64 p-2 bg-brand-dark-alt rounded-card border border-brand-shade3/30 shadow-lg left-0 top-full mt-1">
          <div className="font-semibold text-brand-light text-[11px] mb-0.5">{tool.name}</div>
          <p className="text-[10px] text-brand-shade3 leading-relaxed">{tool.description}</p>
          {tool.risk_warning && (
            <p className="text-[10px] text-brand-accent bg-brand-accent/10 p-1.5 rounded leading-relaxed mt-1">{tool.risk_warning}</p>
          )}
          {tool.hint && (
            <p className="text-[10px] text-amber-400 bg-amber-500/10 p-1.5 rounded leading-relaxed mt-1">{tool.hint}</p>
          )}
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export default function BuilderSidePanel({ agent, agents, onClose, onSaved, onDelete }: BuilderSidePanelProps) {
  const navigate = useNavigate();
  const [tab, setTab] = useState<Tab>('details');
  const [form, setForm] = useState<Partial<CreateAgentRequest>>({});
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState('');
  const [copied, setCopied] = useState(false);
  const [confirmInput, setConfirmInput] = useState('');
  const [dangerousExpanded, setDangerousExpanded] = useState(false);
  const [showDiscardConfirm, setShowDiscardConfirm] = useState(false);
  const [panelWidth, setPanelWidth] = useState(320);
  const resizingRef = useRef(false);
  const resizeStartXRef = useRef(0);
  const resizeStartWidthRef = useRef(320);

  // Data loading
  const { data: models } = useApi<Model[]>(() => api.listModels());
  const { data: toolMetadata } = useApi<ToolMetadata[]>(() => api.listToolMetadata());
  const { data: mcpServers } = useApi<MCPServer[]>(() => api.listMCPServers());
  // Fallback: load agents list when not provided via props
  const { data: fetchedAgents } = useApi<AgentInfo[]>(() => agents ? Promise.resolve(agents) : api.listAgents());
  const resolvedAgents = agents ?? fetchedAgents ?? [];

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
    for (const zone of ZONE_ORDER) {
      grouped[zone].sort((a, b) => a.name.localeCompare(b.name));
    }
    return grouped;
  }, [toolMetadata]);

  // Other agents for can_spawn
  const otherAgents = useMemo(
    () => resolvedAgents.filter((a) => a.name !== agent.name),
    [resolvedAgents, agent.name],
  );

  // Initialize form from agent
  useEffect(() => {
    setForm({
      model_id: agent.model_id,
      system_prompt: agent.system_prompt,
      lifecycle: agent.lifecycle,
      tool_execution: agent.tool_execution,
      max_steps: agent.max_steps,
      max_context_size: agent.max_context_size,
      max_turn_duration: agent.max_turn_duration,
      tools: agent.tools ?? [],
      can_spawn: agent.can_spawn ?? [],
      mcp_servers: agent.mcp_servers ?? [],
      confirm_before: agent.confirm_before ?? [],
    });
    setSaveError('');
    setConfirmInput('');
    setTab('details');

    // Expand dangerous if agent already has dangerous tools
    const dangerousNames = new Set(toolsByZone.dangerous.map((t) => t.name));
    if ((agent.tools ?? []).some((t) => dangerousNames.has(t))) {
      setDangerousExpanded(true);
    } else {
      setDangerousExpanded(false);
    }
  }, [agent.name, toolsByZone]);

  // Field updater
  function updateField<K extends keyof CreateAgentRequest>(key: K, value: CreateAgentRequest[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  // Dirty check — compare all editable fields
  const isDirty = useMemo(() => {
    if (form.model_id !== agent.model_id) return true;
    if ((form.system_prompt ?? '') !== (agent.system_prompt ?? '')) return true;
    if ((form.lifecycle ?? 'persistent') !== (agent.lifecycle ?? 'persistent')) return true;
    if ((form.tool_execution ?? 'sequential') !== (agent.tool_execution ?? 'sequential')) return true;
    if ((form.max_steps ?? 50) !== (agent.max_steps ?? 50)) return true;
    if ((form.max_context_size ?? 16000) !== (agent.max_context_size ?? 16000)) return true;
    if ((form.max_turn_duration ?? 120) !== (agent.max_turn_duration ?? 120)) return true;
    if (!arraysEqual(form.tools, agent.tools)) return true;
    if (!arraysEqual(form.can_spawn, agent.can_spawn)) return true;
    if (!arraysEqual(form.mcp_servers, agent.mcp_servers)) return true;
    if (!arraysEqual(form.confirm_before, agent.confirm_before)) return true;
    return false;
  }, [form, agent]);

  // Resize handlers
  function handleResizeMouseDown(e: React.MouseEvent) {
    e.preventDefault();
    resizingRef.current = true;
    resizeStartXRef.current = e.clientX;
    resizeStartWidthRef.current = panelWidth;

    function onMouseMove(ev: MouseEvent) {
      if (!resizingRef.current) return;
      const delta = resizeStartXRef.current - ev.clientX;
      const newWidth = Math.min(500, Math.max(280, resizeStartWidthRef.current + delta));
      setPanelWidth(newWidth);
    }
    function onMouseUp() {
      resizingRef.current = false;
      window.removeEventListener('mousemove', onMouseMove);
      window.removeEventListener('mouseup', onMouseUp);
    }
    window.addEventListener('mousemove', onMouseMove);
    window.addEventListener('mouseup', onMouseUp);
  }

  // Close with dirty guard
  function handleClose() {
    if (isDirty) {
      setShowDiscardConfirm(true);
    } else {
      onClose();
    }
  }

  // Escape key handler
  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') handleClose();
    }
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isDirty]);

  // Save handler
  async function handleSave() {
    setSaveError('');
    setSaving(true);
    try {
      const updated = await api.updateAgent(agent.name, form);
      onSaved(updated);
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Save failed');
    } finally {
      setSaving(false);
    }
  }

  // Confirm before helpers
  function addConfirmBefore() {
    const trimmed = confirmInput.trim();
    if (!trimmed || (form.confirm_before ?? []).includes(trimmed)) return;
    updateField('confirm_before', [...(form.confirm_before ?? []), trimmed]);
    setConfirmInput('');
  }

  // Build a merged AgentDetail from form state for YAML preview
  const formAsAgent: AgentDetail = {
    ...agent,
    model_id: form.model_id,
    system_prompt: form.system_prompt ?? agent.system_prompt,
    lifecycle: (form.lifecycle as AgentDetail['lifecycle']) ?? agent.lifecycle,
    tool_execution: (form.tool_execution as AgentDetail['tool_execution']) ?? agent.tool_execution,
    max_steps: form.max_steps ?? agent.max_steps,
    max_context_size: form.max_context_size ?? agent.max_context_size,
    max_turn_duration: form.max_turn_duration ?? agent.max_turn_duration,
    tools: form.tools ?? agent.tools,
    can_spawn: form.can_spawn ?? agent.can_spawn,
    mcp_servers: form.mcp_servers ?? agent.mcp_servers,
    confirm_before: form.confirm_before ?? agent.confirm_before,
  };

  // YAML copy
  function handleCopyYaml() {
    navigator.clipboard.writeText(agentToYaml(formAsAgent));
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }

  // Count helpers for section headers
  const toolCount = (form.tools ?? []).length;
  const mcpCount = (form.mcp_servers ?? []).length;
  const spawnCount = (form.can_spawn ?? []).length;
  const confirmCount = (form.confirm_before ?? []).length;

  return (
    <div
      className="relative border-l border-brand-shade3/15 bg-brand-dark-alt flex flex-col h-full flex-shrink-0"
      style={{ width: `${panelWidth}px` }}
    >
      {/* Resize handle */}
      <div
        onMouseDown={handleResizeMouseDown}
        className="absolute left-0 top-0 bottom-0 w-1 cursor-col-resize hover:bg-brand-accent/30 transition-colors z-10"
      />

      {/* Discard confirmation dialog */}
      {showDiscardConfirm && (
        <div className="absolute inset-0 z-50 flex items-center justify-center bg-brand-dark/70">
          <div className="bg-brand-dark-alt border border-brand-shade3/30 rounded-card p-4 mx-4 shadow-xl">
            <p className="text-sm text-brand-light font-medium mb-1">Unsaved changes</p>
            <p className="text-xs text-brand-shade3 mb-3">You have unsaved changes. Discard them?</p>
            <div className="flex gap-2">
              <button
                onClick={() => { setShowDiscardConfirm(false); onClose(); }}
                className="flex-1 py-1.5 bg-brand-accent text-brand-light rounded-card text-xs font-medium hover:bg-brand-accent-hover transition-colors"
              >
                Discard
              </button>
              <button
                onClick={() => setShowDiscardConfirm(false)}
                className="flex-1 py-1.5 bg-brand-dark border border-brand-shade3/30 text-brand-shade2 rounded-card text-xs hover:text-brand-light transition-colors"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
      {/* Header */}
      <div className="px-4 py-3 border-b border-brand-shade3/15 flex items-center justify-between">
        <div className="min-w-0 flex-1">
          <h3 className="text-sm font-semibold text-brand-light truncate">{agent.name}</h3>
          <p className="text-[11px] text-brand-shade3 mt-0.5">
            {form.lifecycle ?? agent.lifecycle} · {form.tool_execution ?? agent.tool_execution}
          </p>
        </div>
        <button
          onClick={handleClose}
          className="ml-2 p-1 text-brand-shade3 hover:text-brand-light transition-colors flex-shrink-0"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      {/* Tabs */}
      <div className="flex border-b border-brand-shade3/15">
        {(['details', 'chat', 'yaml'] as Tab[]).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`flex-1 py-2 text-xs font-medium capitalize transition-colors ${
              tab === t
                ? 'text-brand-accent border-b-2 border-brand-accent'
                : 'text-brand-shade3 hover:text-brand-light'
            }`}
          >
            {t}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div className="flex-1 overflow-hidden flex flex-col min-h-0">
        {tab === 'details' && (
          <div className="flex-1 overflow-y-auto p-4 space-y-3">
            {/* Model */}
            <Section title="Model" defaultOpen={true}>
              <select
                value={form.model_id ?? ''}
                onChange={(e) => updateField('model_id', e.target.value || undefined)}
                className="w-full px-2 py-1.5 bg-brand-dark border border-brand-shade3/30 rounded-card text-xs text-brand-light focus:outline-none focus:border-brand-accent transition-colors"
              >
                <option value="">Default model</option>
                {(models ?? []).map((m) => (
                  <option key={m.id} value={m.id}>{m.name}</option>
                ))}
              </select>
            </Section>

            {/* System Prompt */}
            <Section title="System Prompt" defaultOpen={true}>
              <textarea
                value={form.system_prompt ?? ''}
                onChange={(e) => updateField('system_prompt', e.target.value)}
                rows={10}
                className="w-full px-2 py-1.5 bg-brand-dark border border-brand-shade3/30 rounded-card text-[11px] text-brand-light font-mono focus:outline-none focus:border-brand-accent transition-colors resize-y leading-relaxed min-h-[150px]"
              />
            </Section>

            {/* Tools */}
            <Section title="Tools" defaultOpen={false} count={toolCount}>
              {!toolMetadata ? (
                <p className="text-[10px] text-brand-shade3">Loading...</p>
              ) : (
                <div className="space-y-2">
                  {/* Safe zone */}
                  {toolsByZone.safe.length > 0 && (
                    <div>
                      <p className={`text-[10px] font-medium ${ZONE_STYLE.safe.headerText} mb-1`}>Safe</p>
                      <div className="flex flex-wrap gap-1">
                        {toolsByZone.safe.map((tool) => (
                          <ToolChip
                            key={tool.name}
                            tool={tool}
                            zone="safe"
                            checked={(form.tools ?? []).includes(tool.name)}
                            onToggle={() => updateField('tools', toggleInArray(form.tools ?? [], tool.name))}
                          />
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Caution zone */}
                  {toolsByZone.caution.length > 0 && (
                    <div>
                      <p className={`text-[10px] font-medium ${ZONE_STYLE.caution.headerText} mb-1`}>Caution</p>
                      <div className="flex flex-wrap gap-1">
                        {toolsByZone.caution.map((tool) => (
                          <ToolChip
                            key={tool.name}
                            tool={tool}
                            zone="caution"
                            checked={(form.tools ?? []).includes(tool.name)}
                            onToggle={() => updateField('tools', toggleInArray(form.tools ?? [], tool.name))}
                          />
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Dangerous zone — collapsed by default */}
                  {toolsByZone.dangerous.length > 0 && (
                    <div className="border border-brand-accent/30 rounded-card overflow-hidden">
                      <button
                        type="button"
                        onClick={() => setDangerousExpanded(!dangerousExpanded)}
                        className="w-full flex items-center justify-between px-2 py-1 bg-brand-accent/10 text-brand-accent text-[10px] font-semibold hover:bg-brand-accent/15 transition-colors"
                      >
                        <span>Dangerous</span>
                        <span className="text-brand-accent/60 text-[9px]">{dangerousExpanded ? '\u25B2' : '\u25BC'}</span>
                      </button>
                      {dangerousExpanded && (
                        <div className="p-2">
                          <p className="text-[9px] text-brand-shade3 mb-1.5 leading-relaxed">
                            Filesystem & shell access. Only for trusted agents in isolated environments.
                          </p>
                          <div className="flex flex-wrap gap-1">
                            {toolsByZone.dangerous.map((tool) => (
                              <ToolChip
                                key={tool.name}
                                tool={tool}
                                zone="dangerous"
                                checked={(form.tools ?? []).includes(tool.name)}
                                onToggle={() => updateField('tools', toggleInArray(form.tools ?? [], tool.name))}
                              />
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              )}
            </Section>

            {/* MCP Servers */}
            <Section title="MCP Servers" defaultOpen={false} count={mcpCount}>
              {(mcpServers ?? []).length === 0 ? (
                <p className="text-[10px] text-brand-shade3">No MCP servers configured.</p>
              ) : (
                <div className="space-y-1.5">
                  <div className="flex flex-wrap gap-1">
                    {(mcpServers ?? []).map((s) => {
                      const checked = (form.mcp_servers ?? []).includes(s.name);
                      return (
                        <label
                          key={s.name}
                          className={`inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full border text-[10px] cursor-pointer transition-colors ${
                            checked
                              ? 'border-brand-accent bg-brand-accent/15 text-brand-accent'
                              : 'border-brand-shade3/20 bg-brand-dark text-brand-shade2 hover:text-brand-light'
                          }`}
                        >
                          <input
                            type="checkbox"
                            checked={checked}
                            onChange={() => updateField('mcp_servers', toggleInArray(form.mcp_servers ?? [], s.name))}
                            className="sr-only"
                          />
                          {s.name}
                        </label>
                      );
                    })}
                  </div>
                  {(form.mcp_servers ?? []).includes('admin-api') && (
                    <div className="flex items-start gap-1.5 px-2 py-1.5 bg-amber-500/10 border border-amber-500/20 rounded text-[10px] text-amber-400 leading-relaxed">
                      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="flex-shrink-0 mt-0.5">
                        <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
                        <line x1="12" y1="9" x2="12" y2="13" />
                        <line x1="12" y1="17" x2="12.01" y2="17" />
                      </svg>
                      <span>
                        <strong>Warning:</strong> admin-api grants full platform management access. Only assign to trusted internal agents.
                      </span>
                    </div>
                  )}
                </div>
              )}
            </Section>

            {/* Can Spawn */}
            <Section title="Can Spawn" defaultOpen={false} count={spawnCount}>
              {otherAgents.length === 0 ? (
                <p className="text-[10px] text-brand-shade3">No other agents.</p>
              ) : (
                <div className="flex flex-wrap gap-1">
                  {otherAgents.map((a) => {
                    const checked = (form.can_spawn ?? []).includes(a.name);
                    return (
                      <label
                        key={a.name}
                        className={`inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full border text-[10px] cursor-pointer transition-colors ${
                          checked
                            ? 'border-blue-500 bg-blue-500/15 text-blue-400'
                            : 'border-brand-shade3/20 bg-brand-dark text-brand-shade2 hover:text-brand-light'
                        }`}
                      >
                        <input
                          type="checkbox"
                          checked={checked}
                          onChange={() => updateField('can_spawn', toggleInArray(form.can_spawn ?? [], a.name))}
                          className="sr-only"
                        />
                        {a.name}
                      </label>
                    );
                  })}
                </div>
              )}
            </Section>

            {/* Confirm Before */}
            <Section title="Confirm Before" defaultOpen={false} count={confirmCount}>
              {(() => {
                // Available tools: those from selected MCP servers + built-in tools
                const selectedMcpTools = (toolMetadata ?? []).filter((tool) =>
                  (form.tools ?? []).includes(tool.name)
                );
                const availableTools = selectedMcpTools.length > 0 ? selectedMcpTools : (toolMetadata ?? []);
                const confirmed = form.confirm_before ?? [];

                return (
                  <div className="space-y-1.5">
                    {availableTools.length > 0 && (
                      <div className="flex flex-wrap gap-1">
                        {availableTools.map((tool) => {
                          const checked = confirmed.includes(tool.name);
                          return (
                            <label
                              key={tool.name}
                              className={`inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full border text-[10px] cursor-pointer transition-colors ${
                                checked
                                  ? 'border-amber-500 bg-amber-500/15 text-amber-400'
                                  : 'border-brand-shade3/20 bg-brand-dark text-brand-shade2 hover:text-brand-light'
                              }`}
                            >
                              <input
                                type="checkbox"
                                checked={checked}
                                onChange={() => updateField('confirm_before', toggleInArray(confirmed, tool.name))}
                                className="sr-only"
                              />
                              {tool.name}
                            </label>
                          );
                        })}
                      </div>
                    )}
                    {availableTools.length === 0 && (
                      <div className="flex gap-1 mb-1.5">
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
                          className="flex-1 px-2 py-1 bg-brand-dark border border-brand-shade3/30 rounded text-[10px] text-brand-light placeholder-brand-shade3 focus:outline-none focus:border-brand-accent transition-colors"
                        />
                        <button
                          type="button"
                          onClick={addConfirmBefore}
                          className="px-2 py-1 bg-brand-dark border border-brand-shade3/30 rounded text-[10px] text-brand-shade2 hover:text-brand-light transition-colors"
                        >
                          Add
                        </button>
                      </div>
                    )}
                    {confirmed.filter((t) => !availableTools.some((tool) => tool.name === t)).map((t) => (
                      <span
                        key={t}
                        className="inline-flex items-center gap-0.5 px-1.5 py-0.5 bg-amber-500/10 border border-amber-500/20 rounded-full text-[10px] text-amber-400"
                      >
                        {t}
                        <button
                          type="button"
                          onClick={() => updateField('confirm_before', confirmed.filter((x) => x !== t))}
                          className="text-amber-500 hover:text-amber-300 ml-0.5"
                        >
                          x
                        </button>
                      </span>
                    ))}
                  </div>
                );
              })()}
            </Section>


            {/* Lifecycle */}
            <Section title="Lifecycle" defaultOpen={false}>
              <select
                value={form.lifecycle ?? 'persistent'}
                onChange={(e) => updateField('lifecycle', e.target.value)}
                className="w-full px-2 py-1.5 bg-brand-dark border border-brand-shade3/30 rounded-card text-xs text-brand-light focus:outline-none focus:border-brand-accent transition-colors"
              >
                <option value="persistent">Persistent</option>
                <option value="spawn">Spawn</option>
              </select>
            </Section>

            {/* Tool Execution */}
            <Section title="Tool Execution" defaultOpen={false}>
              <select
                value={form.tool_execution ?? 'sequential'}
                onChange={(e) => updateField('tool_execution', e.target.value)}
                className="w-full px-2 py-1.5 bg-brand-dark border border-brand-shade3/30 rounded-card text-xs text-brand-light focus:outline-none focus:border-brand-accent transition-colors"
              >
                <option value="sequential">Sequential</option>
                <option value="parallel">Parallel</option>
              </select>
            </Section>

            {/* Max Steps + Max Context Size — side by side */}
            <Section title="Limits" defaultOpen={false}>
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label className="block text-[9px] text-brand-shade3 mb-0.5">Max Steps</label>
                  <input
                    type="number"
                    value={form.max_steps ?? 50}
                    onChange={(e) => updateField('max_steps', Number(e.target.value))}
                    min={1}
                    max={500}
                    className="w-full px-2 py-1 bg-brand-dark border border-brand-shade3/30 rounded text-xs text-brand-light focus:outline-none focus:border-brand-accent transition-colors"
                  />
                </div>
                <div>
                  <label className="block text-[9px] text-brand-shade3 mb-0.5">Context Size</label>
                  <input
                    type="number"
                    value={form.max_context_size ?? 16000}
                    onChange={(e) => updateField('max_context_size', Number(e.target.value))}
                    min={1000}
                    max={200000}
                    step={1000}
                    className="w-full px-2 py-1 bg-brand-dark border border-brand-shade3/30 rounded text-xs text-brand-light focus:outline-none focus:border-brand-accent transition-colors"
                  />
                </div>
                <div>
                  <label className="block text-[9px] text-brand-shade3 mb-0.5">Turn Duration (s)</label>
                  <input
                    type="number"
                    value={form.max_turn_duration ?? 120}
                    onChange={(e) => updateField('max_turn_duration', Number(e.target.value))}
                    min={30}
                    max={600}
                    step={10}
                    className="w-full px-2 py-1 bg-brand-dark border border-brand-shade3/30 rounded text-xs text-brand-light focus:outline-none focus:border-brand-accent transition-colors"
                  />
                </div>
              </div>
            </Section>

            {saveError && (
              <p className="text-xs text-red-400">{saveError}</p>
            )}
          </div>
        )}

        {tab === 'chat' && (
          <BuilderChat agentName={agent.name} />
        )}

        {tab === 'yaml' && (
          <div className="flex-1 overflow-y-auto p-4 flex flex-col gap-2 min-h-0">
            <div className="flex justify-end">
              <button
                onClick={handleCopyYaml}
                className="text-xs text-brand-shade3 hover:text-brand-light transition-colors"
              >
                {copied ? 'Copied' : 'Copy'}
              </button>
            </div>
            <pre className="flex-1 text-[11px] font-mono text-brand-shade2 bg-brand-dark border border-brand-shade3/20 rounded-card p-3 overflow-auto whitespace-pre leading-relaxed">
              {agentToYaml(formAsAgent)}
            </pre>
          </div>
        )}
      </div>

      {/* Footer actions */}
      <div className="px-4 py-3 border-t border-brand-shade3/15 flex gap-2">
        {tab === 'details' && (
          <button
            onClick={handleSave}
            disabled={saving || !isDirty}
            className="flex-1 py-1.5 bg-brand-accent text-brand-light rounded-card text-xs font-medium hover:bg-brand-accent-hover disabled:opacity-40 transition-colors"
          >
            {saving ? 'Saving...' : 'Save'}
          </button>
        )}
        <button
          onClick={() => navigate(`/agents/${agent.name}`)}
          className="flex-1 py-1.5 bg-brand-dark border border-brand-shade3/30 text-brand-shade2 rounded-card text-xs hover:text-brand-light hover:border-brand-shade3 transition-colors flex items-center justify-center gap-1"
        >
          Open Full Editor
          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M18 13v6a2 2 0 01-2 2H5a2 2 0 01-2-2V8a2 2 0 012-2h6" />
            <polyline points="15 3 21 3 21 9" />
            <line x1="10" y1="14" x2="21" y2="3" />
          </svg>
        </button>
        <button
          onClick={() => onDelete(agent.name)}
          className="py-1.5 px-3 border border-red-500/20 text-red-400/70 rounded-card text-xs hover:text-red-400 hover:border-red-500/40 transition-colors"
          title="Delete agent"
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <polyline points="3 6 5 6 21 6" />
            <path d="M19 6l-1 14a2 2 0 01-2 2H8a2 2 0 01-2-2L5 6" />
            <path d="M10 11v6M14 11v6" />
            <path d="M9 6V4a1 1 0 011-1h4a1 1 0 011 1v2" />
          </svg>
        </button>
      </div>
    </div>
  );
}
