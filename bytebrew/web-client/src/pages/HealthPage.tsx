import { useState, useEffect, useRef } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api/client';
import { useAuth } from '../hooks/useAuth';
import type { HealthResponse } from '../types';

export function HealthPage() {
  const { logout } = useAuth();
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const fetchHealth = () => {
    api
      .health()
      .then((data) => {
        setHealth(data);
        setError('');
        setLoading(false);
        setLastUpdated(new Date());
      })
      .catch((err: Error) => {
        setError(err.message);
        setLoading(false);
        setLastUpdated(new Date());
      });
  };

  useEffect(() => {
    fetchHealth();
    intervalRef.current = setInterval(fetchHealth, 10_000);
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, []);

  const isHealthy = health?.status === 'ok' || health?.status === 'healthy';
  const dbHealthy = !health?.database || health.database === 'connected' || health.database === 'ok';

  return (
    <div className="min-h-screen bg-brand-dark">
      {/* Header */}
      <header className="border-b border-brand-shade3/15 px-6 py-4">
        <div className="mx-auto flex max-w-5xl items-center justify-between">
          <Link to="/chat" className="text-sm font-bold text-brand-light">
            Byte<span className="text-brand-accent">Brew</span>
          </Link>
          <nav className="flex items-center gap-4">
            <Link to="/chat" className="text-xs text-brand-shade3 hover:text-brand-light">
              Chat
            </Link>
            <Link to="/agents" className="text-xs text-brand-shade3 hover:text-brand-light">
              Agents
            </Link>
            <Link to="/tasks" className="text-xs text-brand-shade3 hover:text-brand-light">
              Tasks
            </Link>
            <button onClick={logout} className="text-xs text-brand-shade3 hover:text-brand-light">
              Logout
            </button>
          </nav>
        </div>
      </header>

      <div className="mx-auto max-w-5xl px-6 py-8">
        <div className="mb-6 flex items-center justify-between">
          <h1 className="text-xl font-bold text-brand-light">Health</h1>
          {lastUpdated && (
            <span className="text-[10px] text-brand-shade3">
              Last updated: {lastUpdated.toLocaleTimeString()} (auto-refresh 10s)
            </span>
          )}
        </div>

        {error && !health && (
          <div className="mb-4 rounded-btn border border-red-500/30 bg-red-500/10 px-4 py-2 text-sm text-red-400">
            {error}
          </div>
        )}

        {loading ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {[1, 2, 3, 4].map((i) => (
              <div key={i} className="h-28 animate-pulse rounded-card bg-brand-dark-alt" />
            ))}
          </div>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {/* Status Card */}
            <div className="rounded-card border border-brand-shade3/15 bg-brand-dark-alt p-5">
              <label className="mb-3 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                Status
              </label>
              <div className="flex items-center gap-3">
                <span
                  className={`h-4 w-4 rounded-full ${
                    error
                      ? 'bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.5)]'
                      : isHealthy
                        ? 'bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)]'
                        : 'bg-amber-500 shadow-[0_0_8px_rgba(245,158,11,0.5)]'
                  }`}
                />
                <span className={`text-lg font-bold ${
                  error ? 'text-red-400' : isHealthy ? 'text-emerald-400' : 'text-amber-400'
                }`}>
                  {error ? 'Error' : health?.status ?? 'Unknown'}
                </span>
              </div>
            </div>

            {/* Version Card */}
            <div className="rounded-card border border-brand-shade3/15 bg-brand-dark-alt p-5">
              <label className="mb-3 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                Version
              </label>
              <p className="text-lg font-bold text-brand-light">{health?.version ?? '-'}</p>
            </div>

            {/* Uptime Card */}
            <div className="rounded-card border border-brand-shade3/15 bg-brand-dark-alt p-5">
              <label className="mb-3 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                Uptime
              </label>
              <p className="text-lg font-bold text-brand-light">{health?.uptime ?? '-'}</p>
            </div>

            {/* Agents Count Card */}
            <div className="rounded-card border border-brand-shade3/15 bg-brand-dark-alt p-5">
              <label className="mb-3 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                Agents
              </label>
              <p className="text-lg font-bold text-brand-accent">{health?.agents_count ?? '-'}</p>
            </div>
          </div>
        )}

        {/* Database Card (full-width, below grid) */}
        {health?.database !== undefined && (
          <div className="mt-4 rounded-card border border-brand-shade3/15 bg-brand-dark-alt p-5">
            <label className="mb-3 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
              Database
            </label>
            <div className="flex items-center gap-3">
              <span
                className={`h-3 w-3 rounded-full ${
                  dbHealthy
                    ? 'bg-emerald-500 shadow-[0_0_6px_rgba(16,185,129,0.5)]'
                    : 'bg-red-500 shadow-[0_0_6px_rgba(239,68,68,0.5)]'
                }`}
              />
              <span className={`text-sm font-medium ${dbHealthy ? 'text-emerald-400' : 'text-red-400'}`}>
                {health.database}
              </span>
            </div>
          </div>
        )}

        {/* Error banner (when health loaded but also has error on refresh) */}
        {error && health && (
          <div className="mt-4 rounded-btn border border-amber-500/30 bg-amber-500/10 px-4 py-2 text-sm text-amber-400">
            Refresh failed: {error}
          </div>
        )}
      </div>
    </div>
  );
}
