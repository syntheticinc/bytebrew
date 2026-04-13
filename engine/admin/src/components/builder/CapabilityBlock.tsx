import React, { useState } from 'react';
import type { CapabilityConfig } from '../../types';
import { CAPABILITY_META } from '../../types';

interface CapabilityBlockProps {
  capability: CapabilityConfig;
  onChange: (updated: CapabilityConfig) => void;
  onRemove: () => void;
  agentName?: string;
  models?: { id: string; name: string; model_name: string }[];
}

const inputCls =
  'w-full bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light px-3 py-2 focus:outline-none focus:border-brand-accent placeholder-brand-shade3';

const selectCls =
  'w-full bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light px-3 py-2 focus:outline-none focus:border-brand-accent';

const labelCls = 'block text-xs text-brand-shade3 mb-1 font-mono';
const descCls = 'text-xs text-brand-shade3 mb-3';
const hintCls = 'text-[11px] text-brand-shade3/70 mt-1';

// ---------------------------------------------------------------------------
// B.1: Capability SVG icons
// ---------------------------------------------------------------------------

/**
 * Returns default config for a capability type so that pre-populated UI defaults
 * are persisted to API on first Save (BUG-003: recovery rules were lost).
 */
export function getCapabilityDefaultConfig(type: string): Record<string, unknown> {
  switch (type) {
    case 'recovery':
      return {
        rules: [
          { failure_type: 'mcp_connection_failed', action: 'retry', retry_count: 1, backoff: 'fixed', fallback_model: '' },
          { failure_type: 'model_unavailable', action: 'fallback', retry_count: 2, backoff: 'exponential', fallback_model: '' },
          { failure_type: 'tool_timeout', action: 'retry', retry_count: 1, backoff: 'fixed', fallback_model: '' },
          { failure_type: 'tool_auth_failure', action: 'block', retry_count: 0, backoff: 'fixed', fallback_model: '' },
          { failure_type: 'context_overflow', action: 'retry', retry_count: 1, backoff: 'fixed', fallback_model: '' },
        ],
      };
    default:
      return {};
  }
}

export function capabilityIcon(name: string): React.ReactElement {
  const props = { width: 18, height: 18, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 1.5, strokeLinecap: 'round' as const, strokeLinejoin: 'round' as const };
  switch (name) {
    case 'brain':
      return <svg {...props}><circle cx="12" cy="12" r="9" /><path d="M9 9c0-1 1-2 3-2s3 1 3 2-1 2-3 2v2" /><circle cx="12" cy="17" r=".5" fill="currentColor" /></svg>;
    case 'book-open':
      return <svg {...props}><path d="M2 3h6a4 4 0 014 4v14a3 3 0 00-3-3H2z" /><path d="M22 3h-6a4 4 0 00-4 4v14a3 3 0 013-3h7z" /></svg>;
    case 'shield-check':
      return <svg {...props}><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" /><path d="M9 12l2 2 4-4" /></svg>;
    case 'file-json':
      return <svg {...props}><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" /><path d="M14 2v6h6" /><path d="M10 13l-1 3 1 3" /><path d="M14 13l1 3-1 3" /></svg>;
    case 'arrow-up-right':
      return <svg {...props}><line x1="7" y1="17" x2="17" y2="7" /><polyline points="7 7 17 7 17 17" /></svg>;
    case 'refresh-cw':
      return <svg {...props}><polyline points="23 4 23 10 17 10" /><polyline points="1 20 1 14 7 14" /><path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15" /></svg>;
    case 'settings':
      return <svg {...props}><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 012.83-2.83l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z" /></svg>;
    default:
      return <span className="text-[10px] font-semibold">{name}</span>;
  }
}


function setKey(cap: CapabilityConfig, key: string, value: unknown): CapabilityConfig {
  return { ...cap, config: { ...cap.config, [key]: value } };
}

function getKey<T>(cap: CapabilityConfig, key: string, fallback: T): T {
  const v = cap.config[key];
  return v !== undefined ? (v as T) : fallback;
}

