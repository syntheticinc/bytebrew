import { useEffect, useRef, useCallback } from 'react';
import { googleLogin } from '../api/auth';
import { ApiError } from '../api/client';

declare global {
  interface Window {
    google?: {
      accounts: {
        id: {
          initialize: (config: GoogleInitConfig) => void;
          renderButton: (element: HTMLElement, config: GoogleButtonConfig) => void;
          cancel: () => void;
        };
      };
    };
  }
}

interface GoogleInitConfig {
  client_id: string;
  callback: (response: GoogleCredentialResponse) => void;
  auto_select?: boolean;
}

interface GoogleButtonConfig {
  theme: 'outline' | 'filled_blue' | 'filled_black';
  size: 'large' | 'medium' | 'small';
  width?: string;
  text?: 'signin_with' | 'signup_with' | 'continue_with' | 'signin';
  shape?: 'rectangular' | 'pill' | 'circle' | 'square';
  logo_alignment?: 'left' | 'center';
}

interface GoogleCredentialResponse {
  credential: string;
  select_by: string;
}

interface GoogleSignInButtonProps {
  onSuccess: (accessToken: string, refreshToken: string, email: string) => void;
  onError: (error: string) => void;
  text?: 'signin_with' | 'signup_with';
}

const GOOGLE_CLIENT_ID = import.meta.env.VITE_GOOGLE_CLIENT_ID;
const GSI_SCRIPT_SRC = 'https://accounts.google.com/gsi/client';

let gsiScriptLoaded = false;
let gsiScriptLoading = false;
const gsiLoadCallbacks: Array<() => void> = [];

function loadGsiScript(): Promise<void> {
  if (gsiScriptLoaded) {
    return Promise.resolve();
  }

  return new Promise((resolve, reject) => {
    gsiLoadCallbacks.push(resolve);

    if (gsiScriptLoading) {
      return;
    }

    gsiScriptLoading = true;

    const script = document.createElement('script');
    script.src = GSI_SCRIPT_SRC;
    script.async = true;
    script.defer = true;
    script.onload = () => {
      gsiScriptLoaded = true;
      gsiScriptLoading = false;
      gsiLoadCallbacks.forEach((cb) => cb());
      gsiLoadCallbacks.length = 0;
    };
    script.onerror = () => {
      gsiScriptLoading = false;
      gsiLoadCallbacks.length = 0;
      reject(new Error('Failed to load Google Sign-In'));
    };

    document.head.appendChild(script);
  });
}

export function GoogleSignInButton({ onSuccess, onError, text = 'signin_with' }: GoogleSignInButtonProps) {
  const buttonRef = useRef<HTMLDivElement>(null);
  const initializedRef = useRef(false);

  const handleCredentialResponse = useCallback(
    async (response: GoogleCredentialResponse) => {
      try {
        const result = await googleLogin(response.credential);
        // Decode email from JWT (id_token payload)
        const payload = JSON.parse(atob(response.credential.split('.')[1]));
        onSuccess(result.access_token, result.refresh_token, payload.email);
      } catch (err) {
        if (err instanceof ApiError) {
          onError(err.message);
        } else {
          onError('Google sign-in failed');
        }
      }
    },
    [onSuccess, onError],
  );

  useEffect(() => {
    if (!GOOGLE_CLIENT_ID) {
      return;
    }

    if (initializedRef.current) {
      return;
    }

    let cancelled = false;

    loadGsiScript()
      .then(() => {
        if (cancelled || !buttonRef.current || !window.google) {
          return;
        }

        initializedRef.current = true;

        window.google.accounts.id.initialize({
          client_id: GOOGLE_CLIENT_ID,
          callback: handleCredentialResponse,
        });

        window.google.accounts.id.renderButton(buttonRef.current, {
          theme: 'outline',
          size: 'large',
          width: buttonRef.current.offsetWidth.toString(),
          text,
          shape: 'rectangular',
        });
      })
      .catch(() => {
        if (!cancelled) {
          onError('Failed to load Google Sign-In');
        }
      });

    return () => {
      cancelled = true;
    };
  }, [handleCredentialResponse, onError, text]);

  if (!GOOGLE_CLIENT_ID) {
    return null;
  }

  return (
    <div className="w-full">
      <div ref={buttonRef} className="flex items-center justify-center [&>div]:!w-full" />
    </div>
  );
}
