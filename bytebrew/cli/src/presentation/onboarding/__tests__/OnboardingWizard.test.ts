import { describe, it, expect, beforeEach, afterEach, spyOn } from 'bun:test';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';

// We test checkLicenseStatus() which internally does:
//   1. new LicenseStorage().load()  -- reads ~/.bytebrew/license.jwt
//   2. parseJwtPayload(jwt)         -- base64-decodes JWT payload
//
// Strategy: mock ByteBrewHome.licenseFile() to return a temp path,
// then write real JWTs with crafted payloads to that path.

import { ByteBrewHome } from '../../../infrastructure/config/ByteBrewHome';
import { checkLicenseStatus } from '../OnboardingWizard';

// --- Helpers ------------------------------------------------------------------

let tempDir: string;
let licenseFilePath: string;
let vectorHomeSpy: ReturnType<typeof spyOn>;
let consoleSpy: ReturnType<typeof spyOn>;

/** Create a minimal JWT (header.payload.signature) with given claims. */
function makeJwt(claims: Record<string, unknown>): string {
  const header = Buffer.from(JSON.stringify({ alg: 'EdDSA' })).toString('base64url');
  const payload = Buffer.from(JSON.stringify(claims)).toString('base64url');
  const signature = 'test-signature';
  return `${header}.${payload}.${signature}`;
}

/** Returns a unix timestamp N seconds from now. */
function nowPlusSeconds(seconds: number): number {
  return Math.floor(Date.now() / 1000) + seconds;
}

/** Write a JWT to the temp license file. */
function writeLicense(jwt: string): void {
  fs.mkdirSync(path.dirname(licenseFilePath), { recursive: true });
  fs.writeFileSync(licenseFilePath, jwt, 'utf-8');
}

// --- Setup / Teardown ---------------------------------------------------------