// ---------------------------------------------------------------------------
// Per-type config panels
// ---------------------------------------------------------------------------

type PanelProps = { cap: CapabilityConfig; onChange: (u: CapabilityConfig) => void; agentName?: string; models?: { id: string; name: string; model_name: string }[] };

function MemoryConfig({ cap, onChange }: PanelProps) {
  const unlimitedRetention = getKey(cap, 'unlimited_retention', false) as boolean;
  const unlimitedEntries = getKey(cap, 'unlimited_entries', false) as boolean;

  return (
    <div className="space-y-3">
      <p className={descCls}>Agent remembers facts across sessions within this schema. Recalled automatically at session start, stored during conversation. Users can also ask the agent to remember things explicitly.</p>
      <div className="bg-brand-dark rounded-card px-3 py-2 space-y-1">
        <span className="text-[11px] text-brand-shade2 font-mono">Scope: per-schema, cross-session</span>
        <p className={hintCls}>Memory is isolated per schema and persists between sessions. Support Schema and Sales Schema have separate memory spaces.</p>
      </div>
      <div className="bg-brand-dark rounded-card px-3 py-2 space-y-1">
        <span className="text-[11px] text-brand-shade2 font-mono">Auto-included tools:</span>
        <div className="flex gap-2 mt-1">
          <span className="text-[10px] px-2 py-0.5 bg-brand-dark-alt border border-brand-shade3/20 rounded-card text-brand-shade2">memory_recall</span>
          <span className="text-[10px] px-2 py-0.5 bg-brand-dark-alt border border-brand-shade3/20 rounded-card text-brand-shade2">memory_store</span>
        </div>
        <p className={hintCls}>These tools are automatically added to agent runtime when Memory is enabled</p>
      </div>
      <div>
        <label className={labelCls}>Retention</label>
        <label className="flex items-center gap-2 text-sm text-brand-shade2 cursor-pointer select-none mb-2">
          <input type="checkbox" className="accent-brand-accent" data-testid="memory-unlimited-retention" checked={unlimitedRetention} onChange={(e) => onChange(setKey(cap, 'unlimited_retention', e.target.checked))} />
          Unlimited
        </label>
        {!unlimitedRetention && (
          <div className="flex items-center gap-2">
            <input type="number" className={inputCls} data-testid="memory-retention-days" min={1} max={365} value={getKey(cap, 'retention_days', 30) as number} onChange={(e) => onChange(setKey(cap, 'retention_days', Number(e.target.value)))} />
            <span className="text-xs text-brand-shade3 shrink-0">days</span>
          </div>
        )}
        <p className={hintCls}>{unlimitedRetention ? 'Memory entries are kept indefinitely' : 'Entries older than this are automatically deleted'}</p>
      </div>
      <div>
        <label className={labelCls}>Max entries</label>
        <label className="flex items-center gap-2 text-sm text-brand-shade2 cursor-pointer select-none mb-2">
          <input type="checkbox" className="accent-brand-accent" data-testid="memory-unlimited-entries" checked={unlimitedEntries} onChange={(e) => onChange(setKey(cap, 'unlimited_entries', e.target.checked))} />
          Unlimited
        </label>
        {!unlimitedEntries && (
          <input type="number" className={inputCls} data-testid="memory-max-entries" min={1} value={getKey(cap, 'max_entries', 500) as number} onChange={(e) => onChange(setKey(cap, 'max_entries', Number(e.target.value)))} />
        )}
        <p className={hintCls}>{unlimitedEntries ? 'No limit on stored entries (bounded by schema storage quota)' : 'Oldest entries removed first (FIFO) when limit reached'}</p>
      </div>
    </div>
  );
}

