import { useQuery } from '@tanstack/react-query';
import { AuthGuard } from '../components/AuthGuard';
import { EnginePricingTable } from '../components/EnginePricingTable';
import { getUsage } from '../api/license';
// createCheckout will be used when EE pricing launches (EnginePricingTable onSelectPlan)
import { /* createCheckout, */ createPortal } from '../api/billing';
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
      <div className="rounded-[12px] border border-brand-shade3/15 bg-brand-dark-alt p-10">
        <h1 className="text-2xl font-bold text-brand-light">ByteBrew Engine</h1>
        <p className="mt-2 text-lg text-emerald-400 font-medium">
          Community Edition — Free Forever
        </p>

        <p className="mt-6 text-sm text-brand-shade2 leading-relaxed">
          ByteBrew Engine Community Edition includes the full AI agent runtime
          with no limits on agents, sessions, tools, or time.
        </p>

        <p className="mt-4 text-sm text-brand-shade2 leading-relaxed">
          Enterprise Edition with AI Observability, Cost Analytics, and
          Compliance Tools is coming soon.
        </p>

        <div className="mt-8 flex flex-col items-center gap-3">
          <a
            href="mailto:enterprise@bytebrew.ai?subject=Enterprise%20Edition%20Waitlist"
            className="inline-block rounded-[10px] bg-brand-accent px-6 py-2 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors"
          >
            Join Waitlist &rarr;
          </a>
          <Link
            to="/dashboard"
            className="text-sm text-brand-shade2 hover:text-brand-shade1 transition-colors"
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

  return (
    <div className="max-w-5xl mx-auto px-4 py-10">
      <h1 className="text-2xl font-bold text-brand-light">Billing</h1>

      {error && (
        <div className="mt-4 rounded-[10px] bg-red-500/10 border border-red-500/20 p-3 text-sm text-red-400">
          {error}
        </div>
      )}

      {/* Current plan info */}
      {usage && (
        <div className="mt-6 rounded-[12px] border border-brand-shade3/15 bg-brand-dark-alt p-5">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-sm font-medium text-brand-shade2">Current Plan</h2>
              <p className="mt-1 text-lg font-semibold text-brand-light capitalize">
                {usage.tier}
              </p>
            </div>
            {usage.tier !== 'trial' && (
              <button
                onClick={handleManageSubscription}
                disabled={loading}
                className="rounded-[10px] border border-brand-shade3/20 px-4 py-2 text-sm font-medium text-brand-shade2 hover:border-brand-shade3/40 hover:text-brand-light transition-colors disabled:opacity-50"
              >
                Manage Subscription
              </button>
            )}
          </div>

          {usage.current_period_end && (
            <p className="mt-2 text-sm text-brand-shade3">
              {usage.tier === 'trial' ? 'Trial ends' : 'Next billing date'}:{' '}
              {new Date(usage.current_period_end).toLocaleDateString()}
            </p>
          )}
        </div>
      )}

      {/* Pricing table — only for trial or no subscription */}
      {(!usage || usage.tier === 'trial') && (
        <div className="mt-12">
          <h2 className="text-xl font-bold text-brand-light text-center mb-2">
            {usage?.tier === 'trial' ? 'Upgrade Your Plan' : 'Available Plans'}
          </h2>
          <p className="text-center text-brand-shade2 mb-10">
            Choose the plan that fits your needs
          </p>
          <EnginePricingTable />
          {loading && (
            <p className="mt-4 text-center text-sm text-brand-shade2">
              Redirecting to checkout...
            </p>
          )}
        </div>
      )}

      {/* For paid subscribers — explain how to change plans */}
      {usage && usage.tier !== 'trial' && (
        <div className="mt-8 rounded-[12px] border border-brand-shade3/15 bg-brand-dark-alt p-5 text-center">
          <p className="text-sm text-brand-shade2">
            To change your plan, cancel your current subscription and subscribe to a new plan after the current period ends.
          </p>
        </div>
      )}
    </div>
  );
}
