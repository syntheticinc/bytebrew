import { useState } from 'react';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type ConditionType = 'auto' | 'human' | 'llm' | 'all_completed';
export type FailureAction = 'block' | 'skip' | 'escalate';

export interface GateConfig {
  label: string;
  conditionType: ConditionType;
  conditionConfig: string;
  maxIterations: number;
  timeout: number;
  onFailure: FailureAction;
  llmModel?: string;
}

interface GateConfigPanelProps {
  gate: Record<string, unknown>;
  onClose: () => void;
  onSave?: (gateId: string, config: GateConfig) => void;
}

const CONDITION_TYPES: { value: ConditionType; label: string; description: string }[] = [
  { value: 'auto', label: 'Auto', description: 'JSON Schema, regex, or contains check' },
  { value: 'human', label: 'Human Approval', description: 'Requires manual approval to pass' },
  { value: 'llm', label: 'LLM Judge', description: 'LLM evaluates output quality' },
  { value: 'all_completed', label: 'Join (All Completed)', description: 'Wait for all incoming edges' },
];

const FAILURE_ACTIONS: { value: FailureAction; label: string; description: string }[] = [
  { value: 'block', label: 'Block', description: 'Stop the pipeline' },
  { value: 'skip', label: 'Skip', description: 'Skip gate, continue flow' },
  { value: 'escalate', label: 'Escalate', description: 'Trigger escalation handler' },
];

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function GateConfigPanel({ gate, onClose, onSave }: GateConfigPanelProps) {
  const [conditionType, setConditionType] = useState<ConditionType>(
    (gate.conditionType as ConditionType) ?? 'auto',
  );
  const [conditionConfig, setConditionConfig] = useState(
    String(gate.conditionConfig ?? ''),
  );
  const [maxIterations, setMaxIterations] = useState(
    Number(gate.maxIterations ?? 3),
  );
  const [timeout, setTimeout_] = useState(
    Number(gate.timeout ?? 0),
  );
  const [onFailure, setOnFailure] = useState<FailureAction>(
    (gate.onFailure as FailureAction) ?? 'block',
  );
  const [llmModel, setLlmModel] = useState(
    String(gate.llmModel ?? ''),
  );

  const label = String(gate.label ?? 'Gate');
  const gateId = String(gate.id ?? gate.nodeId ?? '');

  const handleSave = () => {
    onSave?.(gateId, {
      label,
      conditionType,
      conditionConfig,
      maxIterations,
      timeout: timeout,
      onFailure,
      llmModel: conditionType === 'llm' ? llmModel : undefined,
    });
  };

  const configPlaceholder = (() => {
    switch (conditionType) {
      case 'auto':
        return '{"type":"object","required":["approved"]}\nor regex: ^(yes|approved)$\nor contains: approved';
      case 'human':
        return 'Approval prompt shown to operator:\ne.g. "Review the agent output and approve"';
      case 'llm':
        return 'Judge prompt for LLM:\ne.g. "Evaluate if the output is complete and accurate. Respond with PASS or FAIL."';
      case 'all_completed':
        return 'No configuration needed for Join gate';
    }
  })();

  return (
    <div className="w-80 bg-brand-dark-surface border-l border-brand-shade3/10 flex flex-col h-full flex-shrink-0 animate-slide-in-right">
      {/* Header */}
      <div className="px-4 py-3 border-b border-brand-shade3/15 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <svg width="14" height="14" viewBox="0 0 100 100" className="text-amber-500 flex-shrink-0">
            <polygon points="50,5 95,50 50,95 5,50" fill="none" stroke="currentColor" strokeWidth="8" />
          </svg>
          <span className="text-sm font-semibold text-brand-light font-mono">{label}</span>
        </div>
        <button
          onClick={onClose}
          className="p-1 text-brand-shade3 hover:text-brand-light transition-colors flex-shrink-0"
          title="Close"
          aria-label="Close gate config panel"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M18 6L6 18M6 6l12 12" />
          </svg>
        </button>
      </div>

      {/* Scrollable content */}
      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-4">
        {/* Condition Type */}
        <div>
          <span className="text-[10px] text-brand-shade3 uppercase tracking-wider">Condition Type</span>
          <div className="mt-2 space-y-2">
            {CONDITION_TYPES.map((ct) => (
              <label
                key={ct.value}
                className={`flex items-start gap-2.5 p-2 rounded-card border cursor-pointer transition-colors ${
                  conditionType === ct.value
                    ? 'border-amber-500/40 bg-amber-500/5'
                    : 'border-brand-shade3/15 hover:border-brand-shade3/30'
                }`}
              >
                <input
                  type="radio"
                  name="gate-condition-type"
                  value={ct.value}
                  checked={conditionType === ct.value}
                  onChange={() => setConditionType(ct.value)}
                  className="mt-0.5 accent-amber-500"
                />
                <div className="min-w-0">
                  <div className="text-xs text-brand-light font-medium">{ct.label}</div>
                  <div className="text-[11px] text-brand-shade3 leading-snug mt-0.5">{ct.description}</div>
                </div>
              </label>
            ))}
          </div>
        </div>

        {/* Condition Config (not for all_completed) */}
        {conditionType !== 'all_completed' && (
          <div>
            <span className="text-[10px] text-brand-shade3 uppercase tracking-wider">
              {conditionType === 'auto' ? 'Condition Config' : conditionType === 'human' ? 'Approval Prompt' : 'Judge Prompt'}
            </span>
            <textarea
              value={conditionConfig}
              onChange={(e) => setConditionConfig(e.target.value)}
              placeholder={configPlaceholder}
              rows={4}
              className="mt-1 w-full px-2 py-1.5 bg-brand-dark border border-brand-shade3/30 rounded-card text-xs text-brand-light font-mono focus:outline-none focus:border-amber-500 placeholder-brand-shade3/50 transition-colors resize-y leading-relaxed"
            />
          </div>
        )}

        {/* LLM Model (only for llm type) */}
        {conditionType === 'llm' && (
          <div>
            <span className="text-[10px] text-brand-shade3 uppercase tracking-wider">LLM Model</span>
            <input
              type="text"
              value={llmModel}
              onChange={(e) => setLlmModel(e.target.value)}
              placeholder="e.g. gpt-4o-mini (leave empty for default)"
              className="mt-1 w-full px-2 py-1.5 bg-brand-dark border border-brand-shade3/30 rounded-card text-xs text-brand-light font-mono focus:outline-none focus:border-amber-500 placeholder-brand-shade3/50 transition-colors"
            />
          </div>
        )}

        {/* Max Iterations */}
        <div>
          <span className="text-[10px] text-brand-shade3 uppercase tracking-wider">Max Iterations</span>
          <div className="mt-1 flex items-center gap-2">
            <input
              type="number"
              min={1}
              max={10}
              value={maxIterations}
              onChange={(e) => {
                const v = Math.min(10, Math.max(1, parseInt(e.target.value) || 1));
                setMaxIterations(v);
              }}
              className="w-20 px-2 py-1.5 bg-brand-dark border border-brand-shade3/30 rounded-card text-xs text-brand-light font-mono focus:outline-none focus:border-amber-500 transition-colors"
            />
            <span className="text-[11px] text-brand-shade3">1-10, prevents infinite loops</span>
          </div>
        </div>

        {/* Timeout */}
        <div>
          <span className="text-[10px] text-brand-shade3 uppercase tracking-wider">Timeout (seconds)</span>
          <div className="mt-1 flex items-center gap-2">
            <input
              type="number"
              min={0}
              max={3600}
              value={timeout}
              onChange={(e) => {
                const v = Math.min(3600, Math.max(0, parseInt(e.target.value) || 0));
                setTimeout_(v);
              }}
              className="w-20 px-2 py-1.5 bg-brand-dark border border-brand-shade3/30 rounded-card text-xs text-brand-light font-mono focus:outline-none focus:border-amber-500 transition-colors"
            />
            <span className="text-[11px] text-brand-shade3">0 = no timeout</span>
          </div>
        </div>

        {/* On Failure */}
        <div>
          <span className="text-[10px] text-brand-shade3 uppercase tracking-wider">On Timeout / Failure</span>
          <div className="mt-2 space-y-1.5">
            {FAILURE_ACTIONS.map((fa) => (
              <label
                key={fa.value}
                className={`flex items-center gap-2.5 px-2 py-1.5 rounded-card border cursor-pointer transition-colors ${
                  onFailure === fa.value
                    ? 'border-amber-500/40 bg-amber-500/5'
                    : 'border-brand-shade3/15 hover:border-brand-shade3/30'
                }`}
              >
                <input
                  type="radio"
                  name="gate-on-failure"
                  value={fa.value}
                  checked={onFailure === fa.value}
                  onChange={() => setOnFailure(fa.value)}
                  className="accent-amber-500"
                />
                <div className="flex items-center gap-2 min-w-0">
                  <span className="text-xs text-brand-light font-medium">{fa.label}</span>
                  <span className="text-[10px] text-brand-shade3">— {fa.description}</span>
                </div>
              </label>
            ))}
          </div>
        </div>
      </div>

      {/* Save button */}
      <div className="px-4 py-3 border-t border-brand-shade3/15">
        <button
          onClick={handleSave}
          className="w-full px-3 py-1.5 bg-amber-500 hover:bg-amber-600 text-brand-dark text-xs font-medium rounded-card transition-colors"
        >
          Save Gate Configuration
        </button>
      </div>
    </div>
  );
}
