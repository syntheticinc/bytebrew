import { describe, it, expect } from 'bun:test';
import { MessageId } from '../MessageId.js';

describe('MessageId', () => {
  it('create generates unique ids', () => {
    const id1 = MessageId.create();
    const id2 = MessageId.create();
    const id3 = MessageId.create();

    expect(id1.value).toBeTypeOf('string');
    expect(id1.value).not.toBe('');
    expect(id1.value).not.toBe(id2.value);
    expect(id1.value).not.toBe(id3.value);
    expect(id2.value).not.toBe(id3.value);
  });

  it('from creates with given value', () => {
    const value = 'custom-message-id-123';
    const id = MessageId.from(value);

    expect(id.value).toBe(value);
  });

  it('from throws on empty string', () => {
    expect(() => MessageId.from('')).toThrow('MessageId cannot be empty');
    expect(() => MessageId.from('   ')).toThrow('MessageId cannot be empty');
  });

  it('equals returns true for same value', () => {
    const id1 = MessageId.from('test-id');
    const id2 = MessageId.from('test-id');
    const id3 = MessageId.from('different-id');

    expect(id1.equals(id2)).toBe(true);
    expect(id1.equals(id3)).toBe(false);
    expect(id2.equals(id3)).toBe(false);
  });

  it('toString returns value', () => {
    const value = 'msg-12345';
    const id = MessageId.from(value);

    expect(id.toString()).toBe(value);
    expect(String(id)).toBe(value);
  });
});
