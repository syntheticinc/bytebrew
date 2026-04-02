import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../../api/client';
import { useApi } from '../../hooks/useApi';
import type { AgentDetail, Model, CreateAgentRequest } from '../../types';
import BuilderChat from './BuilderChat';

interface BuilderSidePanelProps {
  agent: AgentDetail;
  onClose: () => void;
  onSaved: (updated: AgentDetail) => void;
  onDelete: (name: string) => void;
}

type Tab = 'details' | 'chat' | 'yaml';

function agentToYaml(agent: AgentDetail): string {
  const lines: string[] = [];
  lines.push(`name: ${agent.name}`);
  if (agent.model_id !== undefined) lines.push(`model_id: ${agent.model_id}`);
  lines.push(`lifecycle: ${agent.lifecycle}`);
  lines.push(`tool_execution: ${agent.tool_execution}`);
  lines.push(`max_steps: ${agent.max_steps}`);
  lines.push(`max_context_size: ${agent.max_context_size}`);

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

export default function BuilderSidePanel({ agent, onClose, onSaved, onDelete }: BuilderSidePanelProps) {
  const navigate = useNavigate();
  const [tab, setTab] = useState<Tab>('details');
  const [form, setForm] = useState<Partial<CreateAgentRequest>>({});
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState('');
  const [copied, setCopied] = useState(false);

  const { data: models } = useApi<Model[]>(() => api.listModels());

  useEffect(() => {
    setForm({
      model_id: agent.model_id,
      system_prompt: agent.system_prompt,
    });
    setSaveError('');
    setTab('details');
  }, [agent.name]);

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

  function handleCopyYaml() {
    navigator.clipboard.writeText(agentToYaml(agent));
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }

  const isDirty =
    form.model_id !== agent.model_id ||
    form.system_prompt !== agent.system_prompt;

  return (
    <div className="w-80 border-l border-brand-shade3/15 bg-brand-dark-alt flex flex-col h-full flex-shrink-0">
      {/* Header */}
      <div className="px-4 py-3 border-b border-brand-shade3/15 flex items-center justify-between">
        <div className="min-w-0 flex-1">
          <h3 className="text-sm font-semibold text-brand-light truncate">{agent.name}</h3>
          <p className="text-[11px] text-brand-shade3 mt-0.5">{agent.lifecycle} · {agent.tool_execution}</p>
        </div>
        <button
          onClick={onClose}
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
          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            {/* Model */}
            <div>
              <label className="block text-xs font-medium text-brand-shade3 mb-1 uppercase tracking-wide">Model</label>
              <select
                value={form.model_id ?? ''}
                onChange={(e) => setForm((p) => ({ ...p, model_id: e.target.value ? Number(e.target.value) : undefined }))}
                className="w-full px-2.5 py-1.5 bg-brand-dark border border-brand-shade3/30 rounded-card text-sm text-brand-light focus:outline-none focus:border-brand-accent transition-colors"
              >
                <option value="">Default model</option>
                {(models ?? []).map((m) => (
                  <option key={m.id} value={m.id}>{m.name}</option>
                ))}
              </select>
            </div>

            {/* System Prompt */}
            <div>
              <label className="block text-xs font-medium text-brand-shade3 mb-1 uppercase tracking-wide">System Prompt</label>
              <textarea
                value={form.system_prompt ?? ''}
                onChange={(e) => setForm((p) => ({ ...p, system_prompt: e.target.value }))}
                rows={8}
                className="w-full px-2.5 py-1.5 bg-brand-dark border border-brand-shade3/30 rounded-card text-xs text-brand-light font-mono focus:outline-none focus:border-brand-accent transition-colors resize-none"
              />
            </div>

            {/* Read-only info */}
            <div className="space-y-2">
              {agent.tools?.length > 0 && (
                <div>
                  <p className="text-xs font-medium text-brand-shade3 uppercase tracking-wide mb-1">Tools</p>
                  <div className="flex flex-wrap gap-1">
                    {agent.tools.map((t) => (
                      <span key={t} className="px-1.5 py-0.5 bg-brand-dark border border-brand-shade3/20 rounded text-[10px] text-brand-shade2">
                        {t}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {agent.can_spawn?.length > 0 && (
                <div>
                  <p className="text-xs font-medium text-brand-shade3 uppercase tracking-wide mb-1">Can Spawn</p>
                  <div className="flex flex-wrap gap-1">
                    {agent.can_spawn.map((a) => (
                      <span key={a} className="px-1.5 py-0.5 bg-blue-500/10 border border-blue-500/20 rounded text-[10px] text-blue-400">
                        {a}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {agent.confirm_before?.length > 0 && (
                <div>
                  <p className="text-xs font-medium text-brand-shade3 uppercase tracking-wide mb-1">Confirm Before</p>
                  <div className="flex flex-wrap gap-1">
                    {agent.confirm_before.map((t) => (
                      <span key={t} className="px-1.5 py-0.5 bg-amber-500/10 border border-amber-500/20 rounded text-[10px] text-amber-400">
                        {t}
                      </span>
                    ))}
                  </div>
                </div>
              )}
            </div>

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
                {copied ? '✓ Copied' : 'Copy'}
              </button>
            </div>
            <pre className="flex-1 text-[11px] font-mono text-brand-shade2 bg-brand-dark border border-brand-shade3/20 rounded-card p-3 overflow-auto whitespace-pre leading-relaxed">
              {agentToYaml(agent)}
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
            {saving ? 'Saving…' : 'Save'}
          </button>
        )}
        <button
          onClick={() => navigate(`/agents/${agent.name}/edit`)}
          className="flex-1 py-1.5 bg-brand-dark border border-brand-shade3/30 text-brand-shade2 rounded-card text-xs hover:text-brand-light hover:border-brand-shade3 transition-colors"
        >
          Full Edit
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
