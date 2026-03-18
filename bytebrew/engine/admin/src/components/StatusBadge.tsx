const colorMap: Record<string, string> = {
  ok: 'bg-status-active/10 text-status-active',
  connected: 'bg-status-active/10 text-status-active',
  completed: 'bg-status-active/10 text-status-active',
  active: 'bg-status-active/10 text-status-active',
  pending: 'bg-yellow-100 text-yellow-800',
  connecting: 'bg-yellow-100 text-yellow-800',
  in_progress: 'bg-brand-accent/10 text-brand-accent',
  needs_input: 'bg-purple-100 text-purple-800',
  error: 'bg-red-100 text-red-800',
  failed: 'bg-red-100 text-red-800',
  cancelled: 'bg-brand-shade1/50 text-brand-shade3',
  disconnected: 'bg-brand-shade1/50 text-brand-shade3',
  escalated: 'bg-orange-100 text-orange-800',
};

interface StatusBadgeProps {
  status: string;
  className?: string;
}

export default function StatusBadge({ status, className = '' }: StatusBadgeProps) {
  const colors = colorMap[status.toLowerCase()] ?? 'bg-brand-shade1/50 text-brand-shade3';
  return (
    <span
      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${colors} ${className}`}
    >
      {status.replace(/_/g, ' ')}
    </span>
  );
}
