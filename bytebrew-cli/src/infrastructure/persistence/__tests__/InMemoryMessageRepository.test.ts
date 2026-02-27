import { describe, it, expect, beforeEach } from 'bun:test';
import { InMemoryMessageRepository } from '../InMemoryMessageRepository.js';
import { Message } from '../../../domain/entities/Message.js';
import { MessageId } from '../../../domain/value-objects/MessageId.js';

describe('InMemoryMessageRepository', () => {
  let repo: InMemoryMessageRepository;

  beforeEach(() => {
    repo = new InMemoryMessageRepository();
  });

  it('save stores message', () => {
    const message = Message.createUser('test content');
    repo.save(message);

    const found = repo.findById(message.id);
    expect(found).toBe(message);
  });

  it('save updates existing message (same id)', () => {
    const id = MessageId.create();
    const message1 = Message.createAssistant(id);
    const message2 = message1.appendContent('updated');

    repo.save(message1);
    repo.save(message2);

    const found = repo.findById(id);
    expect(found).toBe(message2);
    expect(found?.content.value).toBe('updated');
    expect(repo.count()).toBe(1); // Only one message
  });

  it('findById returns correct message', () => {
    const msg1 = Message.createUser('first');
    const msg2 = Message.createUser('second');
    const msg3 = Message.createUser('third');

    repo.save(msg1);
    repo.save(msg2);
    repo.save(msg3);

    expect(repo.findById(msg1.id)).toBe(msg1);
    expect(repo.findById(msg2.id)).toBe(msg2);
    expect(repo.findById(msg3.id)).toBe(msg3);
  });

  it('findById returns undefined for missing', () => {
    const nonExistentId = MessageId.create();
    expect(repo.findById(nonExistentId)).toBeUndefined();
  });

  it('findByToolCallId returns correct message', () => {
    const toolCallMsg = Message.createToolCall({
      callId: 'test-call-123',
      toolName: 'read_file',
      arguments: { path: '/test' },
    });

    repo.save(toolCallMsg);

    const found = repo.findByToolCallId('test-call-123');
    expect(found).toBe(toolCallMsg);
  });

  it('findAll returns messages in insertion order', () => {
    const msg1 = Message.createUser('first');
    const msg2 = Message.createUser('second');
    const msg3 = Message.createUser('third');

    repo.save(msg1);
    repo.save(msg2);
    repo.save(msg3);

    const all = repo.findAll();
    expect(all).toEqual([msg1, msg2, msg3]);
  });

  it('findComplete returns only complete messages', () => {
    const completeMsg = Message.createUser('complete');
    const pendingMsg = Message.createAssistant(); // pending
    const streamingMsg = Message.createAssistant().appendContent('test'); // streaming

    repo.save(completeMsg);
    repo.save(pendingMsg);
    repo.save(streamingMsg);

    const complete = repo.findComplete();
    expect(complete).toHaveLength(1);
    expect(complete[0]).toBe(completeMsg);
  });

  it('findRecent returns last N messages', () => {
    const msg1 = Message.createUser('1');
    const msg2 = Message.createUser('2');
    const msg3 = Message.createUser('3');
    const msg4 = Message.createUser('4');
    const msg5 = Message.createUser('5');

    repo.save(msg1);
    repo.save(msg2);
    repo.save(msg3);
    repo.save(msg4);
    repo.save(msg5);

    const recent3 = repo.findRecent(3);
    expect(recent3).toEqual([msg3, msg4, msg5]);

    const recent2 = repo.findRecent(2);
    expect(recent2).toEqual([msg4, msg5]);

    const recent10 = repo.findRecent(10);
    expect(recent10).toHaveLength(5); // All messages
  });

  it('delete removes message', () => {
    const msg1 = Message.createUser('first');
    const msg2 = Message.createUser('second');

    repo.save(msg1);
    repo.save(msg2);

    expect(repo.count()).toBe(2);

    repo.delete(msg1.id);

    expect(repo.count()).toBe(1);
    expect(repo.findById(msg1.id)).toBeUndefined();
    expect(repo.findById(msg2.id)).toBe(msg2);
  });

  it('clear removes all messages', () => {
    repo.save(Message.createUser('1'));
    repo.save(Message.createUser('2'));
    repo.save(Message.createUser('3'));

    expect(repo.count()).toBe(3);

    repo.clear();

    expect(repo.count()).toBe(0);
    expect(repo.findAll()).toEqual([]);
  });

  it('subscribe notifies on changes', () => {
    const notifications: Message[][] = [];

    // Subscribe returns current state immediately
    const unsubscribe = repo.subscribe((messages) => {
      notifications.push(messages);
    });

    // First notification is immediate with empty state
    expect(notifications).toHaveLength(1);
    expect(notifications[0]).toEqual([]);

    // Add message
    const msg1 = Message.createUser('first');
    repo.save(msg1);

    expect(notifications).toHaveLength(2);
    expect(notifications[1]).toEqual([msg1]);

    // Add another
    const msg2 = Message.createUser('second');
    repo.save(msg2);

    expect(notifications).toHaveLength(3);
    expect(notifications[2]).toEqual([msg1, msg2]);

    // Delete
    repo.delete(msg1.id);

    expect(notifications).toHaveLength(4);
    expect(notifications[3]).toEqual([msg2]);

    // No more notifications after unsubscribe
    unsubscribe();

    repo.save(Message.createUser('third'));
    expect(notifications).toHaveLength(4); // Still 4
  });

  it('count returns correct number', () => {
    expect(repo.count()).toBe(0);

    repo.save(Message.createUser('1'));
    expect(repo.count()).toBe(1);

    repo.save(Message.createUser('2'));
    expect(repo.count()).toBe(2);

    repo.save(Message.createUser('3'));
    expect(repo.count()).toBe(3);

    repo.clear();
    expect(repo.count()).toBe(0);
  });
});
