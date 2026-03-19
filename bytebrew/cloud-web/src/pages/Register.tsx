import { useState, type FormEvent } from 'react';
import { Link } from '@tanstack/react-router';
import { useAuth } from '../lib/auth';
import { register as registerApi } from '../api/auth';
import { ApiError } from '../api/client';

export function RegisterPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');

    if (password !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    if (password.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }

    setLoading(true);

    try {
      const res = await registerApi(email, password);
      login(res.access_token, res.refresh_token, email);
      window.location.href = '/dashboard';
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError('An unexpected error occurred');
      }
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-[calc(100vh-8rem)] items-center justify-center px-4">
      <div className="w-full max-w-sm">
        <h1 className="text-2xl font-bold text-brand-light text-center">Sign Up</h1>
        <p className="mt-2 text-sm text-brand-shade2 text-center">
          Create your account to get started.
        </p>

        <form onSubmit={handleSubmit} className="mt-8 space-y-4">
          {error && (
            <div className="rounded-[10px] bg-red-500/10 border border-red-500/20 p-3 text-sm text-red-400">
              {error}
            </div>
          )}

          <div>
            <label htmlFor="email" className="block text-sm font-medium text-brand-shade2 mb-1">
              Email
            </label>
            <input
              id="email"
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full rounded-[10px] border border-brand-shade3/30 bg-brand-dark px-3 py-2 text-sm text-brand-light placeholder-brand-shade3 focus:border-brand-accent focus:outline-none focus:ring-1 focus:ring-brand-accent"
              placeholder="you@example.com"
            />
          </div>

          <div>
            <label htmlFor="password" className="block text-sm font-medium text-brand-shade2 mb-1">
              Password
            </label>
            <input
              id="password"
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full rounded-[10px] border border-brand-shade3/30 bg-brand-dark px-3 py-2 text-sm text-brand-light placeholder-brand-shade3 focus:border-brand-accent focus:outline-none focus:ring-1 focus:ring-brand-accent"
              placeholder="At least 8 characters"
            />
          </div>

          <div>
            <label
              htmlFor="confirmPassword"
              className="block text-sm font-medium text-brand-shade2 mb-1"
            >
              Confirm Password
            </label>
            <input
              id="confirmPassword"
              type="password"
              required
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              className="w-full rounded-[10px] border border-brand-shade3/30 bg-brand-dark px-3 py-2 text-sm text-brand-light placeholder-brand-shade3 focus:border-brand-accent focus:outline-none focus:ring-1 focus:ring-brand-accent"
              placeholder="Repeat your password"
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-[10px] bg-brand-accent py-2.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? 'Creating account...' : 'Create Account'}
          </button>
        </form>

        <p className="mt-4 text-center text-xs text-brand-shade3">
          By creating an account, you agree to our{' '}
          <Link to="/terms" className="text-brand-accent hover:text-brand-accent-hover">
            Terms of Service
          </Link>
          .
        </p>

        <p className="mt-6 text-center text-sm text-brand-shade2">
          Already have an account?{' '}
          <Link to="/login" className="text-brand-accent hover:text-brand-accent-hover">
            Sign in
          </Link>
        </p>
      </div>
    </div>
  );
}
