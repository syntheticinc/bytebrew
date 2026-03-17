import { describe, it, expect } from 'bun:test';
import { isLifecycleMessage, getLifecycleColor, isSeparatorMessage } from '../messageClassifiers.js';

describe('messageClassifiers', () => {
  describe('isLifecycleMessage', () => {
    it('recognizes + as lifecycle (spawned)', () => {
      expect(isLifecycleMessage('+ Code Agent [abc] spawned: "Task"')).toBe(true);
    });

    it('recognizes ✓ as lifecycle (completed)', () => {
      expect(isLifecycleMessage('✓ Code Agent [xyz] completed: "Done"')).toBe(true);
    });

    it('recognizes ✗ as lifecycle (failed)', () => {
      expect(isLifecycleMessage('✗ Code Agent [err] failed: "Error"')).toBe(true);
    });

    it('recognizes ↻ as lifecycle (restarted)', () => {
      expect(isLifecycleMessage('↻ Code Agent [r1] restarted: "Retry"')).toBe(true);
    });

    it('recognizes ⊕ as lifecycle (legacy spawned)', () => {
      expect(isLifecycleMessage('⊕ Code Agent [old] spawned: "Legacy"')).toBe(true);
    });

    it('does not recognize normal message', () => {
      expect(isLifecycleMessage('Normal content without lifecycle marker')).toBe(false);
    });

    it('does not recognize [Task] prefix', () => {
      expect(isLifecycleMessage('[Task from Supervisor]\nImplement feature')).toBe(false);
    });

    it('handles leading whitespace', () => {
      expect(isLifecycleMessage('  ✓ Code Agent [abc] completed: "Done"')).toBe(true);
    });
  });

  describe('getLifecycleColor', () => {
    it('returns green for ✓ (completed)', () => {
      expect(getLifecycleColor('✓ Completed')).toBe('green');
    });

    it('returns red for ✗ (failed)', () => {
      expect(getLifecycleColor('✗ Failed')).toBe('red');
    });

    it('returns yellow for + (spawned)', () => {
      expect(getLifecycleColor('+ Spawned')).toBe('yellow');
    });

    it('returns yellow for ⊕ (legacy spawned)', () => {
      expect(getLifecycleColor('⊕ Spawned')).toBe('yellow');
    });

    it('returns blue for ↻ (restarted)', () => {
      expect(getLifecycleColor('↻ Restarted')).toBe('blue');
    });

    it('returns gray for unknown marker', () => {
      expect(getLifecycleColor('? Unknown')).toBe('gray');
    });

    it('handles leading whitespace', () => {
      expect(getLifecycleColor('  ✓ Completed')).toBe('green');
    });
  });

  describe('isSeparatorMessage', () => {
    it('recognizes separator with ───', () => {
      expect(isSeparatorMessage('─── Code Agent [abc]: Task description ───')).toBe(true);
    });

    it('recognizes separator without task description', () => {
      expect(isSeparatorMessage('─── Supervisor ───')).toBe(true);
    });

    it('does not recognize normal message', () => {
      expect(isSeparatorMessage('Normal message content')).toBe(false);
    });

    it('does not recognize lifecycle message', () => {
      expect(isSeparatorMessage('✓ Code Agent [abc] completed: "Done"')).toBe(false);
    });
  });
});
