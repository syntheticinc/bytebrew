import { useState, type FormEvent } from 'react';
import { Link, useNavigate } from '@tanstack/react-router';
import { useAuth } from '../lib/auth';
import { login as loginApi, resendVerification } from '../api/auth';
import { ApiError } from '../api/client';
import { GoogleSignInButton } from '../components/GoogleSignInButton';

export function LoginPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [emailNotVerified, setEmailNotVerified] = useState(false);
  const [resendLoading, setResendLoading] = useState(false);
  const [resendMessage, setResendMessage] = useState('');
  const { login } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setEmailNotVerified(false);
    setResendMessage('');
    setLoading(true);

    try {
      const res = await loginApi(email, password);
      login(res.access_token, res.refresh_token, email);
      navigate({ to: '/dashboard' });
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.code === 'EMAIL_NOT_VERIFIED') {
          setEmailNotVerified(true);
        } else {
          setError(err.message);
        }
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
    navigate({ to: '/dashboard' });
  };

  const handleGoogleError = (errorMessage: string) => {
    setError(errorMessage);
  };

  return (
    <div className="flex min-h-[calc(100vh-8rem)] items-center justify-center px-4">
      <div className="w-full max-w-sm">
        <h1 className="text-2xl font-bold text-brand-light text-center">Welcome back</h1>
        <p className="mt-2 text-sm text-brand-shade2 text-center">
          Sign in to your ByteBrew account
        </p>

        <div className="mt-8">
          <GoogleSignInButton
            onSuccess={handleGoogleSuccess}
            onError={handleGoogleError}
            text="signin_with"
          />
        </div>

        <div className="mt-4 flex items-center gap-3">
          <div className="h-px flex-1 bg-brand-shade3/30" />
          <span className="text-xs text-brand-shade3">or</span>
          <div className="h-px flex-1 bg-brand-shade3/30" />
        </div>

        <form onSubmit={handleSubmit} className="mt-4 space-y-4">
          {error && (
            <div className="rounded-[2px] bg-red-500/10 border border-red-500/20 p-3 text-sm text-red-400">
              {error}
            </div>
          )}

          {emailNotVerified && (
            <div className="rounded-[2px] bg-amber-500/10 border border-amber-500/20 p-3 text-sm text-amber-400">
              <p>Please verify your email before signing in.</p>
              <button
                type="button"
                onClick={handleResend}
                disabled={resendLoading}
                className="mt-1 text-brand-accent hover:text-brand-accent-hover transition-colors disabled:opacity-50"
              >
                {resendLoading ? 'Sending...' : 'Resend verification email'}
              </button>
              {resendMessage && (
                <p className="mt-1 text-xs text-brand-shade2">{resendMessage}</p>
              )}
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
              className="w-full rounded-[2px] border border-brand-shade3/30 bg-brand-dark px-3 py-2 text-sm text-brand-light placeholder-brand-shade3 focus:border-brand-accent focus:outline-none focus:ring-1 focus:ring-brand-accent"
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
              className="w-full rounded-[2px] border border-brand-shade3/30 bg-brand-dark px-3 py-2 text-sm text-brand-light placeholder-brand-shade3 focus:border-brand-accent focus:outline-none focus:ring-1 focus:ring-brand-accent"
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
            className="w-full rounded-[2px] bg-brand-accent py-2.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
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
