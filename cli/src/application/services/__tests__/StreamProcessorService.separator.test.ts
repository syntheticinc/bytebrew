// Test separator creation logic for multi-agent
import { describe, it, expect, beforeEach } from 'bun:test';
import { StreamProcessorService } from '../StreamProcessorService.js';
import { MessageAccumulatorService } from '../MessageAccumulatorService.js';
import { AgentStateManager } from '../../../infrastructure/state/AgentStateManager.js';
import {
  MockStreamGateway,
  MockMessageRepository,
  MockToolExecutor,
  MockEventBus,
} from './testHelpers.js';

describe('StreamProcessorService - Separator Logic', () => {
  let gateway: MockStreamGateway;
  let repository: MockMessageRepository;
  let executor: MockToolExecutor;
  let eventBus: MockEventBus;
  let accumulator: MessageAccumulatorService;
  let agentStateManager: AgentStateManager;
  let processor: StreamProcessorService;

  beforeEach(() => {
    gateway = new MockStreamGateway();
    repository = new MockMessageRepository();
    executor = new MockToolExecutor();
    eventBus = new MockEventBus();
    accumulator = new MessageAccumulatorService();
    agentStateManager = new AgentStateManager();

    processor = new StreamProcessorService({
      streamGateway: gateway,
      messageRepository: repository,
      toolExecutor: executor,
      accumulator,
      eventBus,
      agentStateManager,
    });

    processor.initialize();
  });

  function getSeparatorMessages(): any[] {
    const completed = eventBus.getEventsOfType('MessageCompleted');
    return completed.filter((e: any) => e.message.content.value.includes('───'));
  }

  it('should create separator on first agent switch (ANSWER_CHUNK)', () => {
    // 1. Supervisor sends text
    gateway.simulateResponse({
      type: 'ANSWER_CHUNK',
      content: 'Supervisor message',
      agentId: 'supervisor',
        isFinal: false,
      });

    let separators = getSeparatorMessages();
    expect(separators.length).toBe(0); // No separator yet (single agent)

    // 2. Code agent sends text → separator created
    gateway.simulateResponse({
      type: 'ANSWER_CHUNK',
      content: 'Code agent working...',
      agentId: 'code-agent-abc',
        isFinal: false,
      });

    separators = getSeparatorMessages();
    expect(separators.length).toBe(1);
    expect(separators[0].message.content.value).toContain('Code Agent [abc]');
  });

  it('should create separator for TOOL_CALL from new agent', () => {
    // 1. Supervisor sends text
    gateway.simulateResponse({
      type: 'ANSWER_CHUNK',
      content: 'Supervisor thinking',
      agentId: 'supervisor',
        isFinal: false,
      });

    // 2. Code agent sends TOOL_CALL → separator created (agent switch)
    gateway.simulateResponse({
      type: 'TOOL_CALL',
      content: '',
      isFinal: false,
      toolCall: {
        callId: 'tc1',
        toolName: 'write_file',
        arguments: { path: 'hello.go', content: 'package main' },
      },
      agentId: 'code-agent-abc',
    });

    const separators = getSeparatorMessages();
    expect(separators.length).toBe(1); // Separator for agent switch
  });

  it('should create separator for TOOL_RESULT from new agent', () => {
    // 1. Supervisor sends text
    gateway.simulateResponse({
      type: 'ANSWER_CHUNK',
      content: 'Supervisor message',
      agentId: 'supervisor',
        isFinal: false,
      });

    // 2. Code agent sends TOOL_RESULT → separator created (agent switch)
    gateway.simulateResponse({
      type: 'TOOL_RESULT',
      content: '',
      isFinal: false,
      toolResult: {
        callId: 'tc1',
        result: 'File written',
      },
      agentId: 'code-agent-abc',
    });

    const separators = getSeparatorMessages();
    expect(separators.length).toBe(1); // Separator for agent switch
  });

  it('should NOT create separator for lifecycle events', () => {
    // 1. Supervisor lifecycle event
    gateway.simulateResponse({
      type: 'ANSWER_CHUNK',
      content: '[agent_spawned] code-agent-abc: Task description',
      agentId: 'supervisor',
        isFinal: false,
      });

    // 2. Code agent lifecycle event → NO separator (lifecycle event, not text)
    gateway.simulateResponse({
      type: 'ANSWER_CHUNK',
      content: '[agent_completed] code-agent-abc: Done',
      agentId: 'code-agent-abc',
        isFinal: false,
      });

    const separators = getSeparatorMessages();
    expect(separators.length).toBe(0); // No separator for lifecycle events
  });

  it('should create separator on first response from new agent (regardless of type)', () => {
    // 1. Supervisor text
    gateway.simulateResponse({
      type: 'ANSWER_CHUNK',
      content: 'Supervisor',
      agentId: 'supervisor',
        isFinal: false,
      });

    // 2. Code agent TOOL_CALL → separator created (first response from new agent)
    gateway.simulateResponse({
      type: 'TOOL_CALL',
      content: '',
      isFinal: false,
      toolCall: {
        callId: 'tc1',
        toolName: 'read_file',
        arguments: { path: 'hello.go' },
      },
      agentId: 'code-agent-xyz',
    });

    let separators = getSeparatorMessages();
    expect(separators.length).toBe(1);

    // 3. Code agent TOOL_RESULT → no new separator (same agent)
    gateway.simulateResponse({
      type: 'TOOL_RESULT',
      content: '',
      isFinal: false,
      toolResult: {
        callId: 'tc1',
        result: 'package main',
      },
      agentId: 'code-agent-xyz',
    });

    separators = getSeparatorMessages();
    expect(separators.length).toBe(1);

    // 4. Code agent ANSWER_CHUNK → no new separator (same agent)
    gateway.simulateResponse({
      type: 'ANSWER_CHUNK',
      content: 'Agent text response',
      agentId: 'code-agent-xyz',
        isFinal: false,
      });

    separators = getSeparatorMessages();
    expect(separators.length).toBe(1);
    expect(separators[0].message.content.value).toContain('Code Agent [xyz]');
  });

  it('should create only ONE separator per agent switch', () => {
    // 1. Supervisor text
    gateway.simulateResponse({
      type: 'ANSWER_CHUNK',
      content: 'Supervisor',
      agentId: 'supervisor',
        isFinal: false,
      });

    // 2. Code agent first text → separator created
    gateway.simulateResponse({
      type: 'ANSWER_CHUNK',
      content: 'Code agent chunk 1',
      agentId: 'code-agent-abc',
        isFinal: false,
      });

    let separators = getSeparatorMessages();
    expect(separators.length).toBe(1);

    // 3. Code agent second text → NO new separator (same agent)
    gateway.simulateResponse({
      type: 'ANSWER_CHUNK',
      content: 'Code agent chunk 2',
      agentId: 'code-agent-abc',
        isFinal: false,
      });

    separators = getSeparatorMessages();
    expect(separators.length).toBe(1); // Still 1 separator

    // 4. Supervisor text → new separator created (switch back)
    gateway.simulateResponse({
      type: 'ANSWER_CHUNK',
      content: 'Supervisor again',
      agentId: 'supervisor',
        isFinal: false,
      });

    separators = getSeparatorMessages();
    expect(separators.length).toBe(2);
    expect(separators[1].message.content.value).toContain('Supervisor');
  });

  it('should NOT create separator for single agent (no multi-agent)', () => {
    // Only supervisor (single agent) → no separators
    gateway.simulateResponse({
      type: 'ANSWER_CHUNK',
      content: 'Supervisor message 1',
      agentId: 'supervisor',
        isFinal: false,
      });

    gateway.simulateResponse({
      type: 'ANSWER_CHUNK',
      content: 'Supervisor message 2',
      agentId: 'supervisor',
        isFinal: false,
      });

    const separators = getSeparatorMessages();
    expect(separators.length).toBe(0); // No multi-agent scenario
  });
});
