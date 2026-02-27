interface TierBadgeProps {
  tier: string;
}

const TIER_STYLES: Record<string, string> = {
  trial: 'bg-gray-700 text-gray-200',
  personal: 'bg-indigo-600 text-white',
  teams: 'bg-purple-600 text-white',
};

const TIER_LABELS: Record<string, string> = {
  trial: 'Trial',
  personal: 'Personal',
  teams: 'Teams',
};

export function TierBadge({ tier }: TierBadgeProps) {
  const style = TIER_STYLES[tier] ?? 'bg-gray-600 text-gray-300';
  const label = TIER_LABELS[tier] ?? tier;

  return (
    <span
      className={`inline-flex items-center rounded-full px-3 py-1 text-sm font-medium ${style}`}
    >
      {label}
    </span>
  );
}
