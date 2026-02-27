import { describe, it, expect } from 'bun:test';
import { MessageContent } from '../MessageContent.js';

describe('MessageContent', () => {
  it('from creates with value', () => {
    const content = MessageContent.from('Hello, world!');

    expect(content.value).toBe('Hello, world!');
    expect(content.isEmpty).toBe(false);
  });

  it('empty creates empty content', () => {
    const content = MessageContent.empty();

    expect(content.value).toBe('');
    expect(content.isEmpty).toBe(true);
    expect(content.length).toBe(0);
  });

  it('isEmpty returns true for empty', () => {
    const empty = MessageContent.empty();
    const nonEmpty = MessageContent.from('test');

    expect(empty.isEmpty).toBe(true);
    expect(nonEmpty.isEmpty).toBe(false);
  });

  it('length returns string length', () => {
    const content1 = MessageContent.from('test');
    const content2 = MessageContent.from('hello world');
    const empty = MessageContent.empty();

    expect(content1.length).toBe(4);
    expect(content2.length).toBe(11);
    expect(empty.length).toBe(0);
  });

  it('approximateTokens calculates correctly', () => {
    // 4 chars per token
    const content1 = MessageContent.from('test'); // 4 chars = 1 token
    const content2 = MessageContent.from('hello'); // 5 chars = 2 tokens (ceil)
    const content3 = MessageContent.from('hello world test'); // 16 chars = 4 tokens
    const empty = MessageContent.empty();

    expect(content1.approximateTokens).toBe(1);
    expect(content2.approximateTokens).toBe(2);
    expect(content3.approximateTokens).toBe(4);
    expect(empty.approximateTokens).toBe(0);
  });

  it('append returns new content with appended text', () => {
    const content = MessageContent.from('Hello');
    const appended = content.append(', world!');

    expect(appended.value).toBe('Hello, world!');
    expect(content.value).toBe('Hello'); // Original unchanged (immutability)

    // Multiple appends
    const final = content.append(' beautiful').append(' world');
    expect(final.value).toBe('Hello beautiful world');
  });

  it('truncate shortens content with ellipsis', () => {
    const content = MessageContent.from('This is a long message');

    const truncated = content.truncate(10);
    expect(truncated.value).toBe('This is a ...');

    // No truncation if within limit
    const notTruncated = content.truncate(100);
    expect(notTruncated.value).toBe('This is a long message');

    // Exact length
    const exact = content.truncate(22);
    expect(exact.value).toBe('This is a long message'); // No ellipsis
  });

  it('equals compares by value', () => {
    const content1 = MessageContent.from('test');
    const content2 = MessageContent.from('test');
    const content3 = MessageContent.from('different');

    expect(content1.equals(content2)).toBe(true);
    expect(content1.equals(content3)).toBe(false);
  });
});
