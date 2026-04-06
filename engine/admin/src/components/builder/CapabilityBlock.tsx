import { useState, useRef, type DragEvent } from 'react';
import type { CapabilityConfig } from '../../types';
import { CAPABILITY_META } from '../../types';

interface CapabilityBlockProps {
  capability: CapabilityConfig;
  onChange: (updated: CapabilityConfig) => void;
  onRemove: () => void;
}

const inputCls =
  'w-full bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light px-3 py-2 focus:outline-none focus:border-brand-accent placeholder-brand-shade3';

const selectCls =
  'w-full bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light px-3 py-2 focus:outline-none focus:border-brand-accent';

const labelCls = 'block text-xs text-brand-shade3 mb-1 font-mono';
const descCls = 'text-xs text-brand-shade3 mb-3';
const hintCls = 'text-[11px] text-brand-shade3/70 mt-1';

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

type PanelProps = { cap: CapabilityConfig; onChange: (u: CapabilityConfig) => void };

function MemoryConfig({ cap, onChange }: PanelProps) {
  const unlimitedRetention = getKey(cap, 'unlimited_retention', false) as boolean;
  const unlimitedEntries = getKey(cap, 'unlimited_entries', false) as boolean;

  return (
    <div className="space-y-3">
      <p className={descCls}>Agent remembers facts across sessions within this schema. Recalled automatically at session start, stored during conversation. Users can also ask the agent to remember things explicitly.</p>
      <div className="bg-brand-dark rounded-card px-3 py-2 space-y-1">
        <span className="text-[11px] text-brand-shade2 font-mono">Scope: per-schema, cross-session</span>
        <p className={hintCls}>Memory is isolated per schema and persists between sessions. Support Flow and Sales Flow have separate memory spaces.</p>
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
          <input type="checkbox" className="accent-brand-accent" checked={unlimitedRetention} onChange={(e) => onChange(setKey(cap, 'unlimited_retention', e.target.checked))} />
          Unlimited
        </label>
        {!unlimitedRetention && (
          <div className="flex items-center gap-2">
            <input type="number" className={inputCls} min={1} max={365} value={getKey(cap, 'retention_days', 30) as number} onChange={(e) => onChange(setKey(cap, 'retention_days', Number(e.target.value)))} />
            <span className="text-xs text-brand-shade3 shrink-0">days</span>
          </div>
        )}
        <p className={hintCls}>{unlimitedRetention ? 'Memory entries are kept indefinitely' : 'Entries older than this are automatically deleted'}</p>
      </div>
      <div>
        <label className={labelCls}>Max entries</label>
        <label className="flex items-center gap-2 text-sm text-brand-shade2 cursor-pointer select-none mb-2">
          <input type="checkbox" className="accent-brand-accent" checked={unlimitedEntries} onChange={(e) => onChange(setKey(cap, 'unlimited_entries', e.target.checked))} />
          Unlimited
        </label>
        {!unlimitedEntries && (
          <input type="number" className={inputCls} min={1} value={getKey(cap, 'max_entries', 500) as number} onChange={(e) => onChange(setKey(cap, 'max_entries', Number(e.target.value)))} />
        )}
        <p className={hintCls}>{unlimitedEntries ? 'No limit on stored entries (bounded by schema storage quota)' : 'Oldest entries evicted when limit reached'}</p>
      </div>
    </div>
  );
}

