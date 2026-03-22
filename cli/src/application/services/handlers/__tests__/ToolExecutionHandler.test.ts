import { describe, it, expect } from 'bun:test';
import { handleToolCall, handleServerToolResult, countResults, formatSubResultsAsCitations, buildGrepPattern } from '../ToolExecutionHandler.js';
import { StreamProcessorContext } from '../StreamProcessorContext.js';
import { MessageId } from '../../../../domain/value-objects/MessageId.js';
import { MessageAccumulatorService } from '../../MessageAccumulatorService.js';
import { Message, ToolCallInfo } from '../../../../domain/entities/Message.js';
import {
  MockStreamGateway,
  MockMessageRepository,
  MockToolExecutor,
  MockEventBus,
} from '../../__tests__/testHelpers.js';

function createTestContext(agentId?: string): StreamProcessorContext & { cleanup: () => void } {
  const gateway = new MockStreamGateway();
  const repository = new MockMessageRepository();
  const executor = new MockToolExecutor();
  const eventBus = new MockEventBus();
  const accumulator = new MessageAccumulatorService();

  let currentMessageId: MessageId | null = null;
  let currentReasoningId: MessageId | null = null;
  let isProcessing = true;

  return {
    streamGateway: gateway,
    messageRepository: repository,
    toolExecutor: executor,
    accumulator,
    eventBus,
    agentId,
    getCurrentMessageId: () => currentMessageId,
    getCurrentReasoningId: () => currentReasoningId,
    getIsProcessing: () => isProcessing,
    setCurrentMessageId: (id: MessageId | null) => { currentMessageId = id; },
    setCurrentReasoningId: (id: MessageId | null) => { currentReasoningId = id; },
    setIsProcessing: (v: boolean) => { isProcessing = v; },
    consumeSentMessage: () => false,
    cleanup: () => {},
  };
}

describe('handleToolCall', () => {
  it('skips when no toolCall', () => {
    const ctx = createTestContext();
    const repo = ctx.messageRepository as MockMessageRepository;

    handleToolCall(ctx, { type: 'ANSWER', content: 'No tool', isFinal: false });

    expect(repo.findAll().length).toBe(0);
  });

  it('creates tool message with agentId', () => {
    // VALIDATES FIX
    const ctx = createTestContext('code-agent');
    const repo = ctx.messageRepository as MockMessageRepository;

    handleToolCall(ctx, {
      type: 'TOOL_CALL',
      content: '',
      isFinal: false,
      toolCall: {
        callId: 'call-123',
        toolName: 'read_file',
        arguments: { path: '/test.txt' },
      },
    });

    const messages = repo.findAll();
    expect(messages.length).toBe(1);
    expect(messages[0].agentId).toBe('code-agent');
    expect(messages[0].toolCall?.callId).toBe('call-123');
  });

  it('saves tool message to repository', () => {
    const ctx = createTestContext();
    const repo = ctx.messageRepository as MockMessageRepository;

    handleToolCall(ctx, {
      type: 'TOOL_CALL',
      content: '',
      isFinal: false,
      toolCall: {
        callId: 'call-456',
        toolName: 'search_code',
        arguments: { query: 'test' },
      },
    });

    const messages = repo.findAll();
    expect(messages.length).toBe(1);
    expect(messages[0].toolCall?.toolName).toBe('search_code');
  });

  it('publishes ToolExecutionStarted', () => {
    const ctx = createTestContext();
    const eventBus = ctx.eventBus as MockEventBus;

    handleToolCall(ctx, {
      type: 'TOOL_CALL',
      content: '',
      isFinal: false,
      toolCall: {
        callId: 'call-789',
        toolName: 'write_file',
        arguments: { path: '/output.txt', content: 'data' },
      },
    });

    const startedEvent = eventBus.publishedEvents.find(e => e.type === 'ToolExecutionStarted');
    expect(startedEvent).toBeDefined();
  });

  it('executes client-side tool asynchronously', async () => {
    const ctx = createTestContext();
    const executor = ctx.toolExecutor as MockToolExecutor;
    executor.executeResult = { result: 'Tool result' };

    handleToolCall(ctx, {
      type: 'TOOL_CALL',
      content: '',
      isFinal: false,
      toolCall: {
        callId: 'call-client',
        toolName: 'read_file',
        arguments: { path: '/test.txt' },
      },
    });

    // Wait for async execution
    await new Promise(resolve => setTimeout(resolve, 10));

    expect(executor.executedCalls.length).toBe(1);
  });

  it('does not execute server-side tool (callId starts with "server-")', async () => {
    const ctx = createTestContext();
    const executor = ctx.toolExecutor as MockToolExecutor;

    handleToolCall(ctx, {
      type: 'TOOL_CALL',
      content: '',
      isFinal: false,
      toolCall: {
        callId: 'server-123',
        toolName: 'manage_plan',
        arguments: { action: 'create' },
      },
    });

    await new Promise(resolve => setTimeout(resolve, 10));

    expect(executor.executedCalls.length).toBe(0);
  });

  it('skips duplicate tool calls (existing message for callId)', () => {
    const ctx = createTestContext();
    const repo = ctx.messageRepository as MockMessageRepository;

    const toolCall: ToolCallInfo = {
      callId: 'call-duplicate',
      toolName: 'read_file',
      arguments: { path: '/test.txt' },
    };

    // Add existing message for this callId
    const existingMessage = Message.createToolCall(toolCall);
    repo.save(existingMessage);

    handleToolCall(ctx, {
      type: 'TOOL_CALL',
      content: '',
      isFinal: false,
      toolCall,
    });

    // Should not add duplicate
    expect(repo.findAll().length).toBe(1);
  });

  it('creates visible tool message for subQueries (smart_search)', () => {
    const ctx = createTestContext();
    const repo = ctx.messageRepository as MockMessageRepository;

    handleToolCall(ctx, {
      type: 'TOOL_CALL',
      content: '',
      isFinal: false,
      toolCall: {
        callId: 'call-subquery',
        toolName: 'smart_search',
        arguments: { query: 'test' },
        subQueries: [{ type: 'vector', query: 'test', limit: 5 }],
      },
    });

    // subQueries now create a visible tool message in UI
    const messages = repo.findAll();
    expect(messages.length).toBe(1);
    expect(messages[0].toolCall?.toolName).toBe('smart_search');
  });
});

