import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { SHOW_EE_PRICING } from '../lib/feature-flags';
import { getPricing, formatPriceWithInterval, type PricingData } from '../api/pricing';

type Period = 'monthly' | 'annual';
type FeatureStatus = 'available' | 'coming_soon';

interface EEFeature {
  label: string;
  status: FeatureStatus;
}

const CE_FEATURES = [
  'Unlimited agents & spawn',
  'Multi-agent orchestration',
  'MCP servers & declarative tools',
  'Cron, webhooks, background jobs',
  'Task system (manage_tasks)',
  'Knowledge/RAG',
  'Model Registry & recommendations',
  'REST API + SSE + WebSocket',
  'Admin Dashboard (full CRUD)',
  'BYOK (bring your own keys)',
  'API tokens with scopes',
  'Docker / bare metal / Kubernetes',
];

const EE_FEATURES_PREVIEW = [
  'Audit Log',
  'Rate Limiting',
  'Prometheus Metrics',
  'Session Explorer',
  'Cost Analytics',
  'Quality Metrics',
  'PII Redaction',
];

const EE_FEATURES_FULL: EEFeature[] = [
  { label: 'Everything in Community Edition', status: 'available' },
  { label: 'Audit Log (tool calls API)', status: 'available' },
  { label: 'Configurable Rate Limiting', status: 'available' },
  { label: 'Prometheus Metrics', status: 'available' },
  { label: 'Session Explorer & Replay', status: 'coming_soon' },
  { label: 'Cost Analytics per agent', status: 'coming_soon' },
  { label: 'Quality Metrics & Dashboards', status: 'coming_soon' },
  { label: 'Prompt A/B Testing', status: 'coming_soon' },
  { label: 'PII Redaction', status: 'coming_soon' },
  { label: 'Configurable Data Retention', status: 'coming_soon' },
];

const CUSTOM_FEATURES = [
  'SSO / OIDC',
  'Role-based access control (RBAC)',
  'Multi-workspace',
  'Dedicated support (4h SLA)',
  'Custom SLA',
  'Migration assistance',
];

function CheckIcon({ dimmed }: { dimmed?: boolean }) {
  return (
    <svg
      className={`h-4 w-4 mt-0.5 shrink-0 ${dimmed ? 'text-text-tertiary' : 'text-emerald-400'}`}
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
  );
}

function ComingSoonBadge() {
  return (
    <span className="ml-1.5 inline-flex items-center rounded-full bg-border px-1.5 py-0.5 text-[10px] font-medium text-text-tertiary ring-1 ring-inset ring-border">
      Coming Soon
    </span>
  );
}

function FeatureList({ features }: { features: string[] }) {
  return (
    <ul className="mt-6 space-y-2 flex-1">
      {features.map((feature) => (
        <li key={feature} className="flex items-start gap-2 text-sm text-text-secondary">
          <CheckIcon />
          {feature}
        </li>
      ))}
    </ul>
  );
}

function EEFeatureList({ features }: { features: EEFeature[] }) {
  return (
    <ul className="mt-6 space-y-2 flex-1">
      {features.map((feature) => (
        <li
          key={feature.label}
          className={`flex items-start gap-2 text-sm ${
            feature.status === 'coming_soon' ? 'text-text-tertiary' : 'text-text-secondary'
          }`}
        >
          <CheckIcon dimmed={feature.status === 'coming_soon'} />
          <span>
            {feature.label}
            {feature.status === 'coming_soon' && <ComingSoonBadge />}
          </span>
        </li>
      ))}
    </ul>
  );
}

