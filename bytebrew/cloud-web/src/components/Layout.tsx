import { useState } from 'react';
import { Link, Outlet, useNavigate } from '@tanstack/react-router';
import { useAuth } from '../lib/auth';
import { SHOW_EE_PRICING } from '../lib/feature-flags';

export function RootLayout() {
  const { isAuthenticated, email, logout } = useAuth();
  const navigate = useNavigate();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const handleLogout = () => {
    logout();
    navigate({ to: '/' });
  };

  const closeMobileMenu = () => setMobileMenuOpen(false);

  return (
    <div className="min-h-screen flex flex-col">
      {/* Navigation */}
      <nav className="border-b border-brand-shade3/15 bg-brand-dark/80 backdrop-blur-sm sticky top-0 z-50">
        <div className="max-w-6xl mx-auto px-4 sm:px-6">
          <div className="flex items-center justify-between h-14">
            {/* Logo */}
            <Link to="/" className="flex items-center gap-2" onClick={closeMobileMenu}>
              <img src="/logo-dark.svg" alt="ByteBrew" className="h-8" />
            </Link>

            {/* Desktop nav links */}
            <div className="hidden md:flex items-center gap-4">
              {isAuthenticated ? (
                <AuthenticatedNav
                  email={email}
                  onLogout={handleLogout}
                />
              ) : (
                <UnauthenticatedNav />
              )}
            </div>

            {/* Mobile hamburger */}
            <button
              className="md:hidden text-brand-shade2 hover:text-brand-light transition-colors"
              onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
              aria-label="Toggle menu"
            >
              <svg
                className="w-6 h-6"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                {mobileMenuOpen ? (
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M6 18L18 6M6 6l12 12"
                  />
                ) : (
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M4 6h16M4 12h16M4 18h16"
                  />
                )}
              </svg>
            </button>
          </div>

          {/* Mobile menu */}
          {mobileMenuOpen && (
            <div className="md:hidden border-t border-brand-shade3/15 py-3 flex flex-col gap-2">
              {isAuthenticated ? (
                <AuthenticatedNavMobile
                  email={email}
                  onLogout={handleLogout}
                  onLinkClick={closeMobileMenu}
                />
              ) : (
                <UnauthenticatedNavMobile onLinkClick={closeMobileMenu} />
              )}
            </div>
          )}
        </div>
      </nav>

      {/* Content */}
      <main className="flex-1">
        <Outlet />
      </main>

      {/* Footer */}
      <footer className="border-t border-brand-shade3/15 py-6">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 flex flex-col sm:flex-row items-center justify-center gap-3 text-xs text-brand-shade3">
          <span>ByteBrew Engine &mdash; Open infrastructure for AI agents</span>
          <span className="hidden sm:inline">&middot;</span>
          <div className="flex items-center gap-3">
            <Link to="/privacy" className="hover:text-brand-shade2 transition-colors">
              Privacy Policy
            </Link>
            <span>&middot;</span>
            <Link to="/terms" className="hover:text-brand-shade2 transition-colors">
              Terms of Service
            </Link>
          </div>
          <span className="hidden sm:inline">&middot;</span>
          <span>&copy; 2026 ByteBrew</span>
        </div>
      </footer>
    </div>
  );
}

const navLinkClass = 'text-sm text-brand-shade2 hover:text-brand-light transition-colors';
const mobileNavLinkClass = 'block text-sm text-brand-shade2 hover:text-brand-light transition-colors py-1';

function UnauthenticatedNav() {
  return (
    <>
      <Link to="/docs" className={navLinkClass}>
        Docs
      </Link>
      <Link to="/pricing" className={navLinkClass}>
        Pricing
      </Link>
      <Link to="/download" className={navLinkClass}>
        Download
      </Link>
      <Link to="/login" className={navLinkClass}>
        Login
      </Link>
      <Link
        to="/download"
        className="rounded-[10px] bg-brand-accent px-4 py-1.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors"
      >
        Get Started
      </Link>
    </>
  );
}

function UnauthenticatedNavMobile({ onLinkClick }: { onLinkClick: () => void }) {
  return (
    <>
      <Link to="/docs" className={mobileNavLinkClass} onClick={onLinkClick}>
        Docs
      </Link>
      <Link to="/pricing" className={mobileNavLinkClass} onClick={onLinkClick}>
        Pricing
      </Link>
      <Link to="/download" className={mobileNavLinkClass} onClick={onLinkClick}>
        Download
      </Link>
      <Link to="/login" className={mobileNavLinkClass} onClick={onLinkClick}>
        Login
      </Link>
      <Link
        to="/download"
        className="mt-1 inline-block rounded-[10px] bg-brand-accent px-4 py-1.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors"
        onClick={onLinkClick}
      >
        Get Started
      </Link>
    </>
  );
}

function AuthenticatedNav({
  email,
  onLogout,
}: {
  email: string | null;
  onLogout: () => void;
}) {
  return (
    <>
      <Link to="/dashboard" className={navLinkClass}>
        Dashboard
      </Link>
      {SHOW_EE_PRICING && (
        <Link to="/billing" className={navLinkClass}>
          Billing
        </Link>
      )}
      <Link to="/settings" className={navLinkClass}>
        Settings
      </Link>
      <div className="h-4 w-px bg-brand-shade3/30" />
      <span className="text-xs text-brand-shade3">{email}</span>
      <button
        onClick={onLogout}
        className="text-sm text-brand-shade2 hover:text-brand-light transition-colors"
      >
        Logout
      </button>
    </>
  );
}

function AuthenticatedNavMobile({
  email,
  onLogout,
  onLinkClick,
}: {
  email: string | null;
  onLogout: () => void;
  onLinkClick: () => void;
}) {
  return (
    <>
      <Link to="/dashboard" className={mobileNavLinkClass} onClick={onLinkClick}>
        Dashboard
      </Link>
      {SHOW_EE_PRICING && (
        <Link to="/billing" className={mobileNavLinkClass} onClick={onLinkClick}>
          Billing
        </Link>
      )}
      <Link to="/settings" className={mobileNavLinkClass} onClick={onLinkClick}>
        Settings
      </Link>
      <div className="border-t border-brand-shade3/15 my-1" />
      <span className="text-xs text-brand-shade3 py-1">{email}</span>
      <button
        onClick={() => {
          onLogout();
          onLinkClick();
        }}
        className="text-left text-sm text-brand-shade2 hover:text-brand-light transition-colors py-1"
      >
        Logout
      </button>
    </>
  );
}
