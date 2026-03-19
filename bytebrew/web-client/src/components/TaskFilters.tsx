import { useState, useEffect } from 'react';
import { api } from '../api/client';
import type { AgentInfo } from '../types';

export interface TaskFilterValues {
  status: string;
  agent: string;
  source: string;
}

interface TaskFiltersProps {
  filters: TaskFilterValues;
  onChange: (filters: TaskFilterValues) => void;
}

const STATUS_OPTIONS = ['All', 'Pending', 'Running', 'Completed', 'Failed', 'Cancelled'];
const SOURCE_OPTIONS = ['All', 'API', 'Cron', 'Webhook', 'Dashboard'];

export function TaskFilters({ filters, onChange }: TaskFiltersProps) {
  const [agents, setAgents] = useState<AgentInfo[]>([]);

  useEffect(() => {
    api
      .listAgents()
      .then((data) => setAgents(data.filter((a) => a.name)))
      .catch(() => {
        // silently ignore — filters still usable without agent list
      });
  }, []);

  const selectClass =
    'rounded-btn border border-brand-shade3/20 bg-brand-dark px-3 py-1.5 text-xs text-brand-light outline-none transition-colors focus:border-brand-accent/50';

  return (
    <div className="mb-4 flex flex-wrap items-center gap-3">
      {/* Status */}
      <div className="flex items-center gap-1.5">
        <label className="text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
          Status
        </label>
        <select
          value={filters.status}
          onChange={(e) => onChange({ ...filters, status: e.target.value })}
          className={selectClass}
        >
          {STATUS_OPTIONS.map((s) => (
            <option key={s} value={s === 'All' ? '' : s.toLowerCase()}>
              {s}
            </option>
          ))}
        </select>
      </div>

      {/* Agent */}
      <div className="flex items-center gap-1.5">
        <label className="text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
          Agent
        </label>
        <select
          value={filters.agent}
          onChange={(e) => onChange({ ...filters, agent: e.target.value })}
          className={selectClass}
        >
          <option value="">All</option>
          {agents.map((a) => (
            <option key={a.name} value={a.name}>
              {a.name}
            </option>
          ))}
        </select>
      </div>

      {/* Source */}
      <div className="flex items-center gap-1.5">
        <label className="text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
          Source
        </label>
        <select
          value={filters.source}
          onChange={(e) => onChange({ ...filters, source: e.target.value })}
          className={selectClass}
        >
          {SOURCE_OPTIONS.map((s) => (
            <option key={s} value={s === 'All' ? '' : s.toLowerCase()}>
              {s}
            </option>
          ))}
        </select>
      </div>

      {/* Reset */}
      {(filters.status || filters.agent || filters.source) && (
        <button
          onClick={() => onChange({ status: '', agent: '', source: '' })}
          className="text-[10px] text-brand-shade3 transition-colors hover:text-brand-accent"
        >
          Reset
        </button>
      )}
    </div>
  );
}
