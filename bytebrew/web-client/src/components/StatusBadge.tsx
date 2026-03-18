interface StatusBadgeProps {
  status: string;
}

const STATUS_COLORS: Record<string, string> = {
  running: 'bg-green-500',
  completed: 'bg-brand-shade3',
  failed: 'bg-red-500',
  cancelled: 'bg-yellow-500',
  pending: 'bg-blue-500',
};

export function StatusBadge({ status }: StatusBadgeProps) {
  const dotColor = STATUS_COLORS[status.toLowerCase()] ?? 'bg-brand-shade3';

  return (
    <span className="inline-flex items-center gap-1.5 text-xs">
      <span className={`h-1.5 w-1.5 rounded-full ${dotColor}`} />
      <span className="text-brand-shade2 capitalize">{status}</span>
    </span>
  );
}