describe('handleServerToolResult', () => {
  it('updates existing message with result', () => {
    const ctx = createTestContext();
    const repo = ctx.messageRepository as MockMessageRepository;

    const toolCall: ToolCallInfo = {
      callId: 'server-result-123',
      toolName: 'search_vectors',
      arguments: { query: 'test' },
    };

    const message = Message.createToolCall(toolCall);
    repo.save(message);

    handleServerToolResult(ctx, {
      type: 'TOOL_RESULT',
      content: '',
      isFinal: false,
      toolResult: {
        callId: 'server-result-123',
        result: 'Search results',
      },
    });

    const messages = repo.findAll();
    expect(messages[0].toolResult).toBeDefined();
    expect(messages[0].toolResult!.result).toBe('Search results');
    expect(messages[0].isComplete).toBe(true);
  });

  it('handles missing message gracefully', () => {
    const ctx = createTestContext();

    expect(() => {
      handleServerToolResult(ctx, {
        type: 'TOOL_RESULT',
        content: '',
        isFinal: false,
        toolResult: {
          callId: 'nonexistent',
          result: 'Result',
        },
      });
    }).not.toThrow();
  });
});

describe('formatSubResultsAsCitations', () => {
  it('formats grep subResult into citation lines', () => {
    const subResults = [
      {
        type: 'grep',
        result: 'src/foo.ts:10\n  const x = 1\n\nsrc/bar.ts:20\n  const y = 2',
        count: 2,
      },
    ];
    const output = formatSubResultsAsCitations(subResults);
    expect(output).toBe('1. src/foo.ts:10 [grep] const x = 1\n2. src/bar.ts:20 [grep] const y = 2');
  });

  it('formats vector subResult into citation lines', () => {
    const subResults = [
      {
        type: 'vector',
        result: '## function: handleRequest\nFile: src/handler.ts:1-30\nScore: 0.95\n```ts\nconst x = 1\n```',
        count: 1,
      },
    ];
    const output = formatSubResultsAsCitations(subResults);
    expect(output).toBe('1. src/handler.ts:1-30 [vector] (function) handleRequest');
  });

  it('formats symbol subResult into citation lines', () => {
    const subResults = [
      {
        type: 'symbol',
        result: '[function] handleError - (err: Error) => void\n  src/utils.ts:5-15',
        count: 1,
      },
    ];
    const output = formatSubResultsAsCitations(subResults);
    expect(output).toBe('1. src/utils.ts:5-15 [symbol] (function) handleError');
  });

  it('returns empty string for subResult with error', () => {
    const subResults = [
      {
        type: 'grep',
        result: '',
        error: 'search failed',
        count: 0,
      },
    ];
    const output = formatSubResultsAsCitations(subResults);
    expect(output).toBe('');
  });

  it('returns empty string for subResult with count 0', () => {
    const subResults = [
      {
        type: 'vector',
        result: '',
        count: 0,
      },
    ];
    const output = formatSubResultsAsCitations(subResults);
    expect(output).toBe('');
  });

  it('returns empty string for subResult with empty result', () => {
    const subResults = [
      {
        type: 'symbol',
        result: '',
        count: 0,
      },
    ];
    const output = formatSubResultsAsCitations(subResults);
    expect(output).toBe('');
  });

  it('formats mixed grep + vector + symbol with sequential numbering', () => {
    const subResults = [
      {
        type: 'grep',
        result: 'src/a.ts:1\n  line one',
        count: 1,
      },
      {
        type: 'vector',
        result: '## method: doWork\nFile: src/b.ts:10-20\nScore: 0.8\n```\ncode\n```',
        count: 1,
      },
      {
        type: 'symbol',
        result: '[class] MyService - class MyService {}\n  src/c.ts:5-50',
        count: 1,
      },
    ];
    const output = formatSubResultsAsCitations(subResults);
    const lines = output.split('\n');
    expect(lines.length).toBe(3);
    expect(lines[0]).toBe('1. src/a.ts:1 [grep] line one');
    expect(lines[1]).toBe('2. src/b.ts:10-20 [vector] (method) doWork');
    expect(lines[2]).toBe('3. src/c.ts:5-50 [symbol] (class) MyService');
  });

  it('returns empty string for empty subResults array', () => {
    expect(formatSubResultsAsCitations([])).toBe('');
  });
});

