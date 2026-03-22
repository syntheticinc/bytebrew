import { describe, it, expect, beforeEach } from 'bun:test';
import { handleAnswerChunk, handleAnswer } from '../AnswerStreamHandler.js';
import { StreamProcessorContext } from '../StreamProcessorContext.js';
import { MessageId } from '../../../../domain/value-objects/MessageId.js';
import { MessageAccumulatorService } from '../../MessageAccumulatorService.js';
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

describe('handleAnswerChunk', () => {
  it('creates new accumulating message on first chunk', () => {
    const ctx = createTestContext();
    const response = {
      type: 'ANSWER_CHUNK' as const,
      content: 'Hello',
      isFinal: false,
    };

    handleAnswerChunk(ctx, response);

    const messageId = ctx.getCurrentMessageId();
    expect(messageId).not.toBeNull();
    expect(ctx.accumulator.isAccumulating(messageId!)).toBe(true);
  });

  it('appends to existing accumulating message', () => {
    const ctx = createTestContext();

    handleAnswerChunk(ctx, { type: 'ANSWER_CHUNK', content: 'Hello', isFinal: false });
    const messageId = ctx.getCurrentMessageId();

    handleAnswerChunk(ctx, { type: 'ANSWER_CHUNK', content: ' world', isFinal: false });

    const message = ctx.accumulator.complete(messageId!);
    expect(message!.content.value).toBe('Hello world');
  });

  it('publishes MessageStarted on first chunk', () => {
    const ctx = createTestContext();
    const eventBus = ctx.eventBus as MockEventBus;

    handleAnswerChunk(ctx, { type: 'ANSWER_CHUNK', content: 'First chunk', isFinal: false });

    expect(eventBus.publishedEvents.length).toBeGreaterThan(0);
    const startedEvent = eventBus.publishedEvents.find(e => e.type === 'MessageStarted');
    expect(startedEvent).toBeDefined();
  });

  it('publishes StreamingProgress on each chunk', () => {
    const ctx = createTestContext();
    const eventBus = ctx.eventBus as MockEventBus;

    handleAnswerChunk(ctx, { type: 'ANSWER_CHUNK', content: 'Chunk 1', isFinal: false });
    const eventCountAfterFirst = eventBus.publishedEvents.length;

    handleAnswerChunk(ctx, { type: 'ANSWER_CHUNK', content: ' Chunk 2', isFinal: false });

    expect(eventBus.publishedEvents.length).toBeGreaterThan(eventCountAfterFirst);
    const progressEvents = eventBus.publishedEvents.filter(e => e.type === 'StreamingProgress');
    expect(progressEvents.length).toBeGreaterThanOrEqual(2);
  });

  it('passes agentId to startAccumulating (verify created message has agentId)', () => {
    // VALIDATES FIX
    const ctx = createTestContext('supervisor');

    handleAnswerChunk(ctx, { type: 'ANSWER_CHUNK', content: 'Answer from supervisor', isFinal: false });

    const messageId = ctx.getCurrentMessageId();
    const message = ctx.accumulator.complete(messageId!);
    expect(message).not.toBeNull();
    expect(message!.agentId).toBe('supervisor');
  });
});

