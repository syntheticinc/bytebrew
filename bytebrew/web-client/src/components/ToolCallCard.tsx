import { useState, useMemo } from 'react';

interface ToolCallCardProps {
  toolName: string;
  content: string;
  variant: 'call' | 'result';
}

function formatContent(content: string): string {
  if (!content) return '';
  try {
    const parsed = JSON.parse(content);
    return JSON.stringify(parsed, null, 2);
  } catch {
    return content;
  }
}

export function ToolCallCard({ toolName, content, variant }: ToolCallCardProps) {
  const [expanded, setExpanded] = useState(false);
  const icon = variant === 'call' ? '\u{1F527}' : '\u{1F4CB}';
  const label = variant === 'call' ? 'Tool Call' : 'Result';
  const formatted = useMemo(() => formatContent(content), [content]);

  // For collapsed: show a short summary
  const summary = useMemo(() => {
    if (!content) return null;
    if (variant === 'call') {
      try {
        const parsed = JSON.parse(content) as Record<string, string>;
        // Show key params inline
        const parts = Object.entries(parsed)
          .slice(0, 2)
          .map(([k, v]) => `${k}: ${String(v).slice(0, 40)}`);
        return parts.join(' · ');
      } catch {
        return content.slice(0, 60);
      }
    }
    // For result: show truncated text
    const text = content.length > 80 ? content.slice(0, 80) + '...' : content;
    return text;
  }, [content, variant]);

  return (
    <div
      className={`
        rounded-xl border px-3 py-2 text-sm transition-all cursor-pointer
        ${expanded ? 'border-brand-accent/30 bg-brand-dark/50' : 'border-brand-shade3/15'}
        hover:border-brand-shade3/30
      `}
      onClick={() => setExpanded(!expanded)}
    >
      <div className="flex items-center gap-2">
        <span className="text-xs">{icon}</span>
        <span className="font-medium text-brand-shade2">{label}:</span>
        <span className="text-brand-accent">{toolName}</span>
        {!expanded && summary && (
          <span className="ml-2 truncate text-xs text-brand-shade3 opacity-60">
            {summary}
          </span>
        )}
        <button className="ml-auto flex-shrink-0 rounded px-1.5 py-0.5 text-[10px] text-brand-shade3 transition-colors hover:bg-brand-shade3/10 hover:text-brand-light">
          {expanded ? 'collapse' : 'expand'}
        </button>
      </div>
      {expanded && formatted && (
        <pre className="mt-2 rounded-lg border border-brand-shade3/10 bg-brand-dark p-3 text-xs text-brand-shade2 max-h-60 overflow-auto whitespace-pre-wrap break-words">
          {formatted}
        </pre>
      )}
    </div>
  );
}
