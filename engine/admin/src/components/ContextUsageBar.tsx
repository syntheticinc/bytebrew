function formatTokens(n: number): string {
  if (n >= 1000) return `${(n / 1000).toFixed(n % 1000 === 0 ? 0 : 1)}K`;
  return String(n);
}

interface ContextUsageBarProps {
  maxContextTokens: number | null;
}

export default function ContextUsageBar({ maxContextTokens }: ContextUsageBarProps) {
  if (!maxContextTokens) return null;

  return (
    <div className="px-3 py-1 flex items-center gap-2 border-t border-brand-shade3/10 flex-shrink-0">
      <div className="flex-1 h-1 bg-brand-shade3/10 rounded-full overflow-hidden" />
      <span className="text-[10px] text-brand-shade3 whitespace-nowrap">
        &mdash; / {formatTokens(maxContextTokens)} tokens
      </span>
    </div>
  );
}
