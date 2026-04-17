import { useCallback, useEffect, useState } from 'react';
import { api } from '../api/client';
import type { CircuitBreakerState, DeadLetterEntry, StuckAgentEntry } from '../types';

const POLL_INTERVAL_MS = 5000;

function formatRelative(iso?: string | null): string {
  if (!iso) return '-';
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return '-';
  const delta = Math.max(0, Date.now() - then);
  const sec = Math.floor(delta / 1000);
  if (sec < 60) return `${sec}s ago`;
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min}m ago`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr}h ago`;
  return `${Math.floor(hr / 24)}d ago`;
}

function formatDuration(ms?: number): string {
  if (!ms || ms <= 0) return '-';
  if (ms < 1000) return `${ms}ms`;
  const sec = Math.floor(ms / 1000);
  if (sec < 60) return `${sec}s`;
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min}m ${sec % 60}s`;
  return `${Math.floor(min / 60)}h ${min % 60}m`;
}

function StateBadge({ state }: { state: CircuitBreakerState['state'] }) {
  const map: Record<CircuitBreakerState['state'], string> = {
    closed: 'bg-status-active/15 text-status-active',
    open: 'bg-red-500/15 text-red-400',
    half_open: 'bg-amber-500/15 text-amber-400',
  };
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${map[state]}`}>
      {state.replace('_', ' ')}
    </span>
  );
}

