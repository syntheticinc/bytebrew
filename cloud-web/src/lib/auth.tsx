import { createContext, useContext, useState, useEffect, type ReactNode } from 'react';
import { api } from '../api/client';
import { refreshAccessToken } from '../api/auth';

interface AuthState {
  isAuthenticated: boolean;
  isLoading: boolean;
  email: string | null;
  login: (accessToken: string, refreshToken: string, email: string) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthState | null>(null);

const STORAGE_KEY_ACCESS = 'bytebrew_access_token';
const STORAGE_KEY_REFRESH = 'bytebrew_refresh_token';
const STORAGE_KEY_EMAIL = 'bytebrew_email';

export function AuthProvider({ children }: { children: ReactNode }) {
  const [isLoading, setIsLoading] = useState(true);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [email, setEmail] = useState<string | null>(null);

  useEffect(() => {
    const storedAccess = localStorage.getItem(STORAGE_KEY_ACCESS);
    const storedRefresh = localStorage.getItem(STORAGE_KEY_REFRESH);
    const storedEmail = localStorage.getItem(STORAGE_KEY_EMAIL);

    if (storedAccess && storedRefresh) {
      api.setToken(storedAccess);
      api.setRefresher(async () => {
        try {
          const newToken = await refreshAccessToken(storedRefresh);
          localStorage.setItem(STORAGE_KEY_ACCESS, newToken);
          return newToken;
        } catch {
          logout();
          return null;
        }
      });
      setIsAuthenticated(true);
      setEmail(storedEmail);
    }

    setIsLoading(false);
  }, []);

  const login = (accessToken: string, refreshToken: string, userEmail: string) => {
    localStorage.setItem(STORAGE_KEY_ACCESS, accessToken);
    localStorage.setItem(STORAGE_KEY_REFRESH, refreshToken);
    localStorage.setItem(STORAGE_KEY_EMAIL, userEmail);
    api.setToken(accessToken);
    api.setRefresher(async () => {
      try {
        const newToken = await refreshAccessToken(refreshToken);
        localStorage.setItem(STORAGE_KEY_ACCESS, newToken);
        return newToken;
      } catch {
        logout();
        return null;
      }
    });
    setIsAuthenticated(true);
    setEmail(userEmail);
  };

  const logout = () => {
    localStorage.removeItem(STORAGE_KEY_ACCESS);
    localStorage.removeItem(STORAGE_KEY_REFRESH);
    localStorage.removeItem(STORAGE_KEY_EMAIL);
    api.setToken(null);
    setIsAuthenticated(false);
    setEmail(null);
  };

  return (
    <AuthContext.Provider value={{ isAuthenticated, isLoading, email, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return ctx;
}
