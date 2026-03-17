import { useEffect, useRef } from 'react';
import { api } from '../api/client';
import { useApi } from '../hooks/useApi';
import StatusBadge from '../components/StatusBadge';

export default function HealthPage() {
  const { data: health, loading, error, refetch } = useApi(() => api.health());
  const intervalRef = useRef<ReturnType<typeof setInterval> | undefined>(undefined);

  useEffect(() => {
    intervalRef.current = setInterval(refetch, 30000);
    return () => clearInterval(intervalRef.current);
  }, [refetch]);

  if (loading && !health) return <div className="text-gray-500">Loading health...</div>;
  if (error) return <div className="text-red-600">Error: {error}</div>;
  if (!health) return null;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Health</h1>
        <div className="flex items-center gap-3">
          <span className="text-xs text-gray-400">Auto-refreshes every 30s</span>
          <button
            onClick={refetch}
            className="px-4 py-2 text-sm text-gray-700 border border-gray-300 rounded-md hover:bg-gray-50"
          >
            Refresh
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Status */}
        <div className="bg-white rounded-lg shadow p-5">
          <div className="text-sm font-medium text-gray-500 mb-2">Status</div>
          <StatusBadge status={health.status} className="text-sm" />
        </div>

        {/* Version */}
        <div className="bg-white rounded-lg shadow p-5">
          <div className="text-sm font-medium text-gray-500 mb-2">Version</div>
          <div className="text-lg font-semibold text-gray-900">{health.version || 'dev'}</div>
        </div>

        {/* Uptime */}
        <div className="bg-white rounded-lg shadow p-5">
          <div className="text-sm font-medium text-gray-500 mb-2">Uptime</div>
          <div className="text-lg font-semibold text-gray-900">{health.uptime}</div>
        </div>

        {/* Agents */}
        <div className="bg-white rounded-lg shadow p-5">
          <div className="text-sm font-medium text-gray-500 mb-2">Agents</div>
          <div className="text-lg font-semibold text-gray-900">{health.agents_count}</div>
        </div>
      </div>
    </div>
  );
}
