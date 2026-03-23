import { useState, type FormEvent } from 'react';
import { Link } from '@tanstack/react-router';
import { useAuth } from '../lib/auth';
import { registerWithVerification, resendVerification } from '../api/auth';
import { ApiError } from '../api/client';
import { GoogleSignInButton } from '../components/GoogleSignInButton';

export function RegisterPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [registered, setRegistered] = useState(false);
  const [resendLoading, setResendLoading] = useState(false);
  const [resendMessage, setResendMessage] = useState('');
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
      await registerWithVerification(email, password);
      setRegistered(true);
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

  const handleResend = async () => {
    setResendLoading(true);
    setResendMessage('');
    try {
      await resendVerification(email);
      setResendMessage('Verification email sent!');
    } catch (err) {
      if (err instanceof ApiError) {
        setResendMessage(err.message);
      } else {
        setResendMessage('Failed to resend. Please try again.');
      }
    } finally {
      setResendLoading(false);
    }
  };

  const handleGoogleSuccess = (accessToken: string, refreshToken: string, googleEmail: string) => {
    login(accessToken, refreshToken, googleEmail);
    window.location.href = '/dashboard';
  };

  const handleGoogleError = (errorMessage: string) => {
    setError(errorMessage);
  };

  if (registered) {
    return (
      <div className="flex min-h-[calc(100vh-8rem)] items-center justify-center px-4">
        <div className="w-full max-w-sm text-center">
          <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-emerald-500/10 border border-emerald-500/20">
            <svg className="h-8 w-8 text-emerald-400" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" d="M21.75 6.75v10.5a2.25 2.25 0 01-2.25 2.25h-15a2.25 2.25 0 01-2.25-2.25V6.75m19.5 0A2.25 2.25 0 0019.5 4.5h-15a2.25 2.25 0 00-2.25 2.25m19.5 0v.243a2.25 2.25 0 01-1.07 1.916l-7.5 4.615a2.25 2.25 0 01-2.36 0L3.32 8.91a2.25 2.25 0 01-1.07-1.916V6.75" />
            </svg>
          </div>
          <h1 className="mt-4 text-2xl font-bold text-brand-light">Check your email</h1>
          <p className="mt-2 text-sm text-brand-shade2">
            We sent a verification link to{' '}
            <span className="font-medium text-brand-light">{email}</span>.
            <br />
            Click the link to activate your account.
          </p>

          <div className="mt-6">
            <button
              onClick={handleResend}
              disabled={resendLoading}
              className="text-sm text-brand-accent hover:text-brand-accent-hover transition-colors disabled:opacity-50"
            >
              {resendLoading ? 'Sending...' : 'Resend verification email'}
            </button>
            {resendMessage && (
              <p className="mt-2 text-xs text-brand-shade2">{resendMessage}</p>
            )}
          </div>

          <p className="mt-6 text-center text-sm text-brand-shade2">
            <Link to="/login" className="text-brand-accent hover:text-brand-accent-hover">
              Back to Login
            </Link>
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-[calc(100vh-8rem)] items-center justify-center px-4">
      <div className="w-full max-w-sm">
        <h1 className="text-2xl font-bold text-brand-light text-center">Sign Up</h1>
        <p className="mt-2 text-sm text-brand-shade2 text-center">
          Create your account to get started.
        </p>

        <div className="mt-8">
          <GoogleSignInButton
            onSuccess={handleGoogleSuccess}
            onError={handleGoogleError}
            text="signup_with"
          />
        </div>

        <div className="mt-4 flex items-center gap-3">
          <div className="h-px flex-1 bg-brand-shade3/30" />
          <span className="text-xs text-brand-shade3">or</span>
          <div className="h-px flex-1 bg-brand-shade3/30" />
        </div>

        <form onSubmit={handleSubmit} className="mt-4 space-y-4">
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
