import { useState, useEffect, useRef } from 'react';
import { Link, useNavigate } from '@tanstack/react-router';
import { useAuth } from '../lib/auth';
import { verifyEmail, resendVerification } from '../api/auth';
import { ApiError } from '../api/client';

export function VerifyEmailPage() {
  const [status, setStatus] = useState<'verifying' | 'success' | 'error'>('verifying');
  const [error, setError] = useState('');
  const [resendEmail, setResendEmail] = useState('');
  const [resendLoading, setResendLoading] = useState(false);
  const [resendMessage, setResendMessage] = useState('');
  const { login } = useAuth();
  const navigate = useNavigate();
  const verifiedRef = useRef(false);

  useEffect(() => {
    if (verifiedRef.current) {
      return;
    }
    verifiedRef.current = true;

    const params = new URLSearchParams(window.location.search);
    const token = params.get('token');

    if (!token) {
      setStatus('error');
      setError('Verification token is missing.');
      return;
    }

    verifyEmail(token)
      .then((res) => {
        login(res.access_token, res.refresh_token, '');
        setStatus('success');
        setTimeout(() => {
          navigate({ to: '/dashboard' });
        }, 2000);
      })
      .catch((err) => {
        setStatus('error');
        if (err instanceof ApiError) {
          setError(err.message);
        } else {
          setError('Verification failed. The link may be expired or invalid.');
        }
      });
  }, [login, navigate]);

  const handleResend = async () => {
    if (!resendEmail) {
      return;
    }
    setResendLoading(true);
    setResendMessage('');
    try {
      await resendVerification(resendEmail);
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

  return (
    <div className="flex min-h-[calc(100vh-8rem)] items-center justify-center px-4">
      <div className="w-full max-w-sm text-center">
        {status === 'verifying' && (
          <>
            <div className="mx-auto flex h-16 w-16 items-center justify-center">
              <svg className="h-8 w-8 text-brand-shade2 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
            </div>
            <h1 className="mt-4 text-2xl font-bold text-brand-light">Verifying your email...</h1>
            <p className="mt-2 text-sm text-brand-shade2">Please wait a moment.</p>
          </>
        )}

        {status === 'success' && (
          <>
            <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-emerald-500/10 border border-emerald-500/20">
              <svg className="h-8 w-8 text-emerald-400" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 12.75l6 6 9-13.5" />
              </svg>
            </div>
            <h1 className="mt-4 text-2xl font-bold text-brand-light">Email verified!</h1>
            <p className="mt-2 text-sm text-brand-shade2">Redirecting to dashboard...</p>
          </>
        )}

        {status === 'error' && (
          <>
            <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-red-500/10 border border-red-500/20">
              <svg className="h-8 w-8 text-red-400" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </div>
            <h1 className="mt-4 text-2xl font-bold text-brand-light">Verification failed</h1>
            <p className="mt-2 text-sm text-red-400">{error}</p>

            <div className="mt-6 space-y-3">
              <p className="text-sm text-brand-shade2">Need a new verification link?</p>
              <div className="flex gap-2">
                <input
                  type="email"
                  value={resendEmail}
                  onChange={(e) => setResendEmail(e.target.value)}
                  className="flex-1 rounded-[2px] border border-brand-shade3/30 bg-brand-dark px-3 py-2 text-sm text-brand-light placeholder-brand-shade3 focus:border-brand-accent focus:outline-none focus:ring-1 focus:ring-brand-accent"
                  placeholder="you@example.com"
                />
                <button
                  onClick={handleResend}
                  disabled={resendLoading || !resendEmail}
                  className="rounded-[2px] bg-brand-accent px-4 py-2 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {resendLoading ? 'Sending...' : 'Resend'}
                </button>
              </div>
              {resendMessage && (
                <p className="text-xs text-brand-shade2">{resendMessage}</p>
              )}
            </div>

            <p className="mt-6 text-sm text-brand-shade2">
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
