import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { SHOW_EE_PRICING } from '../lib/feature-flags';

type Period = 'monthly' | 'annual';

const CE_FEATURES = [
  'Unlimited agents & spawn',
  'MCP servers & declarative tools',
  'Cron, webhooks, background jobs',
  'Task system (manage_tasks)',
  'Knowledge/RAG',
  'REST API + SSE + WebSocket',
  'Admin Dashboard (full CRUD)',
  'BYOK (bring your own keys)',
  'API tokens with scopes',
  'Docker / bare metal',
];

const EE_FEATURES_PREVIEW = [
  'Session Explorer',
  'Cost Analytics',
  'Quality Metrics',
  'Prompt A/B Testing',
  'Audit Log Export',
  'PII Redaction',
  'Data Retention Policies',
];

const EE_FEATURES_FULL = [
  'Everything in Community Edition',
  'Session Explorer & Replay',
  'Cost Analytics per agent',
  'Quality Metrics & Dashboards',
  'Prompt A/B Testing',
  'Audit Log Export',
  'PII Redaction',
  'Configurable Data Retention',
];

const CUSTOM_FEATURES = [
  'SSO / OIDC',
  'Role-based access control (RBAC)',
  'Multi-workspace',
  'Dedicated support (4h SLA)',
  'Custom SLA',
  'Migration assistance',
];

function CheckIcon() {
  return (
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
  );
}

function FeatureList({ features }: { features: string[] }) {
  return (
    <ul className="mt-6 space-y-2 flex-1">
      {features.map((feature) => (
        <li key={feature} className="flex items-start gap-2 text-sm text-brand-shade2">
          <CheckIcon />
          {feature}
        </li>
      ))}
    </ul>
  );
}

function Badge({ text, color }: { text: string; color: 'accent' | 'gray' }) {
  const colorClasses = {
    accent: 'bg-brand-accent/10 text-brand-accent ring-brand-accent/20',
    gray: 'bg-brand-shade3/10 text-brand-shade3 ring-brand-shade3/20',
  };

  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ring-1 ring-inset ${colorClasses[color]}`}
    >
      {text}
    </span>
  );
}

export function EnginePricingTable() {
  const [period, setPeriod] = useState<Period>('monthly');

  return (
    <div>
      {/* Period toggle — only shown when EE pricing is visible */}
      {SHOW_EE_PRICING && (
        <div className="flex items-center justify-center gap-3 mb-10">
          <span
            className={`text-sm ${period === 'monthly' ? 'text-brand-light font-medium' : 'text-brand-shade2'}`}
          >
            Monthly
          </span>
          <button
            onClick={() => setPeriod(period === 'monthly' ? 'annual' : 'monthly')}
            className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
              period === 'annual' ? 'bg-brand-accent' : 'bg-brand-shade3'
            }`}
          >
            <span
              className={`inline-block h-4 w-4 rounded-full bg-white transition-transform ${
                period === 'annual' ? 'translate-x-6' : 'translate-x-1'
              }`}
            />
          </button>
          <span
            className={`text-sm ${period === 'annual' ? 'text-brand-light font-medium' : 'text-brand-shade2'}`}
          >
            Annual
            <span className="ml-1 text-emerald-400 text-xs font-medium">Save 17%</span>
          </span>
        </div>
      )}

      {/* Plan cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 max-w-5xl mx-auto">
        {/* Community Edition */}
        <div className="rounded-[12px] border border-brand-accent bg-brand-dark-alt ring-1 ring-brand-accent/50 p-5 flex flex-col">
          <Badge text="Free Forever" color="accent" />
          <h3 className="mt-3 text-lg font-semibold text-brand-light">Community Edition</h3>
          <p className="mt-1 text-sm text-brand-shade2">
            Full AI agent runtime. No limits.
          </p>

          <div className="mt-4">
            <span className="text-3xl font-bold text-brand-light">Free</span>
          </div>

          <FeatureList features={CE_FEATURES} />

          <Link
            to="/download"
            className="mt-6 w-full rounded-[10px] bg-brand-accent py-2.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors text-center block"
          >
            Download
          </Link>
          <p className="mt-3 text-xs text-brand-shade3 text-center">
            No credit card required. No time limits.
          </p>
        </div>

        {/* Enterprise Edition */}
        {SHOW_EE_PRICING ? (
          <EECardPriced period={period} />
        ) : (
          <EECardPreview />
        )}

        {/* Custom */}
        <div className="rounded-[12px] border border-brand-shade3/20 bg-brand-dark-alt p-5 flex flex-col">
          <h3 className="text-lg font-semibold text-brand-light">Custom</h3>
          <p className="mt-1 text-sm text-brand-shade2">
            For teams requiring enterprise security.
          </p>

          <div className="mt-4">
            <span className="text-3xl font-bold text-brand-light">Contact Us</span>
          </div>

          <FeatureList features={CUSTOM_FEATURES} />

          <a
            href="mailto:sales@bytebrew.ai"
            className="mt-6 w-full rounded-[10px] bg-brand-shade3/20 py-2.5 text-sm font-medium text-brand-light hover:bg-brand-shade3/30 transition-colors text-center block"
          >
            Talk to Sales
          </a>
          <p className="mt-3 text-xs text-brand-shade3 text-center">
            For teams requiring enterprise security and compliance.
          </p>
        </div>
      </div>
    </div>
  );
}

function EECardPreview() {
  return (
    <div className="rounded-[12px] border border-brand-shade3/20 bg-brand-dark-alt p-5 flex flex-col">
      <Badge text="Coming Soon" color="gray" />
      <h3 className="mt-3 text-lg font-semibold text-brand-light">Enterprise Edition</h3>
      <p className="mt-1 text-sm text-brand-shade2">
        AI Observability, Cost Analytics, Compliance Tools for production at scale.
      </p>

      <FeatureList features={EE_FEATURES_PREVIEW} />

      <a
        href="mailto:hello@bytebrew.ai"
        className="mt-6 w-full rounded-[10px] bg-brand-shade3/20 py-2.5 text-sm font-medium text-brand-light hover:bg-brand-shade3/30 transition-colors text-center block"
      >
        Join Waitlist
      </a>
      <p className="mt-3 text-xs text-brand-shade3 text-center">
        Be the first to know when EE launches.
      </p>
    </div>
  );
}

function EECardPriced({ period }: { period: Period }) {
  const price = period === 'monthly' ? '$499/mo' : '$4,990/yr';

  return (
    <div className="rounded-[12px] border border-brand-accent bg-brand-dark-alt ring-1 ring-brand-accent/50 p-5 flex flex-col">
      <Badge text="Most Popular" color="accent" />
      <h3 className="mt-3 text-lg font-semibold text-brand-light">Enterprise Edition</h3>
      <p className="mt-1 text-sm text-brand-shade2">
        Full observability and compliance toolkit.
      </p>

      <div className="mt-4">
        <span className="text-3xl font-bold text-brand-light">{price}</span>
      </div>

      <FeatureList features={EE_FEATURES_FULL} />

      <Link
        to="/register"
        className="mt-6 w-full rounded-[10px] bg-brand-accent py-2.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors text-center block"
      >
        Start Free Trial
      </Link>
    </div>
  );
}
