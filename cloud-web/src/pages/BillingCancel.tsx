import { Link } from '@tanstack/react-router';

export function BillingCancelPage() {
  return (
    <div className="max-w-lg mx-auto px-4 py-20 text-center">
      <h1 className="text-2xl font-bold text-text-primary">Checkout Cancelled</h1>
      <p className="mt-4 text-text-secondary">
        No worries — you can try again whenever you're ready.
      </p>

      <div className="mt-8 flex gap-4 justify-center">
        <Link
          to="/billing"
          className="rounded-[2px] bg-brand-accent px-6 py-2.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors"
        >
          Back to Billing
        </Link>
        <Link
          to="/"
          className="rounded-[2px] border border-border px-6 py-2.5 text-sm font-medium text-text-secondary hover:border-border-hover hover:text-text-primary transition-colors"
        >
          Home
        </Link>
      </div>
    </div>
  );
}
