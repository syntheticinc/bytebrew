import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import StatusBadge from '../components/StatusBadge';
import { usePrototype } from '../hooks/usePrototype';
import type { SessionTrace, InspectStep, InspectStepKind } from '../types';
import { MOCK_TRACE, MOCK_SESSIONS } from '../mocks/inspect';

// ─── Step icons ───────────────────────────────────────────────────────────────

function StepIcon({ kind }: { kind: InspectStepKind }) {
  const base = 'w-4 h-4 shrink-0';
  if (kind === 'reasoning') return (
    <svg className={base} viewBox="0 0 16 16" fill="none">
      <circle cx="8" cy="8" r="6" stroke="currentColor" strokeWidth="1.5" />
      <path d="M5.5 6.5C5.5 5.12 6.62 4 8 4s2.5 1.12 2.5 2.5c0 1.1-.7 2.04-1.68 2.38L8.5 10h-1l-.32-1.12A2.5 2.5 0 0 1 5.5 6.5Z" fill="currentColor" />
      <circle cx="8" cy="11.5" r=".75" fill="currentColor" />
    </svg>
  );
  if (kind === 'tool_call') return (
    <svg className={base} viewBox="0 0 16 16" fill="none">
      <path d="M9.5 2a3 3 0 0 1 2.83 4.02l2.17 2.17-1.5 1.5-2.17-2.17A3 3 0 1 1 9.5 2ZM9.5 4a1 1 0 1 0 0 2 1 1 0 0 0 0-2Z" fill="currentColor" />
      <path d="M3 9.5l3.5 3.5 1.06-1.06L4.06 8.44 3 9.5Z" fill="currentColor" />
    </svg>
  );
  if (kind === 'memory_recall') return (
    <svg className={base} viewBox="0 0 16 16" fill="none">
      <rect x="2" y="3" width="12" height="10" rx="1.5" stroke="currentColor" strokeWidth="1.5" />
      <path d="M5 7h6M5 9.5h4" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
      <path d="M12 6l2-2M12 6l-2-2" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
    </svg>
  );
  // final_answer
  return (
    <svg className={base} viewBox="0 0 16 16" fill="none">
      <circle cx="8" cy="8" r="6" stroke="currentColor" strokeWidth="1.5" />
      <path d="M5.5 8.5l1.5 1.5 3.5-3.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

const KIND_COLORS: Record<InspectStepKind, string> = {
  reasoning: 'text-blue-400',
  tool_call: 'text-brand-accent',
  memory_recall: 'text-purple-400',
  final_answer: 'text-status-active',
};

// ─── Step card ────────────────────────────────────────────────────────────────

function StepCard({ step, index }: { step: InspectStep; index: number }) {
  const [expanded, setExpanded] = useState(false);
  const hasContent = step.input != null || step.output != null;
  const color = KIND_COLORS[step.kind];
  const durationSec = (step.duration_ms / 1000).toFixed(1);

  return (
    <div className="bg-brand-dark-surface border border-brand-shade3/15 rounded-card">
      <div className="flex items-start gap-3 px-4 py-3">
        {/* Step number */}
        <span className="mt-0.5 text-xs text-brand-shade3 font-mono w-5 shrink-0 text-right">
          {index + 1}
        </span>

        {/* Icon */}
        <span className={`mt-0.5 ${color}`}>
          <StepIcon kind={step.kind} />
        </span>

        {/* Label */}
        <span className="flex-1 text-sm text-brand-light font-mono leading-snug">
          {step.label}
        </span>

        {/* Meta */}
        <div className="flex items-center gap-2 shrink-0 ml-2">
          {step.tokens != null && (
            <span className="text-xs text-brand-shade3 font-mono">
              {step.tokens.toLocaleString()} tok
            </span>
          )}
          <span className="text-xs text-brand-shade2 bg-brand-dark-alt px-2 py-0.5 rounded-card font-mono">
            {durationSec}s
          </span>
          {hasContent && (
            <button
              onClick={() => setExpanded((v) => !v)}
              className="text-xs text-brand-shade3 hover:text-brand-light transition-colors font-mono ml-1"
            >
              {expanded ? '▲ hide' : '▼ show'}
            </button>
          )}
        </div>
      </div>

      {expanded && hasContent && (
        <div className="px-4 pb-3 flex flex-col gap-2">
          {step.input != null && (
            <div>
              <p className="text-xs text-brand-shade3 mb-1 font-mono uppercase tracking-wider">Input</p>
              <pre className="bg-brand-dark-alt px-3 py-2 text-xs text-brand-shade2 overflow-x-auto rounded-card font-mono leading-relaxed">
                {step.input}
              </pre>
            </div>
          )}
          {step.output != null && (
            <div>
              <p className="text-xs text-brand-shade3 mb-1 font-mono uppercase tracking-wider">Output</p>
              <pre className="bg-brand-dark-alt px-3 py-2 text-xs text-brand-shade2 overflow-x-auto rounded-card font-mono leading-relaxed">
                {step.output}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function InspectPage() {
  const { schema, agent, session } = useParams<{ schema: string; agent: string; session: string }>();
  const navigate = useNavigate();
  usePrototype();

  const [selectedSessionId, setSelectedSessionId] = useState(session || MOCK_TRACE.session_id);

  const trace: SessionTrace = MOCK_TRACE;
  const shortId = selectedSessionId.slice(-6);
  const totalSec = (trace.total_duration_ms / 1000).toFixed(1);

  return (
    <div className="max-w-3xl mx-auto">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 mb-6">
        <button
          onClick={() => navigate(`/builder/${schema ?? ''}/${agent ?? ''}`)}
          className="flex items-center gap-1.5 text-sm text-brand-shade3 hover:text-brand-light transition-colors font-mono"
        >
          <svg className="w-4 h-4" viewBox="0 0 16 16" fill="none">
            <path d="M10 12L6 8l4-4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
          {schema} / {agent}
        </button>
        <span className="text-brand-shade3/40 font-mono">/</span>
        <span className="text-sm text-brand-light font-mono">Session #{shortId}</span>
      </div>

      {/* Session selector */}
      <div className="flex items-center gap-1 mb-4">
        {MOCK_SESSIONS.map((s) => (
          <button
            key={s.id}
            type="button"
            onClick={() => setSelectedSessionId(s.id)}
            className={[
              'px-3 py-1.5 rounded-btn text-xs font-mono transition-colors',
              s.id === selectedSessionId
                ? 'bg-brand-accent/20 text-brand-accent border border-brand-accent/40'
                : 'bg-brand-dark-surface text-brand-shade3 border border-brand-shade3/15 hover:text-brand-light hover:border-brand-shade3/30',
            ].join(' ')}
          >
            #{s.id.slice(-6)}
            <span
              className={[
                'ml-1.5 inline-block w-1.5 h-1.5 rounded-full',
                s.status === 'completed' ? 'bg-status-active' : 'bg-brand-accent',
              ].join(' ')}
            />
          </button>
        ))}
      </div>

      {/* Summary bar */}
      <div className="flex items-center gap-4 mb-6 px-4 py-3 bg-brand-dark-surface border border-brand-shade3/15 rounded-card">
        <StatusBadge status={trace.status} />
        <span className="text-sm text-brand-shade2 font-mono">{totalSec}s</span>
        <span className="text-sm text-brand-shade2 font-mono">{trace.total_tokens.toLocaleString()} tokens</span>
        <span className="ml-auto text-xs text-brand-shade3 font-mono">
          {new Date(trace.created_at).toLocaleString()}
        </span>
      </div>

      {/* Steps timeline */}
      <div className="flex flex-col gap-2">
        {trace.steps.map((step, i) => (
          <StepCard key={step.id} step={step} index={i} />
        ))}
      </div>
    </div>
  );
}
