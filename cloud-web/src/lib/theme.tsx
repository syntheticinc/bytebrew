import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react';

type Theme = 'dark' | 'light' | 'system';

interface ThemeContextValue {
  /** The user's chosen setting (dark | light | system) */
  theme: Theme;
  /** The resolved appearance (light | dark) */
  resolved: 'light' | 'dark';
  /** Alias for resolved — matches common convention */
  resolvedTheme: 'light' | 'dark';
  setTheme: (mode: Theme) => void;
  toggleTheme: () => void;
}

const STORAGE_KEY = 'bytebrew-theme';

const ThemeContext = createContext<ThemeContextValue | null>(null);

function getSystemPreference(): 'light' | 'dark' {
  if (typeof window === 'undefined') return 'dark';
  return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
}

function resolveTheme(mode: Theme): 'light' | 'dark' {
  if (mode === 'system') return getSystemPreference();
  return mode;
}

function readStoredTheme(): Theme {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === 'light' || stored === 'dark' || stored === 'system') return stored;
  } catch {
    // localStorage unavailable
  }
  return 'dark';
}

function applyThemeClass(resolved: 'light' | 'dark') {
  const html = document.documentElement;
  html.classList.remove('light', 'dark');
  html.classList.add(resolved);
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(readStoredTheme);
  const [resolved, setResolved] = useState<'light' | 'dark'>(() => resolveTheme(readStoredTheme()));

  const setTheme = useCallback((mode: Theme) => {
    setThemeState(mode);
    try {
      localStorage.setItem(STORAGE_KEY, mode);
      // Sync for docs-site (Starlight)
      localStorage.setItem('starlight-theme', mode === 'system' ? 'auto' : mode);
    } catch {
      // localStorage unavailable
    }
    const r = resolveTheme(mode);
    setResolved(r);
    applyThemeClass(r);
  }, []);

  const toggleTheme = useCallback(() => {
    setTheme(resolved === 'dark' ? 'light' : 'dark');
  }, [resolved, setTheme]);

  // Apply class on mount
  useEffect(() => {
    applyThemeClass(resolved);
  }, [resolved]);

  // Listen for OS theme changes when mode is 'system'
  useEffect(() => {
    if (theme !== 'system') return;

    const mq = window.matchMedia('(prefers-color-scheme: light)');
    const handler = () => {
      const r = resolveTheme('system');
      setResolved(r);
      applyThemeClass(r);
    };
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, [theme]);

  return (
    <ThemeContext.Provider value={{ theme, resolved, resolvedTheme: resolved, setTheme, toggleTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error('useTheme must be used within ThemeProvider');
  return ctx;
}

export type { Theme, ThemeContextValue };
