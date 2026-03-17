import { useMemo } from 'react';
import { LicenseStorage } from '../../infrastructure/license/LicenseStorage.js';
import { parseJwtPayload } from '../../infrastructure/license/parseJwt.js';

export interface LicenseBadgeInfo {
  tier: string;          // 'trial' | 'personal' | 'teams' | 'unknown'
  status: 'active' | 'grace' | 'expired' | 'unknown';
  daysRemaining: number | null; // null if no expiry
  label: string;         // "[Trial 12d]", "[Personal]", "[Expired]"
  color: string;         // 'gray' | 'green' | 'yellow' | 'red'
  proxyStepsRemaining?: number;
  proxyStepsLimit?: number;
  byokEnabled?: boolean;
}

export function useLicenseInfo(
  version = 0,
  jwtLoader: () => string | null = () => new LicenseStorage().load(),
): LicenseBadgeInfo | null {
  return useMemo(() => {
    const jwt = jwtLoader();
    if (!jwt) return null;

    const claims = parseJwtPayload(jwt);
    const tier = (claims.tier as string) ?? 'unknown';
    const now = Math.floor(Date.now() / 1000);

    // Determine days remaining
    let daysRemaining: number | null = null;
    if (claims.exp) {
      daysRemaining = Math.ceil((claims.exp - now) / (60 * 60 * 24));
    }

    // Determine status and color
    let status: LicenseBadgeInfo['status'] = 'active';
    let color = 'green';

    if (claims.exp && claims.exp < now) {
      status = 'expired';
      color = 'red';
    } else if (claims.grace_until) {
      const graceUntil = claims.grace_until as number;
      if (graceUntil > now && claims.exp && claims.exp < now) {
        status = 'grace';
        color = 'yellow';
        daysRemaining = Math.ceil((graceUntil - now) / (60 * 60 * 24));
      }
    }

    // Trial = gray
    if (tier === 'trial' && status === 'active') {
      color = 'gray';
    }

    // Build label
    let label: string;
    switch (status) {
      case 'expired':
        label = '[Expired]';
        break;
      case 'grace':
        label = `[${capitalize(tier)} - ${daysRemaining}d left]`;
        break;
      default:
        if (tier === 'trial' && daysRemaining !== null) {
          label = `[Trial ${daysRemaining}d]`;
        } else {
          label = `[${capitalize(tier)}]`;
        }
    }

    // Proxy and BYOK info from JWT claims
    const proxyStepsRemaining = claims.proxy_steps_remaining as number | undefined;
    const proxyStepsLimit = claims.proxy_steps_limit as number | undefined;
    const byokEnabled = claims.byok_enabled as boolean | undefined;

    return { tier, status, daysRemaining, label, color, proxyStepsRemaining, proxyStepsLimit, byokEnabled };
  }, [version, jwtLoader]); // version changes → re-reads JWT from disk
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}
