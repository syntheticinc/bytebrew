import { useState, type FormEvent } from 'react';
import { Link, useNavigate } from '@tanstack/react-router';
import { useAuth } from '../lib/auth';
import { login as loginApi } from '../api/auth';
import { ApiError } from '../api/client';

export function LoginPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const res = await loginApi(email, password);
      login(res.access_token, res.refresh_token, email);
      navigate({ to: '/dashboard' });
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
        <h1 className="text-2xl font-bold text-brand-light text-center">Welcome back</h1>
        <p className="mt-2 text-sm text-brand-shade2 text-center">
          Sign in to your ByteBrew account
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
              placeholder="Enter your password"
            />
          </div>

          <div className="text-right">
            <Link to="/forgot-password" className="text-sm text-brand-accent hover:text-brand-accent-hover">
              Forgot your password?
            </Link>
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-[10px] bg-brand-accent py-2.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? 'Signing in...' : 'Sign In'}
          </button>
        </form>

        <p className="mt-6 text-center text-sm text-brand-shade2">
          Don't have an account?{' '}
          <Link to="/register" className="text-brand-accent hover:text-brand-accent-hover">
            Sign Up
          </Link>
        </p>
      </div>
    </div>
  );
}
