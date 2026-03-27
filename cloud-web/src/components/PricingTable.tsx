import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { getPricing, formatPrice, type PricingData } from '../api/pricing';

interface PricingTableProps {
  onSelectPlan: (plan: string, period: string) => void;
  currentTier?: string;
}

type Period = 'monthly' | 'annual';

interface PlanInfo {
  id: string;
  name: string;
  description: string;
  features: string[];
  highlighted?: boolean;
  isPerSeat?: boolean;
}

const PLANS: PlanInfo[] = [
  {
    id: 'trial',
    name: 'Trial',
    description: '14 days free, credit card required',
    features: [
      'Full agent functionality',
      'Unlimited proxy steps',
      'Smart routing (GLM-5 / GLM-4.7)',
      'BYOK optional',
      '1 seat',
    ],
  },
  {
    id: 'personal',
    name: 'Personal',
    description: 'For individual developers',
    features: [
      'Full agent functionality',
      '300 proxy steps / month',
      'Smart routing (GLM-5 / GLM-4.7)',
      'BYOK unlimited',
      '1 seat',
      'Relay support',
    ],
    highlighted: true,
  },
  {
    id: 'teams',
    name: 'Teams',
    description: 'For teams and organizations',
    features: [
      'Everything in Personal',
      '300 proxy steps / user / month',
      'Auto-scaling seats',
      'Enterprise SSO (SAML/OIDC)',
      'Admin panel & audit log',
      'Relay support',
    ],
    isPerSeat: true,
  },
];

function getPriceLabel(
  plan: PlanInfo,
  period: Period,
  pricing: PricingData | undefined,
  isLoading: boolean,
): string {
  if (plan.id === 'trial') return 'Free';
  if (isLoading) return '---';

  const planPricing = pricing?.[plan.id];
  const detail = period === 'monthly' ? planPricing?.monthly : planPricing?.annual;

  if (!detail) {
    // Pricing not available — show "Contact Us" instead of hardcoded values
    return 'Contact Us';
  }

  const formatted = formatPrice(detail.amount, detail.currency);
  const intervalSuffix = period === 'monthly' ? '/mo' : '/yr';
  const seatSuffix = plan.isPerSeat ? '/seat' : '';
  return `${formatted}${seatSuffix}${intervalSuffix}`;
}

export function PricingTable({ onSelectPlan, currentTier }: PricingTableProps) {
  const [period, setPeriod] = useState<Period>('monthly');

  const pricingQuery = useQuery({
    queryKey: ['pricing'],
    queryFn: getPricing,
    staleTime: 60 * 60 * 1000, // 1 hour
    retry: 1,
  });

  return (
    <div>
      {/* Period toggle */}
      <div className="flex items-center justify-center gap-3 mb-10">
        <span
          className={`text-sm ${period === 'monthly' ? 'text-text-primary font-medium' : 'text-text-secondary'}`}
        >
          Monthly
        </span>
        <button
          onClick={() => setPeriod(period === 'monthly' ? 'annual' : 'monthly')}
          className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
            period === 'annual' ? 'bg-brand-accent' : 'bg-text-tertiary'
          }`}
        >
          <span
            className={`inline-block h-4 w-4 rounded-full bg-white transition-transform ${
              period === 'annual' ? 'translate-x-6' : 'translate-x-1'
            }`}
          />
        </button>
        <span
          className={`text-sm ${period === 'annual' ? 'text-text-primary font-medium' : 'text-text-secondary'}`}
        >
          Annual
          <span className="ml-1 text-emerald-400 text-xs font-medium">Save ~17%</span>
        </span>
      </div>

      {/* Plan cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 max-w-5xl mx-auto">
        {PLANS.map((plan) => {
          const priceLabel = getPriceLabel(plan, period, pricingQuery.data, pricingQuery.isLoading);
          const isCurrent = currentTier === plan.id;

          return (
            <div
              key={plan.id}
              className={`rounded-[2px] border p-5 flex flex-col ${
                plan.highlighted
                  ? 'border-brand-accent bg-surface-alt ring-1 ring-brand-accent/50'
                  : 'border-border bg-surface-alt'
              }`}
            >
              <h3 className="text-lg font-semibold text-text-primary">{plan.name}</h3>
              <p className="mt-1 text-sm text-text-secondary">{plan.description}</p>

              <div className="mt-4">
                <span
                  className={`text-3xl font-bold ${
                    pricingQuery.isLoading && plan.id !== 'trial'
                      ? 'text-text-tertiary animate-pulse'
                      : 'text-text-primary'
                  }`}
                >
                  {priceLabel}
                </span>
              </div>

              <ul className="mt-6 space-y-2 flex-1">
                {plan.features.map((feature) => (
                  <li key={feature} className="flex items-start gap-2 text-sm text-text-secondary">
                    <svg
                      className="h-4 w-4 mt-0.5 text-emerald-400 shrink-0"
                      fill="none"
                      viewBox="0 0 24 24"
                      strokeWidth={2}
                      stroke="currentColor"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        d="M4.5 12.75l6 6 9-13.5"
                      />
                    </svg>
                    {feature}
                  </li>
                ))}
              </ul>

              <button
                onClick={() => onSelectPlan(plan.id === 'trial' ? 'personal' : plan.id, period)}
                disabled={isCurrent}
                className={`mt-6 w-full rounded-[2px] py-2.5 text-sm font-medium transition-colors ${
                  isCurrent
                    ? 'bg-border text-text-tertiary cursor-not-allowed'
                    : plan.highlighted
                      ? 'bg-brand-accent text-white hover:bg-brand-accent-hover'
                      : 'bg-border text-text-primary hover:bg-border-hover'
                }`}
              >
                {isCurrent ? 'Current Plan' : plan.id === 'trial' ? 'Start Trial' : 'Upgrade'}
              </button>
            </div>
          );
        })}
      </div>
    </div>
  );
}
