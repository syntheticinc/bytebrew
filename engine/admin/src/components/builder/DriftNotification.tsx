import { useState, useEffect, useCallback, useRef } from 'react';
import { api } from '../../api/client';

// ─── Constants ──────────────────────────────────────────────────────────────

const HASH_KEY = 'bytebrew_config_hash';
const HASH_TS_KEY = 'bytebrew_config_hash_ts';

// ─── SHA-256 helper ─────────────────────────────────────────────────────────

async function sha256(text: string): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(text);
  const hashBuffer = await crypto.subtle.digest('SHA-256', data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray.map((b) => b.toString(16).padStart(2, '0')).join('');
}

// ─── Export hash storage ────────────────────────────────────────────────────

/** Call this after a successful export to store the baseline hash. */
export async function storeExportHash(yamlContent: string): Promise<void> {
  const hash = await sha256(yamlContent);
  localStorage.setItem(HASH_KEY, hash);
  localStorage.setItem(HASH_TS_KEY, new Date().toISOString());
}

/** Returns the stored export hash, or null if user has never exported. */
export function getStoredExportHash(): string | null {
  return localStorage.getItem(HASH_KEY);
}

// ─── Component ──────────────────────────────────────────────────────────────

interface DriftNotificationProps {
  /** Increment to trigger a re-check (e.g. after assistant modifies config). */
  checkTrigger?: number;
  /** Called when user clicks "Export Now" inline button. */
  onExport?: () => void;
}

export default function DriftNotification({ checkTrigger, onExport }: DriftNotificationProps) {
  const [drifted, setDrifted] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  const [checking, setChecking] = useState(false);

  // Track the last drift hash so we only reset dismissed when NEW drift is detected
  const lastDriftHashRef = useRef<string | null>(null);

  const checkDrift = useCallback(async () => {
    const storedHash = getStoredExportHash();
    // No baseline — user never exported, don't show notification
    if (!storedHash) return;

    setChecking(true);
    try {
      const yaml = await api.exportConfig();
      const yamlStr = typeof yaml === 'string' ? yaml : String(yaml);
      const currentHash = await sha256(yamlStr);
      const isDrifted = currentHash !== storedHash;

      if (isDrifted) {
        // Only reset dismissed if this is a NEW drift (different hash than last time we showed)
        if (lastDriftHashRef.current !== currentHash) {
          lastDriftHashRef.current = currentHash;
          setDismissed(false);
        }
      }

      setDrifted(isDrifted);
    } catch {
      // If export fails, don't show drift notification
    } finally {
      setChecking(false);
    }
  }, []);

  // Check on mount and when checkTrigger changes — but do NOT reset dismissed here
  useEffect(() => {
    checkDrift();
  }, [checkDrift, checkTrigger]);

  if (!drifted || dismissed || checking) return null;

  return (
    <div className="flex items-center gap-2 px-4 py-1.5 bg-amber-500/10 border-b border-amber-500/20 flex-shrink-0">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-amber-400 flex-shrink-0">
        <circle cx="12" cy="12" r="10" />
        <line x1="12" y1="8" x2="12" y2="12" />
        <line x1="12" y1="16" x2="12.01" y2="16" />
      </svg>
      <span className="text-xs text-amber-300 flex-1">
        Configuration has changed since last export.
      </span>
      {onExport && (
        <button
          onClick={onExport}
          className="text-[11px] text-amber-300 hover:text-amber-100 underline underline-offset-2 transition-colors flex-shrink-0"
        >
          Export Now
        </button>
      )}
      <button
        onClick={() => setDismissed(true)}
        className="p-0.5 text-amber-400/60 hover:text-amber-300 transition-colors flex-shrink-0"
      >
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <line x1="18" y1="6" x2="6" y2="18" />
          <line x1="6" y1="6" x2="18" y2="18" />
        </svg>
      </button>
    </div>
  );
}
