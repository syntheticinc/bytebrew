import { useState } from 'react';
import { Link, Outlet, useNavigate } from '@tanstack/react-router';
import { useAuth } from '../lib/auth';
import { useTheme } from '../lib/theme';
import { SHOW_EE_PRICING } from '../lib/feature-flags';

export function RootLayout() {
  const { isAuthenticated, email, logout } = useAuth();
  const { resolved } = useTheme();
  const navigate = useNavigate();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const logoSrc = resolved === 'light' ? '/logo-light.png' : '/logo-dark.svg';

  const handleLogout = () => {
    logout();
    navigate({ to: '/' });
  };

  const closeMobileMenu = () => setMobileMenuOpen(false);

  return (
    <div className="min-h-screen flex flex-col">
      {/* Navigation */}
      <nav className="border-b border-border bg-surface/80 backdrop-blur-sm sticky top-0 z-50">
        <div className="max-w-6xl mx-auto px-4 sm:px-6">
          <div className="flex items-center justify-between h-14">
            {/* Logo */}
            <Link to="/" className="flex items-center gap-2" onClick={closeMobileMenu}>
              <img src={logoSrc} alt="ByteBrew" className="h-8" />
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
              className="md:hidden text-text-secondary hover:text-text-primary transition-colors"
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
            <div className="md:hidden border-t border-border py-3 flex flex-col gap-2">
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
      <footer className="border-t border-border py-6">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 flex flex-col sm:flex-row items-center justify-center gap-3 text-xs text-text-tertiary">
          <span>ByteBrew Engine &mdash; Open infrastructure for AI agents</span>
          <span className="hidden sm:inline">&middot;</span>
          <div className="flex items-center gap-3">
            <Link to="/privacy" className="hover:text-text-secondary transition-colors">
              Privacy Policy
            </Link>
            <span>&middot;</span>
            <Link to="/terms" className="hover:text-text-secondary transition-colors">
              Terms of Service
            </Link>
          </div>
          <span className="hidden sm:inline">&middot;</span>
          <a href="https://github.com/syntheticinc/bytebrew" target="_blank" rel="noopener noreferrer" className="text-text-tertiary hover:text-text-primary transition-colors" title="GitHub">
            <svg className="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
            </svg>
          </a>
          <span className="hidden sm:inline">&middot;</span>
          <a href="https://github.com/syntheticinc/bytebrew-examples" target="_blank" rel="noopener noreferrer" className="text-text-tertiary hover:text-text-primary transition-colors text-xs">
            Examples on GitHub
          </a>
          <span className="hidden sm:inline">&middot;</span>
          <span>&copy; 2026 ByteBrew</span>
        </div>
      </footer>
    </div>
  );
}

function ThemeToggle() {
  const { resolved, toggleTheme } = useTheme();

  const isDark = resolved === 'dark';
  const label = isDark ? 'Dark theme' : 'Light theme';

  return (
    <button
      onClick={toggleTheme}
      className="cursor-pointer text-text-secondary opacity-60 hover:opacity-100 transition-opacity p-1"
      aria-label={label}
      title={label}
    >
      <div className="relative w-[18px] h-[18px]">
        {/* Sun icon — visible in dark mode (click switches to light) */}
        <svg
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth={2}
          className={`absolute inset-0 transition-all duration-300 ${isDark ? 'opacity-100 rotate-0' : 'opacity-0 -rotate-90'}`}
        >
          <circle cx="12" cy="12" r="5" />
          <line x1="12" y1="1" x2="12" y2="3" />
          <line x1="12" y1="21" x2="12" y2="23" />
          <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
          <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
          <line x1="1" y1="12" x2="3" y2="12" />
          <line x1="21" y1="12" x2="23" y2="12" />
          <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
          <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
        </svg>
        {/* Moon icon — visible in light mode (click switches to dark) */}
        <svg
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth={2}
          className={`absolute inset-0 transition-all duration-300 ${isDark ? 'opacity-0 rotate-90' : 'opacity-100 rotate-0'}`}
        >
          <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
        </svg>
      </div>
    </button>
  );
}

const navLinkClass = 'text-sm text-text-secondary hover:text-text-primary transition-colors';
const mobileNavLinkClass = 'block text-sm text-text-secondary hover:text-text-primary transition-colors py-1';

function UnauthenticatedNav() {
  return (
    <>
      <a href="/docs" className={navLinkClass}>
        Docs
      </a>
      <Link to="/examples" className={navLinkClass}>
        Examples
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
      <ThemeToggle />
      <Link
        to="/download"
        className="rounded-[2px] bg-brand-accent px-4 py-1.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors"
      >
        Get Started
      </Link>
    </>
  );
}

function UnauthenticatedNavMobile({ onLinkClick }: { onLinkClick: () => void }) {
  return (
    <>
      <a href="/docs" className={mobileNavLinkClass} onClick={onLinkClick}>
        Docs
      </a>
      <Link to="/examples" className={mobileNavLinkClass} onClick={onLinkClick}>
        Examples
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
      <div className="flex items-center gap-2 py-1">
        <ThemeToggle />
        <span className="text-xs text-text-tertiary">Theme</span>
      </div>
      <Link
        to="/download"
        className="mt-1 inline-block rounded-[2px] bg-brand-accent px-4 py-1.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors"
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
      <a href="/docs" className={navLinkClass}>
        Docs
      </a>
      <Link to="/examples" className={navLinkClass}>
        Examples
      </Link>
      <Link to="/pricing" className={navLinkClass}>
        Pricing
      </Link>
      <Link to="/download" className={navLinkClass}>
        Download
      </Link>
      <div className="h-4 w-px bg-border-hover" />
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
      <div className="h-4 w-px bg-border-hover" />
      <span className="text-xs text-text-tertiary">{email}</span>
      <button
        onClick={onLogout}
        className="text-sm text-text-secondary hover:text-text-primary transition-colors"
      >
        Logout
      </button>
      <ThemeToggle />
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
      <a href="/docs" className={mobileNavLinkClass} onClick={onLinkClick}>
        Docs
      </a>
      <Link to="/examples" className={mobileNavLinkClass} onClick={onLinkClick}>
        Examples
      </Link>
      <Link to="/pricing" className={mobileNavLinkClass} onClick={onLinkClick}>
        Pricing
      </Link>
      <Link to="/download" className={mobileNavLinkClass} onClick={onLinkClick}>
        Download
      </Link>
      <div className="border-t border-border my-1" />
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
      <div className="border-t border-border my-1" />
      <span className="text-xs text-text-tertiary py-1">{email}</span>
      <button
        onClick={() => {
          onLogout();
          onLinkClick();
        }}
        className="text-left text-sm text-text-secondary hover:text-text-primary transition-colors py-1"
      >
        Logout
      </button>
      <div className="flex items-center gap-2 py-1">
        <ThemeToggle />
        <span className="text-xs text-text-tertiary">Theme</span>
      </div>
    </>
  );
}