function KnowledgeConfig({ cap, onChange }: PanelProps) {
  return (
    <div className="space-y-3">
      <p className={descCls}>RAG: agent searches linked knowledge bases before answering</p>

      <div className="bg-brand-dark rounded-card px-3 py-2 space-y-1">
        <span className="text-[11px] text-brand-shade2 font-mono">Auto-included tools:</span>
        <div className="flex gap-2 mt-1">
          <span className="text-[10px] px-2 py-0.5 bg-brand-dark-alt border border-brand-shade3/20 rounded-card text-brand-shade2">knowledge_search</span>
        </div>
        <p className={hintCls}>Automatically available to agent when Knowledge is enabled</p>
      </div>

      <div className="bg-brand-dark-alt border border-brand-shade3/20 rounded-card px-3 py-3 space-y-2">
        <div className="flex items-start gap-2">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" className="text-brand-accent shrink-0 mt-0.5"><path d="M2 3h6a4 4 0 014 4v14a3 3 0 00-3-3H2z" /><path d="M22 3h-6a4 4 0 00-4 4v14a3 3 0 013-3h7z" /></svg>
          <div>
            <p className="text-xs text-brand-light font-medium">Knowledge Bases</p>
            <p className="text-[10px] text-brand-shade3 mt-1">
              Documents and files are managed through Knowledge Bases. Link one or more KBs to this agent on the Knowledge page.
            </p>
          </div>
        </div>
        <a
          href={import.meta.env.BASE_URL + 'knowledge'}
          className="inline-flex items-center gap-1 px-3 py-1.5 bg-brand-accent text-brand-light rounded-btn text-xs font-medium hover:bg-brand-accent-hover transition-colors"
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 13v6a2 2 0 01-2 2H5a2 2 0 01-2-2V8a2 2 0 012-2h6" /><polyline points="15 3 21 3 21 9" /><line x1="10" y1="14" x2="21" y2="3" /></svg>
          Manage Knowledge Bases
        </a>
      </div>

      <div>
        <label className={labelCls}>Top-K</label>
        <input type="number" className={inputCls} data-testid="knowledge-top-k" min={1} max={20} value={getKey(cap, 'top_k', 5) as number} onChange={(e) => onChange(setKey(cap, 'top_k', Number(e.target.value)))} />
        <p className={hintCls}>Number of most relevant document chunks retrieved per query</p>
      </div>
      <div>
        <label className={labelCls}>Similarity threshold</label>
        <input type="number" className={inputCls} data-testid="knowledge-threshold" min={0} max={1} step={0.05} value={getKey(cap, 'similarity_threshold', 0.75) as number} onChange={(e) => onChange(setKey(cap, 'similarity_threshold', Number(e.target.value)))} />
        <p className={hintCls}>0 = return all chunks, 1 = exact match only. Recommended: 0.7-0.85</p>
      </div>
    </div>
  );
}

