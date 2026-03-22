import { useState, useEffect, useCallback, type FormEvent } from 'react';
import { login as loginApi, register as registerApi, forgotPassword } from '../api/auth';
import { ApiError } from '../api/client';
import { GoogleSignInButton } from './GoogleSignInButton';

type AuthView = 'login' | 'register' | 'forgot';

interface AuthPopupProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: (accessToken: string, refreshToken: string, email: string) => void;
  title?: string;
}

const inputClass =
  'w-full rounded-[10px] border border-brand-shade3/30 bg-brand-dark px-3 py-2 text-sm text-brand-light placeholder-brand-shade3 focus:border-brand-accent focus:outline-none focus:ring-1 focus:ring-brand-accent';
const labelClass = 'block text-sm font-medium text-brand-shade2 mb-1';
const submitBtnClass =
  'w-full rounded-[10px] bg-brand-accent py-2.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors disabled:opacity-50 disabled:cursor-not-allowed';
const linkBtnClass = 'text-brand-accent hover:text-brand-accent-hover transition-colors';

export function AuthPopup({ isOpen, onClose, onSuccess, title }: AuthPopupProps) {
  const [view, setView] = useState<AuthView>('login');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  // Login fields
  const [loginEmail, setLoginEmail] = useState('');
  const [loginPassword, setLoginPassword] = useState('');

  // Register fields
  const [regEmail, setRegEmail] = useState('');
  const [regPassword, setRegPassword] = useState('');
  const [regConfirm, setRegConfirm] = useState('');

  // Forgot fields
  const [forgotEmail, setForgotEmail] = useState('');
  const [forgotSent, setForgotSent] = useState(false);

  // Reset state when popup opens/closes
  useEffect(() => {
    if (isOpen) {
      setView('login');
      setError('');
      setLoading(false);
      setLoginEmail('');
      setLoginPassword('');
      setRegEmail('');
      setRegPassword('');
      setRegConfirm('');
      setForgotEmail('');
      setForgotSent(false);
    }
  }, [isOpen]);

  // Close on Escape
  useEffect(() => {
    if (!isOpen) {
      return;
    }

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  // Prevent body scroll when open
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden';
      return () => {
        document.body.style.overflow = '';
      };
    }
  }, [isOpen]);

  const switchView = useCallback((newView: AuthView) => {
    setView(newView);
    setError('');
  }, []);

  const handleGoogleSuccess = useCallback(
    (accessToken: string, refreshToken: string, email: string) => {
      onSuccess(accessToken, refreshToken, email);
    },
    [onSuccess],
  );

  const handleGoogleError = useCallback((errorMsg: string) => {
    setError(errorMsg);
  }, []);

  const handleLogin = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const res = await loginApi(loginEmail, loginPassword);
      onSuccess(res.access_token, res.refresh_token, loginEmail);
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

  const handleRegister = async (e: FormEvent) => {
    e.preventDefault();
    setError('');

    if (regPassword !== regConfirm) {
      setError('Passwords do not match');
      return;
    }

    if (regPassword.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }

    setLoading(true);

    try {
      const res = await registerApi(regEmail, regPassword);
      onSuccess(res.access_token, res.refresh_token, regEmail);
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

  const handleForgotPassword = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await forgotPassword(forgotEmail);
      setForgotSent(true);
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

  if (!isOpen) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm animate-in fade-in duration-200"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="relative z-10 w-full max-w-sm mx-4 rounded-[12px] border border-brand-shade3/20 bg-brand-dark-alt shadow-2xl animate-in zoom-in-95 fade-in duration-200">
        {/* Close button */}
        <button
          onClick={onClose}
          className="absolute right-3 top-3 text-brand-shade3 hover:text-brand-light transition-colors"
          aria-label="Close"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>

        <div className="p-6">
          {/* Header */}
          <h2 className="text-xl font-bold text-brand-light text-center">
            {title ?? getDefaultTitle(view)}
          </h2>
          <p className="mt-1 text-sm text-brand-shade2 text-center">
            {getSubtitle(view)}
          </p>

          <div className="mt-6">
            {view === 'login' && (
              <LoginView
                email={loginEmail}
                password={loginPassword}
                error={error}
                loading={loading}
                onEmailChange={setLoginEmail}
                onPasswordChange={setLoginPassword}
                onSubmit={handleLogin}
                onSwitchToRegister={() => switchView('register')}
                onSwitchToForgot={() => switchView('forgot')}
                onGoogleSuccess={handleGoogleSuccess}
                onGoogleError={handleGoogleError}
              />
            )}

            {view === 'register' && (
              <RegisterView
                email={regEmail}
                password={regPassword}
                confirmPassword={regConfirm}
                error={error}
                loading={loading}
                onEmailChange={setRegEmail}
                onPasswordChange={setRegPassword}
                onConfirmChange={setRegConfirm}
                onSubmit={handleRegister}
                onSwitchToLogin={() => switchView('login')}
                onGoogleSuccess={handleGoogleSuccess}
                onGoogleError={handleGoogleError}
              />
            )}

            {view === 'forgot' && (
              <ForgotView
                email={forgotEmail}
                error={error}
                loading={loading}
                sent={forgotSent}
                onEmailChange={setForgotEmail}
                onSubmit={handleForgotPassword}
                onSwitchToLogin={() => switchView('login')}
              />
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function getDefaultTitle(view: AuthView): string {
  switch (view) {
    case 'login':
      return 'Welcome back';
    case 'register':
      return 'Create account';
    case 'forgot':
      return 'Reset password';
  }
}

function getSubtitle(view: AuthView): string {
  switch (view) {
    case 'login':
      return 'Sign in to your ByteBrew account';
    case 'register':
      return 'Get started with ByteBrew';
    case 'forgot':
      return "Enter your email and we'll send you a reset link";
  }
}

// --- Sub-views ---

function ErrorBanner({ message }: { message: string }) {
  if (!message) {
    return null;
  }

  return (
    <div className="rounded-[10px] bg-red-500/10 border border-red-500/20 p-3 text-sm text-red-400">
      {message}
    </div>
  );
}

function OrDivider() {
  return (
    <div className="relative my-5">
      <div className="absolute inset-0 flex items-center">
        <div className="w-full border-t border-brand-shade3/20" />
      </div>
      <div className="relative flex justify-center text-xs">
        <span className="bg-brand-dark-alt px-3 text-brand-shade3">or</span>
      </div>
    </div>
  );
}

interface LoginViewProps {
  email: string;
  password: string;
  error: string;
  loading: boolean;
  onEmailChange: (v: string) => void;
  onPasswordChange: (v: string) => void;
  onSubmit: (e: FormEvent) => void;
  onSwitchToRegister: () => void;
  onSwitchToForgot: () => void;
  onGoogleSuccess: (accessToken: string, refreshToken: string, email: string) => void;
  onGoogleError: (error: string) => void;
}

function LoginView({
  email,
  password,
  error,
  loading,
  onEmailChange,
  onPasswordChange,
  onSubmit,
  onSwitchToRegister,
  onSwitchToForgot,
  onGoogleSuccess,
  onGoogleError,
}: LoginViewProps) {
  return (
    <div className="space-y-4">
      <GoogleSignInButton onSuccess={onGoogleSuccess} onError={onGoogleError} text="signin_with" />
      <OrDivider />

      <form onSubmit={onSubmit} className="space-y-4">
        <ErrorBanner message={error} />

        <div>
          <label className={labelClass}>Email</label>
          <input
            type="email"
            required
            value={email}
            onChange={(e) => onEmailChange(e.target.value)}
            className={inputClass}
            placeholder="you@example.com"
          />
        </div>

        <div>
          <label className={labelClass}>Password</label>
          <input
            type="password"
            required
            value={password}
            onChange={(e) => onPasswordChange(e.target.value)}
            className={inputClass}
            placeholder="Enter your password"
          />
        </div>

        <div className="text-right">
          <button type="button" onClick={onSwitchToForgot} className={`text-sm ${linkBtnClass}`}>
            Forgot your password?
          </button>
        </div>

        <button type="submit" disabled={loading} className={submitBtnClass}>
          {loading ? 'Signing in...' : 'Sign In'}
        </button>
      </form>

      <p className="text-center text-sm text-brand-shade2">
        Don't have an account?{' '}
        <button type="button" onClick={onSwitchToRegister} className={linkBtnClass}>
          Sign Up
        </button>
      </p>
    </div>
  );
}

interface RegisterViewProps {
  email: string;
  password: string;
  confirmPassword: string;
  error: string;
  loading: boolean;
  onEmailChange: (v: string) => void;
  onPasswordChange: (v: string) => void;
  onConfirmChange: (v: string) => void;
  onSubmit: (e: FormEvent) => void;
  onSwitchToLogin: () => void;
  onGoogleSuccess: (accessToken: string, refreshToken: string, email: string) => void;
  onGoogleError: (error: string) => void;
}

function RegisterView({
  email,
  password,
  confirmPassword,
  error,
  loading,
  onEmailChange,
  onPasswordChange,
  onConfirmChange,
  onSubmit,
  onSwitchToLogin,
  onGoogleSuccess,
  onGoogleError,
}: RegisterViewProps) {
  return (
    <div className="space-y-4">
      <GoogleSignInButton onSuccess={onGoogleSuccess} onError={onGoogleError} text="signup_with" />
      <OrDivider />

      <form onSubmit={onSubmit} className="space-y-4">
        <ErrorBanner message={error} />

        <div>
          <label className={labelClass}>Email</label>
          <input
            type="email"
            required
            value={email}
            onChange={(e) => onEmailChange(e.target.value)}
            className={inputClass}
            placeholder="you@example.com"
          />
        </div>

        <div>
          <label className={labelClass}>Password</label>
          <input
            type="password"
            required
            value={password}
            onChange={(e) => onPasswordChange(e.target.value)}
            className={inputClass}
            placeholder="At least 8 characters"
          />
        </div>

        <div>
          <label className={labelClass}>Confirm Password</label>
          <input
            type="password"
            required
            value={confirmPassword}
            onChange={(e) => onConfirmChange(e.target.value)}
            className={inputClass}
            placeholder="Repeat your password"
          />
        </div>

        <button type="submit" disabled={loading} className={submitBtnClass}>
          {loading ? 'Creating account...' : 'Create Account'}
        </button>
      </form>

      <p className="text-center text-sm text-brand-shade2">
        Already have an account?{' '}
        <button type="button" onClick={onSwitchToLogin} className={linkBtnClass}>
          Sign in
        </button>
      </p>
    </div>
  );
}

interface ForgotViewProps {
  email: string;
  error: string;
  loading: boolean;
  sent: boolean;
  onEmailChange: (v: string) => void;
  onSubmit: (e: FormEvent) => void;
  onSwitchToLogin: () => void;
}

function ForgotView({
  email,
  error,
  loading,
  sent,
  onEmailChange,
  onSubmit,
  onSwitchToLogin,
}: ForgotViewProps) {
  if (sent) {
    return (
      <div className="space-y-4">
        <div className="rounded-[10px] bg-emerald-500/10 border border-emerald-500/20 p-4 text-sm text-emerald-400">
          If an account exists with this email, you'll receive a reset link.
        </div>
        <p className="text-center text-sm text-brand-shade2">
          <button type="button" onClick={onSwitchToLogin} className={linkBtnClass}>
            Back to Sign In
          </button>
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <form onSubmit={onSubmit} className="space-y-4">
        <ErrorBanner message={error} />

        <div>
          <label className={labelClass}>Email</label>
          <input
            type="email"
            required
            value={email}
            onChange={(e) => onEmailChange(e.target.value)}
            className={inputClass}
            placeholder="you@example.com"
          />
        </div>

        <button type="submit" disabled={loading} className={submitBtnClass}>
          {loading ? 'Sending...' : 'Send Reset Link'}
        </button>
      </form>

      <p className="text-center text-sm text-brand-shade2">
        <button type="button" onClick={onSwitchToLogin} className={linkBtnClass}>
          Back to Sign In
        </button>
      </p>
    </div>
  );
}
