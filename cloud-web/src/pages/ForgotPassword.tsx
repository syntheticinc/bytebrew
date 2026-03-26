import { useState, type FormEvent } from 'react';
import { Link } from '@tanstack/react-router';
import { forgotPassword } from '../api/auth';
import { ApiError } from '../api/client';

export function ForgotPasswordPage() {
  const [email, setEmail] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [sent, setSent] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await forgotPassword(email);
      setSent(true);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError('An unexpected error occurred');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-[calc(100vh-8rem)] items-center justify-center px-4">
      <div className="w-full max-w-sm">
        <h1 className="text-2xl font-bold text-text-primary text-center">Reset Password</h1>
        <p className="mt-2 text-sm text-text-secondary text-center">
          Enter your email and we'll send you a reset link.
        </p>

        {sent ? (
          <div className="mt-8">
            <div className="rounded-[2px] bg-emerald-500/10 border border-emerald-500/20 p-4 text-sm text-emerald-400">
              If an account exists with this email, you'll receive a reset link.
            </div>
            <p className="mt-6 text-center text-sm text-text-secondary">
              <Link to="/login" className="text-brand-accent hover:text-brand-accent-hover">
                Back to Login
              </Link>
            </p>
          </div>
        ) : (
          <>
            <form onSubmit={handleSubmit} className="mt-8 space-y-4">
              {error && (
                <div className="rounded-[2px] bg-red-500/10 border border-red-500/20 p-3 text-sm text-red-400">
                  {error}
                </div>
              )}

              <div>
                <label htmlFor="email" className="block text-sm font-medium text-text-secondary mb-1">
                  Email
                </label>
                <input
                  id="email"
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="w-full rounded-[2px] border border-border-hover bg-surface px-3 py-2 text-sm text-text-primary placeholder-text-tertiary focus:border-brand-accent focus:outline-none focus:ring-1 focus:ring-brand-accent"
                  placeholder="you@example.com"
                />
              </div>

              <button
                type="submit"
                disabled={loading}
                className="w-full rounded-[2px] bg-brand-accent py-2.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {loading ? 'Sending...' : 'Send Reset Link'}
              </button>
            </form>

            <p className="mt-6 text-center text-sm text-text-secondary">
              <Link to="/login" className="text-brand-accent hover:text-brand-accent-hover">
                Back to Login
              </Link>
            </p>
          </>
        )}
      </div>
    </div>
  );
}
