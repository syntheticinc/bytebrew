import { useState, type FormEvent } from 'react';
import { useAuth } from '../hooks/useAuth';
import { useNavigate } from 'react-router-dom';

export function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await login(username, password);
      navigate('/chat');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-brand-dark px-4">
      <div className="w-full max-w-sm">
        <div className="rounded-card bg-brand-dark-alt p-8 shadow-lg">
          {/* Logo */}
          <div className="mb-8 flex flex-col items-center">
            <img src="/logo-dark.svg" alt="ByteBrew" className="mb-3 h-10" />
            <p className="text-sm text-brand-shade3">Connect to ByteBrew Engine</p>
          </div>

          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            {error && (
              <div className="rounded-btn border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">
                {error}
              </div>
            )}

            <div>
              <label htmlFor="username" className="mb-1 block text-xs font-medium text-brand-shade2">
                Username
              </label>
              <input
                id="username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="w-full rounded-btn border border-brand-shade3/30 bg-brand-dark px-3 py-2 text-sm text-brand-light placeholder-brand-shade3 outline-none transition-colors focus:border-brand-accent"
                placeholder="admin"
                autoFocus
                required
              />
            </div>

            <div>
              <label htmlFor="password" className="mb-1 block text-xs font-medium text-brand-shade2">
                Password
              </label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full rounded-btn border border-brand-shade3/30 bg-brand-dark px-3 py-2 text-sm text-brand-light placeholder-brand-shade3 outline-none transition-colors focus:border-brand-accent"
                placeholder="password"
                required
              />
            </div>

            <button
              type="submit"
              disabled={loading || !username || !password}
              className="mt-2 w-full rounded-btn bg-brand-accent py-2.5 text-sm font-medium text-white transition-colors hover:bg-brand-accent-hover disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {loading ? 'Signing in...' : 'Sign in'}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
