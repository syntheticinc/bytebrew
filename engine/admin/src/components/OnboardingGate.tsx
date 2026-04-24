import { useEffect, useState } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { api } from '../api/client';

// OnboardingGate runs AFTER the ProtectedRoute auth check. It blocks the
// normal admin surface until the tenant has at least one configured LLM
// model. Keeps the logic out of Layout so layout stays concerned only with
// chrome — and so the wizard page itself can mount outside this gate.
//
// State machine:
//   "checking"      — first network call in flight
//   "has-models"    — proceed, render children
//   "no-models"     — redirect to /onboarding (unless already there)
//   "error"         — fail-open: render children; surfacing the error here
//                     would permanently strand the user if /models ever
//                     returns 5xx. The admin UI handles its own load errors.
type GateState = 'checking' | 'has-models' | 'no-models' | 'error';

// Once the wizard registers a model we write this session-storage flag so the
// gate trusts the onboarded state across its own remounts (the route tree has
// two gate instances — one for /onboarding, one for /*). Without this, a
// read-after-write race against POST /models on the very next navigation can
// surface an empty list and redirect the user back into the wizard.
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
        // API says zero models but the wizard recently succeeded — trust the
        // session flag and keep the user on the surface they asked for.
        if (readOnboardedFlag()) {
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
    // Guard against loops: the wizard itself must not be gated.
    if (location.pathname === '/onboarding') {
      return <>{children}</>;
    }
    return <Navigate to="/onboarding" replace />;
  }

  return <>{children}</>;
}
