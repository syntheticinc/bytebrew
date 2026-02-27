import { describe, it, expect } from 'bun:test';
import { handleReasoning } from '../ReasoningHandler.js';
import { handleAnswerChunk } from '../AnswerStreamHandler.js';
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
    cleanup: () => {},
  };
}

describe('handleReasoning', () => {
  it('skips when no reasoning', () => {
    const ctx = createTestContext();
    const eventBus = ctx.eventBus as MockEventBus;

    handleReasoning(ctx, { type: 'ANSWER', content: 'No reasoning', isFinal: false });

    expect(ctx.getCurrentReasoningId()).toBeNull();
    expect(eventBus.publishedEvents.length).toBe(0);
  });

  it('completes current answer before reasoning', () => {
    const ctx = createTestContext();
    const repo = ctx.messageRepository as MockMessageRepository;

    handleAnswerChunk(ctx, { type: 'ANSWER_CHUNK', content: 'Current answer', isFinal: false });

    handleReasoning(ctx, {
      type: 'REASONING',
      content: '',
      isFinal: false,
      reasoning: { thinking: 'Thinking...', isComplete: false },
    });

    expect(ctx.getCurrentMessageId()).toBeNull();
    const messages = repo.findAll();
    expect(messages.length).toBe(1);
    expect(messages[0].content.value).toBe('Current answer');
  });

  it('creates new reasoning accumulation', () => {
    const ctx = createTestContext();

    handleReasoning(ctx, {
      type: 'REASONING',
      content: '',
      isFinal: false,
      reasoning: { thinking: 'Analyzing...', isComplete: false },
    });

    const reasoningId = ctx.getCurrentReasoningId();
    expect(reasoningId).not.toBeNull();
    expect(ctx.accumulator.isAccumulating(reasoningId!)).toBe(true);
  });

  it('passes agentId to startAccumulating', () => {
    // VALIDATES FIX
    const ctx = createTestContext('planner');

    handleReasoning(ctx, {
      type: 'REASONING',
      content: '',
      isFinal: false,
      reasoning: { thinking: 'Planning steps', isComplete: false },
    });

    const reasoningId = ctx.getCurrentReasoningId();
    const message = ctx.accumulator.complete(reasoningId!);
    expect(message).not.toBeNull();
    expect(message!.agentId).toBe('planner');
  });

  it('publishes MessageStarted for reasoning', () => {
    const ctx = createTestContext();
    const eventBus = ctx.eventBus as MockEventBus;

    handleReasoning(ctx, {
      type: 'REASONING',
      content: '',
      isFinal: false,
      reasoning: { thinking: 'First thought', isComplete: false },
    });

    const startedEvent = eventBus.publishedEvents.find(e => e.type === 'MessageStarted');
    expect(startedEvent).toBeDefined();
    expect((startedEvent as any)?.role).toBe('reasoning');
  });

  it('updates reasoning content', () => {
    const ctx = createTestContext();

    handleReasoning(ctx, {
      type: 'REASONING',
      content: '',
      isFinal: false,
      reasoning: { thinking: 'Initial thought', isComplete: false },
    });

    const reasoningId = ctx.getCurrentReasoningId();

    handleReasoning(ctx, {
      type: 'REASONING',
      content: '',
      isFinal: false,
      reasoning: { thinking: 'Updated thought', isComplete: false },
    });

    expect(ctx.getCurrentReasoningId()).toEqual(reasoningId);
  });

  it('completes reasoning when isComplete', () => {
    const ctx = createTestContext();
    const repo = ctx.messageRepository as MockMessageRepository;

    handleReasoning(ctx, {
      type: 'REASONING',
      content: '',
      isFinal: false,
      reasoning: { thinking: 'Incomplete', isComplete: false },
    });

    handleReasoning(ctx, {
      type: 'REASONING',
      content: '',
      isFinal: false,
      reasoning: { thinking: 'Complete thought', isComplete: true },
    });

    expect(ctx.getCurrentReasoningId()).toBeNull();
    const messages = repo.findAll();
    expect(messages.length).toBe(1);
    expect(messages[0].reasoning!.thinking).toBe('Complete thought');
    expect(messages[0].reasoning!.isComplete).toBe(true);
  });

  it('completed reasoning has correct agentId', () => {
    // VALIDATES FIX
    const ctx = createTestContext('analyzer');
    const repo = ctx.messageRepository as MockMessageRepository;

    handleReasoning(ctx, {
      type: 'REASONING',
      content: '',
      isFinal: false,
      reasoning: { thinking: 'Analysis', isComplete: false },
    });

    handleReasoning(ctx, {
      type: 'REASONING',
      content: '',
      isFinal: false,
      reasoning: { thinking: 'Analysis complete', isComplete: true },
    });

    const messages = repo.findAll();
    expect(messages.length).toBe(1);
    expect(messages[0].agentId).toBe('analyzer');
  });

  it('saves completed reasoning to repository', () => {
    const ctx = createTestContext();
    const repo = ctx.messageRepository as MockMessageRepository;

    handleReasoning(ctx, {
      type: 'REASONING',
      content: '',
      isFinal: false,
      reasoning: { thinking: 'Thinking', isComplete: true },
    });

    const messages = repo.findAll();
    expect(messages.length).toBe(1);
    expect(messages[0].reasoning).toBeDefined();
    expect(messages[0].reasoning!.thinking).toBe('Thinking');
  });

  it('publishes MessageCompleted on complete', () => {
    const ctx = createTestContext();
    const eventBus = ctx.eventBus as MockEventBus;

    handleReasoning(ctx, {
      type: 'REASONING',
      content: '',
      isFinal: false,
      reasoning: { thinking: 'Final thought', isComplete: true },
    });

    const completedEvent = eventBus.publishedEvents.find(e => e.type === 'MessageCompleted');
    expect(completedEvent).toBeDefined();
  });
});
