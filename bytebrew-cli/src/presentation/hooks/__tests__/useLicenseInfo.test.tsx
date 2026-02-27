import { describe, it, expect, beforeAll } from 'bun:test';
import React from 'react';
import { Text } from 'ink';
import { useLicenseInfo, type LicenseBadgeInfo } from '../useLicenseInfo.js';

let render: typeof import('ink-testing-library').render;

beforeAll(async () => {
  const inkTesting = await import('ink-testing-library');
  render = inkTesting.render;
});

function makeFakeJwt(claims: Record<string, unknown>): string {
  const header = Buffer.from(JSON.stringify({ alg: 'EdDSA', typ: 'JWT' })).toString('base64url');
  const payload = Buffer.from(JSON.stringify(claims)).toString('base64url');
  return `${header}.${payload}.fakesig`;
}

let capturedInfo: LicenseBadgeInfo | null = null;

function TestComponent({ jwtLoader }: { jwtLoader: () => string | null }) {
  const info = useLicenseInfo(0, jwtLoader);
  capturedInfo = info;
  return <Text>done</Text>;
}

describe('useLicenseInfo', () => {
  it('returns null when no jwt', () => {
    capturedInfo = undefined as unknown as LicenseBadgeInfo | null;
    const inst = render(<TestComponent jwtLoader={() => null} />);
    expect(capturedInfo).toBeNull();
    inst.unmount();
  });

  it('parses tier from JWT', () => {
    const jwt = makeFakeJwt({ tier: 'personal', exp: Math.floor(Date.now() / 1000) + 86400 * 30 });
    const inst = render(<TestComponent jwtLoader={() => jwt} />);
    expect(capturedInfo).not.toBeNull();
    expect(capturedInfo!.tier).toBe('personal');
    expect(capturedInfo!.status).toBe('active');
    expect(capturedInfo!.color).toBe('green');
    expect(capturedInfo!.label).toBe('[Personal]');
    inst.unmount();
  });

  it('detects expired license', () => {
    const jwt = makeFakeJwt({ tier: 'personal', exp: Math.floor(Date.now() / 1000) - 86400 });
    const inst = render(<TestComponent jwtLoader={() => jwt} />);
    expect(capturedInfo!.status).toBe('expired');
    expect(capturedInfo!.color).toBe('red');
    expect(capturedInfo!.label).toBe('[Expired]');
    inst.unmount();
  });

  it('trial has gray color and days remaining in label', () => {
    const jwt = makeFakeJwt({ tier: 'trial', exp: Math.floor(Date.now() / 1000) + 86400 * 12 });
    const inst = render(<TestComponent jwtLoader={() => jwt} />);
    expect(capturedInfo!.tier).toBe('trial');
    expect(capturedInfo!.color).toBe('gray');
    expect(capturedInfo!.label).toMatch(/Trial.*12d/);
    inst.unmount();
  });

  it('includes proxy and byok info from claims', () => {
    const jwt = makeFakeJwt({
      tier: 'personal',
      exp: Math.floor(Date.now() / 1000) + 86400 * 30,
      proxy_steps_remaining: 200,
      proxy_steps_limit: 300,
      byok_enabled: true,
    });
    const inst = render(<TestComponent jwtLoader={() => jwt} />);
    expect(capturedInfo!.proxyStepsRemaining).toBe(200);
    expect(capturedInfo!.proxyStepsLimit).toBe(300);
    expect(capturedInfo!.byokEnabled).toBe(true);
    inst.unmount();
  });

  it('defaults to unknown tier', () => {
    const jwt = makeFakeJwt({ exp: Math.floor(Date.now() / 1000) + 86400 });
    const inst = render(<TestComponent jwtLoader={() => jwt} />);
    expect(capturedInfo!.tier).toBe('unknown');
    inst.unmount();
  });
});
