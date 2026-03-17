const colorMap: Record<string, string> = {
  ok: 'bg-green-100 text-green-800',
  connected: 'bg-green-100 text-green-800',
  completed: 'bg-green-100 text-green-800',
  active: 'bg-green-100 text-green-800',
  pending: 'bg-yellow-100 text-yellow-800',
  connecting: 'bg-yellow-100 text-yellow-800',
  in_progress: 'bg-blue-100 text-blue-800',
  needs_input: 'bg-purple-100 text-purple-800',
  error: 'bg-red-100 text-red-800',
  failed: 'bg-red-100 text-red-800',
  cancelled: 'bg-gray-100 text-gray-800',
  disconnected: 'bg-gray-100 text-gray-800',
  escalated: 'bg-orange-100 text-orange-800',
};

interface StatusBadgeProps {
  status: string;
  className?: string;
}

export default function StatusBadge({ status, className = '' }: StatusBadgeProps) {
  const colors = colorMap[status.toLowerCase()] ?? 'bg-gray-100 text-gray-800';
  return (
    <span
      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${colors} ${className}`}
    >
      {status.replace(/_/g, ' ')}
    </span>
  );
}
