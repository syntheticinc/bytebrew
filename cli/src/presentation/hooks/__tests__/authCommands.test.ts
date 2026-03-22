import { describe, it, expect } from 'bun:test';
import {
  handleLogoutCommand,
  handleStatusCommand,
  handleLoginCommand,
  handleActivateCommand,
  type AuthDeps,
  type AuthStorageLike,
  type LicenseStorageLike,
  type CloudApiClientLike,
} from '../authCommands.js';
import type { AuthTokens } from '../../../infrastructure/auth/AuthStorage.js';

// --- Helpers ---

function makeFakeJwt(claims: Record<string, unknown>): string {
  const header = Buffer.from(JSON.stringify({ alg: 'EdDSA', typ: 'JWT' })).toString('base64url');
  const payload = Buffer.from(JSON.stringify(claims)).toString('base64url');
  return `${header}.${payload}.fakesig`;
}

function makeMemoryAuthStorage(initial?: AuthTokens): AuthStorageLike {
  let stored: AuthTokens | null = initial ?? null;
  return {
    load: () => stored,
    save: (tokens: AuthTokens) => { stored = tokens; },
    clear: () => { stored = null; },
  };
}

function makeMemoryLicenseStorage(initial?: string): LicenseStorageLike {
  let stored: string | null = initial ?? null;
  return {
    load: () => stored,
    save: (jwt: string) => { stored = jwt; },
    clear: () => { stored = null; },
  };
}

function makeMockDeps(overrides?: {
  auth?: AuthTokens;
  license?: string;
  apiClient?: Partial<CloudApiClientLike>;
}): AuthDeps {
  const authStorage = makeMemoryAuthStorage(overrides?.auth);
  const licenseStorage = makeMemoryLicenseStorage(overrides?.license);

  const defaultClient: CloudApiClientLike = {
    login: async () => { throw new Error('login not mocked'); },
    activateLicense: async () => { throw new Error('activateLicense not mocked'); },
    refreshLicense: async () => { throw new Error('refreshLicense not mocked'); },
  };

  const client = { ...defaultClient, ...overrides?.apiClient };

  return {
    authStorage,
    licenseStorage,
    createApiClient: () => client,
  };
}

// --- handleLogoutCommand ---

describe('handleLogoutCommand', () => {
  it('returns confirmation message', () => {
    const deps = makeMockDeps();
    const result = handleLogoutCommand(deps);
    expect(result).toBe('Logged out. Auth tokens and license cleared.');
  });

  it('clears auth storage', () => {
    const deps = makeMockDeps({
      auth: { accessToken: 'at', refreshToken: 'rt', email: 'test@example.com', userId: 'u1' },
    });
    expect(deps.authStorage.load()).not.toBeNull();

    handleLogoutCommand(deps);

    expect(deps.authStorage.load()).toBeNull();
  });

  it('clears license storage', () => {
    const deps = makeMockDeps({ license: 'some.jwt.token' });
    expect(deps.licenseStorage.load()).not.toBeNull();

    handleLogoutCommand(deps);

    expect(deps.licenseStorage.load()).toBeNull();
  });

  it('works when storages are empty', () => {
    const deps = makeMockDeps();
    const result = handleLogoutCommand(deps);
    expect(result).toBe('Logged out. Auth tokens and license cleared.');
  });
});

// --- handleStatusCommand ---

describe('handleStatusCommand', () => {
  it('not logged in — no auth, no JWT', () => {
    const deps = makeMockDeps();
    const result = handleStatusCommand(deps);
    expect(result).toBe('Not logged in. Use /login <email> <password> to authenticate.');
  });

  it('email only — auth exists, no JWT', () => {
    const deps = makeMockDeps({
      auth: { accessToken: 'at', refreshToken: 'rt', email: 'alice@example.com', userId: 'u1' },
    });

    const result = handleStatusCommand(deps);
    expect(result).toContain('Email: alice@example.com');
    expect(result).toContain('License: not activated');
    expect(result).toContain('/activate');
  });

  it('full status — auth + JWT with all claims', () => {
    const expTimestamp = Math.floor(Date.now() / 1000) + 86400 * 30; // 30 days from now
    const jwt = makeFakeJwt({
      tier: 'personal',
      exp: expTimestamp,
      proxy_steps_remaining: 250,
      proxy_steps_limit: 300,
      byok_enabled: true,
    });

    const deps = makeMockDeps({
      auth: { accessToken: 'at', refreshToken: 'rt', email: 'bob@example.com', userId: 'u2' },
      license: jwt,
    });

    const result = handleStatusCommand(deps);
    expect(result).toContain('Email: bob@example.com');
    expect(result).toContain('Tier: personal');
    expect(result).toContain('30 days remaining');
    expect(result).toContain('Proxy steps: 250/300');
    expect(result).toContain('BYOK: enabled');
  });

  it('expired license — JWT with exp in the past', () => {
    const expTimestamp = Math.floor(Date.now() / 1000) - 86400 * 5; // 5 days ago
    const jwt = makeFakeJwt({
      tier: 'trial',
      exp: expTimestamp,
    });

    const deps = makeMockDeps({
      auth: { accessToken: 'at', refreshToken: 'rt', email: 'expired@example.com', userId: 'u3' },
      license: jwt,
    });

    const result = handleStatusCommand(deps);
    expect(result).toContain('Tier: trial');
    // Days remaining should be negative (e.g. -4 or -5)
    expect(result).toMatch(/-\d+ days remaining/);
  });

  it('JWT only (no auth) — shows tier but no email', () => {
    const jwt = makeFakeJwt({ tier: 'teams', exp: Math.floor(Date.now() / 1000) + 86400 });
    const deps = makeMockDeps({ license: jwt });

    const result = handleStatusCommand(deps);
    expect(result).toContain('Tier: teams');
    expect(result).not.toContain('Email:');
  });

  it('BYOK disabled', () => {
    const jwt = makeFakeJwt({
      tier: 'personal',
      exp: Math.floor(Date.now() / 1000) + 86400,
      byok_enabled: false,
    });

    const deps = makeMockDeps({
      auth: { accessToken: 'at', refreshToken: 'rt', email: 'test@test.com', userId: 'u1' },
      license: jwt,
    });

    const result = handleStatusCommand(deps);
    expect(result).toContain('BYOK: disabled');
  });
});

