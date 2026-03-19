const colorMap: Record<string, string> = {
  ok: 'bg-status-active/15 text-status-active',
  connected: 'bg-status-active/15 text-status-active',
  completed: 'bg-status-active/15 text-status-active',
  active: 'bg-status-active/15 text-status-active',
  pending: 'bg-yellow-500/15 text-yellow-400',
  connecting: 'bg-yellow-500/15 text-yellow-400',
  in_progress: 'bg-brand-accent/15 text-brand-accent',
  needs_input: 'bg-purple-500/15 text-purple-400',
  error: 'bg-red-500/15 text-red-400',
  failed: 'bg-red-500/15 text-red-400',
  cancelled: 'bg-brand-shade3/15 text-brand-shade3',
  disconnected: 'bg-brand-shade3/15 text-brand-shade3',
  escalated: 'bg-orange-500/15 text-orange-400',
};

interface StatusBadgeProps {
  status: string;
  className?: string;
}

export default function StatusBadge({ status, className = '' }: StatusBadgeProps) {
  const colors = colorMap[status.toLowerCase()] ?? 'bg-brand-shade3/15 text-brand-shade3';
  return (
    <span
      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${colors} ${className}`}
    >
      {status.replace(/_/g, ' ')}
    </span>
  );
}
