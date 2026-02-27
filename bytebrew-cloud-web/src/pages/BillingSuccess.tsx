import { Link } from '@tanstack/react-router';

export function BillingSuccessPage() {
  return (
    <div className="max-w-lg mx-auto px-4 py-20 text-center">
      <div className="text-5xl mb-6">&#10003;</div>
      <h1 className="text-2xl font-bold text-white">Payment Successful</h1>
      <p className="mt-4 text-gray-400">
        Your subscription is now active. You can start using ByteBrew right away.
      </p>

      <div className="mt-8 rounded-xl border border-gray-800 bg-gray-900/50 p-6 text-left">
        <h2 className="text-sm font-medium text-gray-400 mb-3">Next steps</h2>
        <ol className="space-y-3 text-sm text-gray-300">
          <li>
            <span className="font-mono text-white">1.</span>{' '}
            Open a terminal and run{' '}
            <code className="rounded bg-gray-800 px-2 py-0.5 text-blue-400">bytebrew login</code>
          </li>
          <li>
            <span className="font-mono text-white">2.</span>{' '}
            Enter your email and password
          </li>
          <li>
            <span className="font-mono text-white">3.</span>{' '}
            Start coding:{' '}
            <code className="rounded bg-gray-800 px-2 py-0.5 text-blue-400">bytebrew</code>
          </li>
        </ol>
      </div>

      <div className="mt-8 flex gap-4 justify-center">
        <Link
          to="/dashboard"
          className="rounded-lg bg-blue-600 px-6 py-2.5 text-sm font-medium text-white hover:bg-blue-500 transition-colors"
        >
          Go to Dashboard
        </Link>
        <Link
          to="/billing"
          className="rounded-lg border border-gray-700 px-6 py-2.5 text-sm font-medium text-gray-300 hover:border-gray-500 hover:text-white transition-colors"
        >
          View Billing
        </Link>
      </div>
    </div>
  );
}
