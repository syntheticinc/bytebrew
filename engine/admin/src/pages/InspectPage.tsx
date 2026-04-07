import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import StatusBadge from '../components/StatusBadge';
import { api } from '../api/client';
import type { SessionSummary, SessionTrace, InspectStep, InspectStepKind, SessionStatus } from '../types';

// ─── Step icons ───────────────────────────────────────────────────────────────

const STEP_ICON_MAP: Record<InspectStepKind, { emoji: string; color: string }> = {
  reasoning:        { emoji: '\u{1F4AD}', color: 'text-blue-400' },
  tool_call:        { emoji: '\u{1F527}', color: 'text-brand-accent' },
  memory_recall:    { emoji: '\u{1F9E0}', color: 'text-purple-400' },
  knowledge_search: { emoji: '\u{1F4DA}', color: 'text-amber-400' },
  guardrail_check:  { emoji: '\u{1F6E1}\uFE0F', color: 'text-red-400' },
  final_answer:     { emoji: '\u2705',    color: 'text-status-active' },
  error:            { emoji: '\u26A0\uFE0F', color: 'text-red-500' },
  escalation:       { emoji: '\u{1F6A8}', color: 'text-orange-400' },
  task_dispatch:    { emoji: '\u{1F4E4}', color: 'text-cyan-400' },
  task_timeout:     { emoji: '\u23F0',    color: 'text-amber-500' },
};

// ─── Step card ────────────────────────────────────────────────────────────────