function GuardrailConfig({ cap, onChange }: PanelProps) {
  const mode = getKey(cap, 'mode', 'json_schema') as string;
  const modeHints: Record<string, string> = {
    json_schema: 'Agent output must match this JSON Schema',
    llm_check: 'A secondary LLM evaluates the output quality',
    webhook: 'Output sent to webhook for external validation',
  };
  const configLabels: Record<string, string> = {
    json_schema: 'Schema',
    llm_check: 'Judge LLM Prompt',
    webhook: 'Webhook URL',
  };
  const placeholders: Record<string, string> = {
    json_schema: '{"type":"object","required":["answer"]}',
    llm_check: 'Judge LLM Prompt \u2014 a separate LLM evaluates the agent\'s response',
    webhook: 'https://validate.example.com/check',
  };
  // Map UI mode to the backend config key that the guardrail pipeline reads
  const configKeyForMode: Record<string, string> = {
    json_schema: 'json_schema',
    llm_check: 'judge_prompt',
    webhook: 'webhook_url',
  };
  const configKey = configKeyForMode[mode] ?? 'json_schema';
  return (
    <div className="space-y-3">
      <p className={descCls}>Validates agent output before sending to user</p>
      <div>
        <label className={labelCls}>Mode</label>
        <select className={selectCls} data-testid="guardrail-mode" value={mode} onChange={(e) => onChange(setKey(cap, 'mode', e.target.value))}>
          <option value="json_schema">JSON Schema</option>
          <option value="llm_check">LLM Check</option>
          <option value="webhook">Webhook</option>
        </select>
        <p className={hintCls}>{modeHints[mode]}</p>
      </div>
      <div>
        <label className={labelCls}>{configLabels[mode] ?? 'Config'}</label>
        <textarea className={`${inputCls} font-mono resize-y`} data-testid="guardrail-config" rows={4} placeholder={placeholders[mode]} value={getKey(cap, configKey, '') as string} onChange={(e) => onChange(setKey(cap, configKey, e.target.value))} />
      </div>
      {mode === 'webhook' && (
        <div className="bg-brand-dark rounded-card px-3 py-2">
          <p className="text-[10px] text-brand-shade3 font-mono mb-1">Contract</p>
          <pre className="text-[10px] text-brand-shade2 font-mono leading-relaxed whitespace-pre">
{`Request:  POST { event, agent, session_id, response }
Response: { pass: boolean, reason?: string }`}
          </pre>
        </div>
      )}
      <div>
        <label className={labelCls}>On failure</label>
        <select className={selectCls} data-testid="guardrail-on-failure" value={getKey(cap, 'on_failure', 'retry') as string} onChange={(e) => onChange(setKey(cap, 'on_failure', e.target.value))}>
          <option value="retry">Retry (agent regenerates response, max 3 attempts)</option>
          <option value="error">Error (return validation error to user, no response)</option>
          <option value="fallback">Fallback (use simpler prompt for safe response)</option>
        </select>
        <p className={hintCls}>What happens when validation fails</p>
      </div>
      <label className="flex items-center gap-2 text-sm text-brand-shade2 cursor-pointer select-none">
        <input type="checkbox" className="accent-brand-accent" checked={getKey(cap, 'strict', false) as boolean} onChange={(e) => onChange(setKey(cap, 'strict', e.target.checked))} />
        Strict mode
      </label>
      <p className="text-[11px] text-brand-shade3/70 ml-6">When enabled, completely blocks output that fails validation</p>
    </div>
  );
}

function EscalationConfig({ cap, onChange }: PanelProps) {
  const action = getKey(cap, 'action', 'transfer_to_user') as string;
  const actionHints: Record<string, string> = {
    transfer_to_user: 'Session marked as needing human, agent stops responding',
    notify_webhook: 'Webhook called with session context, agent continues',
  };

  return (
    <div className="space-y-3">
      <p className={descCls}>Defines when and how agent escalates to human or another system</p>
      <div className="bg-brand-dark rounded-card px-3 py-2 space-y-1">
        <span className="text-[11px] text-brand-shade2 font-mono">Auto-included tools:</span>
        <div className="flex gap-2 mt-1">
          <span className="text-[10px] px-2 py-0.5 bg-brand-dark-alt border border-brand-shade3/20 rounded-card text-brand-shade2">escalate</span>
        </div>
        <p className={hintCls}>Agent can trigger escalation when conditions are met</p>
      </div>
      <div>
        <label className={labelCls}>Action</label>
        <select className={selectCls} data-testid="escalation-action" value={action} onChange={(e) => onChange(setKey(cap, 'action', e.target.value))}>
          <option value="transfer_to_user">Transfer to User</option>
          <option value="notify_webhook">Notify</option>
        </select>
        <p className={hintCls}>{actionHints[action]}</p>
      </div>
      {action === 'notify_webhook' && (
        <div>
          <label className={labelCls}>Webhook URL</label>
          <input className={inputCls} placeholder="https://hooks.example.com/escalate" value={getKey(cap, 'webhook_url', '') as string} onChange={(e) => onChange(setKey(cap, 'webhook_url', e.target.value))} />
          <p className={hintCls}>Called when escalation triggers. Receives full session context.</p>
        </div>
      )}
    </div>
  );
}

