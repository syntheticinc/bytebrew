import { describe, it, expect } from 'bun:test';
import { UpdateChecker } from './UpdateChecker.js';

describe('UpdateChecker.isNewer', () => {
  const checker = new UpdateChecker('0.0.0');

  const cases: Array<{ latest: string; current: string; expected: boolean }> = [
    { latest: '1.0.0', current: '0.9.9', expected: true },
    { latest: '0.3.0', current: '0.2.0', expected: true },
    { latest: '0.2.1', current: '0.2.0', expected: true },
    { latest: '0.2.0', current: '0.2.0', expected: false },
    { latest: '0.1.0', current: '0.2.0', expected: false },
    { latest: '0.2.0', current: '0.3.0', expected: false },
    { latest: '2.0.0', current: '1.9.9', expected: true },
    { latest: '1.0.0', current: '1.0.0', expected: false },
    { latest: '0.0.1', current: '0.0.0', expected: true },
    { latest: '0.0.0', current: '0.0.1', expected: false },
    // Edge: malformed versions default to 0
    { latest: '1', current: '0.0.0', expected: true },
    { latest: '1.2', current: '1.1.0', expected: true },
  ];

  for (const { latest, current, expected } of cases) {
    it(`${latest} > ${current} = ${expected}`, () => {
      expect(checker.isNewer(latest, current)).toBe(expected);
    });
  }
});
