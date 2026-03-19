import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { AuthGuard } from '../components/AuthGuard';
import { TierBadge } from '../components/TierBadge';
import { getUsage, downloadLicense } from '../api/license';
import { ApiError } from '../api/client';
import { Link } from '@tanstack/react-router';
import { useAuth } from '../lib/auth';
import { SHOW_EE_PRICING } from '../lib/feature-flags';

export function DashboardPage() {
  return (
    <AuthGuard>
      <DashboardContent />
    </AuthGuard>
  );
}

function DashboardContent() {
  const { email } = useAuth();
  const [downloadError, setDownloadError] = useState('');

  const searchParams = new URLSearchParams(window.location.search);
  const checkoutFailed = searchParams.get('checkout_failed') === '1';

  const usageQuery = useQuery({
    queryKey: ['usage'],
    queryFn: getUsage,
    refetchInterval: 60_000,
  });

  const usage = usageQuery.data;
  const is403 =
    usageQuery.error instanceof ApiError && usageQuery.error.status === 403;
  const hasSubscription = !!usage && !is403;

  const tier = SHOW_EE_PRICING && usage?.tier ? usage.tier : 'ce';

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
      <h1 className="text-2xl font-bold text-brand-light">Dashboard</h1>

      {checkoutFailed && (
        <div className="mt-4 rounded-[10px] bg-yellow-500/10 border border-yellow-500/20 p-3 text-sm text-yellow-400">
          Checkout could not be completed. You can try again from the{' '}
          <Link to="/billing" className="underline hover:text-yellow-300">
            billing page
          </Link>
          .
        </div>
      )}

      {usageQuery.isLoading && (
        <div className="mt-8 text-brand-shade2">Loading...</div>
      )}

      {usageQuery.error && !is403 && (
        <div className="mt-4 rounded-[10px] bg-red-500/10 border border-red-500/20 p-3 text-sm text-red-400">
          Failed to load usage data
        </div>
      )}

      {is403 && (
        <div className="mt-6 rounded-[10px] bg-emerald-500/10 border border-emerald-500/20 p-3 text-sm text-emerald-400">
          You're using ByteBrew Engine Community Edition — free forever.
        </div>
      )}

      <div className="mt-8 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {/* Card 1: Account Status */}
        <div className="rounded-[12px] border border-brand-shade3/15 bg-brand-dark-alt p-5">
          <h2 className="text-sm font-medium text-brand-shade2">Account</h2>
          <p className="mt-3 text-lg font-semibold text-brand-light truncate">
            {email ?? 'Unknown'}
          </p>
          <div className="mt-3">
            <TierBadge tier={tier} />
          </div>
        </div>

        {/* Card 2: License (only when subscription exists) */}
        {hasSubscription && (
          <div className="rounded-[12px] border border-brand-shade3/15 bg-brand-dark-alt p-5">
            <h2 className="text-sm font-medium text-brand-shade2">License</h2>
            <p className="mt-2 text-sm text-brand-shade2">
              Download your license file for offline activation.
            </p>
            <button
              onClick={handleDownloadLicense}
              className="mt-4 rounded-[10px] border border-brand-shade3/20 px-4 py-2 text-sm font-medium text-brand-shade2 hover:border-brand-shade3/40 hover:text-brand-light transition-colors"
            >
              Download license.jwt
            </button>
            {downloadError && (
              <p className="mt-2 text-xs text-red-400">{downloadError}</p>
            )}
            <p className="mt-2 text-xs text-brand-shade3">
              Place in ~/.bytebrew/license.jwt
            </p>
          </div>
        )}

        {/* Card 3: Quick Start */}
        <div className="rounded-[12px] border border-brand-shade3/15 bg-brand-dark-alt p-5">
          <h2 className="text-sm font-medium text-brand-shade2">Quick Start</h2>
          <ul className="mt-4 space-y-3">
            <li>
              <Link
                to="/download"
                className="flex items-center justify-between text-sm text-brand-shade2 hover:text-brand-light transition-colors"
              >
                <span>Installation Guide</span>
                <span className="text-brand-shade3">&rarr;</span>
              </Link>
            </li>
            <li>
              <a
                href="https://docs.bytebrew.ai"
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center justify-between text-sm text-brand-shade2 hover:text-brand-light transition-colors"
              >
                <span>Documentation</span>
                <span className="text-brand-shade3">&rarr;</span>
              </a>
            </li>
            <li>
              <a
                href="#"
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center justify-between text-sm text-brand-shade2 hover:text-brand-light transition-colors"
              >
                <span>GitHub</span>
                <span className="text-brand-shade3">&rarr;</span>
              </a>
            </li>
          </ul>
        </div>
      </div>
    </div>
  );
}