type RecoveryRule = { failure_type: string; action: string; retry_count: number; backoff: string; fallback_model: string };
const defaultRules: RecoveryRule[] = [
  { failure_type: 'mcp_connection_failed', action: 'retry', retry_count: 1, backoff: 'fixed', fallback_model: '' },
  { failure_type: 'model_unavailable', action: 'fallback', retry_count: 2, backoff: 'exponential', fallback_model: '' },
  { failure_type: 'tool_timeout', action: 'retry', retry_count: 1, backoff: 'fixed', fallback_model: '' },
  { failure_type: 'tool_auth_failure', action: 'block', retry_count: 0, backoff: 'fixed', fallback_model: '' },
  { failure_type: 'context_overflow', action: 'retry', retry_count: 1, backoff: 'fixed', fallback_model: '' },
];
const failureMeta: Record<string, [string, string]> = {
  mcp_connection_failed: ['MCP Connection Failed', 'MCP server unreachable -- reconnect, then degrade'],
  model_unavailable: ['Model Unavailable', 'LLM API returns error -- retry with backoff, then fallback'],
  tool_timeout: ['Tool Timeout', 'Tool call exceeds timeout -- retry if idempotent'],
  tool_auth_failure: ['Tool Auth Failure', 'Invalid credentials -- no retry, inform user'],
  context_overflow: ['Context Overflow', 'Context too large -- auto-compact and retry'],
};

function RecoveryConfig({ cap, onChange }: PanelProps) {
  const rules = getKey<RecoveryRule[]>(cap, 'rules', defaultRules);

  function updateRule(i: number, field: keyof RecoveryRule, value: string | number) {
    const next = rules.map((r, j) => (j === i ? { ...r, [field]: value } : r));
    onChange(setKey(cap, 'rules', next));
  }

  return (
    <div className="space-y-3">
      <p className={descCls}>Defines automatic recovery behavior per failure type. Each failure type has its own recovery strategy.</p>
      <div className="bg-brand-dark rounded-card px-3 py-2 space-y-1">
        <p className="text-[11px] text-brand-shade2 font-mono">Degrade mode applies until end of current session</p>
        <p className="text-[11px] text-brand-shade3/70">1 automatic recovery attempt before escalation</p>
      </div>
      {rules.map((rule, i) => (
        <div key={rule.failure_type} className="bg-brand-dark rounded-card p-3 space-y-2">
          <span className="text-xs font-semibold text-brand-shade2">{failureMeta[rule.failure_type]?.[0] ?? rule.failure_type}</span>
          <p className="text-[11px] text-brand-shade3/70">{failureMeta[rule.failure_type]?.[1]}</p>
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className={labelCls}>Action</label>
              <select className={selectCls} value={rule.action} onChange={(e) => updateRule(i, 'action', e.target.value)}>
                <option value="retry">Retry (attempt the operation again)</option>
                <option value="fallback">Fallback (switch to backup model)</option>
                <option value="degrade">Degrade (continue without the failed component)</option>
                <option value="block">Block (stop agent, return error to user)</option>
              </select>
            </div>
            {(rule.action === 'retry' || rule.action === 'fallback') && (
              <div>
                <label className={labelCls}>Retries</label>
                <input type="number" className={inputCls} min={0} max={10} value={rule.retry_count} onChange={(e) => updateRule(i, 'retry_count', Number(e.target.value))} />
              </div>
            )}
          </div>
          {(rule.action === 'retry' || rule.action === 'fallback') && (
            <div>
              <label className={labelCls}>Backoff</label>
              <select className={selectCls} value={rule.backoff} onChange={(e) => updateRule(i, 'backoff', e.target.value)}>
                <option value="fixed">Fixed</option>
                <option value="exponential">Exponential</option>
              </select>
            </div>
          )}
          {/* fallback_model: hidden until runtime model-switching is implemented */}
        </div>
      ))}
    </div>
  );
}

