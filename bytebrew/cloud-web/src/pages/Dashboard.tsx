import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { AuthGuard } from '../components/AuthGuard';
import { TierBadge } from '../components/TierBadge';
import { getUsage, downloadLicense, getLicenseStatus } from '../api/license';
import { ApiError } from '../api/client';
import { Link } from '@tanstack/react-router';

export function DashboardPage() {
  return (
    <AuthGuard>
      <DashboardContent />
    </AuthGuard>
  );
}

function DashboardContent() {
  const [downloadError, setDownloadError] = useState('');

  const searchParams = new URLSearchParams(window.location.search);
  const checkoutFailed = searchParams.get('checkout_failed') === '1';

  const usageQuery = useQuery({
    queryKey: ['usage'],
    queryFn: getUsage,
    refetchInterval: 60_000,
  });

  const licenseQuery = useQuery({
    queryKey: ['license-status'],
    queryFn: getLicenseStatus,
  });

  const usage = usageQuery.data;
  const license = licenseQuery.data;

  const handleDownloadLicense = async () => {
    setDownloadError('');
    try {
      const blob = await downloadLicense();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'license.jwt';
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      setDownloadError('Failed to download license. Please try again.');
    }
  };

  return (
    <div className="max-w-4xl mx-auto px-4 py-10">
      <h1 className="text-2xl font-bold text-white">Dashboard</h1>

      {checkoutFailed && (
        <div className="mt-4 rounded-lg bg-yellow-500/10 border border-yellow-500/20 p-3 text-sm text-yellow-400">
          Checkout could not be completed. You can try again from the{' '}
          <Link to="/billing" className="underline hover:text-yellow-300">
            billing page
          </Link>
          .
        </div>
      )}

      {(usageQuery.isLoading || licenseQuery.isLoading) && (
        <div className="mt-8 text-gray-400">Loading...</div>
      )}

      {usageQuery.error && (
        usageQuery.error instanceof ApiError && usageQuery.error.status === 403 ? (
          <div className="mt-8 rounded-xl border border-gray-800 bg-gray-900/50 p-6 text-center">
            <h2 className="text-lg font-semibold text-white">No Active Subscription</h2>
            <p className="mt-2 text-sm text-gray-400">
              Choose a plan to start using ByteBrew.
            </p>
            <Link
              to="/billing"
              className="mt-4 inline-block rounded-lg bg-indigo-600 px-6 py-2 text-sm font-medium text-white hover:bg-indigo-500 transition-colors"
            >
              View Plans
            </Link>
          </div>
        ) : (
          <div className="mt-4 rounded-lg bg-red-500/10 border border-red-500/20 p-3 text-sm text-red-400">
            Failed to load usage data
          </div>
        )
      )}

      {usage && (
        <div className="mt-8 grid grid-cols-1 md:grid-cols-2 gap-6">
          {/* Subscription Status */}
          <div className="rounded-xl border border-gray-800 bg-gray-900/50 p-6">
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-medium text-gray-400">Subscription</h2>
              <TierBadge tier={usage.tier} />
            </div>

            {license && (
              <div className="mt-4 space-y-2">
                {usage.tier === 'trial' && license.expires_at && (
                  <TrialCountdown expiresAt={license.expires_at} />
                )}
                {license.grace_until && (
                  <div className="rounded-lg bg-yellow-500/10 border border-yellow-500/20 p-2 text-sm text-yellow-400">
                    Grace period ends {new Date(license.grace_until).toLocaleDateString()}
                  </div>
                )}
              </div>
            )}

            <div className="mt-4">
              <Link
                to="/billing"
                className="text-sm text-indigo-400 hover:text-indigo-300"
              >
                {usage.tier === 'trial' ? 'Upgrade plan' : 'Manage subscription'}
              </Link>
            </div>
          </div>

          {/* Proxy Usage */}
          <div className="rounded-xl border border-gray-800 bg-gray-900/50 p-6">
            <h2 className="text-sm font-medium text-gray-400">Proxy Steps</h2>

            <div className="mt-4">
              <div className="flex items-end justify-between">
                <span className="text-2xl font-bold text-white">
                  {usage.proxy_steps_used}
                </span>
                <span className="text-sm text-gray-500">
                  / {usage.proxy_steps_limit === 0 ? 'unlimited' : usage.proxy_steps_limit}
                </span>
              </div>

              {usage.proxy_steps_limit > 0 && (
                <div className="mt-3">
                  <div className="h-2 rounded-full bg-gray-800 overflow-hidden">
                    <div
                      className={`h-full rounded-full transition-all ${
                        usage.proxy_steps_remaining <= 10
                          ? 'bg-red-500'
                          : usage.proxy_steps_remaining <= 50
                            ? 'bg-yellow-500'
                            : 'bg-indigo-500'
                      }`}
                      style={{
                        width: `${Math.min(100, (usage.proxy_steps_used / usage.proxy_steps_limit) * 100)}%`,
                      }}
                    />
                  </div>
                  <p className="mt-2 text-xs text-gray-500">
                    {usage.proxy_steps_remaining} steps remaining
                  </p>
                </div>
              )}

              {usage.current_period_end && (
                <p className="mt-2 text-xs text-gray-500">
                  Resets on {new Date(usage.current_period_end).toLocaleDateString()}
                </p>
              )}
            </div>
          </div>

          {/* License Download */}
          <div className="rounded-xl border border-gray-800 bg-gray-900/50 p-6">
            <h2 className="text-sm font-medium text-gray-400">License</h2>
            <p className="mt-2 text-sm text-gray-300">
              Download your license file for offline activation.
            </p>
            <button
              onClick={handleDownloadLicense}
              className="mt-4 rounded-lg border border-gray-700 px-4 py-2 text-sm font-medium text-gray-300 hover:border-gray-500 hover:text-white transition-colors"
            >
              Download license.jwt
            </button>
            {downloadError && (
              <p className="mt-2 text-xs text-red-400">{downloadError}</p>
            )}
            <p className="mt-2 text-xs text-gray-600">
              Place in ~/.bytebrew/license.jwt or use{' '}
              <code className="text-gray-500">bytebrew activate --file license.jwt</code>
            </p>
          </div>

          {/* BYOK Status */}
          <div className="rounded-xl border border-gray-800 bg-gray-900/50 p-6">
            <h2 className="text-sm font-medium text-gray-400">BYOK (API Key)</h2>
            <div className="mt-2">
              {usage.byok_enabled ? (
                <span className="inline-flex items-center gap-1.5 text-sm text-emerald-400">
                  <span className="h-2 w-2 rounded-full bg-emerald-400" />
                  Configured
                </span>
              ) : (
                <span className="inline-flex items-center gap-1.5 text-sm text-gray-500">
                  <span className="h-2 w-2 rounded-full bg-gray-600" />
                  Not configured
                </span>
              )}
            </div>
            <Link
              to="/settings"
              className="mt-4 inline-block text-sm text-indigo-400 hover:text-indigo-300"
            >
              Configure BYOK
            </Link>
          </div>
        </div>
      )}
    </div>
  );
}

function TrialCountdown({ expiresAt }: { expiresAt: string }) {
  const now = new Date();
  const expiry = new Date(expiresAt);
  const daysLeft = Math.max(0, Math.ceil((expiry.getTime() - now.getTime()) / (1000 * 60 * 60 * 24)));

  const color =
    daysLeft <= 3
      ? 'text-red-400 bg-red-500/10 border-red-500/20'
      : daysLeft <= 7
        ? 'text-yellow-400 bg-yellow-500/10 border-yellow-500/20'
        : 'text-gray-300 bg-gray-800/50 border-gray-700';

  return (
    <div className={`rounded-lg border p-2 text-sm ${color}`}>
      {daysLeft === 0
        ? 'Trial expires today'
        : `${daysLeft} day${daysLeft !== 1 ? 's' : ''} remaining in trial`}
    </div>
  );
}