export default function ResiliencePage() {
  const [breakers, setBreakers] = useState<CircuitBreakerState[]>([]);
  const [deadLetters, setDeadLetters] = useState<DeadLetterEntry[]>([]);
  const [stuckAgents, setStuckAgents] = useState<StuckAgentEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [toast, setToast] = useState<string | null>(null);

  const fetchAll = useCallback(async () => {
    try {
      const [cb, dl, sa] = await Promise.all([
        api.listCircuitBreakers(),
        api.listDeadLetterTasks(),
        api.listStuckAgents(),
      ]);
      setBreakers(cb);
      setDeadLetters(dl);
      setStuckAgents(sa);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchAll();
    const id = setInterval(fetchAll, POLL_INTERVAL_MS);
    return () => clearInterval(id);
  }, [fetchAll]);

  useEffect(() => {
    if (!toast) return;
    const id = setTimeout(() => setToast(null), 4000);
    return () => clearTimeout(id);
  }, [toast]);

  async function handleReset(name: string) {
    try {
      await api.resetCircuitBreaker(name);
      setToast(`Circuit breaker ${name} reset`);
      fetchAll();
    } catch (e) {
      setToast(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-brand-light">Resilience</h1>
          <p className="text-sm text-brand-shade3 mt-1">
            Platform observability: circuit breakers, dead-letter queue, stuck agents. Polls every {POLL_INTERVAL_MS / 1000}s.
          </p>
        </div>
        <button
          onClick={fetchAll}
          className="px-4 py-2 text-sm text-brand-shade2 border border-brand-shade3/30 rounded-btn hover:bg-brand-dark-alt hover:text-brand-light transition-colors"
        >
          Refresh
        </button>
      </div>

      {error && (
        <div className="mb-4 p-3 bg-red-500/10 border border-red-500/30 rounded-btn text-sm text-red-400">
          Error: {error}
        </div>
      )}

      {toast && (
        <div className="mb-4 p-3 bg-brand-accent/10 border border-brand-accent/30 rounded-btn text-sm text-brand-light">
          {toast}
        </div>
      )}

      <Section title="Circuit Breakers" hint="One breaker per MCP server. Opens after consecutive failures; half-open probes recovery.">
        {loading && breakers.length === 0 ? (
          <Loading />
        ) : breakers.length === 0 ? (
          <Empty message="No circuit breakers registered." />
        ) : (
          <table className="w-full text-sm">
            <thead className="text-left text-xs text-brand-shade3 border-b border-brand-shade3/15">
              <tr>
                <th className="px-4 py-2 font-medium">MCP Server</th>
                <th className="px-4 py-2 font-medium">State</th>
                <th className="px-4 py-2 font-medium">Failures</th>
                <th className="px-4 py-2 font-medium">Last failure</th>
                <th className="px-4 py-2 font-medium text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {breakers.map((b) => (
                <tr key={b.name} className="border-b border-brand-shade3/10 last:border-0">
                  <td className="px-4 py-3 text-brand-light font-mono text-xs">{b.name}</td>
                  <td className="px-4 py-3"><StateBadge state={b.state} /></td>
                  <td className="px-4 py-3 text-brand-shade2">{b.failure_count}</td>
                  <td className="px-4 py-3 text-brand-shade3 text-xs">{formatRelative(b.last_failure)}</td>
                  <td className="px-4 py-3 text-right">
                    {b.state !== 'closed' && (
                      <button
                        onClick={() => handleReset(b.name)}
                        className="px-3 py-1 text-xs text-brand-shade2 border border-brand-shade3/30 rounded-btn hover:bg-brand-dark hover:text-brand-light transition-colors"
                      >
                        Reset
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Section>

      <Section title="Dead Letter Queue" hint="Tasks that exceeded their runtime budget and were abandoned by the task runner.">
        {loading && deadLetters.length === 0 ? (
          <Loading />
        ) : deadLetters.length === 0 ? (
          <Empty message="No dead-letter tasks." />
        ) : (
          <table className="w-full text-sm">
            <thead className="text-left text-xs text-brand-shade3 border-b border-brand-shade3/15">
              <tr>
                <th className="px-4 py-2 font-medium">Task</th>
                <th className="px-4 py-2 font-medium">Agent</th>
                <th className="px-4 py-2 font-medium">Reason</th>
                <th className="px-4 py-2 font-medium">Elapsed</th>
                <th className="px-4 py-2 font-medium">Moved</th>
              </tr>
            </thead>
            <tbody>
              {deadLetters.map((t) => (
                <tr key={t.task_id} className="border-b border-brand-shade3/10 last:border-0">
                  <td className="px-4 py-3 text-brand-light font-mono text-xs" title={t.task_id}>
                    {t.task_id.length > 8 ? `${t.task_id.slice(0, 8)}…` : t.task_id}
                  </td>
                  <td className="px-4 py-3 text-brand-shade2 text-xs">{t.agent_name || t.agent_id}</td>
                  <td className="px-4 py-3 text-brand-shade3 text-xs truncate max-w-xs" title={t.last_error || t.reason}>
                    {t.reason || '-'}
                  </td>
                  <td className="px-4 py-3 text-brand-shade3 text-xs">{formatDuration(t.elapsed_ms)}</td>
                  <td className="px-4 py-3 text-brand-shade3 text-xs">{formatRelative(t.moved_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Section>

      <Section title="Stuck Agents" hint="Agents whose last heartbeat is older than the stuck threshold (~2x heartbeat interval).">
        {loading && stuckAgents.length === 0 ? (
          <Loading />
        ) : stuckAgents.length === 0 ? (
          <Empty message="No stuck agents." />
        ) : (
          <table className="w-full text-sm">
            <thead className="text-left text-xs text-brand-shade3 border-b border-brand-shade3/15">
              <tr>
                <th className="px-4 py-2 font-medium">Agent</th>
                <th className="px-4 py-2 font-medium">Type</th>
                <th className="px-4 py-2 font-medium">Current step</th>
                <th className="px-4 py-2 font-medium">Last heartbeat</th>
                <th className="px-4 py-2 font-medium">Elapsed</th>
                <th className="px-4 py-2 font-medium">Status</th>
              </tr>
            </thead>
            <tbody>
              {stuckAgents.map((a) => (
                <tr key={a.agent_id} className="border-b border-brand-shade3/10 last:border-0">
                  <td className="px-4 py-3 text-brand-light text-xs">{a.agent_name || a.agent_id}</td>
                  <td className="px-4 py-3 text-brand-shade2 text-xs">{a.agent_type}</td>
                  <td className="px-4 py-3 text-brand-shade3 text-xs">{a.current_step || '-'}</td>
                  <td className="px-4 py-3 text-brand-shade3 text-xs">{formatRelative(a.last_heartbeat)}</td>
                  <td className="px-4 py-3 text-brand-shade3 text-xs">{formatDuration(a.elapsed_ms)}</td>
                  <td className="px-4 py-3">
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-amber-500/15 text-amber-400">
                      {a.status}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Section>
    </div>
  );
}

function Section({ title, hint, children }: { title: string; hint?: string; children: React.ReactNode }) {
  return (
    <div className="mb-8 bg-brand-dark-alt rounded-card border border-brand-shade3/15">
      <div className="px-4 py-3 border-b border-brand-shade3/15">
        <h2 className="text-sm font-semibold text-brand-light">{title}</h2>
        {hint && <p className="text-xs text-brand-shade3 mt-0.5">{hint}</p>}
      </div>
      {children}
    </div>
  );
}

function Loading() {
  return <div className="px-4 py-6 text-xs text-brand-shade3">Loading…</div>;
}

function Empty({ message }: { message: string }) {
  return <div className="px-4 py-6 text-xs text-brand-shade3">{message}</div>;
}