// Nested format matching backend policyRuleRaw struct
type PolicyRule = {
  enabled: boolean;
  condition: { type: string; pattern?: string; start?: string; end?: string };
  action: { type: string; message?: string; webhook_url?: string; headers?: Record<string, string> };
};

const conditionHints: Record<string, string> = {
  before_tool_call: 'Runs before any tool is invoked. Can block or modify the call',
  after_tool_call: 'Runs after a tool returns its result. Can log or validate output',
  tool_matches: 'Triggers only when the tool name matches a glob pattern (e.g. delete_*)',
  time_range: 'Triggers only during specified hours (24h format, UTC)',
  error_occurred: 'Triggers on any agent error: tool failures, model errors, timeouts',
};
const actionHintsPolicy: Record<string, string> = {
  log_to_webhook: 'Sends event details (tool name, params, result) to a webhook URL for logging',
  block: 'Blocks the tool call from executing. Agent receives "tool blocked by policy" message',
  notify: 'Sends real-time notification to webhook URL. Agent continues working',
  inject_header: 'Adds a custom HTTP header to outgoing MCP tool requests (e.g. auth tokens)',
  write_audit: 'Records policy trigger event in admin Audit Log page with full context',
};

function PoliciesConfig({ cap, onChange }: PanelProps) {
  const rules = getKey<PolicyRule[]>(cap, 'rules', []);

  function setRules(next: PolicyRule[]) { onChange(setKey(cap, 'rules', next)); }

  function updateCondition(i: number, field: string, value: string) {
    const next = rules.map((r, j) => j === i ? { ...r, condition: { ...r.condition, [field]: value } } : r);
    setRules(next);
  }

  function updateAction(i: number, field: string, value: string) {
    const next = rules.map((r, j) => j === i ? { ...r, action: { ...r.action, [field]: value } } : r);
    setRules(next);
  }

  function updateHeader(i: number, name: string, value: string) {
    const next = rules.map((r, j) => j === i ? { ...r, action: { ...r.action, headers: { [name]: value } } } : r);
    setRules(next);
  }

  // Helper: extract header name/value from headers map for display
  const getHeader = (rule: PolicyRule): [string, string] => {
    const entries = Object.entries(rule.action.headers ?? {});
    return entries.length > 0 ? [entries[0]![0], entries[0]![1]] : ['', ''];
  };

  // Helper: format time range from start/end for display
  const getTimeRange = (rule: PolicyRule): string => {
    const s = rule.condition.start ?? '';
    const e = rule.condition.end ?? '';
    if (s && e) return `${s}-${e}`;
    return s || e || '';
  };

  const parseTimeRange = (i: number, val: string) => {
    const parts = val.split('-');
    const next = rules.map((r, j) => j === i ? { ...r, condition: { ...r.condition, start: parts[0]?.trim() ?? '', end: parts[1]?.trim() ?? '' } } : r);
    setRules(next);
  };

  return (
    <div className="space-y-3">
      <p className={descCls}>Visual rules: When [condition] occurs -- Do [action]</p>
      {rules.map((rule, i) => (
        <div key={rule.condition.type + '-' + rule.action.type + '-' + i} className="space-y-2">
          <div className="flex gap-2 items-center">
            <select className={`${selectCls} flex-1`} value={rule.condition.type} onChange={(e) => updateCondition(i, 'type', e.target.value)}>
              <option value="before_tool_call">Before tool call</option>
              <option value="after_tool_call">After tool call</option>
              <option value="tool_matches">Tool matches</option>
              <option value="time_range">Time range</option>
              <option value="error_occurred">Error occurred</option>
            </select>
            <select className={`${selectCls} flex-1`} value={rule.action.type} onChange={(e) => updateAction(i, 'type', e.target.value)}>
              <option value="log_to_webhook">Log to webhook</option>
              <option value="block">Block</option>
              <option value="notify">Notify</option>
              <option value="inject_header">Inject header</option>
              <option value="write_audit">Write audit</option>
            </select>
            <button type="button" onClick={() => setRules(rules.filter((_, j) => j !== i))} className="text-brand-shade3 hover:text-brand-accent text-lg leading-none flex-shrink-0">x</button>
          </div>
          <p className="text-[11px] text-brand-shade3/70">{conditionHints[rule.condition.type]}</p>
          <p className="text-[11px] text-brand-shade3/70">{actionHintsPolicy[rule.action.type]}</p>
          {rule.condition.type === 'tool_matches' && (
            <div>
              <label className={labelCls}>Tool pattern (glob)</label>
              <input className={inputCls} placeholder="delete_*, send_email, admin_*" value={rule.condition.pattern ?? ''} onChange={(e) => updateCondition(i, 'pattern', e.target.value)} />
              <p className={hintCls}>Use * as wildcard. Matches tool names like delete_user, delete_cache</p>
            </div>
          )}
          {rule.condition.type === 'time_range' && (
            <div>
              <label className={labelCls}>Time window (UTC, 24h)</label>
              <input className={inputCls} placeholder="09:00-17:00" value={getTimeRange(rule)} onChange={(e) => parseTimeRange(i, e.target.value)} />
              <p className={hintCls}>Format: HH:MM-HH:MM. Example: 09:00-17:00 for working hours only</p>
            </div>
          )}
          {(rule.action.type === 'log_to_webhook' || rule.action.type === 'notify') && (
            <div>
              <label className={labelCls}>Webhook URL</label>
              <input className={inputCls} placeholder="https://hooks.example.com/events" value={rule.action.webhook_url ?? ''} onChange={(e) => updateAction(i, 'webhook_url', e.target.value)} />
              <p className={hintCls}>Receives JSON payload with event type, tool name, agent name, timestamp</p>
            </div>
          )}
          {rule.action.type === 'inject_header' && (
            <div>
              <label className={labelCls}>HTTP Header (added to MCP tool requests)</label>
              <div className="flex gap-2">
                <input className={`${inputCls} flex-1`} placeholder="X-Request-ID" value={getHeader(rule)[0]} onChange={(e) => updateHeader(i, e.target.value, getHeader(rule)[1])} />
                <input className={`${inputCls} flex-1`} placeholder="correlation-id-123" value={getHeader(rule)[1]} onChange={(e) => updateHeader(i, getHeader(rule)[0], e.target.value)} />
              </div>
              <p className={hintCls}>Left: header name. Right: header value. Injected into all outgoing MCP requests</p>
            </div>
          )}
        </div>
      ))}
      <button type="button" onClick={() => setRules([...rules, { enabled: true, condition: { type: 'before_tool_call' }, action: { type: 'log_to_webhook' } }])} className="text-xs text-brand-shade3 hover:text-brand-light transition-colors">
        + Add rule
      </button>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Summary
// ---------------------------------------------------------------------------

function getSummary(cap: CapabilityConfig): string {
  const c = cap.config ?? {} as Record<string, unknown>;
  switch (cap.type) {
    case 'memory': {
      const parts: string[] = ['Per-schema'];
      if (c.unlimited_retention) parts.push('unlimited retention');
      else if (c.retention_days) parts.push(`${c.retention_days}d retention`);
      else parts.push('unlimited retention');
      if (c.unlimited_entries) parts.push('unlimited entries');
      else parts.push(`max ${c.max_entries ?? 500}`);
      return parts.join(', ');
    }
    case 'knowledge': {
      const sources = c.sources as string[] | undefined;
      const parts: string[] = [];
      const first = sources?.[0];
      if (first) parts.push(first);
      const topK = c.top_k as number | undefined;
      if (topK) parts.push(`top-k: ${topK}`);
      return parts.length > 0 ? parts.join(', ') : 'No sources configured';
    }
    case 'guardrail': {
      const parts: string[] = [];
      if (c.mode) parts.push(String(c.mode).replace('_', ' '));
      const onFail = c.on_failure as string | undefined;
      if (onFail) parts.push(`on fail: ${onFail.replace('_', ' ')}`);
      return parts.length > 0 ? parts.join(', ') : 'JSON Schema validation';
    }
    case 'escalation': {
      const parts: string[] = [];
      parts.push(c.action ? String(c.action) : 'transfer_to_user');
      const triggers = c.triggers_text as string | undefined;
      if (triggers) {
        const count = triggers.split(',').filter(Boolean).length;
        if (count > 0) parts.push(`${count} trigger${count > 1 ? 's' : ''}`);
      }
      return parts.join(', ');
    }
    case 'recovery': {
      const rules = c.rules as RecoveryRule[] | undefined;
      const count = rules?.length ?? 5;
      return `${count} failure types configured`;
    }
    case 'policies': {
      const rules = c.rules as unknown[] | undefined;
      return rules?.length ? `${rules.length} rule(s)` : 'No rules';
    }
    default: return '';
  }
}

// ---------------------------------------------------------------------------
// Main block
// ---------------------------------------------------------------------------

const configMap: Record<string, React.FC<PanelProps>> = {
  memory: MemoryConfig,
  knowledge: KnowledgeConfig,
  guardrail: GuardrailConfig,

  escalation: EscalationConfig,
  recovery: RecoveryConfig,
  policies: PoliciesConfig,
};

export default function CapabilityBlock({ capability, onChange, onRemove, agentName, models }: CapabilityBlockProps) {
  const [open, setOpen] = useState(false);
  const meta = CAPABILITY_META[capability.type] ?? { label: capability.type, icon: 'brain', description: 'Unknown capability type' };
  const summary = getSummary(capability);
  const ConfigPanel = configMap[capability.type];

  return (
    <div className="bg-brand-dark-alt border border-brand-shade3/20 rounded-card font-mono">
      {/* Header — click anywhere to expand/collapse */}
      <div
        className="flex items-center justify-between px-3 py-2.5 cursor-pointer hover:bg-brand-dark-surface/50 transition-colors"
        onClick={() => setOpen((v) => !v)}
      >
        <div className="flex items-center gap-2 text-sm min-w-0">
          <span className="text-brand-shade3 shrink-0">{capabilityIcon(meta.icon)}</span>
          <span className={`shrink-0 ${capability.enabled ? 'text-brand-light' : 'text-brand-shade3'}`}>{meta.label}</span>
          {!open && summary && <span className="text-[11px] text-brand-shade3 truncate ml-1">{summary}</span>}
        </div>
        <div className="flex items-center gap-1.5">
          {/* Chevron indicator */}
          <svg
            width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"
            className={`text-brand-shade3 transition-transform ${open ? 'rotate-180' : ''}`}
          >
            <path d="M6 9l6 6 6-6" />
          </svg>
          {/* Toggle — stopPropagation to prevent expand/collapse */}
          <button
            type="button"
            onClick={(e) => { e.stopPropagation(); onChange({ ...capability, enabled: !capability.enabled }); }}
            className={`relative inline-flex h-4 w-7 items-center rounded-full transition-colors ${capability.enabled ? 'bg-brand-accent' : 'bg-brand-shade3/40'}`}
            title={capability.enabled ? 'Disable' : 'Enable'}
          >
            <span className={`inline-block h-3 w-3 rounded-full bg-white transition-transform ${capability.enabled ? 'translate-x-3.5' : 'translate-x-0.5'}`} />
          </button>
          {/* Remove — stopPropagation */}
          <button
            type="button"
            onClick={(e) => { e.stopPropagation(); onRemove(); }}
            className="p-1 text-brand-shade3 hover:text-brand-light transition-colors"
            title="Remove"
            aria-label="Remove capability"
          >
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 6L6 18M6 6l12 12" /></svg>
          </button>
        </div>
      </div>
      {open && ConfigPanel && (
        <div className="px-3 py-3 border-t border-brand-shade3/10">
          <ConfigPanel cap={capability} onChange={onChange} agentName={agentName} models={models} />
        </div>
      )}
    </div>
  );
}
