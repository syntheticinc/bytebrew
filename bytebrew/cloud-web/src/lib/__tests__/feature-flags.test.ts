import { describe, it, expect, vi, afterEach } from 'vitest';

describe('feature-flags', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.resetModules();
  });

  it('defaults SHOW_EE_PRICING to false when env var is not set', async () => {
    vi.stubEnv('VITE_SHOW_EE_PRICING', '');

    const { SHOW_EE_PRICING } = await import('../feature-flags');

    expect(SHOW_EE_PRICING).toBe(false);
  });

  it('defaults SHOW_CODE_SITE to false when env var is not set', async () => {
    vi.stubEnv('VITE_SHOW_CODE_SITE', '');

    const { SHOW_CODE_SITE } = await import('../feature-flags');

    expect(SHOW_CODE_SITE).toBe(false);
  });

  it('SHOW_EE_PRICING is true when env var is "true"', async () => {
    vi.stubEnv('VITE_SHOW_EE_PRICING', 'true');

    const { SHOW_EE_PRICING } = await import('../feature-flags');

    expect(SHOW_EE_PRICING).toBe(true);
  });

  it('SHOW_CODE_SITE is true when env var is "true"', async () => {
    vi.stubEnv('VITE_SHOW_CODE_SITE', 'true');

    const { SHOW_CODE_SITE } = await import('../feature-flags');

    expect(SHOW_CODE_SITE).toBe(true);
  });

  it('SHOW_EE_PRICING is false for non-"true" values', async () => {
    vi.stubEnv('VITE_SHOW_EE_PRICING', '1');

    const { SHOW_EE_PRICING } = await import('../feature-flags');

    expect(SHOW_EE_PRICING).toBe(false);
  });
});
