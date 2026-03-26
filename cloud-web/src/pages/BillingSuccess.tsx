import { Link } from '@tanstack/react-router';

export function BillingSuccessPage() {
  return (
    <div className="max-w-lg mx-auto px-4 py-20 text-center">
      <div className="text-5xl mb-6">&#10003;</div>
      <h1 className="text-2xl font-bold text-text-primary">Payment Successful</h1>
      <p className="mt-4 text-text-secondary">
        Your subscription is now active. You can start using ByteBrew Engine right away.
      </p>

      <div className="mt-8 rounded-[2px] border border-border bg-surface-alt p-5 text-left">
        <h2 className="text-sm font-medium text-text-secondary mb-3">Next steps</h2>
        <ol className="space-y-3 text-sm text-text-secondary">
          <li>
            <span className="font-mono text-text-primary">1.</span>{' '}
            Download your license from the{' '}
            <Link to="/dashboard" className="text-brand-accent hover:text-brand-accent-hover">
              Dashboard
            </Link>
          </li>
          <li>
            <span className="font-mono text-text-primary">2.</span>{' '}
            Place the license file:{' '}
            <code className="rounded bg-surface-alt px-2 py-0.5 text-brand-accent text-xs">~/.bytebrew/license.jwt</code>
          </li>
          <li>
            <span className="font-mono text-text-primary">3.</span>{' '}
            Restart your Engine instance to activate Enterprise features
          </li>
        </ol>
      </div>

      <div className="mt-8 flex gap-4 justify-center">
        <Link
          to="/dashboard"
          className="rounded-[2px] bg-brand-accent px-6 py-2.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors"
        >
          Go to Dashboard
        </Link>
        <Link
          to="/billing"
          className="rounded-[2px] border border-border px-6 py-2.5 text-sm font-medium text-text-secondary hover:border-border-hover hover:text-text-primary transition-colors"
        >
          View Billing
        </Link>
      </div>
    </div>
  );
}
