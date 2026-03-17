import { Link, Outlet, useNavigate } from '@tanstack/react-router';
import { useAuth } from '../lib/auth';

export function RootLayout() {
  const { isAuthenticated, email, logout } = useAuth();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate({ to: '/' });
  };

  return (
    <div className="min-h-screen flex flex-col">
      {/* Navigation */}
      <nav className="border-b border-gray-800 bg-gray-950/80 backdrop-blur-sm sticky top-0 z-50">
        <div className="max-w-6xl mx-auto px-4 sm:px-6">
          <div className="flex items-center justify-between h-14">
            {/* Logo */}
            <Link to="/" className="flex items-center gap-2">
              <span className="text-lg font-bold text-white">ByteBrew</span>
              <span className="text-xs text-gray-500 font-mono">AI Agent</span>
            </Link>

            {/* Nav links */}
            <div className="flex items-center gap-4">
              {isAuthenticated ? (
                <>
                  <Link
                    to="/dashboard"
                    className="text-sm text-gray-300 hover:text-white transition-colors"
                  >
                    Dashboard
                  </Link>
                  <Link
                    to="/billing"
                    className="text-sm text-gray-300 hover:text-white transition-colors"
                  >
                    Billing
                  </Link>
                  <Link
                    to="/team"
                    className="text-sm text-gray-300 hover:text-white transition-colors"
                  >
                    Team
                  </Link>
                  <Link
                    to="/settings"
                    className="text-sm text-gray-300 hover:text-white transition-colors"
                  >
                    Settings
                  </Link>
                  <div className="h-4 w-px bg-gray-700" />
                  <span className="text-xs text-gray-500">{email}</span>
                  <button
                    onClick={handleLogout}
                    className="text-sm text-gray-400 hover:text-white transition-colors"
                  >
                    Logout
                  </button>
                </>
              ) : (
                <>
                  <Link
                    to="/login"
                    className="text-sm text-gray-300 hover:text-white transition-colors"
                  >
                    Login
                  </Link>
                  <Link
                    to="/register"
                    className="rounded-lg bg-indigo-600 px-4 py-1.5 text-sm font-medium text-white hover:bg-indigo-500 transition-colors"
                  >
                    Start Trial
                  </Link>
                </>
              )}
            </div>
          </div>
        </div>
      </nav>

      {/* Content */}
      <main className="flex-1">
        <Outlet />
      </main>

      {/* Footer */}
      <footer className="border-t border-gray-800 py-6">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 flex items-center justify-center gap-3 text-xs text-gray-600">
          <span>ByteBrew AI Agent &mdash; Built for software engineers</span>
          <span>&middot;</span>
          <Link to="/terms" className="hover:text-gray-400 transition-colors">
            Terms of Service
          </Link>
        </div>
      </footer>
    </div>
  );
}
