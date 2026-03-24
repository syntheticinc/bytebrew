interface TierBadgeProps {
  tier: number;
  className?: string;
}

const TIER_CONFIG: Record<number, { label: string; colorClass: string }> = {
  1: {
    label: 'Tier 1 - Orchestrator',
    colorClass: 'bg-green-500/15 text-green-400',
  },
  2: {
    label: 'Tier 2 - Sub-agent',
    colorClass: 'bg-blue-500/15 text-blue-400',
  },
  3: {
    label: 'Tier 3 - Utility',
    colorClass: 'bg-brand-shade3/15 text-brand-shade3',
  },
};

export default function TierBadge({ tier, className = '' }: TierBadgeProps) {
  const config = TIER_CONFIG[tier];
  if (!config) return null;

  return (
    <span
      className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${config.colorClass} ${className}`}
    >
      {config.label}
    </span>
  );
}

export function CustomModelBadge({ className = '' }: { className?: string }) {
  return (
    <span
      className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-yellow-500/15 text-yellow-400 ${className}`}
    >
      Custom
    </span>
  );
}
