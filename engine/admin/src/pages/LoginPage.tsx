import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';

export default function LoginPage() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await login(username, password);
      navigate('/health');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div
      className="flex min-h-screen items-center justify-center bg-brand-dark px-4 relative"
      style={{
        backgroundImage: 'radial-gradient(circle, #87867F12 1px, transparent 1px)',
        backgroundSize: '24px 24px',
      }}
    >
      <div className="w-full max-w-md animate-fade-in">
        <div className="rounded-card bg-brand-dark-alt p-10 shadow-2xl border border-brand-shade3/10 relative overflow-hidden">
          {/* Brand accent line at top */}
          <div className="absolute top-0 left-0 right-0 h-0.5 bg-brand-accent" />

          {/* Logo */}
          <div className="mb-10 flex flex-col items-center">
            <img src={import.meta.env.BASE_URL + 'logo-dark.svg'} alt="ByteBrew" className="mb-4 h-10" />
            <p className="text-sm text-brand-shade3">Sign in to manage your engine</p>
          </div>

          {error && (
            <div className="mb-5 p-3 rounded-btn border border-red-500/30 bg-red-500/10 text-sm text-red-400">
              {error}
            </div>
          )}

          <form onSubmit={handleSubmit} className="flex flex-col gap-5">
            <div>
              <label htmlFor="username" className="mb-1.5 block text-xs font-medium text-brand-shade2">
                Username
              </label>
              <input
                id="username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                autoFocus
                placeholder="admin"
                className="w-full rounded-btn border border-brand-shade3/30 bg-brand-dark px-3.5 py-2.5 text-sm text-brand-light placeholder-brand-shade3/60 outline-none transition-all duration-150 focus:border-brand-accent focus:ring-1 focus:ring-brand-accent/30"
              />
            </div>

            <div>
              <label htmlFor="password" className="mb-1.5 block text-xs font-medium text-brand-shade2">
                Password
              </label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                placeholder="password"
                className="w-full rounded-btn border border-brand-shade3/30 bg-brand-dark px-3.5 py-2.5 text-sm text-brand-light placeholder-brand-shade3/60 outline-none transition-all duration-150 focus:border-brand-accent focus:ring-1 focus:ring-brand-accent/30"
              />
            </div>

            <button
              type="submit"
              disabled={loading || !username || !password}
              className="mt-3 w-full rounded-btn bg-brand-accent py-2.5 text-sm font-medium text-white transition-all duration-150 hover:bg-brand-accent-hover disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {loading ? 'Signing in...' : 'Sign in'}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
