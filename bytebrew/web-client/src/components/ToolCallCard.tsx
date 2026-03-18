import { useState } from 'react';

interface ToolCallCardProps {
  toolName: string;
  content: string;
  variant: 'call' | 'result';
}

export function ToolCallCard({ toolName, content, variant }: ToolCallCardProps) {
  const [expanded, setExpanded] = useState(false);
  const icon = variant === 'call' ? '\u{1F527}' : '\u{1F4CB}';
  const label = variant === 'call' ? 'Tool Call' : 'Result';
  const preview = content.length > 80 ? content.slice(0, 80) + '...' : content;

  return (
    <div
      className={`
        rounded-xl border px-3 py-2 text-sm transition-colors cursor-pointer
        ${expanded ? 'border-brand-accent/40' : 'border-brand-shade3/20'}
        hover:border-brand-shade3/40
      `}
      onClick={() => setExpanded(!expanded)}
    >
      <div className="flex items-center gap-2">
        <span>{icon}</span>
        <span className="font-medium text-brand-shade2">{label}:</span>
        <span className="text-brand-accent">{toolName}</span>
        <span className="ml-auto text-brand-shade3 text-xs">
          {expanded ? 'collapse' : 'expand'}
        </span>
      </div>
      {expanded ? (
        <pre className="mt-2 whitespace-pre-wrap break-all text-xs text-brand-shade2 max-h-60 overflow-y-auto">
          {content}
        </pre>
      ) : (
        <p className="mt-1 text-xs text-brand-shade3 truncate">{preview}</p>
      )}
    </div>
  );
}
