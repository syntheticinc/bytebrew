import { createContext, useContext, useState, useEffect, useRef, useCallback, type ReactNode } from 'react';
import { api } from '../api/client';
import { refreshAccessToken } from '../api/auth';
import { AuthPopup } from '../components/AuthPopup';

interface AuthState {
  isAuthenticated: boolean;
  isLoading: boolean;
  email: string | null;
  login: (accessToken: string, refreshToken: string, email: string) => void;
  logout: () => void;
  showAuthPopup: boolean;
  triggerAuthPopup: (onSuccess?: () => void, title?: string) => void;
  closeAuthPopup: () => void;
}

const AuthContext = createContext<AuthState | null>(null);

const STORAGE_KEY_ACCESS = 'bytebrew_access_token';
const STORAGE_KEY_REFRESH = 'bytebrew_refresh_token';
const STORAGE_KEY_EMAIL = 'bytebrew_email';

function clearAuthStorage() {
  localStorage.removeItem(STORAGE_KEY_ACCESS);
  localStorage.removeItem(STORAGE_KEY_REFRESH);
  localStorage.removeItem(STORAGE_KEY_EMAIL);
  api.setToken(null);
}

function setupRefresher(refreshToken: string, onFail: () => void) {
  api.setRefresher(async () => {
    try {
      const newToken = await refreshAccessToken(refreshToken);
      localStorage.setItem(STORAGE_KEY_ACCESS, newToken);
      return newToken;
    } catch {
      onFail();
      return null;
    }
  });
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [isLoading, setIsLoading] = useState(true);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [email, setEmail] = useState<string | null>(null);
  const [showAuthPopup, setShowAuthPopup] = useState(false);
  const [popupTitle, setPopupTitle] = useState<string | undefined>();

  const onSuccessCallbackRef = useRef<(() => void) | undefined>(undefined);

  const logout = useCallback(() => {
    clearAuthStorage();
    setIsAuthenticated(false);
    setEmail(null);
  }, []);

  useEffect(() => {
    const storedAccess = localStorage.getItem(STORAGE_KEY_ACCESS);
    const storedRefresh = localStorage.getItem(STORAGE_KEY_REFRESH);
    const storedEmail = localStorage.getItem(STORAGE_KEY_EMAIL);

    if (storedAccess && storedRefresh) {
      api.setToken(storedAccess);
      setupRefresher(storedRefresh, logout);
      setIsAuthenticated(true);
      setEmail(storedEmail);
    }

    setIsLoading(false);
  }, [logout]);

  const login = useCallback(
    (accessToken: string, refreshToken: string, userEmail: string) => {
      localStorage.setItem(STORAGE_KEY_ACCESS, accessToken);
      localStorage.setItem(STORAGE_KEY_REFRESH, refreshToken);
      localStorage.setItem(STORAGE_KEY_EMAIL, userEmail);
      api.setToken(accessToken);
      setupRefresher(refreshToken, logout);
      setIsAuthenticated(true);
      setEmail(userEmail);
    },
    [logout],
  );

  const triggerAuthPopup = useCallback((onSuccess?: () => void, title?: string) => {
    onSuccessCallbackRef.current = onSuccess;
    setPopupTitle(title);
    setShowAuthPopup(true);
  }, []);

  const closeAuthPopup = useCallback(() => {
    setShowAuthPopup(false);
    onSuccessCallbackRef.current = undefined;
    setPopupTitle(undefined);
  }, []);

  const handlePopupSuccess = useCallback(
    (accessToken: string, refreshToken: string, userEmail: string) => {
      login(accessToken, refreshToken, userEmail);
      setShowAuthPopup(false);

      const callback = onSuccessCallbackRef.current;
      onSuccessCallbackRef.current = undefined;
      setPopupTitle(undefined);

      if (callback) {
        callback();
      }
    },
    [login],
  );

  return (
    <AuthContext.Provider
      value={{
        isAuthenticated,
        isLoading,
        email,
        login,
        logout,
        showAuthPopup,
        triggerAuthPopup,
        closeAuthPopup,
      }}
    >
      {children}
      <AuthPopup
        isOpen={showAuthPopup}
        onClose={closeAuthPopup}
        onSuccess={handlePopupSuccess}
        title={popupTitle}
      />
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
