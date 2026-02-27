import { describe, it, expect } from 'bun:test';
import { computeLineDiff } from '../diff.js';

describe('computeLineDiff', () => {
  it('returns empty array for identical content', () => {
    const result = computeLineDiff('hello\nworld', 'hello\nworld');
    expect(result).toEqual([]);
  });

  it('detects single line replacement', () => {
    const result = computeLineDiff('old line\nkeep', 'new line\nkeep');
    expect(result.length).toBeGreaterThan(0);
    expect(result.some((l) => l.type === '-' && l.content.includes('old'))).toBe(true);
    expect(result.some((l) => l.type === '+' && l.content.includes('new'))).toBe(true);
  });

  it('detects multiple added lines', () => {
    const result = computeLineDiff('line1', 'line1\nline2\nline3');
    expect(result.some((l) => l.type === '+')).toBe(true);
  });

  it('detects multiple removed lines', () => {
    const result = computeLineDiff('line1\nline2\nline3', 'line1');
    expect(result.some((l) => l.type === '-')).toBe(true);
  });

  it('truncates output to maxLines', () => {
    const old = Array.from({ length: 50 }, (_, i) => `old ${i}`).join('\n');
    const newContent = Array.from({ length: 50 }, (_, i) => `new ${i}`).join('\n');
    const result = computeLineDiff(old, newContent, 20);
    expect(result.length).toBeLessThanOrEqual(21); // 20 + "... N more lines"
    expect(result[result.length - 1].content).toContain('more lines');
  });

  it('handles new file (empty old content)', () => {
    const result = computeLineDiff('', 'new\ncontent');
    expect(result.every((l) => l.type === '+')).toBe(true);
  });

  it('handles file deletion (empty new content)', () => {
    const result = computeLineDiff('old\ncontent', '');
    expect(result.every((l) => l.type === '-')).toBe(true);
  });

  it('skips binary content', () => {
    const binaryOld = 'hello\x00world';
    const binaryNew = 'hello\x00universe';
    const result = computeLineDiff(binaryOld, binaryNew);
    expect(result).toEqual([]);
  });
});
