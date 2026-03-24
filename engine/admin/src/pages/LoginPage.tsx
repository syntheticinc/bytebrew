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
    <div className="flex min-h-screen items-center justify-center bg-brand-dark px-4">
      <div className="w-full max-w-sm">
        <div className="rounded-card bg-brand-dark-alt p-8 shadow-lg">
          {/* Logo */}
          <div className="mb-8 flex flex-col items-center">
            <img src={import.meta.env.BASE_URL + 'logo-dark.svg'} alt="ByteBrew" className="mb-3 h-10" />
            <p className="text-sm text-brand-shade3">Sign in to manage your engine</p>
          </div>

          {error && (
            <div className="mb-4 p-3 rounded-btn border border-red-500/30 bg-red-500/10 text-sm text-red-400">
              {error}
            </div>
          )}

          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <div>
              <label htmlFor="username" className="mb-1 block text-xs font-medium text-brand-shade2">
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
                className="w-full rounded-btn border border-brand-shade3/50 bg-brand-dark-alt px-3 py-2 text-sm text-brand-light placeholder-brand-shade3 outline-none transition-colors focus:border-brand-accent"
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
                required
                placeholder="password"
                className="w-full rounded-btn border border-brand-shade3/50 bg-brand-dark-alt px-3 py-2 text-sm text-brand-light placeholder-brand-shade3 outline-none transition-colors focus:border-brand-accent"
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
