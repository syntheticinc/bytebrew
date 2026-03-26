import { useQuery } from '@tanstack/react-query';
import { AuthGuard } from '../components/AuthGuard';
import { getUsage } from '../api/license';
import { createPortal } from '../api/billing';
import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { SHOW_EE_PRICING } from '../lib/feature-flags';

export function BillingPage() {
  return (
    <AuthGuard>
      <BillingContent />
    </AuthGuard>
  );
}

function BillingContent() {
  if (!SHOW_EE_PRICING) {
    return <CommunityBilling />;
  }

  return <EnterpriseBilling />;
}

function CommunityBilling() {
  return (
    <div className="max-w-2xl mx-auto px-4 py-20 text-center">
      <div className="rounded-[2px] border border-border bg-surface-alt p-10">
        <h1 className="text-2xl font-bold text-text-primary">ByteBrew Engine</h1>
        <p className="mt-2 text-lg text-emerald-400 font-medium">
          Community Edition — Free Forever
        </p>

        <p className="mt-6 text-sm text-text-secondary leading-relaxed">
          ByteBrew Engine Community Edition includes the full AI agent runtime
          with no limits on agents, sessions, tools, or time.
        </p>

        <p className="mt-4 text-sm text-text-secondary leading-relaxed">
          Enterprise Edition with AI Observability, Cost Analytics, and
          Compliance Tools is coming soon.
        </p>

        <div className="mt-8 flex flex-col items-center gap-3">
          <a
            href="mailto:enterprise@bytebrew.ai?subject=Enterprise%20Edition%20Waitlist"
            className="inline-block rounded-[2px] bg-brand-accent px-6 py-2 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors"
          >
            Join Waitlist &rarr;
          </a>
          <Link
            to="/dashboard"
            className="text-sm text-text-secondary hover:text-text-secondary transition-colors"
          >
            &larr; Back to Dashboard
          </Link>
        </div>
      </div>
    </div>
  );
}

