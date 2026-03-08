import { describe, test, expect } from 'bun:test';
import { MobilePairingBlock } from '../MobilePairingBlock';
import { OnboardingStateStore } from '../../../../infrastructure/onboarding/OnboardingStateStore';

// --- In-memory mock for OnboardingStateStore ---
// Overrides filesystem methods to keep state in memory.

class MockOnboardingStateStore extends OnboardingStateStore {
  private blocks: Record<string, {
    completedAt?: string;
    lastSkippedAt?: string;
    skipCount: number;
    reofferAfter?: string;
  }> = {};

  override getBlockState(blockId: string) {
    return this.blocks[blockId];
  }

  override markCompleted(blockId: string): void {
    this.blocks[blockId] = {
      ...this.blocks[blockId],
      completedAt: new Date().toISOString(),
      skipCount: this.blocks[blockId]?.skipCount ?? 0,
    };
  }

  override markSkipped(blockId: string, reofferDays: number): void {
    const now = new Date();
    const reofferAfter = new Date(now.getTime() + reofferDays * 24 * 60 * 60 * 1000);
    const current = this.blocks[blockId];
    this.blocks[blockId] = {
      ...current,
      lastSkippedAt: now.toISOString(),
      skipCount: (current?.skipCount ?? 0) + 1,
      reofferAfter: reofferAfter.toISOString(),
    };
  }

  /** Helper: set block state directly for test setup */
  setBlockState(blockId: string, state: {
    completedAt?: string;
    lastSkippedAt?: string;
    skipCount?: number;
    reofferAfter?: string;
  }): void {
    this.blocks[blockId] = {
      skipCount: 0,
      ...state,
    };
  }
}

function createBlock() {
  const stateStore = new MockOnboardingStateStore();
  const block = new MobilePairingBlock(stateStore);
  return { block, stateStore };
}

// ---------------------------------------------------------------------------
// Properties
// ---------------------------------------------------------------------------

describe('MobilePairingBlock — properties', () => {
  test('id is "mobile-pairing"', () => {
    const { block } = createBlock();
    expect(block.id).toBe('mobile-pairing');
  });

  test('displayName is "Mobile App"', () => {
    const { block } = createBlock();
    expect(block.displayName).toBe('Mobile App');
  });

  test('description is set', () => {
    const { block } = createBlock();
    expect(block.description).toBe('Set up ByteBrew mobile app for remote access');
  });

  test('priority is 10', () => {
    const { block } = createBlock();
    expect(block.priority).toBe(10);
  });

  test('skippable is true', () => {
    const { block } = createBlock();
    expect(block.skippable).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// check() — needs onboarding
// ---------------------------------------------------------------------------

describe('MobilePairingBlock — check()', () => {
  test('needs onboarding when no state exists', () => {
    const { block } = createBlock();

    const result = block.check();

    expect(result.needsOnboarding).toBe(true);
    expect(result.skipCount).toBe(0);
    expect(result.lastSkippedAt).toBeUndefined();
  });

  test('no onboarding needed when completed', () => {
    const { block, stateStore } = createBlock();
    stateStore.setBlockState('mobile-pairing', {
      completedAt: new Date().toISOString(),
    });

    const result = block.check();

    expect(result.needsOnboarding).toBe(false);
    expect(result.skipCount).toBe(0);
  });

  test('needs onboarding when skipped (not completed)', () => {
    const { block, stateStore } = createBlock();
    stateStore.setBlockState('mobile-pairing', {
      lastSkippedAt: '2026-03-01T00:00:00.000Z',
      skipCount: 1,
    });

    const result = block.check();

    expect(result.needsOnboarding).toBe(true);
    expect(result.skipCount).toBe(1);
    expect(result.lastSkippedAt).toEqual(new Date('2026-03-01T00:00:00.000Z'));
  });

  test('reports correct skip count after multiple skips', () => {
    const { block, stateStore } = createBlock();
    stateStore.setBlockState('mobile-pairing', {
      lastSkippedAt: '2026-03-05T12:00:00.000Z',
      skipCount: 3,
    });

    const result = block.check();

    expect(result.needsOnboarding).toBe(true);
    expect(result.skipCount).toBe(3);
    expect(result.lastSkippedAt).toEqual(new Date('2026-03-05T12:00:00.000Z'));
  });

  test('completed takes precedence over skipped state', () => {
    const { block, stateStore } = createBlock();
    stateStore.setBlockState('mobile-pairing', {
      completedAt: '2026-03-06T00:00:00.000Z',
      lastSkippedAt: '2026-03-05T00:00:00.000Z',
      skipCount: 2,
    });

    const result = block.check();

    expect(result.needsOnboarding).toBe(false);
    // When completed, skipCount is always 0 per the implementation
    expect(result.skipCount).toBe(0);
  });

  test('no lastSkippedAt when state has no skips', () => {
    const { block, stateStore } = createBlock();
    stateStore.setBlockState('mobile-pairing', {
      skipCount: 0,
    });

    const result = block.check();

    expect(result.needsOnboarding).toBe(true);
    expect(result.lastSkippedAt).toBeUndefined();
  });
});

// ---------------------------------------------------------------------------
// check() — after markCompleted / markSkipped on mock store
// ---------------------------------------------------------------------------

describe('MobilePairingBlock — check() after state transitions', () => {
  test('check returns needsOnboarding=false after markCompleted', () => {
    const { block, stateStore } = createBlock();

    // Initially needs onboarding
    expect(block.check().needsOnboarding).toBe(true);

    stateStore.markCompleted('mobile-pairing');

    const result = block.check();
    expect(result.needsOnboarding).toBe(false);
  });

  test('check still needs onboarding after markSkipped', () => {
    const { block, stateStore } = createBlock();

    stateStore.markSkipped('mobile-pairing', 7);

    const result = block.check();
    expect(result.needsOnboarding).toBe(true);
    expect(result.skipCount).toBe(1);
    expect(result.lastSkippedAt).toBeDefined();
  });

  test('skipCount increments with each markSkipped call', () => {
    const { block, stateStore } = createBlock();

    stateStore.markSkipped('mobile-pairing', 7);
    stateStore.markSkipped('mobile-pairing', 7);
    stateStore.markSkipped('mobile-pairing', 7);

    const result = block.check();
    expect(result.skipCount).toBe(3);
  });
});

// ---------------------------------------------------------------------------
// OnboardingBlock interface conformance
// ---------------------------------------------------------------------------

describe('MobilePairingBlock — interface conformance', () => {
  test('implements OnboardingBlock interface (has all required members)', () => {
    const { block } = createBlock();

    // Properties
    expect(typeof block.id).toBe('string');
    expect(typeof block.displayName).toBe('string');
    expect(typeof block.description).toBe('string');
    expect(typeof block.priority).toBe('number');
    expect(typeof block.skippable).toBe('boolean');

    // Methods
    expect(typeof block.check).toBe('function');
    expect(typeof block.run).toBe('function');
  });

  test('check() returns OnboardingCheckResult shape', () => {
    const { block } = createBlock();

    const result = block.check();

    expect(typeof result.needsOnboarding).toBe('boolean');
    expect(typeof result.skipCount).toBe('number');
    // lastSkippedAt is optional (Date | undefined)
  });
});
