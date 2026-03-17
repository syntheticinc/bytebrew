import { useQuery } from '@tanstack/react-query';
import { AuthGuard } from '../components/AuthGuard';
import { PricingTable } from '../components/PricingTable';
import { getUsage } from '../api/license';
import { createCheckout, createPortal } from '../api/billing';
import { useState } from 'react';

export function BillingPage() {
  return (
    <AuthGuard>
      <BillingContent />
    </AuthGuard>
  );
}

function BillingContent() {
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const usageQuery = useQuery({
    queryKey: ['usage'],
    queryFn: getUsage,
  });

  const usage = usageQuery.data;

  const handleSelectPlan = async (plan: string, period: string) => {
    setError('');
    setLoading(true);
    try {
      const res = await createCheckout(plan, period);
      window.location.href = res.checkout_url;
    } catch {
      setError('Failed to create checkout session. Please try again.');
      setLoading(false);
    }
  };

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
      <h1 className="text-2xl font-bold text-white">Billing</h1>

      {error && (
        <div className="mt-4 rounded-lg bg-red-500/10 border border-red-500/20 p-3 text-sm text-red-400">
          {error}
        </div>
      )}

      {/* Current plan info */}
      {usage && (
        <div className="mt-6 rounded-xl border border-gray-800 bg-gray-900/50 p-6">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-sm font-medium text-gray-400">Current Plan</h2>
              <p className="mt-1 text-lg font-semibold text-white capitalize">
                {usage.tier}
              </p>
            </div>
            {usage.tier !== 'trial' && (
              <button
                onClick={handleManageSubscription}
                disabled={loading}
                className="rounded-lg border border-gray-700 px-4 py-2 text-sm font-medium text-gray-300 hover:border-gray-500 hover:text-white transition-colors disabled:opacity-50"
              >
                Manage Subscription
              </button>
            )}
          </div>

          {usage.current_period_end && (
            <p className="mt-2 text-sm text-gray-500">
              {usage.tier === 'trial' ? 'Trial ends' : 'Next billing date'}:{' '}
              {new Date(usage.current_period_end).toLocaleDateString()}
            </p>
          )}
        </div>
      )}

      {/* Pricing table — only for trial or no subscription */}
      {(!usage || usage.tier === 'trial') && (
        <div className="mt-12">
          <h2 className="text-xl font-bold text-white text-center mb-2">
            {usage?.tier === 'trial' ? 'Upgrade Your Plan' : 'Available Plans'}
          </h2>
          <p className="text-center text-gray-400 mb-10">
            Choose the plan that fits your needs
          </p>
          <PricingTable
            onSelectPlan={handleSelectPlan}
            currentTier={usage?.tier}
          />
          {loading && (
            <p className="mt-4 text-center text-sm text-gray-400">
              Redirecting to checkout...
            </p>
          )}
        </div>
      )}

      {/* For paid subscribers — explain how to change plans */}
      {usage && usage.tier !== 'trial' && (
        <div className="mt-8 rounded-xl border border-gray-800 bg-gray-900/50 p-6 text-center">
          <p className="text-sm text-gray-400">
            To change your plan, cancel your current subscription and subscribe to a new plan after the current period ends.
          </p>
        </div>
      )}
    </div>
  );
}
