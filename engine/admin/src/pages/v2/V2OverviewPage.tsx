import { Link } from 'react-router-dom';
import {
  v2OverviewEvents,
  v2Sessions,
  v2Schemas,
  v2Triggers,
  getSchemaById,
} from '../../mocks/v2';

function formatRelativeTime(iso: string) {
  const diff = Date.now() - new Date(iso).getTime();
  if (diff < 60_000) return `${Math.floor(diff / 1000)}s ago`;
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
  return `${Math.floor(diff / 86_400_000)}d ago`;
}

function Stat({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div className="bg-brand-dark-surface border border-brand-shade3/15 rounded-card px-5 py-4">
      <div className="text-[10px] uppercase tracking-[0.2em] text-brand-shade3 mb-2">{label}</div>
      <div className="text-2xl font-semibold text-brand-light leading-tight">{value}</div>
      {hint && <div className="text-[11px] text-brand-shade3 mt-1">{hint}</div>}
    </div>
  );
}

const eventColor: Record<string, string> = {
  trigger_fired: 'text-amber-300 border-amber-500/30',
  delegation: 'text-purple-300 border-purple-500/30',
  session_completed: 'text-emerald-300 border-emerald-500/30',
  agent_error: 'text-red-300 border-red-500/30',
  flow_entered: 'text-blue-300 border-blue-500/30',
};

export default function V2OverviewPage() {
  const activeSessions = v2Sessions.filter((s) => s.status === 'active');
  const completedSessions = v2Sessions.filter((s) => s.status === 'completed');
  const failedSessions = v2Sessions.filter((s) => s.status === 'failed');
  const sessionsToday = v2Schemas.reduce((sum, s) => sum + s.sessionsToday, 0);
  const enabledTriggers = v2Triggers.filter((t) => t.enabled).length;
  const finishedTotal = completedSessions.length + failedSessions.length;
  const successRate = finishedTotal > 0
    ? Math.round((completedSessions.length / finishedTotal) * 100)
    : null;

  return (
    <div className="max-w-[1200px] mx-auto">
      <div className="mb-8">
        <h1 className="text-2xl font-semibold text-brand-light">Overview</h1>
        <p className="text-sm text-brand-shade3 mt-1">
          Live picture of what your agents are doing right now.
        </p>
      </div>

      {/* Stats grid — derived from existing data only (sessions, schemas, triggers) */}
      <div className="grid grid-cols-4 gap-4 mb-6">
        <Stat
          label="Active Sessions"
          value={String(activeSessions.length)}
          hint={`${v2Schemas.length} schemas`}
        />
        <Stat
          label="Sessions Today"
          value={sessionsToday.toLocaleString()}
          hint="across all schemas"
        />
        <Stat
          label="Enabled Triggers"
          value={`${enabledTriggers} / ${v2Triggers.length}`}
          hint={`${v2Triggers.length - enabledTriggers} paused`}
        />
        <Stat
          label="Success Rate"
          value={successRate !== null ? `${successRate}%` : '—'}
          hint={`${completedSessions.length} ok · ${failedSessions.length} failed`}
        />
      </div>

      <div className="grid grid-cols-[1.4fr_1fr] gap-6">
        {/* Live sessions */}
        <div className="bg-brand-dark-surface border border-brand-shade3/15 rounded-card">
          <div className="flex items-center justify-between px-5 py-3 border-b border-brand-shade3/10">
            <h2 className="text-sm font-semibold text-brand-light">Live Sessions</h2>
            <span className="flex items-center gap-1.5 text-[11px] text-emerald-400">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
              {activeSessions.length} active
            </span>
          </div>
          <div className="divide-y divide-brand-shade3/10">
            {activeSessions.length === 0 && (
              <div className="px-5 py-6 text-center text-[12px] text-brand-shade3">
                No live sessions right now. Create your first{' '}
                <Link to="/v2/schemas" className="text-brand-accent hover:underline">
                  schema
                </Link>{' '}
                to get started.
              </div>
            )}
            {activeSessions.map((s) => {
              const schema = getSchemaById(s.schemaId);
              return (
                <Link
                  key={s.id}
                  to={`/v2/schemas/${s.schemaId}?session=${s.id}`}
                  className="flex items-center gap-3 px-5 py-3 hover:bg-brand-shade3/5 transition-colors group"
                >
                  <span className="font-mono text-[11px] text-brand-shade3 shrink-0">{s.id}</span>
                  <div className="min-w-0 flex-1">
                    <div className="text-[13px] text-brand-light truncate">{s.title}</div>
                    <div className="text-[10px] text-brand-shade3 mt-0.5">
                      {schema?.name} · {s.participantAgentIds.length} agents · started {formatRelativeTime(s.startedAt)}
                    </div>
                  </div>
                  <span className="text-[11px] text-brand-shade3 group-hover:text-brand-accent transition-colors shrink-0">
                    Debug →
                  </span>
                </Link>
              );
            })}
            {activeSessions.length === 0 && (
              <div className="px-5 py-6 text-center text-[12px] text-brand-shade3">
                No active sessions right now.
              </div>
            )}
          </div>
        </div>

        {/* Events feed */}
        <div className="bg-brand-dark-surface border border-brand-shade3/15 rounded-card">
          <div className="px-5 py-3 border-b border-brand-shade3/10">
            <h2 className="text-sm font-semibold text-brand-light">Recent Events</h2>
          </div>
          <div className="divide-y divide-brand-shade3/10 max-h-[400px] overflow-y-auto">
            {v2OverviewEvents.map((e, idx) => (
              <div key={idx} className="px-5 py-2.5 hover:bg-brand-shade3/5 transition-colors">
                <div className="flex items-center gap-2 mb-1">
                  <span className={`text-[9px] uppercase tracking-wider border rounded px-1.5 py-0.5 ${eventColor[e.kind] ?? 'text-brand-shade3 border-brand-shade3/30'}`}>
                    {e.kind.replace('_', ' ')}
                  </span>
                  <span className="text-[10px] text-brand-shade3 font-mono ml-auto">
                    {formatRelativeTime(e.timestamp)}
                  </span>
                </div>
                <div className="text-[12px] text-brand-shade2 leading-snug">{e.summary}</div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Schemas quick access */}
      <div className="mt-6">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-semibold text-brand-light">Schemas</h2>
          <Link to="/v2/schemas" className="text-[11px] text-brand-shade3 hover:text-brand-accent transition-colors">
            View all →
          </Link>
        </div>
        <div className="grid grid-cols-3 gap-3">
          {v2Schemas.map((s) => (
            <Link
              key={s.id}
              to={`/v2/schemas/${s.id}`}
              className="bg-brand-dark-surface border border-brand-shade3/15 rounded-card px-4 py-3 hover:border-brand-shade3/35 transition-colors"
            >
              <div className="text-[13px] font-semibold text-brand-light truncate">{s.name}</div>
              <div className="flex items-center gap-2 mt-2 text-[10px] text-brand-shade3">
                <span>{s.agentIds.length} agents</span>
                <span className="text-brand-shade3/40">·</span>
                <span>{s.sessionsToday} today</span>
                <span className="text-brand-shade3/40">·</span>
                <span className={s.activeSessions > 0 ? 'text-emerald-400' : 'text-brand-shade3'}>
                  {s.activeSessions > 0 ? `${s.activeSessions} active` : 'idle'}
                </span>
              </div>
            </Link>
          ))}
        </div>
      </div>
    </div>
  );
}
