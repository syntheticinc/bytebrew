import { Link } from '@tanstack/react-router';

export function BillingCancelPage() {
  return (
    <div className="max-w-lg mx-auto px-4 py-20 text-center">
      <h1 className="text-2xl font-bold text-white">Checkout Cancelled</h1>
      <p className="mt-4 text-gray-400">
        No worries — you can try again whenever you're ready.
      </p>

      <div className="mt-8 flex gap-4 justify-center">
        <Link
          to="/billing"
          className="rounded-lg bg-blue-600 px-6 py-2.5 text-sm font-medium text-white hover:bg-blue-500 transition-colors"
        >
          Back to Billing
        </Link>
        <Link
          to="/"
          className="rounded-lg border border-gray-700 px-6 py-2.5 text-sm font-medium text-gray-300 hover:border-gray-500 hover:text-white transition-colors"
        >
          Home
        </Link>
      </div>
    </div>
  );
}