function EnterpriseBilling() {
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const usageQuery = useQuery({
    queryKey: ['usage'],
    queryFn: getUsage,
  });

  const usage = usageQuery.data;

  const handleManageSubscription = async () => {
    setError('');
    setLoading(true);
    try {
      const res = await createPortal();
      window.location.href = res.portal_url;
    } catch {
      setError('Failed to open billing portal. Please try again.');
      setLoading(false);
    }
  };

  if (usageQuery.isLoading) {
    return (
      <div className="max-w-2xl mx-auto px-4 py-20 text-center">
        <p className="text-text-secondary">Loading billing information...</p>
      </div>
    );
  }

  if (usageQuery.isError) {
    return (
      <div className="max-w-2xl mx-auto px-4 py-20 text-center">
        <div className="rounded-[2px] border border-red-500/20 bg-red-500/5 p-6">
          <p className="text-sm text-red-400">Failed to load billing information. Please try again later.</p>
          <button
            onClick={() => usageQuery.refetch()}
            className="mt-4 rounded-[2px] border border-border px-4 py-2 text-sm font-medium text-text-secondary hover:border-border-hover hover:text-text-primary transition-colors"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  const hasSubscription = usage && usage.tier !== 'trial';

  if (!hasSubscription) {
    return <FreeTierBilling />;
  }

  return <SubscriptionBilling usage={usage} error={error} loading={loading} onManage={handleManageSubscription} />;
}

function FreeTierBilling() {
  return (
    <div className="max-w-2xl mx-auto px-4 py-20">
      <h1 className="text-2xl font-bold text-text-primary">Billing</h1>

      <div className="mt-8 rounded-[2px] border border-border bg-surface-alt p-8 text-center">
        <div className="inline-flex items-center justify-center w-12 h-12 rounded-full bg-emerald-400/10 mb-4">
          <svg className="h-6 w-6 text-emerald-400" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>

        <h2 className="text-lg font-semibold text-text-primary">
          You're on the Free Trial
        </h2>
        <p className="mt-2 text-sm text-text-secondary leading-relaxed">
          You have access to ByteBrew Engine with a trial license.
          Upgrade to Enterprise Edition for full observability and compliance features.
        </p>

        <div className="mt-8">
          <Link
            to="/pricing"
            className="inline-block rounded-[2px] bg-brand-accent px-6 py-2.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors"
          >
            View pricing plans &rarr;
          </Link>
        </div>
      </div>
    </div>
  );
}

interface SubscriptionBillingProps {
  usage: {
    tier: string;
    proxy_steps_used: number;
    proxy_steps_limit: number;
    proxy_steps_remaining: number;
    byok_enabled: boolean;
    current_period_end?: string;
  };
  error: string;
  loading: boolean;
  onManage: () => void;
}

function SubscriptionBilling({ usage, error, loading, onManage }: SubscriptionBillingProps) {
  const usagePercent = usage.proxy_steps_limit > 0
    ? Math.round((usage.proxy_steps_used / usage.proxy_steps_limit) * 100)
    : 0;

  return (
    <div className="max-w-2xl mx-auto px-4 py-20">
      <h1 className="text-2xl font-bold text-text-primary">Billing</h1>

      {error && (
        <div className="mt-4 rounded-[2px] bg-red-500/10 border border-red-500/20 p-3 text-sm text-red-400">
          {error}
        </div>
      )}

      {/* Current plan card */}
      <div className="mt-8 rounded-[2px] border border-border bg-surface-alt p-6">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm text-text-tertiary">Current Plan</p>
            <h2 className="mt-1 text-xl font-semibold text-text-primary capitalize">
              {usage.tier}
            </h2>
          </div>
          <span className="inline-flex items-center rounded-full bg-emerald-400/10 px-3 py-1 text-xs font-medium text-emerald-400 ring-1 ring-inset ring-emerald-400/20">
            Active
          </span>
        </div>

        {usage.current_period_end && (
          <p className="mt-3 text-sm text-text-secondary">
            Next billing date: {new Date(usage.current_period_end).toLocaleDateString('en-US', {
              year: 'numeric',
              month: 'long',
              day: 'numeric',
            })}
          </p>
        )}

        <div className="mt-6">
          <button
            onClick={onManage}
            disabled={loading}
            className="rounded-[2px] bg-brand-accent px-5 py-2.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors disabled:opacity-50"
          >
            {loading ? 'Redirecting...' : 'Manage Subscription'}
          </button>
          <p className="mt-2 text-xs text-text-tertiary">
            Update payment method, change plan, or cancel via Stripe.
          </p>
        </div>
      </div>

      {/* Usage stats */}
      {usage.proxy_steps_limit > 0 && (
        <div className="mt-6 rounded-[2px] border border-border bg-surface-alt p-6">
          <h3 className="text-sm font-medium text-text-secondary">Proxy Steps Usage</h3>

          <div className="mt-4">
            <div className="flex items-baseline justify-between">
              <span className="text-2xl font-bold text-text-primary">
                {usage.proxy_steps_used.toLocaleString()}
              </span>
              <span className="text-sm text-text-tertiary">
                of {usage.proxy_steps_limit.toLocaleString()}
              </span>
            </div>

            <div className="mt-3 h-2 rounded-full bg-border overflow-hidden">
              <div
                className={`h-full rounded-full transition-all ${
                  usagePercent >= 90 ? 'bg-red-400' : usagePercent >= 70 ? 'bg-yellow-400' : 'bg-emerald-400'
                }`}
                style={{ width: `${Math.min(usagePercent, 100)}%` }}
              />
            </div>

            <p className="mt-2 text-xs text-text-tertiary">
              {usage.proxy_steps_remaining.toLocaleString()} steps remaining ({usagePercent}% used)
            </p>
          </div>
        </div>
      )}

      {/* BYOK status */}
      <div className="mt-6 rounded-[2px] border border-border bg-surface-alt p-6">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-sm font-medium text-text-secondary">Bring Your Own Keys (BYOK)</h3>
            <p className="mt-1 text-xs text-text-tertiary">
              Use your own LLM API keys for unlimited usage.
            </p>
          </div>
          <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ring-1 ring-inset ${
            usage.byok_enabled
              ? 'bg-emerald-400/10 text-emerald-400 ring-emerald-400/20'
              : 'bg-border text-text-tertiary ring-border'
          }`}>
            {usage.byok_enabled ? 'Enabled' : 'Disabled'}
          </span>
        </div>
      </div>
    </div>
  );
}