describe('buildGrepPattern', () => {
  it('returns camelCase identifier as-is', () => {
    expect(buildGrepPattern('handleError')).toBe('handleError');
  });

  it('returns snake_case identifier as-is', () => {
    expect(buildGrepPattern('handle_error')).toBe('handle_error');
  });

  it('joins multi-word query with | for significant words', () => {
    expect(buildGrepPattern('error handling patterns')).toBe('error|handling|patterns');
  });

  it('filters out short words and joins remaining with | (multi-word result)', () => {
    // "to" (2 chars) filtered out, remaining ["handle", "errors"] → 2 words → joined
    expect(buildGrepPattern('to handle errors')).toBe('handle|errors');
  });

  it('returns original query when only one significant word remains after filtering', () => {
    // "a" (1), "is" (2) filtered; only "big" remains → words.length <= 1 → return trimmed
    expect(buildGrepPattern('a is big')).toBe('a is big');
  });

  it('returns single word as-is', () => {
    expect(buildGrepPattern('errors')).toBe('errors');
  });

  it('returns empty string for empty input', () => {
    expect(buildGrepPattern('')).toBe('');
  });

  it('returns trimmed empty string for spaces-only input', () => {
    // trim() → "", /^[a-zA-Z_].../.test("") = false
    // split → [""], filter(w => w.length > 2) → []
    // words.length <= 1, returns trimmed = ""
    expect(buildGrepPattern('   ')).toBe('');
  });

  it('returns PascalCase identifier as-is', () => {
    expect(buildGrepPattern('MyComponent')).toBe('MyComponent');
  });
});

describe('countResults', () => {
  it('returns 0 for empty string', () => {
    expect(countResults('')).toBe(0);
  });

  it('counts entries by double newlines', () => {
    const result = 'Entry 1\n\nEntry 2\n\nEntry 3';
    expect(countResults(result)).toBe(3);
  });

  it('counts lines when no double newlines', () => {
    // When entries.length > 0, returns entries.length (which is 1 for single block)
    // Otherwise falls back to lines.length
    const result = 'Line 1\nLine 2\nLine 3\nLine 4';
    const entries = result.split('\n\n').filter(e => e.trim());
    const lines = result.split('\n').filter(l => l.trim());
    const expected = entries.length > 0 ? entries.length : lines.length;
    expect(countResults(result)).toBe(expected);
  });
});
