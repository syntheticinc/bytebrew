import { useEffect, useState } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { api } from '../api/client';

// Runs AFTER ProtectedRoute. Blocks the admin surface until the tenant has
// at least one LLM. `error` state fails open — surfacing /models 5xx here
// would permanently strand the user.
type GateState = 'checking' | 'has-models' | 'no-models' | 'error';

// Sticky session flag — the route tree mounts two gate instances
// (/onboarding wrapper and /* wrapper); without it a read-after-write race
// against POST /models bounces the user back into the wizard.
const ONBOARDED_FLAG = 'bb_onboarded';

function readOnboardedFlag(): boolean {
  try { return sessionStorage.getItem(ONBOARDED_FLAG) === '1'; } catch { return false; }
}

export default function OnboardingGate({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<GateState>(() =>
    readOnboardedFlag() ? 'has-models' : 'checking',
  );
  const location = useLocation();

  useEffect(() => {
    // Sticky check: once the wizard's Step 1 sets the session flag (or any
    // earlier API call confirmed at least one model), trust it absolutely on
    // every subsequent gate mount. Prevents the read-after-write race against
    // postgres on Skip → /schemas navigation.
    if (readOnboardedFlag()) {
      setState('has-models');
      return;
    }
    let cancelled = false;
    api
      .listModels()
      .then((models) => {
        if (cancelled) return;
        const hasModels = !!models && models.length > 0;
        if (hasModels) {
          try { sessionStorage.setItem(ONBOARDED_FLAG, '1'); } catch { /* no-op */ }
          setState('has-models');
          return;
        }
        setState('no-models');
      })
      .catch(() => {
        if (cancelled) return;
        setState('error');
      });
    return () => {
      cancelled = true;
    };
    // Re-run on path change so finishing the wizard (navigate elsewhere)
    // re-checks and unlocks the normal surface without a full reload.
  }, [location.pathname]);

  if (state === 'checking') {
    return (
      <div className="fixed inset-0 bg-brand-dark flex items-center justify-center">
        <div className="text-sm text-brand-shade3 font-mono">Loading workspace…</div>
      </div>
    );
  }

  if (state === 'no-models') {
    // Synchronous re-check of the session flag before redirecting. The
    // useEffect that flips state from 'no-models' → 'has-models' runs AFTER
    // this render commits, so when the wizard's Step 1 sets the flag and
    // immediately calls navigate('/schemas'), the very next render here
    // still has stale state='no-models'. Without this check we'd fire
    // <Navigate to="/onboarding" replace /> before the effect can update
    // state, sending the user back into the wizard in a loop.
    if (readOnboardedFlag()) {
      return <>{children}</>;
    }
    // Guard against loops: the wizard itself must not be gated.
    if (location.pathname === '/onboarding') {
      return <>{children}</>;
    }
    return <Navigate to="/onboarding" replace />;
  }

  return <>{children}</>;
}
