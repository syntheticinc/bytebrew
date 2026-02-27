import { describe, it, expect, mock, beforeEach } from 'bun:test';
import { refreshLicenseBackground } from '../backgroundRefresh.js';
import type { AuthStorage } from '../../auth/AuthStorage.js';
import type { LicenseStorage } from '../LicenseStorage.js';
import type { CloudApiClient } from '../../api/CloudApiClient.js';

function makeAuthStorage(tokens: ReturnType<AuthStorage['load']>): AuthStorage {
  return {
    load: mock(() => tokens),
    save: mock(() => {}),
    clear: mock(() => {}),
  } as unknown as AuthStorage;
}

function makeLicenseStorage(jwt: string | null): LicenseStorage & { saveMock: ReturnType<typeof mock> } {
  const saveMock = mock(() => {});
  return {
    load: mock(() => jwt),
    save: saveMock,
    clear: mock(() => {}),
    saveMock,
  } as unknown as LicenseStorage & { saveMock: ReturnType<typeof mock> };
}

function makeClientFactory(
  refreshResult: string | (() => Promise<string>),
  shouldThrow?: Error,
): [(tokens: { accessToken: string; refreshToken: string }) => CloudApiClient, ReturnType<typeof mock>] {
  const refreshLicense = shouldThrow
    ? mock(() => Promise.reject(shouldThrow))
    : typeof refreshResult === 'function'
      ? mock(refreshResult)
      : mock(() => Promise.resolve(refreshResult as string));

  const factory = mock((_tokens: { accessToken: string; refreshToken: string }) => ({
    refreshLicense,
  } as unknown as CloudApiClient));

  return [factory, refreshLicense];
}

describe('refreshLicenseBackground', () => {
  it('does nothing when no auth tokens', async () => {
    const auth = makeAuthStorage(null);
    const licenseStore = makeLicenseStorage('existing-jwt');
    const [factory] = makeClientFactory('new-jwt');

    await refreshLicenseBackground(auth, licenseStore, factory);

    expect((factory as ReturnType<typeof mock>).mock.calls.length).toBe(0);
    expect(licenseStore.saveMock.mock.calls.length).toBe(0);
  });

  it('does nothing when no current JWT', async () => {
    const auth = makeAuthStorage({
      accessToken: 'access',
      refreshToken: 'refresh',
      email: 'user@example.com',
      userId: 'user-1',
    });
    const licenseStore = makeLicenseStorage(null);
    const [factory] = makeClientFactory('new-jwt');

    await refreshLicenseBackground(auth, licenseStore, factory);

    expect((factory as ReturnType<typeof mock>).mock.calls.length).toBe(0);
    expect(licenseStore.saveMock.mock.calls.length).toBe(0);
  });

  it('calls refreshLicense when tokens and JWT are present', async () => {
    const auth = makeAuthStorage({
      accessToken: 'access',
      refreshToken: 'refresh',
      email: 'user@example.com',
      userId: 'user-1',
    });
    const licenseStore = makeLicenseStorage('current-jwt');
    const [factory, refreshLicense] = makeClientFactory('new-jwt');

    await refreshLicenseBackground(auth, licenseStore, factory);

    expect((refreshLicense as ReturnType<typeof mock>).mock.calls.length).toBe(1);
    expect((refreshLicense as ReturnType<typeof mock>).mock.calls[0]).toEqual(['current-jwt']);
  });

  it('saves new JWT when it differs from current', async () => {
    const auth = makeAuthStorage({
      accessToken: 'access',
      refreshToken: 'refresh',
      email: 'user@example.com',
      userId: 'user-1',
    });
    const licenseStore = makeLicenseStorage('current-jwt');
    const [factory] = makeClientFactory('updated-jwt');

    await refreshLicenseBackground(auth, licenseStore, factory);

    expect(licenseStore.saveMock.mock.calls.length).toBe(1);
    expect(licenseStore.saveMock.mock.calls[0]).toEqual(['updated-jwt']);
  });

  it('does not save when returned JWT is same as current', async () => {
    const auth = makeAuthStorage({
      accessToken: 'access',
      refreshToken: 'refresh',
      email: 'user@example.com',
      userId: 'user-1',
    });
    const licenseStore = makeLicenseStorage('same-jwt');
    const [factory] = makeClientFactory('same-jwt');

    await refreshLicenseBackground(auth, licenseStore, factory);

    expect(licenseStore.saveMock.mock.calls.length).toBe(0);
  });

  it('silently ignores API errors (does not throw)', async () => {
    const auth = makeAuthStorage({
      accessToken: 'access',
      refreshToken: 'refresh',
      email: 'user@example.com',
      userId: 'user-1',
    });
    const licenseStore = makeLicenseStorage('current-jwt');
    const [factory] = makeClientFactory('', new Error('Network error'));

    await expect(
      refreshLicenseBackground(auth, licenseStore, factory),
    ).rejects.toThrow('Network error');

    // The save must NOT have been called
    expect(licenseStore.saveMock.mock.calls.length).toBe(0);
  });
});
