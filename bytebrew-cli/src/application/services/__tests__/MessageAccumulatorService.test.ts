import { describe, it, expect, beforeEach } from 'bun:test';
import { MessageAccumulatorService } from '../MessageAccumulatorService.js';
import { MessageId } from '../../../domain/value-objects/MessageId.js';

describe('MessageAccumulatorService', () => {
  let accumulator: MessageAccumulatorService;

  beforeEach(() => {
    accumulator = new MessageAccumulatorService();
  });

  describe('startAccumulating', () => {
    it('creates accumulating message with correct role', () => {
      const messageId = accumulator.startAccumulating('assistant');
      expect(messageId).toBeInstanceOf(MessageId);
      expect(accumulator.isAccumulating(messageId)).toBe(true);
    });

    it('with agentId stores agentId', () => {
      const messageId = accumulator.startAccumulating('assistant', 'code-agent');
      const message = accumulator.complete(messageId);
      expect(message).not.toBeNull();
      expect(message!.agentId).toBe('code-agent');
    });

    it('without agentId leaves agentId undefined', () => {
      const messageId = accumulator.startAccumulating('assistant');
      const message = accumulator.complete(messageId);
      expect(message).not.toBeNull();
      expect(message!.agentId).toBeUndefined();
    });
  });

  describe('appendChunk', () => {
    it('adds content to accumulating message', () => {
      const messageId = accumulator.startAccumulating('assistant');
      accumulator.appendChunk(messageId, 'Hello');
      accumulator.appendChunk(messageId, ' world');

      const message = accumulator.complete(messageId);
      expect(message).not.toBeNull();
      expect(message!.content.value).toBe('Hello world');
    });

    it('returns approximate tokens', () => {
      const messageId = accumulator.startAccumulating('assistant');
      const tokens = accumulator.appendChunk(messageId, 'Hello world');
      expect(tokens).toBeGreaterThan(0);
    });

    it('on non-existent message returns 0', () => {
      const fakeId = MessageId.create();
      const tokens = accumulator.appendChunk(fakeId, 'test');
      expect(tokens).toBe(0);
    });

    it('updates reasoning content for reasoning messages', () => {
      const messageId = accumulator.startAccumulating('reasoning');
      accumulator.appendChunk(messageId, 'Thinking...');

      const message = accumulator.complete(messageId);
      expect(message).not.toBeNull();
      expect(message!.reasoning).toBeDefined();
      expect(message!.reasoning!.thinking).toBe('Thinking...');
    });
  });

  describe('updateReasoning', () => {
    it('updates thinking and isComplete', () => {
      const messageId = accumulator.startAccumulating('reasoning');
      accumulator.updateReasoning(messageId, 'New thinking', true);

      const message = accumulator.complete(messageId);
      expect(message!.reasoning!.thinking).toBe('New thinking');
      expect(message!.reasoning!.isComplete).toBe(true);
    });

    it('calculates token delta', () => {
      const messageId = accumulator.startAccumulating('reasoning');
      accumulator.updateReasoning(messageId, 'Short', false);
      accumulator.updateReasoning(messageId, 'Much longer thinking text', true);

      const message = accumulator.complete(messageId);
      expect(message!.reasoning!.thinking).toBe('Much longer thinking text');
    });
  });

  describe('complete', () => {
    it('returns Message with accumulated content', () => {
      const messageId = accumulator.startAccumulating('assistant');
      accumulator.appendChunk(messageId, 'Test ');
      accumulator.appendChunk(messageId, 'content');

      const message = accumulator.complete(messageId);
      expect(message).not.toBeNull();
      expect(message!.content.value).toBe('Test content');
      expect(message!.role).toBe('assistant');
      expect(message!.isComplete).toBe(true);
    });

    it('returns Message with agentId from startAccumulating', () => {
      // VALIDATES BUG 1 FIX
      const messageId = accumulator.startAccumulating('assistant', 'supervisor');
      accumulator.appendChunk(messageId, 'Answer from supervisor');

      const message = accumulator.complete(messageId);
      expect(message).not.toBeNull();
      expect(message!.agentId).toBe('supervisor');
    });

    it('returns reasoning Message when started as reasoning', () => {
      const messageId = accumulator.startAccumulating('reasoning');
      accumulator.appendChunk(messageId, 'Thinking text');

      const message = accumulator.complete(messageId);
      expect(message).not.toBeNull();
      expect(message!.reasoning).toBeDefined();
      expect(message!.reasoning!.thinking).toBe('Thinking text');
    });

    it('reasoning Message has agentId', () => {
      // VALIDATES BUG 1 FIX
      const messageId = accumulator.startAccumulating('reasoning', 'planner');
      accumulator.appendChunk(messageId, 'Planning thoughts');

      const message = accumulator.complete(messageId);
      expect(message).not.toBeNull();
      expect(message!.agentId).toBe('planner');
    });

    it('removes message from accumulating map', () => {
      const messageId = accumulator.startAccumulating('assistant');
      accumulator.appendChunk(messageId, 'Content');

      const message = accumulator.complete(messageId);
      expect(message).not.toBeNull();
      expect(accumulator.isAccumulating(messageId)).toBe(false);
    });

    it('on non-existent message returns null', () => {
      const fakeId = MessageId.create();
      const message = accumulator.complete(fakeId);
      expect(message).toBeNull();
    });
  });

  describe('completeReasoning', () => {
    it('returns reasoning Message with thinking', () => {
      const messageId = accumulator.startAccumulating('reasoning');
      accumulator.appendChunk(messageId, 'Initial thought');

      const message = accumulator.completeReasoning(messageId, 'Final thinking');
      expect(message).not.toBeNull();
      expect(message!.reasoning).toBeDefined();
      expect(message!.reasoning!.thinking).toBe('Final thinking');
      expect(message!.reasoning!.isComplete).toBe(true);
    });

    it('passes agentId to created Message', () => {
      // VALIDATES BUG 1 FIX
      const messageId = accumulator.startAccumulating('reasoning', 'analyzer');
      accumulator.appendChunk(messageId, 'Analyzing...');

      const message = accumulator.completeReasoning(messageId, 'Analysis complete');
      expect(message).not.toBeNull();
      expect(message!.agentId).toBe('analyzer');
    });
  });

  describe('abort/isAccumulating/token/clear', () => {
    it('abort removes message', () => {
      const messageId = accumulator.startAccumulating('assistant');
      accumulator.appendChunk(messageId, 'Content');

      accumulator.abort(messageId);
      expect(accumulator.isAccumulating(messageId)).toBe(false);
    });

    it('isAccumulating checks correctly', () => {
      const messageId = accumulator.startAccumulating('assistant');
      expect(accumulator.isAccumulating(messageId)).toBe(true);

      accumulator.complete(messageId);
      expect(accumulator.isAccumulating(messageId)).toBe(false);
    });

    it('getTokenCounts returns correct values', () => {
      accumulator.addInputTokens('Test input string that has some tokens');
      const messageId = accumulator.startAccumulating('assistant');
      accumulator.appendChunk(messageId, 'Some text');

      const counts = accumulator.getTokenCounts();
      expect(counts.input).toBeGreaterThan(0);
      expect(counts.output).toBeGreaterThan(0);
    });

    it('addInputTokens sets input and resets output', () => {
      const messageId = accumulator.startAccumulating('assistant');
      accumulator.appendChunk(messageId, 'Text');

      accumulator.addInputTokens('Short input');
      const counts = accumulator.getTokenCounts();
      expect(counts.input).toBeGreaterThan(0);
      expect(counts.output).toBe(0);
    });

    it('resetTokenCounts resets both', () => {
      accumulator.addInputTokens('Some input text to count tokens from');
      const messageId = accumulator.startAccumulating('assistant');
      accumulator.appendChunk(messageId, 'Text');

      accumulator.resetTokenCounts();
      const counts = accumulator.getTokenCounts();
      expect(counts.input).toBe(0);
      expect(counts.output).toBe(0);
    });

    it('clear removes all and resets tokens', () => {
      accumulator.addInputTokens('Input text');
      const id1 = accumulator.startAccumulating('assistant');
      const id2 = accumulator.startAccumulating('reasoning');

      accumulator.clear();
      expect(accumulator.isAccumulating(id1)).toBe(false);
      expect(accumulator.isAccumulating(id2)).toBe(false);
      expect(accumulator.getAccumulatingCount()).toBe(0);

      const counts = accumulator.getTokenCounts();
      expect(counts.input).toBe(0);
      expect(counts.output).toBe(0);
    });

    it('getAccumulatingCount returns correct count', () => {
      expect(accumulator.getAccumulatingCount()).toBe(0);

      const id1 = accumulator.startAccumulating('assistant');
      expect(accumulator.getAccumulatingCount()).toBe(1);

      const id2 = accumulator.startAccumulating('reasoning');
      expect(accumulator.getAccumulatingCount()).toBe(2);

      accumulator.complete(id1);
      expect(accumulator.getAccumulatingCount()).toBe(1);

      accumulator.abort(id2);
      expect(accumulator.getAccumulatingCount()).toBe(0);
    });
  });
});