describe('checkLicenseStatus', () => {
  beforeEach(() => {
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'onboarding-test-'));
    licenseFilePath = path.join(tempDir, 'license.jwt');

    // Redirect ByteBrewHome.licenseFile() to our temp path
    vectorHomeSpy = spyOn(ByteBrewHome, 'licenseFile').mockReturnValue(licenseFilePath);

    // Suppress console.log output from grace period warnings
    consoleSpy = spyOn(console, 'log').mockImplementation(() => {});
  });

  afterEach(() => {
    consoleSpy.mockRestore();
    vectorHomeSpy.mockRestore();
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true });
    }
  });

  // --- Tests ------------------------------------------------------------------

  it('returns "missing" when no JWT stored', () => {
    const result = checkLicenseStatus();
    expect(result).toBe('missing');
  });

  it('returns "valid" when JWT exists and exp is in the future', () => {
    writeLicense(
      makeJwt({
        tier: 'personal',
        exp: nowPlusSeconds(30 * 86400),
      }),
    );

    const result = checkLicenseStatus();
    expect(result).toBe('valid');
  });

  it('returns "valid" when JWT has no exp claim', () => {
    writeLicense(makeJwt({ tier: 'personal' }));

    const result = checkLicenseStatus();
    expect(result).toBe('valid');
  });

  it('returns "expired" when JWT expired and no grace_until', () => {
    writeLicense(
      makeJwt({
        tier: 'personal',
        exp: nowPlusSeconds(-3600),
      }),
    );

    const result = checkLicenseStatus();
    expect(result).toBe('expired');
  });

  it('returns "expired" when JWT expired and grace_until also passed', () => {
    writeLicense(
      makeJwt({
        tier: 'personal',
        exp: nowPlusSeconds(-7200),
        grace_until: nowPlusSeconds(-3600),
      }),
    );

    const result = checkLicenseStatus();
    expect(result).toBe('expired');
  });

  it('returns "grace" when JWT expired but within grace_until', () => {
    writeLicense(
      makeJwt({
        tier: 'personal',
        exp: nowPlusSeconds(-3600),
        grace_until: nowPlusSeconds(86400),
      }),
    );

    const result = checkLicenseStatus();
    expect(result).toBe('grace');
  });

  it('returns "grace" and logs warning with days remaining', () => {
    // Restore the beforeEach spy so we get a fresh one for assertions
    consoleSpy.mockRestore();
    const logSpy = spyOn(console, 'log').mockImplementation(() => {});

    const graceSeconds = 3 * 86400; // exactly 3 days
    writeLicense(
      makeJwt({
        tier: 'personal',
        exp: nowPlusSeconds(-3600),
        grace_until: nowPlusSeconds(graceSeconds),
      }),
    );

    const result = checkLicenseStatus();
    expect(result).toBe('grace');

    const allCalls = logSpy.mock.calls.map((args: unknown[]) => (args as string[]).join(' '));
    const warningLine = allCalls.find((line: string) => line.includes('Grace period ends'));
    expect(warningLine).toBeDefined();
    // 3 * 86400 seconds = 3 days. Math.ceil should give exactly 3.
    expect(warningLine).toContain('3 day');

    // Reassign consoleSpy for afterEach cleanup
    consoleSpy = logSpy;
  });

  it('returns "grace" and shows singular "day" for 1 day remaining', () => {
    consoleSpy.mockRestore();
    const logSpy = spyOn(console, 'log').mockImplementation(() => {});

    writeLicense(
      makeJwt({
        tier: 'personal',
        exp: nowPlusSeconds(-3600),
        grace_until: nowPlusSeconds(86400),
      }),
    );

    const result = checkLicenseStatus();
    expect(result).toBe('grace');

    const allCalls = logSpy.mock.calls.map((args: unknown[]) => (args as string[]).join(' '));
    const warningLine = allCalls.find((line: string) => line.includes('Grace period ends'));
    expect(warningLine).toBeDefined();
    expect(warningLine).toContain('1 day.');
    expect(warningLine).not.toContain('1 days');

    consoleSpy = logSpy;
  });

  it('returns "expired" when grace_until is 0 (falsy)', () => {
    writeLicense(
      makeJwt({
        tier: 'personal',
        exp: nowPlusSeconds(-3600),
        grace_until: 0,
      }),
    );

    const result = checkLicenseStatus();
    expect(result).toBe('expired');
  });

  it('returns "expiring_soon" when exp equals current time (boundary)', () => {
    consoleSpy.mockRestore();
    const logSpy = spyOn(console, 'log').mockImplementation(() => {});

    const now = Math.floor(Date.now() / 1000);
    writeLicense(makeJwt({ tier: 'personal', exp: now }));

    const result = checkLicenseStatus();
    // exp === now means `claims.exp < now` is false, but daysLeft=0 <= threshold=7
    expect(result).toBe('expiring_soon');

    consoleSpy = logSpy;
  });

  it('returns "expiring_soon" for trial expiring in 2 days', () => {
    consoleSpy.mockRestore();
    const logSpy = spyOn(console, 'log').mockImplementation(() => {});

    writeLicense(
      makeJwt({
        tier: 'trial',
        exp: nowPlusSeconds(2 * 86400),
      }),
    );

    const result = checkLicenseStatus();
    expect(result).toBe('expiring_soon');

    const allCalls = logSpy.mock.calls.map((args: unknown[]) => (args as string[]).join(' '));
    const warningLine = allCalls.find((line: string) => line.includes('trial expires'));
    expect(warningLine).toBeDefined();
    expect(warningLine).toContain('2 days');

    consoleSpy = logSpy;
  });

  it('returns "expiring_soon" for paid subscription expiring in 5 days', () => {
    consoleSpy.mockRestore();
    const logSpy = spyOn(console, 'log').mockImplementation(() => {});

    writeLicense(
      makeJwt({
        tier: 'personal',
        exp: nowPlusSeconds(5 * 86400),
      }),
    );

    const result = checkLicenseStatus();
    expect(result).toBe('expiring_soon');

    const allCalls = logSpy.mock.calls.map((args: unknown[]) => (args as string[]).join(' '));
    const warningLine = allCalls.find((line: string) => line.includes('subscription expires'));
    expect(warningLine).toBeDefined();
    expect(warningLine).toContain('5 days');

    consoleSpy = logSpy;
  });

  it('returns "missing" when license file is empty', () => {
    fs.mkdirSync(path.dirname(licenseFilePath), { recursive: true });
    fs.writeFileSync(licenseFilePath, '', 'utf-8');

    const result = checkLicenseStatus();
    expect(result).toBe('missing');
  });
});