function StepCard({ step, index }: { step: InspectStep; index: number }) {
  const [expanded, setExpanded] = useState(false);
  const hasContent = step.input != null || step.output != null;
  const info = STEP_ICON_MAP[step.kind] ?? { emoji: '\u2753', color: 'text-brand-shade3' };
  const durationSec = (step.duration_ms / 1000).toFixed(1);

  return (
    <div className="bg-brand-dark-surface border border-brand-shade3/15 rounded-card">
      <div className="flex items-start gap-3 px-4 py-3">
        <span className="mt-0.5 text-xs text-brand-shade3 font-mono w-5 shrink-0 text-right">
          {index + 1}
        </span>
        <span className={`mt-0.5 text-sm ${info.color}`} title={step.kind}>
          {info.emoji}
        </span>
        <span className="flex-1 text-sm text-brand-light font-mono leading-snug">
          {step.label}
        </span>
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
              {expanded ? 'hide' : 'show'}
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

// ─── Status filter ────────────────────────────────────────────────────────────

const ALL_STATUSES: SessionStatus[] = ['completed', 'running', 'failed', 'blocked', 'timeout'];

function StatusFilter({
  selected,
  onChange,
}: {
  selected: SessionStatus[];
  onChange: (s: SessionStatus[]) => void;
}) {
  const [open, setOpen] = useState(false);

  const toggle = (s: SessionStatus) => {
    if (selected.includes(s)) {
      onChange(selected.filter((x) => x !== s));
    } else {
      onChange([...selected, s]);
    }
  };

  return (
    <div className="relative">
      <button
        onClick={() => setOpen((o) => !o)}
        className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-btn text-xs font-mono text-brand-shade2 border border-brand-shade3/20 hover:border-brand-shade3/40 transition-colors"
      >
        Status
        {selected.length > 0 && (
          <span className="bg-brand-accent/20 text-brand-accent px-1.5 rounded text-[10px] font-medium">
            {selected.length}
          </span>
        )}
        <svg width="10" height="10" viewBox="0 0 14 14" fill="none" className={`transition-transform ${open ? 'rotate-180' : ''}`}>
          <path d="M3 5L7 9L11 5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      </button>
      {open && (
        <div className="absolute top-full left-0 mt-1 bg-brand-dark-alt border border-brand-shade3/20 rounded-card shadow-xl z-50 py-1 min-w-[150px]">
          {ALL_STATUSES.map((s) => (
            <label
              key={s}
              className="flex items-center gap-2 px-3 py-1.5 text-xs text-brand-shade2 hover:bg-brand-dark-surface cursor-pointer transition-colors"
            >
              <input
                type="checkbox"
                checked={selected.includes(s)}
                onChange={() => toggle(s)}
                className="accent-brand-accent"
              />
              <StatusBadge status={s} />
            </label>
          ))}
          {selected.length > 0 && (
            <button
              onClick={() => onChange([])}
              className="w-full px-3 py-1.5 text-xs text-brand-shade3 hover:text-brand-light text-left border-t border-brand-shade3/10 mt-1"
            >
              Clear all
            </button>
          )}
        </div>
      )}
    </div>
  );
}

// ─── Session list (paginated table) ──────────────────────────────────────────

function SessionList({
  onSelectSession,
}: {
  onSelectSession: (id: string) => void;
}) {
  const [sessions, setSessions] = useState<SessionSummary[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<SessionStatus[]>([]);
  const [loading, setLoading] = useState(true);
  const perPage = 20;

  const fetchSessions = useCallback(() => {
    setLoading(true);
    api
      .listSessions({
        page,
        per_page: perPage,
        search: search || undefined,
        status: statusFilter.length > 0 ? statusFilter : undefined,
        sort_by: 'created_at',
        sort_dir: 'desc',
      })
      .then((res) => {
        setSessions(res.sessions);
        setTotal(res.total);
      })
      .catch(() => {
        setSessions([]);
        setTotal(0);
      })
      .finally(() => setLoading(false));
  }, [page, search, statusFilter]);

  useEffect(() => {
    fetchSessions();
  }, [fetchSessions]);

  // Reset to page 1 when filters change
  useEffect(() => {
    setPage(1);
  }, [search, statusFilter]);

  const totalPages = Math.max(1, Math.ceil(total / perPage));

  return (
    <div>
      {/* Search + filters */}
      <div className="flex items-center gap-3 mb-4">
        <div className="relative flex-1 max-w-sm">
          <svg
            className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-brand-shade3"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
          >
            <circle cx="11" cy="11" r="8" />
            <path d="M21 21l-4.35-4.35" />
          </svg>
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search by session ID or agent..."
            className="w-full pl-8 pr-3 py-1.5 bg-brand-dark-surface border border-brand-shade3/20 rounded-btn text-xs text-brand-light font-mono focus:outline-none focus:border-brand-accent placeholder-brand-shade3/50 transition-colors"
          />
        </div>
        <StatusFilter selected={statusFilter} onChange={setStatusFilter} />
        <span className="text-xs text-brand-shade3 font-mono ml-auto">
          {total} session{total !== 1 ? 's' : ''}
        </span>
      </div>

      {/* Table */}
      <div className="border border-brand-shade3/15 rounded-card overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="border-b border-brand-shade3/15 bg-brand-dark-surface">
              <th className="text-left text-[10px] text-brand-shade3 uppercase tracking-wider font-mono px-4 py-2.5">Session ID</th>
              <th className="text-left text-[10px] text-brand-shade3 uppercase tracking-wider font-mono px-4 py-2.5">Entry Agent</th>
              <th className="text-left text-[10px] text-brand-shade3 uppercase tracking-wider font-mono px-4 py-2.5">Status</th>
              <th className="text-left text-[10px] text-brand-shade3 uppercase tracking-wider font-mono px-4 py-2.5">Duration</th>
              <th className="text-left text-[10px] text-brand-shade3 uppercase tracking-wider font-mono px-4 py-2.5">Tokens</th>
              <th className="text-left text-[10px] text-brand-shade3 uppercase tracking-wider font-mono px-4 py-2.5">Created</th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-xs text-brand-shade3 font-mono">
                  Loading sessions...
                </td>
              </tr>
            )}
            {!loading && sessions.length === 0 && (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-xs text-brand-shade3 font-mono">
                  No sessions found
                </td>
              </tr>
            )}
            {!loading &&
              sessions.map((s) => (
                <tr
                  key={s.session_id}
                  onClick={() => onSelectSession(s.session_id)}
                  className="border-b border-brand-shade3/10 hover:bg-brand-dark-surface/50 cursor-pointer transition-colors"
                >
                  <td className="px-4 py-2.5">
                    <span className="text-xs text-brand-accent font-mono">
                      #{s.session_id.slice(-8)}
                    </span>
                  </td>
                  <td className="px-4 py-2.5">
                    <span className="text-xs text-brand-light font-mono">{s.entry_agent}</span>
                  </td>
                  <td className="px-4 py-2.5">
                    <StatusBadge status={s.status} />
                  </td>
                  <td className="px-4 py-2.5">
                    <span className="text-xs text-brand-shade2 font-mono">
                      {(s.duration_ms / 1000).toFixed(1)}s
                    </span>
                  </td>
                  <td className="px-4 py-2.5">
                    <span className="text-xs text-brand-shade2 font-mono">
                      {s.total_tokens.toLocaleString()}
                    </span>
                  </td>
                  <td className="px-4 py-2.5">
                    <span className="text-xs text-brand-shade3 font-mono">
                      {new Date(s.created_at).toLocaleString()}
                    </span>
                  </td>
                </tr>
              ))}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between mt-3">
          <span className="text-xs text-brand-shade3 font-mono">
            Page {page} of {totalPages}
          </span>
          <div className="flex items-center gap-1">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1}
              className="px-2.5 py-1 rounded-btn text-xs font-mono text-brand-shade2 border border-brand-shade3/20 hover:border-brand-shade3/40 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
            >
              Prev
            </button>
            {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
              const start = Math.max(1, Math.min(page - 2, totalPages - 4));
              const p = start + i;
              if (p > totalPages) return null;
              return (
                <button
                  key={p}
                  onClick={() => setPage(p)}
                  className={[
                    'px-2.5 py-1 rounded-btn text-xs font-mono border transition-colors',
                    p === page
                      ? 'bg-brand-accent/20 text-brand-accent border-brand-accent/40'
                      : 'text-brand-shade2 border-brand-shade3/20 hover:border-brand-shade3/40',
                  ].join(' ')}
                >
                  {p}
                </button>
              );
            })}
            <button
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page >= totalPages}
              className="px-2.5 py-1 rounded-btn text-xs font-mono text-brand-shade2 border border-brand-shade3/20 hover:border-brand-shade3/40 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
            >
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