function KnowledgeConfig({ cap, onChange }: PanelProps) {
  const sources = getKey<string[]>(cap, 'sources', []);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [dragOver, setDragOver] = useState(false);

  function addFiles(files: FileList | null) {
    if (!files || files.length === 0) return;
    const names = Array.from(files).map(f => f.name);
    onChange(setKey(cap, 'sources', [...sources, ...names]));
  }

  function handleDrop(e: DragEvent<HTMLDivElement>) {
    e.preventDefault();
    setDragOver(false);
    addFiles(e.dataTransfer.files);
  }

  return (
    <div className="space-y-3">
      <p className={descCls}>RAG: agent searches uploaded documents before answering</p>
      <div className="bg-brand-dark rounded-card px-3 py-2 space-y-1">
        <span className="text-[11px] text-brand-shade2 font-mono">Auto-included tools:</span>
        <div className="flex gap-2 mt-1">
          <span className="text-[10px] px-2 py-0.5 bg-brand-dark-alt border border-brand-shade3/20 rounded-card text-brand-shade2">search_knowledge</span>
        </div>
        <p className={hintCls}>Automatically available to agent when Knowledge is enabled</p>
      </div>
      <input ref={fileInputRef} type="file" multiple accept=".pdf,.txt,.md" className="hidden" onChange={(e) => { addFiles(e.target.files); e.target.value = ''; }} />
      <div
        className={`border border-dashed rounded-card px-4 py-6 text-center text-xs cursor-pointer transition-colors ${
          dragOver ? 'border-brand-accent bg-brand-accent/5 text-brand-accent' : 'border-brand-shade3/30 text-brand-shade3 hover:border-brand-shade3/50'
        }`}
        onClick={() => fileInputRef.current?.click()}
        onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
        onDragLeave={() => setDragOver(false)}
        onDrop={handleDrop}
      >
        {dragOver ? 'Drop files to add' : 'Drop files here or click to upload (PDF, TXT, MD)'}
      </div>
      {sources.length > 0 && (
        <ul className="space-y-1">
          {sources.map((src, i) => (
            <li key={src + '-' + i} className="flex items-center justify-between text-xs text-brand-shade2 bg-brand-dark-alt px-2 py-1 rounded-card">
              <span className="truncate">{src}</span>
              <button type="button" onClick={() => onChange(setKey(cap, 'sources', sources.filter((_, j) => j !== i)))} className="ml-2 text-brand-shade3 hover:text-brand-accent">x</button>
            </li>
          ))}
        </ul>
      )}
      <div>
        <label className={labelCls}>Top-K</label>
        <input type="number" className={inputCls} min={1} max={20} value={getKey(cap, 'top_k', 5) as number} onChange={(e) => onChange(setKey(cap, 'top_k', Number(e.target.value)))} />
        <p className={hintCls}>Number of most relevant document chunks retrieved per query</p>
      </div>
      <div>
        <label className={labelCls}>Similarity threshold</label>
        <input type="number" className={inputCls} min={0} max={1} step={0.05} value={getKey(cap, 'similarity_threshold', 0.75) as number} onChange={(e) => onChange(setKey(cap, 'similarity_threshold', Number(e.target.value)))} />
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
  const placeholders: Record<string, string> = {
    json_schema: '{"type":"object","required":["answer"]}',
    llm_check: 'Is this response professional and accurate? Reply YES or NO.',
    webhook: 'https://validate.example.com/check',
  };
  return (
    <div className="space-y-3">
      <p className={descCls}>Validates agent output before sending to user</p>
      <div>
        <label className={labelCls}>Mode</label>
        <select className={selectCls} value={mode} onChange={(e) => onChange(setKey(cap, 'mode', e.target.value))}>
          <option value="json_schema">JSON Schema</option>
          <option value="llm_check">LLM Check</option>
          <option value="webhook">Webhook</option>
        </select>
        <p className={hintCls}>{modeHints[mode]}</p>
      </div>
      <div>
        <label className={labelCls}>Config</label>
        <textarea className={`${inputCls} font-mono resize-y`} rows={4} placeholder={placeholders[mode]} value={getKey(cap, 'config_text', '') as string} onChange={(e) => onChange(setKey(cap, 'config_text', e.target.value))} />
      </div>
      <div>
        <label className={labelCls}>On failure</label>
        <select className={selectCls} value={getKey(cap, 'on_failure', 'retry_max_3') as string} onChange={(e) => onChange(setKey(cap, 'on_failure', e.target.value))}>
          <option value="retry_max_3">Retry (agent regenerates response, max 3 attempts)</option>
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

function OutputSchemaConfig({ cap, onChange }: PanelProps) {
  const fmt = getKey(cap, 'format', 'json_schema') as string;
  return (
    <div className="space-y-3">
      <p className={descCls}>Enforces structured JSON output format from agent</p>
      <div>
        <label className={labelCls}>Format</label>
        <select className={selectCls} value={fmt} onChange={(e) => onChange(setKey(cap, 'format', e.target.value))}>
          <option value="json_schema">JSON Schema</option>
          <option value="plain_text">Plain Text</option>
        </select>
        <p className={hintCls}>JSON Schema: agent must output valid JSON matching the schema. Plain Text: validates basic format rules (length, no code blocks) without JSON</p>
      </div>
      <label className="flex items-center gap-2 text-sm text-brand-shade2 cursor-pointer select-none">
        <input type="checkbox" className="accent-brand-accent" checked={getKey(cap, 'enforce', false) as boolean} onChange={(e) => onChange(setKey(cap, 'enforce', e.target.checked))} />
        Enforce
      </label>
      <p className="text-[11px] text-brand-shade3/70 ml-6">When enabled, response is rejected and error shown to user if output doesn't conform</p>
      {fmt === 'json_schema' && (
        <div>
          <label className={labelCls}>Schema</label>
          <textarea className={`${inputCls} font-mono resize-y`} rows={6} placeholder='{"type":"object","properties":{}}' value={getKey(cap, 'schema', '') as string} onChange={(e) => onChange(setKey(cap, 'schema', e.target.value))} />
        </div>
      )}
    </div>
  );
}

function EscalationConfig({ cap, onChange }: PanelProps) {
  const action = getKey(cap, 'action', 'transfer_to_human') as string;
  const actionHints: Record<string, string> = {
    transfer_to_human: 'Session marked as needing human, agent stops responding',
    notify: 'Webhook called with session context, agent continues',
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
        <select className={selectCls} value={action} onChange={(e) => onChange(setKey(cap, 'action', e.target.value))}>
          <option value="transfer_to_human">Transfer to Human</option>
          <option value="notify">Notify</option>
        </select>
        <p className={hintCls}>{actionHints[action]}</p>
      </div>
      {action !== 'transfer_to_human' && (
        <div>
          <label className={labelCls}>Webhook URL</label>
          <input className={inputCls} placeholder="https://hooks.example.com/escalate" value={getKey(cap, 'webhook_url', '') as string} onChange={(e) => onChange(setKey(cap, 'webhook_url', e.target.value))} />
          <p className={hintCls}>Called when escalation triggers. Receives full session context.</p>
        </div>
      )}
      <div>
        <label className={labelCls}>Triggers</label>
        <textarea className={`${inputCls} resize-y`} rows={2} placeholder="confidence < 0.4, user requests human" value={getKey(cap, 'triggers_text', '') as string} onChange={(e) => onChange(setKey(cap, 'triggers_text', e.target.value))} />
        <p className={hintCls}>Comma-separated conditions. Examples: 'confidence {'<'} 0.4', 'user requests human', 'sentiment negative'</p>
      </div>
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
          {rule.action === 'fallback' && (
            <div>
              <label className={labelCls}>Fallback model</label>
              <input className={inputCls} placeholder="e.g. gpt-4o-mini" value={rule.fallback_model} onChange={(e) => updateRule(i, 'fallback_model', e.target.value)} />
              <p className="text-[11px] text-brand-shade3/70 mt-1">Model name as shown on Models page (e.g. gpt-4o-mini, claude-haiku-3)</p>
            </div>
          )}
        </div>
      ))}
    </div>
  );
}

