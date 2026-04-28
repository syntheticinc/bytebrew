import { useState, useEffect } from 'react';
import { api } from '../api/client';
import type { UsageData, UsageMetric } from '../types';

function getBarColor(pct: number): string {
  if (pct >= 100) return 'bg-red-500';
  if (pct >= 80) return 'bg-amber-500';
  return 'bg-brand-accent';
}

function getPctColor(pct: number): string {
  if (pct >= 100) return 'text-red-400';
  if (pct >= 80) return 'text-amber-400';
  return 'text-brand-shade3';
}

function formatUsed(metric: UsageMetric): string {
  if (metric.unit === 'GB') return `${metric.used.toFixed(1)} ${metric.unit}`;
  return metric.used.toLocaleString();
}

function formatLimit(metric: UsageMetric): string {
  if (metric.limit < 0) return 'Unlimited';
  if (metric.unit === 'GB') return `${metric.limit} ${metric.unit}`;
  return metric.limit.toLocaleString();
}

function UsageBar({ metric }: { metric: UsageMetric }) {
  const pct = metric.limit > 0 ? Math.min(100, (metric.used / metric.limit) * 100) : 0;
  const barColor = getBarColor(pct);
  const usedLabel = formatUsed(metric);
  const limitLabel = formatLimit(metric);

  return (
    <div className="bg-brand-dark border border-brand-shade3/15 rounded-card p-4">
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm font-medium text-brand-light font-mono">{metric.label}</span>
        <span className="text-xs text-brand-shade3 font-mono">
          {usedLabel} / {limitLabel}
          {metric.unit && metric.unit !== 'GB' ? ` ${metric.unit}` : ''}
        </span>
      </div>
      <div className="h-3 bg-brand-dark-alt rounded-full overflow-hidden">
        <div
          className={`h-full rounded-full transition-all duration-500 ${barColor}`}
          style={{ width: `${pct}%` }}
        />
      </div>
      <div className="flex justify-end mt-1">
        <span className={`text-[10px] font-mono ${getPctColor(pct)}`}>
          {pct.toFixed(0)}%
        </span>
      </div>
    </div>
  );
}

const PLAN_COLORS: Record<string, string> = {
  Free: 'bg-brand-shade3/15 text-brand-shade3',
  Pro: 'bg-brand-accent/15 text-brand-accent',
  Business: 'bg-purple-500/15 text-purple-400',
  Enterprise: 'bg-amber-500/15 text-amber-400',
};

export default function UsageDashboard() {
  const [usage, setUsage] = useState<UsageData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api
      .getUsage()
      .then(setUsage)
      .catch(() => setUsage(null))
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <span className="text-sm text-brand-shade3 font-mono">Loading usage data...</span>
      </div>
    );
  }

  if (!usage) {
    return (
      <div className="flex items-center justify-center py-12">
        <span className="text-sm text-brand-shade3 font-mono">Usage data unavailable</span>
      </div>
    );
  }

  const planColor = PLAN_COLORS[usage.plan] ?? PLAN_COLORS['Free']!;
  const cycleStart = new Date(usage.billing_cycle_start).toLocaleDateString();
  const cycleEnd = new Date(usage.billing_cycle_end).toLocaleDateString();

  return (
    <div className="space-y-6">
      {/* Plan info */}
      <div className="flex items-center justify-between bg-brand-dark-surface border border-brand-shade3/10 rounded-card p-4">
        <div className="flex items-center gap-3">
          <span className={`px-3 py-1 rounded-full text-xs font-semibold font-mono ${planColor}`}>
            {usage.plan}
          </span>
          <span className="text-xs text-brand-shade3 font-mono">
            Billing cycle: {cycleStart} — {cycleEnd}
          </span>
        </div>
        <button
          onClick={() => {
            if (usage.stripe_portal_url) {
              window.open(usage.stripe_portal_url, '_blank');
            }
          }}
          className="px-4 py-1.5 bg-brand-accent hover:bg-brand-accent-hover text-brand-light rounded-btn text-xs font-medium font-mono transition-colors"
        >
          Manage Plan
        </button>
      </div>

      {/* Usage bars */}
      <div className="grid grid-cols-2 gap-4">
        {usage.metrics.map((m) => (
          <UsageBar key={m.name} metric={m} />
        ))}
      </div>
    </div>
  );
}