// ─── Session detail (step timeline) ──────────────────────────────────────────

function SessionDetail({
  sessionId,
  onBack,
}: {
  sessionId: string;
  onBack: () => void;
}) {
  const [trace, setTrace] = useState<SessionTrace | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    api
      .getSessionTrace(sessionId)
      .then(setTrace)
      .catch(() => setTrace(null))
      .finally(() => setLoading(false));
  }, [sessionId]);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16">
        <span className="text-sm text-brand-shade3 font-mono">Loading session trace...</span>
      </div>
    );
  }

  if (!trace) {
    return (
      <div className="flex flex-col items-center justify-center py-16 gap-3">
        <span className="text-sm text-brand-shade3 font-mono">Session not found</span>
        <button onClick={onBack} className="text-xs text-brand-accent hover:underline font-mono">
          Back to sessions
        </button>
      </div>
    );
  }

  const shortId = sessionId.slice(-8);
  const totalSec = (trace.total_duration_ms / 1000).toFixed(1);

  return (
    <div className="max-w-3xl mx-auto">
      {/* Back button */}
      <button
        onClick={onBack}
        className="flex items-center gap-1.5 text-sm text-brand-shade3 hover:text-brand-light transition-colors font-mono mb-4"
      >
        <svg className="w-4 h-4" viewBox="0 0 16 16" fill="none">
          <path d="M10 12L6 8l4-4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
        Back to sessions
      </button>

      {/* Session header */}
      <div className="flex items-center gap-2 mb-4">
        <span className="text-sm font-semibold text-brand-light font-mono">Session #{shortId}</span>
        <span className="text-xs text-brand-shade3 font-mono">({trace.agent_name})</span>
      </div>

      {/* Summary bar */}
      <div className="flex items-center gap-4 mb-6 px-4 py-3 bg-brand-dark-surface border border-brand-shade3/15 rounded-card">
        <StatusBadge status={trace.status} />
        <span className="text-sm text-brand-shade2 font-mono">{totalSec}s</span>
        <span className="text-sm text-brand-shade2 font-mono">{trace.total_tokens.toLocaleString()} tokens</span>
        <span className="text-sm text-brand-shade2 font-mono">{trace.steps.length} steps</span>
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

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function InspectPage() {
  const { session } = useParams<{ schema: string; agent: string; session: string }>();
  const navigate = useNavigate();

  const [selectedSession, setSelectedSession] = useState<string | null>(session ?? null);

  const handleSelectSession = useCallback(
    (id: string) => {
      setSelectedSession(id);
      // Update URL without full navigation
      navigate(`/inspect/${id}`, { replace: true });
    },
    [navigate],
  );

  const handleBack = useCallback(() => {
    setSelectedSession(null);
    navigate('/inspect', { replace: true });
  }, [navigate]);

  if (selectedSession) {
    return <SessionDetail sessionId={selectedSession} onBack={handleBack} />;
  }

  return (
    <div>
      <h1 className="text-lg font-semibold text-brand-light mb-6 font-mono">Sessions</h1>
      <SessionList onSelectSession={handleSelectSession} />
    </div>
  );
}
