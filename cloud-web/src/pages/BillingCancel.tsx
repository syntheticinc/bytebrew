import { Link } from '@tanstack/react-router';

export function BillingCancelPage() {
  return (
    <div className="max-w-lg mx-auto px-4 py-20 text-center">
      <h1 className="text-2xl font-bold text-brand-light">Checkout Cancelled</h1>
      <p className="mt-4 text-brand-shade2">
        No worries — you can try again whenever you're ready.
      </p>

      <div className="mt-8 flex gap-4 justify-center">
        <Link
          to="/billing"
          className="rounded-[10px] bg-brand-accent px-6 py-2.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors"
        >
          Back to Billing
        </Link>
        <Link
          to="/"
          className="rounded-[10px] border border-brand-shade3/20 px-6 py-2.5 text-sm font-medium text-brand-shade2 hover:border-brand-shade3/40 hover:text-brand-light transition-colors"
        >
          Home
        </Link>
      </div>
    </div>
  );
}