function Badge({ text, color }: { text: string; color: 'accent' | 'gray' }) {
  const colorClasses = {
    accent: 'bg-brand-accent/10 text-brand-accent ring-brand-accent/20',
    gray: 'bg-border text-text-tertiary ring-border',
  };

  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ring-1 ring-inset ${colorClasses[color]}`}
    >
      {text}
    </span>
  );
}

/** Invisible placeholder that matches Badge height to keep card titles aligned */
function BadgePlaceholder() {
  return <span className="inline-flex h-[22px]" aria-hidden="true" />;
}

interface EnginePricingTableProps {
  onSelectPlan?: (plan: string, period: Period) => void;
}

export function EnginePricingTable({ onSelectPlan }: EnginePricingTableProps = {}) {
  const [period, setPeriod] = useState<Period>('monthly');

  const pricingQuery = useQuery({
    queryKey: ['pricing'],
    queryFn: getPricing,
    staleTime: 60 * 60 * 1000, // 1 hour
    retry: 1,
    enabled: SHOW_EE_PRICING,
  });

  return (
    <div>
      {/* Period toggle — only shown when EE pricing is visible */}
      {SHOW_EE_PRICING && (
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
            <span className="ml-1 text-emerald-400 text-xs font-medium">Save 17%</span>
          </span>
        </div>
      )}

      {/* Plan cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 max-w-5xl mx-auto items-stretch">
        {/* Community Edition */}
        <div className="rounded-[2px] border border-brand-accent bg-surface-alt ring-1 ring-brand-accent/50 p-5 flex flex-col">
          <Badge text="Free Forever" color="accent" />
          <h3 className="mt-3 text-lg font-semibold text-text-primary">Community Edition</h3>
          <p className="mt-1 text-sm text-text-secondary">
            Full AI agent runtime. No limits.
          </p>

          <div className="mt-4">
            <span className="text-3xl font-bold text-text-primary">Free</span>
          </div>

          <FeatureList features={CE_FEATURES} />

          <div className="mt-auto pt-6">
            <Link
              to="/download"
              className="w-full rounded-[2px] bg-brand-accent py-2.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors text-center block"
            >
              Download
            </Link>
            <p className="mt-3 text-xs text-text-tertiary text-center">
              No credit card required. No time limits.
            </p>
          </div>
        </div>

        {/* Enterprise Edition */}
        {SHOW_EE_PRICING ? (
          <EECardPriced
            period={period}
            onSelectPlan={onSelectPlan}
            pricing={pricingQuery.data}
            isLoading={pricingQuery.isLoading}
          />
        ) : (
          <EECardPreview />
        )}

        {/* Custom */}
        <div className="rounded-[2px] border border-border bg-surface-alt p-5 flex flex-col">
          <BadgePlaceholder />
          <h3 className="mt-3 text-lg font-semibold text-text-primary">Custom</h3>
          <p className="mt-1 text-sm text-text-secondary">
            For teams requiring enterprise security.
          </p>

          <div className="mt-4">
            <span className="text-3xl font-bold text-text-primary">Contact Us</span>
          </div>

          <FeatureList features={CUSTOM_FEATURES} />

          <div className="mt-auto pt-6">
            <a
              href="mailto:info@bytebrew.ai"
              className="w-full rounded-[2px] bg-border py-2.5 text-sm font-medium text-text-primary hover:bg-border-hover transition-colors text-center block"
            >
              Talk to Sales
            </a>
            <p className="mt-3 text-xs text-text-tertiary text-center">
              For teams requiring enterprise security and compliance.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

function EECardPreview() {
  return (
    <div className="rounded-[2px] border border-border bg-surface-alt p-5 flex flex-col">
      <Badge text="Coming Soon" color="gray" />
      <h3 className="mt-3 text-lg font-semibold text-text-primary">Enterprise Edition</h3>
      <p className="mt-1 text-sm text-text-secondary">
        AI Observability, Cost Analytics, Compliance Tools for production at scale.
      </p>

      <FeatureList features={EE_FEATURES_PREVIEW} />

      <div className="mt-auto pt-6">
        <a
          href="mailto:info@bytebrew.ai"
          className="w-full rounded-[2px] bg-border py-2.5 text-sm font-medium text-text-primary hover:bg-border-hover transition-colors text-center block"
        >
          Join Waitlist
        </a>
        <p className="mt-3 text-xs text-text-tertiary text-center">
          Be the first to know when EE launches.
        </p>
      </div>
    </div>
  );
}

interface EECardPricedProps {
  period: Period;
  onSelectPlan?: (plan: string, period: Period) => void;
  pricing?: PricingData;
  isLoading: boolean;
}

function getEEPrice(pricing: PricingData | undefined, period: Period): string {
  const plan = pricing?.engine_ee;
  if (!plan) return '';

  const detail = period === 'monthly' ? plan.monthly : plan.annual;
  if (!detail) return '';

  return formatPriceWithInterval(detail.amount, detail.currency, detail.interval);
}

function EECardPriced({ period, onSelectPlan, pricing, isLoading }: EECardPricedProps) {
  const price = getEEPrice(pricing, period);

  const handleClick = () => {
    if (onSelectPlan) {
      onSelectPlan('engine_ee', period);
    }
  };

  return (
    <div className="rounded-[2px] border border-brand-accent bg-surface-alt ring-1 ring-brand-accent/50 p-5 flex flex-col">
      <Badge text="Most Popular" color="accent" />
      <h3 className="mt-3 text-lg font-semibold text-text-primary">Enterprise Edition</h3>
      <p className="mt-1 text-sm text-text-secondary">
        Full observability and compliance toolkit.
      </p>

      <div className="mt-4">
        {isLoading ? (
          <span className="text-3xl font-bold text-text-tertiary animate-pulse">---</span>
        ) : price ? (
          <span className="text-3xl font-bold text-text-primary">{price}</span>
        ) : (
          <span className="text-3xl font-bold text-text-tertiary">---</span>
        )}
      </div>

      <EEFeatureList features={EE_FEATURES_FULL} />

      <div className="mt-auto pt-6">
        {onSelectPlan ? (
          <button
            onClick={handleClick}
            className="w-full rounded-[2px] bg-brand-accent py-2.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors text-center block"
          >
            Subscribe
          </button>
        ) : (
          <Link
            to="/register"
            className="w-full rounded-[2px] bg-brand-accent py-2.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors text-center block"
          >
            Start Free Trial
          </Link>
        )}
        <p className="mt-3 text-xs text-text-tertiary text-center">
          14-day free trial. No credit card required.
        </p>
      </div>
    </div>
  );
}
