import { describe, it, expect, beforeEach } from 'bun:test';
import { handleToolCall, executeToolAsync } from '../ToolExecutionHandler.js';
import { StreamProcessorContext } from '../StreamProcessorContext.js';
import { MockStreamGateway, MockToolExecutor, MockEventBus } from '../../__tests__/testHelpers.js';
import { InMemoryMessageRepository } from '../../../../infrastructure/persistence/InMemoryMessageRepository.js';
import { MessageAccumulatorService } from '../../MessageAccumulatorService.js';
import { ToolCallInfo } from '../../../../domain/entities/Message.js';
import { MessageId } from '../../../../domain/value-objects/MessageId.js';

describe('ToolExecutionHandler - ask_user tool', () => {
  let gateway: MockStreamGateway;
  let repository: InMemoryMessageRepository;
  let executor: MockToolExecutor;
  let eventBus: MockEventBus;
  let accumulator: MessageAccumulatorService;
  let context: StreamProcessorContext;

  beforeEach(() => {
    gateway = new MockStreamGateway();
    repository = new InMemoryMessageRepository();
    executor = new MockToolExecutor();
    eventBus = new MockEventBus();
    accumulator = new MessageAccumulatorService();

    // Mock context state
    let currentMessageId: MessageId | null = null;
    let currentReasoningId: MessageId | null = null;
    let isProcessing = false;

    context = {
      streamGateway: gateway,
      messageRepository: repository,
      toolExecutor: executor,
      accumulator,
      eventBus,
      getCurrentMessageId: () => currentMessageId,
      getCurrentReasoningId: () => currentReasoningId,
      getIsProcessing: () => isProcessing,
      setCurrentMessageId: (id) => { currentMessageId = id; },
      setCurrentReasoningId: (id) => { currentReasoningId = id; },
      setIsProcessing: (value) => { isProcessing = value; },
      agentId: 'test-agent',
    };
  });

  describe('ask_user saves question and response in chat history', () => {
    it('should save question as assistant message in repository', async () => {
      const askUserToolCall: ToolCallInfo = {
        callId: 'client-ask_user-1',
        toolName: 'ask_user',
        arguments: { question: 'Do you approve?' },
      };

      executor.executeResult = { result: 'yes' };

      handleToolCall(context, {
        type: 'TOOL_CALL',
        content: '',
        isFinal: false,
        toolCall: askUserToolCall,
      });

      await new Promise(resolve => setTimeout(resolve, 10));

      const allMessages = repository.findAll();
      const questionMsg = allMessages.find(m =>
        m.role === 'assistant' && m.content.value.includes('Do you approve?')
      );
      expect(questionMsg).toBeDefined();
    });

    it('should save user response as user message in repository', async () => {
      const askUserToolCall: ToolCallInfo = {
        callId: 'client-ask_user-2',
        toolName: 'ask_user',
        arguments: { question: 'Proceed?' },
      };

      executor.executeResult = { result: 'approved' };

      handleToolCall(context, {
        type: 'TOOL_CALL',
        content: '',
        isFinal: false,
        toolCall: askUserToolCall,
      });

      await new Promise(resolve => setTimeout(resolve, 10));

      const allMessages = repository.findAll();
      const userMsg = allMessages.find(m =>
        m.role === 'user' && m.content.value.includes('approved')
      );
      expect(userMsg).toBeDefined();
    });

    it('should NOT create tool message for ask_user (no spinner in UI)', async () => {
      const askUserToolCall: ToolCallInfo = {
        callId: 'client-ask_user-3',
        toolName: 'ask_user',
        arguments: { question: 'Test?' },
      };

      executor.executeResult = { result: 'yes' };

      handleToolCall(context, {
        type: 'TOOL_CALL',
        content: '',
        isFinal: false,
        toolCall: askUserToolCall,
      });

      await new Promise(resolve => setTimeout(resolve, 10));

      // No tool messages should exist
      const allMessages = repository.findAll();
      const askUserToolMessages = allMessages.filter(m => m.toolCall?.toolName === 'ask_user');
      expect(askUserToolMessages.length).toBe(0);

      // But question + response messages should exist
      expect(allMessages.length).toBeGreaterThanOrEqual(2);
    });

    it('should publish MessageCompleted for both question and response', async () => {
      const askUserToolCall: ToolCallInfo = {
        callId: 'client-ask_user-4',
        toolName: 'ask_user',
        arguments: { question: 'Approve?' },
      };

      executor.executeResult = { result: 'yes' };

      handleToolCall(context, {
        type: 'TOOL_CALL',
        content: '',
        isFinal: false,
        toolCall: askUserToolCall,
      });

      await new Promise(resolve => setTimeout(resolve, 10));

      const completedEvents = eventBus.getEventsOfType('MessageCompleted');
      // At least 2: question message + user response message
      expect(completedEvents.length).toBeGreaterThanOrEqual(2);
    });
  });

  describe('ask_user execution and server communication', () => {
    it('should execute ask_user tool via toolExecutor', async () => {
      const askUserToolCall: ToolCallInfo = {
        callId: 'client-ask_user-exec',
        toolName: 'ask_user',
        arguments: { question: 'Proceed?' },
      };

      executor.executeResult = { result: 'yes' };

      handleToolCall(context, {
        type: 'TOOL_CALL',
        content: '',
        isFinal: false,
        toolCall: askUserToolCall,
      });

      await new Promise(resolve => setTimeout(resolve, 10));

      expect(executor.executedCalls.length).toBe(1);
      expect(executor.executedCalls[0].toolName).toBe('ask_user');
    });

    it('should send ask_user result back to server', async () => {
      const askUserToolCall: ToolCallInfo = {
        callId: 'client-ask_user-send',
        toolName: 'ask_user',
        arguments: { question: 'Continue?' },
      };

      executor.executeResult = { result: 'approved' };

      handleToolCall(context, {
        type: 'TOOL_CALL',
        content: '',
        isFinal: false,
        toolCall: askUserToolCall,
      });

      await new Promise(resolve => setTimeout(resolve, 10));

      expect(gateway.sentToolResults.length).toBe(1);
      expect(gateway.sentToolResults[0].callId).toBe('client-ask_user-send');
      expect(gateway.sentToolResults[0].result).toBe('approved');
    });

    it('should NOT publish ToolExecutionStarted event (no spinner)', async () => {
      const askUserToolCall: ToolCallInfo = {
        callId: 'client-ask_user-nostart',
        toolName: 'ask_user',
        arguments: { question: 'Test?' },
      };

      handleToolCall(context, {
        type: 'TOOL_CALL',
        content: '',
        isFinal: false,
        toolCall: askUserToolCall,
      });

      await new Promise(resolve => setTimeout(resolve, 10));

      const startedEvents = eventBus.getEventsOfType('ToolExecutionStarted');
      expect(startedEvents.length).toBe(0);
    });

    it('should publish ToolExecutionCompleted after ask_user finishes', async () => {
      const askUserToolCall: ToolCallInfo = {
        callId: 'client-ask_user-complete',
        toolName: 'ask_user',
        arguments: { question: 'Approve?' },
      };

      executor.executeResult = { result: 'yes' };

      handleToolCall(context, {
        type: 'TOOL_CALL',
        content: '',
        isFinal: false,
        toolCall: askUserToolCall,
      });

      await new Promise(resolve => setTimeout(resolve, 10));

      const completedEvents = eventBus.getEventsOfType('ToolExecutionCompleted');
      expect(completedEvents.length).toBe(1);
      expect((completedEvents[0] as any).execution.toolName).toBe('ask_user');
      expect((completedEvents[0] as any).execution.isCompleted).toBe(true);
    });
  });

  describe('ask_user with preamble text', () => {
    it('should save preamble text before ask_user question', async () => {
      // LLM streams text before the ask_user tool call
      const msgId = accumulator.startAccumulating('assistant');
      accumulator.appendChunk(msgId, 'I need to ask you something...');
      context.setCurrentMessageId(msgId);

      const askUserToolCall: ToolCallInfo = {
        callId: 'client-ask_user-preamble',
        toolName: 'ask_user',
        arguments: { question: 'Approve?' },
      };

      executor.executeResult = { result: 'yes' };

      handleToolCall(context, {
        type: 'TOOL_CALL',
        content: '',
        isFinal: false,
        toolCall: askUserToolCall,
      });

      await new Promise(resolve => setTimeout(resolve, 10));

      // Preamble should be saved via completeCurrentMessage
      const allMessages = repository.findAll();
      const preambleMsg = allMessages.find(m =>
        m.role === 'assistant' && m.content.value.includes('I need to ask')
      );
      expect(preambleMsg).toBeDefined();

      // Question should also be saved as separate message
      const questionMsg = allMessages.find(m =>
        m.role === 'assistant' && m.content.value.includes('Approve?')
      );
      expect(questionMsg).toBeDefined();

      // currentMessageId should be cleared
      expect(context.getCurrentMessageId()).toBeNull();
    });
  });

  describe('ask_user error handling', () => {
    it('should send error result back to server if ask_user fails', async () => {
      const askUserToolCall: ToolCallInfo = {
        callId: 'client-ask_user-error',
        toolName: 'ask_user',
        arguments: { question: 'Test?' },
      };

      executor.executeResult = {
        result: '',
        error: new Error('User cancelled')
      };

      handleToolCall(context, {
        type: 'TOOL_CALL',
        content: '',
        isFinal: false,
        toolCall: askUserToolCall,
      });

      await new Promise(resolve => setTimeout(resolve, 10));

      expect(gateway.sentToolResults.length).toBe(1);
      expect(gateway.sentToolResults[0].callId).toBe('client-ask_user-error');
      expect(gateway.sentToolResults[0].error).toBeDefined();
    });

    it('should NOT save user response when ask_user returns error', async () => {
      const askUserToolCall: ToolCallInfo = {
        callId: 'client-ask_user-err-noresp',
        toolName: 'ask_user',
        arguments: { question: 'Test?' },
      };

      executor.executeResult = {
        result: '',
        error: new Error('User cancelled')
      };

      handleToolCall(context, {
        type: 'TOOL_CALL',
        content: '',
        isFinal: false,
        toolCall: askUserToolCall,
      });

      await new Promise(resolve => setTimeout(resolve, 10));

      // Neither question nor user response saved on error
      const allMessages = repository.findAll();
      const userMessages = allMessages.filter(m => m.role === 'user');
      expect(userMessages.length).toBe(0);
    });

    it('should publish ToolExecutionCompleted with error status', async () => {
      const askUserToolCall: ToolCallInfo = {
        callId: 'client-ask_user-fail',
        toolName: 'ask_user',
        arguments: { question: 'Test?' },
      };

      executor.executeResult = {
        result: '',
        error: new Error('Execution failed')
      };

      handleToolCall(context, {
        type: 'TOOL_CALL',
        content: '',
        isFinal: false,
        toolCall: askUserToolCall,
      });

      await new Promise(resolve => setTimeout(resolve, 10));

      const completedEvents = eventBus.getEventsOfType('ToolExecutionCompleted');
      expect(completedEvents.length).toBe(1);
      expect((completedEvents[0] as any).execution.toolName).toBe('ask_user');
      expect((completedEvents[0] as any).execution.isFailed).toBe(true);
    });
  });

  describe('Regular tool call - unchanged behavior', () => {
    it('should create tool message in repository for regular tools (e.g., read_file)', async () => {
      const readFileToolCall: ToolCallInfo = {
        callId: 'client-read_file-1',
        toolName: 'read_file',
        arguments: { file_path: '/test.txt' },
      };

      handleToolCall(context, {
        type: 'TOOL_CALL',
        content: '',
        isFinal: false,
        toolCall: readFileToolCall,
      });

      await new Promise(resolve => setTimeout(resolve, 10));

      const allMessages = repository.findAll();
      const toolMessages = allMessages.filter(m => m.toolCall?.toolName === 'read_file');
      expect(toolMessages.length).toBe(1);

      const messageByCallId = repository.findByToolCallId('client-read_file-1');
      expect(messageByCallId).toBeDefined();
      expect(messageByCallId?.toolCall?.toolName).toBe('read_file');
    });

    it('should publish ToolExecutionStarted event for regular tools', async () => {
      const readFileToolCall: ToolCallInfo = {
        callId: 'client-read_file-2',
        toolName: 'read_file',
        arguments: { file_path: '/test.txt' },
      };

      handleToolCall(context, {
        type: 'TOOL_CALL',
        content: '',
        isFinal: false,
        toolCall: readFileToolCall,
      });

      const startedEvents = eventBus.getEventsOfType('ToolExecutionStarted');
      expect(startedEvents.length).toBe(1);
      expect((startedEvents[0] as any).execution.toolName).toBe('read_file');
    });

    it('should execute regular tool and send result back', async () => {
      const searchToolCall: ToolCallInfo = {
        callId: 'client-search_code-1',
        toolName: 'search_code',
        arguments: { query: 'test' },
      };

      executor.executeResult = { result: 'search results here' };

      handleToolCall(context, {
        type: 'TOOL_CALL',
        content: '',
        isFinal: false,
        toolCall: searchToolCall,
      });

      await new Promise(resolve => setTimeout(resolve, 10));

      expect(executor.executedCalls.length).toBe(1);
      expect(executor.executedCalls[0].toolName).toBe('search_code');

      expect(gateway.sentToolResults.length).toBe(1);
      expect(gateway.sentToolResults[0].callId).toBe('client-search_code-1');
      expect(gateway.sentToolResults[0].result).toBe('search results here');

      const messageByCallId = repository.findByToolCallId('client-search_code-1');
      expect(messageByCallId).toBeDefined();
      expect(messageByCallId?.toolResult).toBeDefined();
      expect(messageByCallId?.toolResult?.result).toBe('search results here');
      expect(messageByCallId?.isComplete).toBe(true);
    });
  });
});