type PolicyRule = { condition: string; action: string; pattern?: string; webhook_url?: string; header_name?: string; header_value?: string; time_range?: string };

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

  function updateRule(i: number, field: keyof PolicyRule, value: string) {
    const next = rules.map((r, j) => (j === i ? { ...r, [field]: value } : r));
    onChange(setKey(cap, 'rules', next));
  }

  return (
    <div className="space-y-3">
      <p className={descCls}>Visual rules: When [condition] occurs -- Do [action]</p>
      {rules.map((rule, i) => (
        <div key={rule.condition + '-' + rule.action + '-' + i} className="space-y-2">
          <div className="flex gap-2 items-center">
            <select className={`${selectCls} flex-1`} value={rule.condition} onChange={(e) => updateRule(i, 'condition', e.target.value)}>
              <option value="before_tool_call">Before tool call</option>
              <option value="after_tool_call">After tool call</option>
              <option value="tool_matches">Tool matches</option>
              <option value="time_range">Time range</option>
              <option value="error_occurred">Error occurred</option>
            </select>
            <select className={`${selectCls} flex-1`} value={rule.action} onChange={(e) => updateRule(i, 'action', e.target.value)}>
              <option value="log_to_webhook">Log to webhook</option>
              <option value="block">Block</option>
              <option value="notify">Notify</option>
              <option value="inject_header">Inject header</option>
              <option value="write_audit">Write audit</option>
            </select>
            <button type="button" onClick={() => onChange(setKey(cap, 'rules', rules.filter((_, j) => j !== i)))} className="text-brand-shade3 hover:text-brand-accent text-lg leading-none flex-shrink-0">x</button>
          </div>
          <p className="text-[11px] text-brand-shade3/70">{conditionHints[rule.condition]}</p>
          <p className="text-[11px] text-brand-shade3/70">{actionHintsPolicy[rule.action]}</p>
          {rule.condition === 'tool_matches' && (
            <div>
              <label className={labelCls}>Tool pattern (glob)</label>
              <input className={inputCls} placeholder="delete_*, send_email, admin_*" value={rule.pattern ?? ''} onChange={(e) => updateRule(i, 'pattern', e.target.value)} />
              <p className={hintCls}>Use * as wildcard. Matches tool names like delete_user, delete_cache</p>
            </div>
          )}
          {rule.condition === 'time_range' && (
            <div>
              <label className={labelCls}>Time window (UTC, 24h)</label>
              <input className={inputCls} placeholder="09:00-17:00" value={rule.time_range ?? ''} onChange={(e) => updateRule(i, 'time_range', e.target.value)} />
              <p className={hintCls}>Format: HH:MM-HH:MM. Example: 09:00-17:00 for working hours only</p>
            </div>
          )}
          {(rule.action === 'log_to_webhook' || rule.action === 'notify') && (
            <div>
              <label className={labelCls}>Webhook URL</label>
              <input className={inputCls} placeholder="https://hooks.example.com/events" value={rule.webhook_url ?? ''} onChange={(e) => updateRule(i, 'webhook_url', e.target.value)} />
              <p className={hintCls}>Receives JSON payload with event type, tool name, agent name, timestamp</p>
            </div>
          )}
          {rule.action === 'inject_header' && (
            <div>
              <label className={labelCls}>HTTP Header (added to MCP tool requests)</label>
              <div className="flex gap-2">
                <input className={`${inputCls} flex-1`} placeholder="X-Request-ID" value={rule.header_name ?? ''} onChange={(e) => updateRule(i, 'header_name', e.target.value)} />
                <input className={`${inputCls} flex-1`} placeholder="correlation-id-123" value={rule.header_value ?? ''} onChange={(e) => updateRule(i, 'header_value', e.target.value)} />
              </div>
              <p className={hintCls}>Left: header name. Right: header value. Injected into all outgoing MCP requests</p>
            </div>
          )}
        </div>
      ))}
      <button type="button" onClick={() => onChange(setKey(cap, 'rules', [...rules, { condition: 'before_tool_call', action: 'log_to_webhook' }]))} className="text-xs text-brand-shade3 hover:text-brand-light transition-colors">
        + Add rule
      </button>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Summary
// ---------------------------------------------------------------------------

function getSummary(cap: CapabilityConfig): string {
  const c = cap.config;
  switch (cap.type) {
    case 'memory': {
      const parts: string[] = ['Per-schema'];
      if (c.unlimited_retention) parts.push('unlimited retention');
      else parts.push(`${c.retention_days ?? 30}d retention`);
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
    case 'output_schema': {
      const parts: string[] = [];
      const fmt = c.format as string | undefined;
      if (fmt) parts.push(fmt.replace('_', ' '));
      if (c.enforce) parts.push('enforced');
      return parts.length > 0 ? parts.join(', ') : 'No schema defined';
    }
    case 'escalation': {
      const parts: string[] = [];
      parts.push(c.action ? String(c.action) : 'transfer_to_human');
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
  output_schema: OutputSchemaConfig,
  escalation: EscalationConfig,
  recovery: RecoveryConfig,
  policies: PoliciesConfig,
};

export default function CapabilityBlock({ capability, onChange, onRemove }: CapabilityBlockProps) {
  const [open, setOpen] = useState(false);
  const meta = CAPABILITY_META[capability.type];
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
          <span className="text-[10px] font-semibold text-brand-shade3 bg-brand-dark px-1.5 py-0.5 rounded-card shrink-0">{meta.abbr}</span>
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
          <ConfigPanel cap={capability} onChange={onChange} />
        </div>
      )}
    </div>
  );
}
