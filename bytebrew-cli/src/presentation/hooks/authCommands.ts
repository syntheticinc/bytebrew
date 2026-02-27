import { AuthStorage } from '../../infrastructure/auth/AuthStorage.js';
import { LicenseStorage } from '../../infrastructure/license/LicenseStorage.js';
import { CloudApiClient } from '../../infrastructure/api/CloudApiClient.js';
import { parseJwtPayload } from '../../infrastructure/license/parseJwt.js';
import type { AuthTokens } from '../../infrastructure/auth/AuthStorage.js';

// Consumer-side interfaces (ISP)

export interface AuthStorageLike {
  load(): AuthTokens | null;
  save(tokens: AuthTokens): void;
  clear(): void;
}

export interface LicenseStorageLike {
  load(): string | null;
  save(jwt: string): void;
  clear(): void;
}

export interface CloudApiClientLike {
  login(email: string, password: string): Promise<AuthTokens>;
  activateLicense(): Promise<string>;
  refreshLicense(currentJwt: string): Promise<string>;
}

export interface CloudApiClientOptions {
  accessToken?: string;
  refreshToken?: string;
  email?: string;
  userId?: string;
  onTokenRefreshed?: (tokens: AuthTokens) => void;
}

export interface AuthDeps {
  authStorage: AuthStorageLike;
  licenseStorage: LicenseStorageLike;
  createApiClient: (opts?: CloudApiClientOptions) => CloudApiClientLike;
}

function defaultDeps(): AuthDeps {
  return {
    authStorage: new AuthStorage(),
    licenseStorage: new LicenseStorage(),
    createApiClient: (opts) => new CloudApiClient(opts),
  };
}

export function handleLogoutCommand(deps: AuthDeps = defaultDeps()): string {
  deps.authStorage.clear();
  deps.licenseStorage.clear();
  return 'Logged out. Auth tokens and license cleared.';
}

export function handleStatusCommand(deps: AuthDeps = defaultDeps()): string {
  const auth = deps.authStorage.load();
  const jwt = deps.licenseStorage.load();

  if (!auth && !jwt) {
    return 'Not logged in. Use /login <email> <password> to authenticate.';
  }

  const lines: string[] = [];

  if (auth) {
    lines.push(`Email: ${auth.email}`);
  }

  if (!jwt) {
    lines.push('License: not activated. Run /activate');
    return lines.join('\n');
  }

  const claims = parseJwtPayload(jwt);
  const tier = (claims.tier as string) ?? 'unknown';
  lines.push(`Tier: ${tier}`);

  if (claims.exp) {
    const expDate = new Date(claims.exp * 1000);
    const days = Math.ceil((expDate.getTime() - Date.now()) / (1000 * 60 * 60 * 24));
    const dateStr = expDate.toISOString().split('T')[0];
    lines.push(`Expires: ${dateStr} (${days} days remaining)`);
  }

  if (claims.proxy_steps_remaining !== undefined) {
    const limit = claims.proxy_steps_limit ?? '?';
    lines.push(`Proxy steps: ${claims.proxy_steps_remaining}/${limit}`);
  }

  if (claims.byok_enabled !== undefined) {
    lines.push(`BYOK: ${claims.byok_enabled ? 'enabled' : 'disabled'}`);
  }

  return lines.join('\n');
}

export async function handleLoginCommand(args: string, deps: AuthDeps = defaultDeps()): Promise<string> {
  const spaceIdx = args.indexOf(' ');
  if (spaceIdx === -1) {
    return 'Usage: /login <email> <password>';
  }

  const email = args.slice(0, spaceIdx).trim();
  const password = args.slice(spaceIdx + 1).trim();

  if (!email || !password) {
    return 'Usage: /login <email> <password>';
  }

  const client = deps.createApiClient();
  let tokens;
  try {
    tokens = await client.login(email, password);
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err);
    return `Login failed: ${msg}`;
  }

  deps.authStorage.save(tokens);

  try {
    const jwt = await client.activateLicense();
    deps.licenseStorage.save(jwt);
    const claims = parseJwtPayload(jwt);
    const tier = (claims.tier as string) ?? 'unknown';
    return `Logged in as ${email}\nLicense activated. Tier: ${tier}`;
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err);
    return `Logged in as ${email}\nLicense activation failed: ${msg}. Run /activate to retry.`;
  }
}

export async function handleActivateCommand(deps: AuthDeps = defaultDeps()): Promise<string> {
  const auth = deps.authStorage.load();
  if (!auth) {
    return 'Not logged in. Use /login first.';
  }

  const existingJwt = deps.licenseStorage.load();

  const client = deps.createApiClient({
    accessToken: auth.accessToken,
    refreshToken: auth.refreshToken,
    email: auth.email,
    userId: auth.userId,
    onTokenRefreshed: (refreshed) => deps.authStorage.save(refreshed),
  });

  try {
    const jwt = existingJwt
      ? await client.refreshLicense(existingJwt)
      : await client.activateLicense();

    deps.licenseStorage.save(jwt);

    const claims = parseJwtPayload(jwt);
    const tier = (claims.tier as string) ?? 'unknown';
    const lines = ['License activated.', `Tier: ${tier}`];

    if (claims.exp) {
      const expDate = new Date(claims.exp * 1000);
      const days = Math.ceil((expDate.getTime() - Date.now()) / (1000 * 60 * 60 * 24));
      const dateStr = expDate.toISOString().split('T')[0];
      lines.push(`Expires: ${dateStr} (${days} days remaining)`);
    }

    return lines.join('\n');
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err);
    return `Activation failed: ${msg}`;
  }
}