// --- handleLoginCommand ---

describe('handleLoginCommand', () => {
  it('no args — no space in input', async () => {
    const deps = makeMockDeps();
    const result = await handleLoginCommand('emailonly', deps);
    expect(result).toBe('Usage: /login <email> <password>');
  });

  it('empty email — spaces only before space', async () => {
    const deps = makeMockDeps();
    const result = await handleLoginCommand('  password', deps);
    expect(result).toBe('Usage: /login <email> <password>');
  });

  it('empty password — spaces only after space', async () => {
    const deps = makeMockDeps();
    const result = await handleLoginCommand('email@test.com   ', deps);
    expect(result).toBe('Usage: /login <email> <password>');
  });

  it('successful login + activate', async () => {
    const activateJwt = makeFakeJwt({ tier: 'personal', exp: Math.floor(Date.now() / 1000) + 86400 * 30 });

    const loginTokens: AuthTokens = {
      accessToken: 'at123',
      refreshToken: 'rt123',
      email: 'alice@example.com',
      userId: 'uid1',
    };

    const deps = makeMockDeps({
      apiClient: {
        login: async () => loginTokens,
        activateLicense: async () => activateJwt,
      },
    });

    const result = await handleLoginCommand('alice@example.com mypassword', deps);

    expect(result).toContain('Logged in as alice@example.com');
    expect(result).toContain('License activated');
    expect(result).toContain('Tier: personal');

    // Verify auth was persisted
    const auth = deps.authStorage.load();
    expect(auth).not.toBeNull();
    expect(auth!.email).toBe('alice@example.com');
    expect(auth!.accessToken).toBe('at123');

    // Verify license was persisted
    const license = deps.licenseStorage.load();
    expect(license).toBe(activateJwt);
  });

  it('login failure', async () => {
    const deps = makeMockDeps({
      apiClient: {
        login: async () => { throw new Error('Invalid email or password'); },
      },
    });

    const result = await handleLoginCommand('bad@example.com wrongpw', deps);

    expect(result).toContain('Login failed:');
    expect(result).toContain('Invalid email or password');
  });

  it('login ok, activate failure', async () => {
    const loginTokens: AuthTokens = {
      accessToken: 'at',
      refreshToken: 'rt',
      email: 'test@test.com',
      userId: 'uid',
    };

    const deps = makeMockDeps({
      apiClient: {
        login: async () => loginTokens,
        activateLicense: async () => { throw new Error('Internal server error'); },
      },
    });

    const result = await handleLoginCommand('test@test.com password', deps);

    expect(result).toContain('Logged in as test@test.com');
    expect(result).toContain('License activation failed:');
    expect(result).toContain('/activate to retry');

    // Auth should still be saved
    const auth = deps.authStorage.load();
    expect(auth).not.toBeNull();
    expect(auth!.email).toBe('test@test.com');
  });
});

// --- handleActivateCommand ---

describe('handleActivateCommand', () => {
  it('not logged in — no auth', async () => {
    const deps = makeMockDeps();
    const result = await handleActivateCommand(deps);
    expect(result).toBe('Not logged in. Use /login first.');
  });

  it('activate success (no existing JWT)', async () => {
    const activateJwt = makeFakeJwt({
      tier: 'personal',
      exp: Math.floor(Date.now() / 1000) + 86400 * 30,
    });

    const deps = makeMockDeps({
      auth: { accessToken: 'at', refreshToken: 'rt', email: 'user@example.com', userId: 'u1' },
      apiClient: {
        activateLicense: async () => activateJwt,
      },
    });

    const result = await handleActivateCommand(deps);

    expect(result).toContain('License activated');
    expect(result).toContain('Tier: personal');
    expect(result).toContain('days remaining');

    // Verify license persisted
    const license = deps.licenseStorage.load();
    expect(license).toBe(activateJwt);
  });

  it('refresh success (existing JWT)', async () => {
    const oldJwt = makeFakeJwt({ tier: 'trial', exp: Math.floor(Date.now() / 1000) + 86400 });
    const newJwt = makeFakeJwt({
      tier: 'personal',
      exp: Math.floor(Date.now() / 1000) + 86400 * 30,
    });

    const deps = makeMockDeps({
      auth: { accessToken: 'at', refreshToken: 'rt', email: 'user@example.com', userId: 'u1' },
      license: oldJwt,
      apiClient: {
        refreshLicense: async () => newJwt,
      },
    });

    const result = await handleActivateCommand(deps);

    expect(result).toContain('License activated');
    expect(result).toContain('Tier: personal');

    // Verify new license persisted
    const license = deps.licenseStorage.load();
    expect(license).toBe(newJwt);
  });

  it('activation failure', async () => {
    const deps = makeMockDeps({
      auth: { accessToken: 'at', refreshToken: 'rt', email: 'user@example.com', userId: 'u1' },
      apiClient: {
        activateLicense: async () => { throw new Error('No subscription found'); },
      },
    });

    const result = await handleActivateCommand(deps);

    expect(result).toContain('Activation failed:');
    expect(result).toContain('No subscription found');
  });
});
