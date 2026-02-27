export interface JwtPayload {
  tier?: string;
  exp?: number;
  iat?: number;
  email?: string;
  status?: string;
  features?: Record<string, unknown>;
  proxy_steps_remaining?: number;
  proxy_steps_limit?: number;
  byok_enabled?: boolean;
  [key: string]: unknown;
}

export function parseJwtPayload(jwt: string): JwtPayload {
  const parts = jwt.split('.');
  if (parts.length !== 3) return {};

  try {
    const payload = Buffer.from(parts[1], 'base64url').toString('utf-8');
    return JSON.parse(payload) as JwtPayload;
  } catch {
    return {};
  }
}

export function showLicenseInfo(jwt: string): void {
  const claims = parseJwtPayload(jwt);
  console.log(`  Tier: ${claims.tier ?? 'unknown'}`);
  if (claims.exp) {
    const exp = new Date(claims.exp * 1000);
    const days = Math.ceil((exp.getTime() - Date.now()) / (1000 * 60 * 60 * 24));
    console.log(`  Expires: ${exp.toISOString().split('T')[0]} (in ${days} days)`);
  }
}
