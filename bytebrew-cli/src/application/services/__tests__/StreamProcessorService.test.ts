import { describe, it, expect, beforeEach, mock } from 'bun:test';
import { StreamProcessorService, StreamProcessorOptions } from '../StreamProcessorService.js';
import { MessageAccumulatorService } from '../MessageAccumulatorService.js';
import { MockStreamGateway, MockMessageRepository, MockToolExecutor, MockEventBus } from './testHelpers.js';

describe('StreamProcessorService', () => {
  let gateway: MockStreamGateway;
  let repository: MockMessageRepository;
  let executor: MockToolExecutor;
  let eventBus: MockEventBus;
  let accumulator: MessageAccumulatorService;
  let processor: StreamProcessorService;

  beforeEach(() => {
    gateway = new MockStreamGateway();
    repository = new MockMessageRepository();
    executor = new MockToolExecutor();
    eventBus = new MockEventBus();
    accumulator = new MessageAccumulatorService();

    processor = new StreamProcessorService({
      streamGateway: gateway,
      messageRepository: repository,
      toolExecutor: executor,
      accumulator,
      eventBus,
    });

    processor.initialize();
  });

  describe('handleResponse', () => {
    describe('Answer handling', () => {
      it('should handle ANSWER response and complete message', () => {
        gateway.simulateResponse({
          type: 'ANSWER',
          content: 'This is the answer',
          isFinal: true,
        });

        const events = eventBus.getEventsOfType('MessageCompleted');
        expect(events.length).toBe(1);
        expect((events[0] as any).message.content.value).toBe('This is the answer');
      });

      it('should handle ANSWER_CHUNK and accumulate content', () => {
        // Send chunks
        gateway.simulateResponse({
          type: 'ANSWER_CHUNK',
          content: 'Hello ',
          isFinal: false,
        });

        gateway.simulateResponse({
          type: 'ANSWER_CHUNK',
          content: 'World!',
          isFinal: false,
        });

        // Complete with final answer
        gateway.simulateResponse({
          type: 'ANSWER',
          content: '',
          isFinal: true,
        });

        const events = eventBus.getEventsOfType('MessageCompleted');
        expect(events.length).toBe(1);
        expect((events[0] as any).message.content.value).toBe('Hello World!');
      });

      it('should publish MessageStarted when streaming begins', () => {
        gateway.simulateResponse({
          type: 'ANSWER_CHUNK',
          content: 'First chunk',
          isFinal: false,
        });

        const events = eventBus.getEventsOfType('MessageStarted');
        expect(events.length).toBe(1);
        expect((events[0] as any).role).toBe('assistant');
      });
    });

    describe('Tool call handling', () => {
      it('should execute client-side tool and send result back', async () => {
        const toolCall = {
          callId: 'client-read_file-1',
          toolName: 'read_file',
          arguments: { file_path: '/test.txt' },
        };

        gateway.simulateResponse({
          type: 'TOOL_CALL',
          content: '',
          isFinal: false,
          toolCall,
        });

        // Wait for async execution
        await new Promise(resolve => setTimeout(resolve, 10));

        // Tool should be executed
        expect(executor.executedCalls.length).toBe(1);
        expect(executor.executedCalls[0].toolName).toBe('read_file');

        // Result should be sent back
        expect(gateway.sentToolResults.length).toBe(1);
        expect(gateway.sentToolResults[0].callId).toBe('client-read_file-1');
      });

      it('should NOT execute server-side tool (waits for TOOL_RESULT)', async () => {
        const toolCall = {
          callId: 'server-smart_search-1',
          toolName: 'smart_search',
          arguments: { query: 'test' },
        };

        gateway.simulateResponse({
          type: 'TOOL_CALL',
          content: '',
          isFinal: false,
          toolCall,
        });

        // Wait a bit
        await new Promise(resolve => setTimeout(resolve, 10));

        // Tool should NOT be executed (it's server-side)
        expect(executor.executedCalls.length).toBe(0);

        // Message should be created and visible
        const messages = repository.findAll();
        expect(messages.some(m => m.toolCall?.callId === 'server-smart_search-1')).toBe(true);
      });

      it('should skip duplicate tool calls', async () => {
        const toolCall = {
          callId: 'client-read_file-1',
          toolName: 'read_file',
          arguments: { file_path: '/test.txt' },
        };

        // Send same tool call twice
        gateway.simulateResponse({
          type: 'TOOL_CALL',
          content: '',
          isFinal: false,
          toolCall,
        });

        gateway.simulateResponse({
          type: 'TOOL_CALL',
          content: '',
          isFinal: false,
          toolCall,
        });

        // Wait for async execution
        await new Promise(resolve => setTimeout(resolve, 10));

        // Should only execute once
        expect(executor.executedCalls.length).toBe(1);
      });

      it('should execute subQueries and create visible tool message for grouped search', async () => {
        const toolCall = {
          callId: 'smart_search-1',
          toolName: 'smart_search',
          arguments: {},
          subQueries: [
            { type: 'grep', query: 'pattern', limit: 10 },
            { type: 'vector', query: 'semantic', limit: 5 },
          ],
        };

        gateway.simulateResponse({
          type: 'TOOL_CALL',
          content: '',
          isFinal: false,
          toolCall,
        });

        // Wait for async execution
        await new Promise(resolve => setTimeout(resolve, 20));

        // SubQueries should be executed
        expect(executor.executedCalls.length).toBe(2);

        // Result should be sent back with subResults
        expect(gateway.sentToolResults.length).toBe(1);
        expect(gateway.sentToolResults[0].subResults?.length).toBe(2);

        // Tool message should be created (visible in UI for smart_search)
        const toolMessages = repository.findAll().filter(m => m.toolCall);
        expect(toolMessages.length).toBe(1);
        expect(toolMessages[0].toolCall?.toolName).toBe('smart_search');
      });
    });

    describe('Server tool result handling', () => {
      it('should update message when TOOL_RESULT arrives for server-side tool', () => {
        // First, create the tool call message
        const toolCall = {
          callId: 'server-smart_search-1',
          toolName: 'smart_search',
          arguments: { query: 'test' },
        };

        gateway.simulateResponse({
          type: 'TOOL_CALL',
          content: '',
          isFinal: false,
          toolCall,
        });

        // Now simulate server sending the result
        gateway.simulateResponse({
          type: 'TOOL_RESULT',
          content: '',
          isFinal: false,
          toolResult: {
            callId: 'server-smart_search-1',
            result: 'Search result here',
          },
        });

        // Find the message
        const message = repository.findByToolCallId('server-smart_search-1');
        expect(message).toBeDefined();
        expect(message?.toolResult?.result).toBe('Search result here');

        // Should publish ToolExecutionCompleted
        const events = eventBus.getEventsOfType('ToolExecutionCompleted');
        expect(events.length).toBe(1);
      });
    });

    describe('Reasoning handling', () => {
      it('should accumulate reasoning content', () => {
        gateway.simulateResponse({
          type: 'REASONING',
          content: '',
          isFinal: false,
          reasoning: {
            thinking: 'Thinking about this problem...',
            isComplete: false,
          },
        });

        const events = eventBus.getEventsOfType('MessageStarted');
        expect(events.length).toBe(1);
        expect((events[0] as any).role).toBe('reasoning');
      });

      it('should complete reasoning when isComplete is true', () => {
        // Start reasoning
        gateway.simulateResponse({
          type: 'REASONING',
          content: '',
          isFinal: false,
          reasoning: {
            thinking: 'Partial thought',
            isComplete: false,
          },
        });

        // Complete reasoning
        gateway.simulateResponse({
          type: 'REASONING',
          content: '',
          isFinal: false,
          reasoning: {
            thinking: 'Complete thought here.',
            isComplete: true,
          },
        });

        const events = eventBus.getEventsOfType('MessageCompleted');
        expect(events.length).toBe(1);
        expect((events[0] as any).message.reasoning?.isComplete).toBe(true);
      });
    });

    describe('Error handling', () => {
      it('should handle ERROR response type', () => {
        gateway.simulateResponse({
          type: 'ERROR',
          content: 'Error occurred', // Need content to not be skipped
          isFinal: false, // isFinal with empty content is skipped
          error: {
            message: 'Something went wrong',
            code: 'INTERNAL_ERROR',
          },
        });

        const errorEvents = eventBus.getEventsOfType('ErrorOccurred');
        expect(errorEvents.length).toBe(1);
        expect((errorEvents[0] as any).error.message).toBe('Something went wrong');

        // Should save error message
        const messages = repository.findAll();
        expect(messages.some(m => m.content.value.includes('Error:'))).toBe(true);
      });

      it('should propagate stream errors', () => {
        gateway.simulateError(new Error('Connection lost'));

        const events = eventBus.getEventsOfType('ErrorOccurred');
        expect(events.length).toBe(1);
        expect((events[0] as any).error.message).toBe('Connection lost');
      });
    });
  });

  describe('sendMessage', () => {
    it('should send message when connected', () => {
      processor.sendMessage('Hello');

      expect(gateway.sentMessages).toContain('Hello');
    });

    it('should create user message in repository', () => {
      processor.sendMessage('User question');

      const messages = repository.findAll();
      expect(messages.some(m => m.role === 'user' && m.content.value === 'User question')).toBe(true);
    });

    it('should publish ProcessingStarted event', () => {
      processor.sendMessage('Hello');

      const events = eventBus.getEventsOfType('ProcessingStarted');
      expect(events.length).toBe(1);
    });

    it('should not send when disconnected', () => {
      gateway.disconnect();
      processor.sendMessage('Hello');

      expect(gateway.sentMessages.length).toBe(0);
    });

    it('should track input tokens', () => {
      processor.sendMessage('Test message');

      const tokens = accumulator.getTokenCounts();
      expect(tokens.input).toBeGreaterThan(0);
    });
  });

  describe('cancel', () => {
    it('should abort accumulating messages', () => {
      // Start accumulating
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Partial',
        isFinal: false,
      });

      processor.cancel();

      expect(processor.getIsProcessing()).toBe(false);
    });

    it('should publish ProcessingStopped event', () => {
      processor.sendMessage('Hello');
      processor.cancel();

      const events = eventBus.getEventsOfType('ProcessingStopped');
      expect(events.length).toBe(1);
    });
  });

  describe('getIsProcessing', () => {
    it('should return false initially', () => {
      expect(processor.getIsProcessing()).toBe(false);
    });

    it('should return true after sending message', () => {
      processor.sendMessage('Hello');

      expect(processor.getIsProcessing()).toBe(true);
    });

    it('should return false after final response', () => {
      processor.sendMessage('Hello');

      gateway.simulateResponse({
        type: 'ANSWER',
        content: 'Response',
        isFinal: true,
      });

      expect(processor.getIsProcessing()).toBe(false);
    });
  });

  describe('dispose', () => {
    it('should unsubscribe from gateway events', () => {
      processor.dispose();

      // Events should no longer be processed
      eventBus.clear();
      gateway.simulateResponse({
        type: 'ANSWER',
        content: 'Test',
        isFinal: true,
      });

      expect(eventBus.publishedEvents.length).toBe(0);
    });
  });

  describe('Multi-agent separator', () => {
    it('inserts separator when agentId changes in multi-agent scenario', () => {
      // First agent: supervisor
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Supervisor response',
        isFinal: false,
        agentId: 'supervisor',
      });

      // Switch to code-agent
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Code agent response',
        isFinal: false,
        agentId: 'code-agent-abc123',
      });

      // Check messages in repository
      const messages = repository.findAll();
      const separators = messages.filter(m => m.content.value.includes('───'));

      expect(separators.length).toBe(1);
      expect(separators[0].content.value).toContain('Code Agent');
      expect(separators[0].content.value).toContain('abc123');
    });

    it('does NOT insert separator if only one agent (hasMultipleAgents=false)', () => {
      // Only supervisor (single agent)
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'First chunk',
        isFinal: false,
        agentId: 'supervisor',
      });

      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Second chunk',
        isFinal: false,
        agentId: 'supervisor',
      });

      // No separators should be inserted
      const messages = repository.findAll();
      const separators = messages.filter(m => m.content.value.includes('───'));

      expect(separators.length).toBe(0);
    });

    it('lastAgentIdInStream resets on sendMessage', () => {
      // First stream: supervisor → code-agent-1
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Supervisor',
        isFinal: false,
        agentId: 'supervisor',
      });

      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Code agent',
        isFinal: false,
        agentId: 'code-agent-1',
      });

      // Complete first stream (final response from code-agent-1, no switch back to supervisor)
      gateway.simulateResponse({
        type: 'ANSWER',
        content: 'Done',
        isFinal: true,
        agentId: 'code-agent-1',
      });

      const messagesAfterFirst = repository.findAll();
      const separatorsAfterFirst = messagesAfterFirst.filter(m => m.content.value.includes('───'));
      // One separator: supervisor → code-agent-1
      expect(separatorsAfterFirst.length).toBe(1);

      // New message (resets lastAgentIdInStream to 'supervisor')
      processor.sendMessage('Second question');

      // New stream: supervisor (no separator, lastAgentIdInStream was reset) → code-agent-2 (separator)
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Supervisor again',
        isFinal: false,
        agentId: 'supervisor',
      });

      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'New code agent',
        isFinal: false,
        agentId: 'code-agent-2',
      });

      const messagesAfterSecond = repository.findAll();
      const separatorsAfterSecond = messagesAfterSecond.filter(m => m.content.value.includes('───'));

      // Should have 2 separators total: code-agent-1 from first stream, code-agent-2 from second
      expect(separatorsAfterSecond.length).toBe(2);
    });

    it('separator includes task description if available', () => {
      // Create agent with lifecycle event first
      const agentStateManager = processor.getAgentStateManager();
      agentStateManager.updateLifecycle('code-agent-test', 'agent_spawned', 'Implement feature X');

      // Supervisor first
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Delegating...',
        isFinal: false,
        agentId: 'supervisor',
      });

      // Switch to code-agent with task description
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Working...',
        isFinal: false,
        agentId: 'code-agent-test',
      });

      const messages = repository.findAll();
      const separators = messages.filter(m => m.content.value.includes('───'));

      expect(separators.length).toBe(1);
      expect(separators[0].content.value).toContain('Code Agent [test]');
      expect(separators[0].content.value).toContain('Implement feature X');
    });

    it('no separator inserted when switching back to same agent', () => {
      // supervisor → code-agent → supervisor → code-agent (only 2 separators)
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Supervisor 1',
        isFinal: false,
        agentId: 'supervisor',
      });

      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Code 1',
        isFinal: false,
        agentId: 'code-agent-1',
      });

      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Supervisor 2',
        isFinal: false,
        agentId: 'supervisor',
      });

      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Code 2',
        isFinal: false,
        agentId: 'code-agent-1',
      });

      const messages = repository.findAll();
      const separators = messages.filter(m => m.content.value.includes('───'));

      // 3 separators: supervisor→code, code→supervisor, supervisor→code
      expect(separators.length).toBe(3);
    });
  });
});
