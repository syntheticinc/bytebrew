import { useEffect, useRef, useState } from 'react';
import { api } from '../api/client';
import { useApi } from '../hooks/useApi';
import StatusBadge from '../components/StatusBadge';

const iconClass = "w-4 h-4 text-brand-shade3";

const statIcons = {
  status: (
    <svg className={iconClass} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M22 12h-4l-3 9L9 3l-3 9H2" />
    </svg>
  ),
  version: (
    <svg className={iconClass} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="16 18 22 12 16 6" />
      <polyline points="8 6 2 12 8 18" />
    </svg>
  ),
  uptime: (
    <svg className={iconClass} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="10" />
      <polyline points="12 6 12 12 16 14" />
    </svg>
  ),
  agents: (
    <svg className={iconClass} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <rect x="4" y="4" width="16" height="16" rx="2" />
      <rect x="9" y="9" width="6" height="6" rx="1" />
      <path d="M9 1v3M15 1v3M9 20v3M15 20v3M20 9h3M20 15h3M1 9h3M1 15h3" />
    </svg>
  ),
  database: (
    <svg className={iconClass} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <ellipse cx="12" cy="5" rx="9" ry="3" />
      <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3" />
      <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5" />
    </svg>
  ),
};

function statusColor(status: string): string {
  switch (status?.toLowerCase()) {
    case 'ok':
    case 'healthy':
    case 'active':
      return 'bg-status-active';
    case 'warning':
    case 'degraded':
      return 'bg-amber-400';
    default:
      return 'bg-status-attention';
  }
}

export default function HealthPage() {
  const { data: health, loading, error, refetch } = useApi(() => api.health());
  const intervalRef = useRef<ReturnType<typeof setInterval> | undefined>(undefined);
  const [updateDismissed, setUpdateDismissed] = useState(false);

  useEffect(() => {
    intervalRef.current = setInterval(refetch, 30000);
    return () => clearInterval(intervalRef.current);
  }, [refetch]);

  if (loading && !health) return <div className="text-brand-shade3">Loading health...</div>;
  if (error) return <div className="text-red-400">Error: {error}</div>;
  if (!health) return null;

  return (
    <div>
      {health.update_available && !updateDismissed && (
        <div className="mb-6 flex items-start gap-3 rounded-card border border-amber-500/30 bg-amber-500/8 px-5 py-4 relative">
          <svg className="w-5 h-5 text-amber-400 mt-0.5 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
            <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
            <line x1="12" y1="9" x2="12" y2="13" />
            <line x1="12" y1="17" x2="12.01" y2="17" />
          </svg>
          <div className="text-sm flex-1">
            <p className="text-amber-300 font-semibold">
              Update available: v{health.update_available}
            </p>
            <code className="text-amber-400/70 text-xs mt-1.5 block font-mono">
              docker pull bytebrew/engine:{health.update_available}
            </code>
          </div>
          <button
            onClick={() => setUpdateDismissed(true)}
            className="text-amber-400/50 hover:text-amber-400 transition-colors p-1"
          >
            <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>
      )}

      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-brand-light">Health</h1>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <span className="w-1.5 h-1.5 rounded-full bg-status-active animate-pulse-dot" />
            <span className="text-xs text-brand-shade3">Auto-refreshes every 30s</span>
          </div>
          <button
            onClick={refetch}
            className="px-4 py-2 text-sm text-brand-shade2 border border-brand-shade3/20 rounded-btn hover:bg-brand-dark-alt hover:text-brand-light hover:border-brand-shade3/30 transition-all duration-150"
          >
            Refresh
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Status */}
        <div className="bg-brand-dark-alt rounded-card border border-brand-shade3/10 p-5 relative overflow-hidden">
          <div className={`absolute top-0 left-0 right-0 h-0.5 ${statusColor(health.status)}`} />
          <div className="flex items-center gap-2 mb-3">
            {statIcons.status}
            <span className="text-[11px] font-medium text-brand-shade3 uppercase tracking-wider">Status</span>
          </div>
          <StatusBadge status={health.status} className="text-sm" />
        </div>

        {/* Version */}
        <div className="bg-brand-dark-alt rounded-card border border-brand-shade3/10 p-5 relative overflow-hidden">
          <div className="absolute top-0 left-0 right-0 h-0.5 bg-brand-shade3/30" />
          <div className="flex items-center gap-2 mb-3">
            {statIcons.version}
            <span className="text-[11px] font-medium text-brand-shade3 uppercase tracking-wider">Version</span>
          </div>
          <div className="text-2xl font-bold text-brand-light">{health.version || 'dev'}</div>
        </div>

        {/* Uptime */}
        <div className="bg-brand-dark-alt rounded-card border border-brand-shade3/10 p-5 relative overflow-hidden">
          <div className="absolute top-0 left-0 right-0 h-0.5 bg-brand-shade3/30" />
          <div className="flex items-center gap-2 mb-3">
            {statIcons.uptime}
            <span className="text-[11px] font-medium text-brand-shade3 uppercase tracking-wider">Uptime</span>
          </div>
          <div className="text-2xl font-bold text-brand-light">{health.uptime}</div>
        </div>

        {/* Agents */}
        <div className="bg-brand-dark-alt rounded-card border border-brand-shade3/10 p-5 relative overflow-hidden">
          <div className="absolute top-0 left-0 right-0 h-0.5 bg-brand-shade3/30" />
          <div className="flex items-center gap-2 mb-3">
            {statIcons.agents}
            <span className="text-[11px] font-medium text-brand-shade3 uppercase tracking-wider">Agents</span>
          </div>
          <div className="text-2xl font-bold text-brand-light">{health.agents_count}</div>
        </div>
      </div>

      {/* Database status (if available from API) */}
      {(() => {
        const db = (health as unknown as Record<string, unknown>).database as { status?: string } | undefined;
        if (!db) return null;
        return (
          <div className="mt-4">
            <div className="bg-brand-dark-alt rounded-card border border-brand-shade3/10 p-5 relative overflow-hidden">
              <div className={`absolute top-0 left-0 right-0 h-0.5 ${statusColor(db.status || 'ok')}`} />
              <div className="flex items-center gap-2 mb-3">
                {statIcons.database}
                <span className="text-[11px] font-medium text-brand-shade3 uppercase tracking-wider">Database</span>
              </div>
              <div className="text-lg font-semibold text-brand-light">{db.status || 'Connected'}</div>
            </div>
          </div>
        );
      })()}
    </div>
  );
}
