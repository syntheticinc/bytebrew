function formatTokens(n: number): string {
  if (n >= 1000) return `${(n / 1000).toFixed(n % 1000 === 0 ? 0 : 1)}K`;
  return String(n);
}

function usageColor(pct: number): string {
  if (pct >= 85) return 'bg-red-500';
  if (pct >= 60) return 'bg-yellow-500';
  return 'bg-emerald-500';
}

interface ContextUsageBarProps {
  maxContextTokens: number | null;
  totalTokens?: number | null;
}

export default function ContextUsageBar({ maxContextTokens, totalTokens }: ContextUsageBarProps) {
  if (!maxContextTokens) return null;

  const pct = totalTokens ? Math.min(100, (totalTokens / maxContextTokens) * 100) : 0;

  return (
    <div className="px-3 py-1 flex items-center gap-2 border-t border-brand-shade3/10 flex-shrink-0">
      <div className="flex-1 h-1 bg-brand-shade3/10 rounded-full overflow-hidden">
        {pct > 0 && (
          <div
            className={`h-full rounded-full transition-all duration-300 ${usageColor(pct)}`}
            style={{ width: `${pct}%` }}
          />
        )}
      </div>
      <span className="text-[10px] text-brand-shade3 whitespace-nowrap">
        {totalTokens ? formatTokens(totalTokens) : '\u2014'} / {formatTokens(maxContextTokens)} tokens
      </span>
    </div>
  );
}
