import { useState } from 'react';

interface PricingTableProps {
  onSelectPlan: (plan: string, period: string) => void;
  currentTier?: string;
}

type Period = 'monthly' | 'annual';

interface PlanInfo {
  id: string;
  name: string;
  monthlyPrice: number;
  annualPrice: number;
  description: string;
  features: string[];
  highlighted?: boolean;
}

const PLANS: PlanInfo[] = [
  {
    id: 'trial',
    name: 'Trial',
    monthlyPrice: 0,
    annualPrice: 0,
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
    monthlyPrice: 20,
    annualPrice: 200,
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
    monthlyPrice: 30,
    annualPrice: 300,
    description: 'For teams and organizations',
    features: [
      'Everything in Personal',
      '300 proxy steps / user / month',
      'Auto-scaling seats',
      'Enterprise SSO (SAML/OIDC)',
      'Admin panel & audit log',
      'Relay support',
    ],
  },
];

export function PricingTable({ onSelectPlan, currentTier }: PricingTableProps) {
  const [period, setPeriod] = useState<Period>('monthly');

  return (
    <div>
      {/* Period toggle */}
      <div className="flex items-center justify-center gap-3 mb-10">
        <span
          className={`text-sm ${period === 'monthly' ? 'text-white font-medium' : 'text-gray-400'}`}
        >
          Monthly
        </span>
        <button
          onClick={() => setPeriod(period === 'monthly' ? 'annual' : 'monthly')}
          className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
            period === 'annual' ? 'bg-indigo-600' : 'bg-gray-600'
          }`}
        >
          <span
            className={`inline-block h-4 w-4 rounded-full bg-white transition-transform ${
              period === 'annual' ? 'translate-x-6' : 'translate-x-1'
            }`}
          />
        </button>
        <span
          className={`text-sm ${period === 'annual' ? 'text-white font-medium' : 'text-gray-400'}`}
        >
          Annual
          <span className="ml-1 text-emerald-400 text-xs font-medium">Save ~17%</span>
        </span>
      </div>

      {/* Plan cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 max-w-5xl mx-auto">
        {PLANS.map((plan) => {
          const price =
            period === 'monthly' ? plan.monthlyPrice : plan.annualPrice;
          const priceLabel =
            plan.id === 'trial'
              ? 'Free'
              : plan.id === 'teams'
                ? `$${price}${period === 'annual' ? '/seat/yr' : '/seat/mo'}`
                : `$${price}${period === 'annual' ? '/yr' : '/mo'}`;
          const isCurrent = currentTier === plan.id;

          return (
            <div
              key={plan.id}
              className={`rounded-xl border p-6 flex flex-col ${
                plan.highlighted
                  ? 'border-indigo-500 bg-gray-900/80 ring-1 ring-indigo-500/50'
                  : 'border-gray-700 bg-gray-900/50'
              }`}
            >
              <h3 className="text-lg font-semibold text-white">{plan.name}</h3>
              <p className="mt-1 text-sm text-gray-400">{plan.description}</p>

              <div className="mt-4">
                <span className="text-3xl font-bold text-white">{priceLabel}</span>
              </div>

              <ul className="mt-6 space-y-2 flex-1">
                {plan.features.map((feature) => (
                  <li key={feature} className="flex items-start gap-2 text-sm text-gray-300">
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
                className={`mt-6 w-full rounded-lg py-2.5 text-sm font-medium transition-colors ${
                  isCurrent
                    ? 'bg-gray-700 text-gray-400 cursor-not-allowed'
                    : plan.highlighted
                      ? 'bg-indigo-600 text-white hover:bg-indigo-500'
                      : 'bg-gray-700 text-white hover:bg-gray-600'
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