describe('handleAnswer', () => {
  it('completes accumulated message and saves', () => {
    const ctx = createTestContext();
    const repo = ctx.messageRepository as MockMessageRepository;

    handleAnswerChunk(ctx, { type: 'ANSWER_CHUNK', content: 'Accumulated', isFinal: false });
    handleAnswer(ctx, { type: 'ANSWER', content: '', isFinal: false });

    const messages = repo.findAll();
    expect(messages.length).toBe(1);
    expect(messages[0].content.value).toBe('Accumulated');
  });

  it('completed message has correct agentId', () => {
    // VALIDATES FIX
    const ctx = createTestContext('code-agent');
    const repo = ctx.messageRepository as MockMessageRepository;

    handleAnswerChunk(ctx, { type: 'ANSWER_CHUNK', content: 'Code analysis', isFinal: false });
    handleAnswer(ctx, { type: 'ANSWER', content: ' complete', isFinal: false });

    const messages = repo.findAll();
    expect(messages.length).toBe(1);
    expect(messages[0].agentId).toBe('code-agent');
  });

  it('publishes MessageCompleted event', () => {
    const ctx = createTestContext();
    const eventBus = ctx.eventBus as MockEventBus;

    handleAnswerChunk(ctx, { type: 'ANSWER_CHUNK', content: 'Test', isFinal: false });
    handleAnswer(ctx, { type: 'ANSWER', content: '', isFinal: false });

    const completedEvent = eventBus.publishedEvents.find(e => e.type === 'MessageCompleted');
    expect(completedEvent).toBeDefined();
  });

  it('skips FinalizeAccumulatedText duplicate (isFinal=false, no active chunks)', () => {
    // Server sends ANSWER with isFinal=false after chunks were already completed
    // by completeCurrentMessage (from TOOL_CALL handler). Should NOT create duplicate.
    const ctx = createTestContext();
    const repo = ctx.messageRepository as MockMessageRepository;

    handleAnswer(ctx, { type: 'ANSWER', content: 'Already saved text', isFinal: false });

    const messages = repo.findAll();
    expect(messages.length).toBe(0); // Duplicate skipped
  });

  it('creates message for code agent intermediate text (isFinal=false)', () => {
    // Code agents use non-streaming mode. FinalizeAccumulatedText sends ANSWER
    // with isFinal=false. Unlike supervisor, code agents should create messages.
    const ctx = createTestContext('code-agent-abc');
    const repo = ctx.messageRepository as MockMessageRepository;

    handleAnswer(ctx, { type: 'ANSWER', content: 'Analyzing the code...', isFinal: false });

    const messages = repo.findAll();
    expect(messages.length).toBe(1);
    expect(messages[0].content.value).toBe('Analyzing the code...');
    expect(messages[0].agentId).toBe('code-agent-abc');
  });

  it('creates direct answer for non-streaming mode (isFinal=true)', () => {
    // Non-streaming ANSWER with isFinal=true and content — genuine standalone answer
    const ctx = createTestContext();
    const repo = ctx.messageRepository as MockMessageRepository;

    handleAnswer(ctx, { type: 'ANSWER', content: 'Direct answer', isFinal: true });

    const messages = repo.findAll();
    expect(messages.length).toBe(1);
    expect(messages[0].content.value).toBe('Direct answer');
  });

  it('direct non-streaming answer includes agentId', () => {
    const ctx = createTestContext('planner');
    const repo = ctx.messageRepository as MockMessageRepository;

    handleAnswer(ctx, { type: 'ANSWER', content: 'Planning complete', isFinal: true });

    const messages = repo.findAll();
    expect(messages.length).toBe(1);
    expect(messages[0].agentId).toBe('planner');
  });

  it('isFinal from supervisor stops processing', () => {
    const ctx = createTestContext('supervisor');

    handleAnswer(ctx, { type: 'ANSWER', content: 'Final answer', isFinal: true, agentId: 'supervisor' });

    expect(ctx.getIsProcessing()).toBe(false);
  });

  it('isFinal from code-agent does NOT stop processing', () => {
    const ctx = createTestContext('code-agent');

    handleAnswer(ctx, { type: 'ANSWER', content: 'Sub-answer', isFinal: true, agentId: 'code-agent' });

    expect(ctx.getIsProcessing()).toBe(true);
  });

  it('empty content with isFinal from supervisor stops', () => {
    const ctx = createTestContext('supervisor');
    const repo = ctx.messageRepository as MockMessageRepository;

    handleAnswer(ctx, { type: 'ANSWER', content: '', isFinal: true, agentId: 'supervisor' });

    expect(ctx.getIsProcessing()).toBe(false);
    const messages = repo.findAll();
    expect(messages.length).toBe(0); // Empty content not saved
  });

  it('skips empty/whitespace accumulated messages', () => {
    // When chunks contain only whitespace, complete should skip saving
    const ctx = createTestContext();
    const repo = ctx.messageRepository as MockMessageRepository;

    handleAnswerChunk(ctx, { type: 'ANSWER_CHUNK', content: '   ', isFinal: false });
    handleAnswer(ctx, { type: 'ANSWER', content: '', isFinal: false });

    const messages = repo.findAll();
    expect(messages.length).toBe(0); // Whitespace-only not saved
  });
});
