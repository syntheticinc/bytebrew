import { describe, it, expect } from 'bun:test';
import { Message, ToolCallInfo } from '../Message.js';
import { MessageId } from '../../value-objects/MessageId.js';

describe('Message', () => {
  describe('factory methods', () => {
    it('createUser creates user message with complete state', () => {
      const msg = Message.createUser('Hello');
      expect(msg.role).toBe('user');
      expect(msg.content.value).toBe('Hello');
      expect(msg.isComplete).toBe(true);
      expect(msg.agentId).toBeUndefined();
    });

    it('createAssistant creates pending assistant message', () => {
      const msg = Message.createAssistant();
      expect(msg.role).toBe('assistant');
      expect(msg.content.isEmpty).toBe(true);
      expect(msg.isComplete).toBe(false);
    });

    it('createAssistantWithContent creates complete message with content', () => {
      const msg = Message.createAssistantWithContent('Response text');
      expect(msg.role).toBe('assistant');
      expect(msg.content.value).toBe('Response text');
      expect(msg.isComplete).toBe(true);
    });

    it('createAssistantWithContent with agentId', () => {
      // VALIDATES FIX
      const msg = Message.createAssistantWithContent('Answer', 'supervisor');
      expect(msg.agentId).toBe('supervisor');
      expect(msg.role).toBe('assistant');
      expect(msg.isComplete).toBe(true);
    });

    it('createAssistantWithContent without agentId → undefined', () => {
      const msg = Message.createAssistantWithContent('Answer');
      expect(msg.agentId).toBeUndefined();
    });

    it('createReasoning creates message with reasoning info', () => {
      const msg = Message.createReasoning('Thinking...', false);
      expect(msg.role).toBe('assistant');
      expect(msg.reasoning).toBeDefined();
      expect(msg.reasoning!.thinking).toBe('Thinking...');
      expect(msg.reasoning!.isComplete).toBe(false);
    });

    it('createReasoning with agentId', () => {
      // VALIDATES FIX
      const msg = Message.createReasoning('Analyzing', true, 'planner');
      expect(msg.agentId).toBe('planner');
      expect(msg.reasoning).toBeDefined();
    });

    it('createToolCall creates tool message', () => {
      const toolCall: ToolCallInfo = {
        callId: 'call-123',
        toolName: 'read_file',
        arguments: { path: '/test.txt' },
      };

      const msg = Message.createToolCall(toolCall);
      expect(msg.role).toBe('tool');
      expect(msg.toolCall).toBe(toolCall);
      expect(msg.isComplete).toBe(false);
    });

    it('createToolCall with agentId', () => {
      // VALIDATES FIX
      const toolCall: ToolCallInfo = {
        callId: 'call-456',
        toolName: 'search_code',
        arguments: { query: 'test' },
      };

      const msg = Message.createToolCall(toolCall, 'code-agent');
      expect(msg.agentId).toBe('code-agent');
      expect(msg.toolCall).toBe(toolCall);
    });

    it('fromSnapshot restores all fields including agentId', () => {
      const id = MessageId.create();
      const snapshot = {
        id: id.value,
        role: 'assistant' as const,
        content: 'Restored content',
        timestamp: new Date(),
        isComplete: true,
        agentId: 'supervisor',
        reasoning: {
          thinking: 'Restored thinking',
          isComplete: true,
        },
      };

      const msg = Message.fromSnapshot(snapshot);
      expect(msg.agentId).toBe('supervisor');
      expect(msg.content.value).toBe('Restored content');
      expect(msg.reasoning).toBeDefined();
      expect(msg.reasoning!.thinking).toBe('Restored thinking');
    });
  });

  describe('behavior methods', () => {
    it('appendContent returns new message with appended content', () => {
      const msg1 = Message.createAssistant();
      const msg2 = msg1.appendContent('Hello');
      const msg3 = msg2.appendContent(' world');

      expect(msg1.content.value).toBe('');
      expect(msg2.content.value).toBe('Hello');
      expect(msg3.content.value).toBe('Hello world');
      expect(msg1.id).toEqual(msg3.id); // Same entity
    });

    it('withContent replaces content', () => {
      const msg1 = Message.createAssistantWithContent('Old');
      const msg2 = msg1.withContent('New');

      expect(msg1.content.value).toBe('Old');
      expect(msg2.content.value).toBe('New');
    });

    it('markComplete transitions to complete state', () => {
      const msg1 = Message.createAssistant();
      expect(msg1.isComplete).toBe(false);

      const msg2 = msg1.markComplete();
      expect(msg2.isComplete).toBe(true);
    });

    it('markAborted transitions to aborted state', () => {
      const msg1 = Message.createAssistant();
      const msg2 = msg1.markAborted();

      expect(msg2.streamingState.isAborted).toBe(true);
    });

    it('withToolResult adds result to tool message', () => {
      const toolCall: ToolCallInfo = {
        callId: 'call-789',
        toolName: 'write_file',
        arguments: { path: '/test.txt', content: 'data' },
      };

      const msg1 = Message.createToolCall(toolCall);
      expect(msg1.toolResult).toBeUndefined();

      const msg2 = msg1.withToolResult('Success');
      expect(msg2.toolResult).toBeDefined();
      expect(msg2.toolResult!.result).toBe('Success');
      expect(msg2.isComplete).toBe(true);
    });

    it('withToolResult throws on non-tool message', () => {
      const msg = Message.createAssistantWithContent('Not a tool');
      expect(() => msg.withToolResult('Result')).toThrow();
    });

    it('updateReasoning updates reasoning info', () => {
      const msg1 = Message.createReasoning('Initial', false);
      const msg2 = msg1.updateReasoning('Updated', true);

      expect(msg1.reasoning!.thinking).toBe('Initial');
      expect(msg1.reasoning!.isComplete).toBe(false);

      expect(msg2.reasoning!.thinking).toBe('Updated');
      expect(msg2.reasoning!.isComplete).toBe(true);
    });
  });

  describe('serialization', () => {
    it('toSnapshot includes agentId', () => {
      const msg = Message.createAssistantWithContent('Test', 'code-agent');
      const snapshot = msg.toSnapshot();

      expect(snapshot.agentId).toBe('code-agent');
      expect(snapshot.content).toBe('Test');
      expect(snapshot.role).toBe('assistant');
    });

    it('equals compares by id', () => {
      const msg1 = Message.createUser('Hello');
      const msg2 = msg1.appendContent(' world');
      const msg3 = Message.createUser('Different');

      expect(msg1.equals(msg2)).toBe(true);
      expect(msg1.equals(msg3)).toBe(false);
    });

    it('getter methods work correctly', () => {
      const toolCall: ToolCallInfo = {
        callId: 'call-xyz',
        toolName: 'test_tool',
        arguments: {},
      };

      const assistantMsg = Message.createAssistantWithContent('Hi');
      const toolMsg = Message.createToolCall(toolCall);
      const userMsg = Message.createUser('Query');

      expect(assistantMsg.isAssistantMessage).toBe(true);
      expect(assistantMsg.isToolMessage).toBe(false);
      expect(assistantMsg.isUserMessage).toBe(false);

      expect(toolMsg.isToolMessage).toBe(true);
      expect(toolMsg.isAssistantMessage).toBe(false);

      expect(userMsg.isUserMessage).toBe(true);
      expect(userMsg.isAssistantMessage).toBe(false);
    });
  });
});
