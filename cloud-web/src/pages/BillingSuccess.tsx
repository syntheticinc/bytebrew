import { Link } from '@tanstack/react-router';

export function BillingSuccessPage() {
  return (
    <div className="max-w-lg mx-auto px-4 py-20 text-center">
      <div className="text-5xl mb-6">&#10003;</div>
      <h1 className="text-2xl font-bold text-brand-light">Payment Successful</h1>
      <p className="mt-4 text-brand-shade2">
        Your subscription is now active. You can start using ByteBrew Engine right away.
      </p>

      <div className="mt-8 rounded-[2px] border border-brand-shade3/15 bg-brand-dark-alt p-5 text-left">
        <h2 className="text-sm font-medium text-brand-shade2 mb-3">Next steps</h2>
        <ol className="space-y-3 text-sm text-brand-shade2">
          <li>
            <span className="font-mono text-brand-light">1.</span>{' '}
            Download your license from the{' '}
            <Link to="/dashboard" className="text-brand-accent hover:text-brand-accent-hover">
              Dashboard
            </Link>
          </li>
          <li>
            <span className="font-mono text-brand-light">2.</span>{' '}
            Place the license file:{' '}
            <code className="rounded bg-brand-dark-alt px-2 py-0.5 text-brand-accent text-xs">~/.bytebrew/license.jwt</code>
          </li>
          <li>
            <span className="font-mono text-brand-light">3.</span>{' '}
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
          className="rounded-[2px] border border-brand-shade3/20 px-6 py-2.5 text-sm font-medium text-brand-shade2 hover:border-brand-shade3/40 hover:text-brand-light transition-colors"
        >
          View Billing
        </Link>
      </div>
    </div>
  );
}
