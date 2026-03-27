interface TierBadgeProps {
  tier: string;
}

const TIER_STYLES: Record<string, string> = {
  trial: 'bg-border text-text-secondary',
  personal: 'bg-brand-accent text-white',
  teams: 'bg-purple-600 text-white',
  ce: 'bg-brand-accent/10 text-brand-accent border border-brand-accent/20',
  ee: 'bg-purple-500/10 text-purple-400 border border-purple-500/20',
};

const TIER_LABELS: Record<string, string> = {
  trial: 'Trial',
  personal: 'Personal',
  teams: 'Teams',
  ce: 'Community',
  ee: 'Enterprise',
};

export function TierBadge({ tier }: TierBadgeProps) {
  const style = TIER_STYLES[tier] ?? 'bg-border text-text-secondary';
  const label = TIER_LABELS[tier] ?? tier;

  return (
    <span
      className={`inline-flex items-center rounded-full px-3 py-1 text-sm font-medium ${style}`}
    >
      {label}
    </span>
  );
}
