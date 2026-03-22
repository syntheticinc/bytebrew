import { describe, it, expect } from 'bun:test';
import { filterMessagesForView, ViewMode } from '../MessageViewFilter.js';
import { MessageViewModel } from '../MessageViewMapper.js';

// Helper function to create MessageViewModel with defaults
function makeMsg(
  overrides: Partial<MessageViewModel> & { id: string }
): MessageViewModel {
  return {
    role: 'assistant',
    content: '',
    timestamp: new Date(),
    isStreaming: false,
    isComplete: true,
    ...overrides,
  };
}

describe('MessageViewFilter', () => {
  describe('Supervisor view', () => {
    const supervisorView: ViewMode = { type: 'supervisor' };

    it('показывает user messages', () => {
      const messages: MessageViewModel[] = [
        makeMsg({ id: 'm1', role: 'user', content: 'Hello', agentId: undefined }),
        makeMsg({ id: 'm2', role: 'user', content: 'Fix bug', agentId: undefined }),
      ];

      const result = filterMessagesForView(messages, supervisorView);

      expect(result).toHaveLength(2);
      expect(result[0].id).toBe('m1');
      expect(result[1].id).toBe('m2');
    });

    it('показывает supervisor text messages', () => {
      const messages: MessageViewModel[] = [
        makeMsg({ id: 'm1', role: 'assistant', content: 'I will help you', agentId: 'supervisor' }),
        makeMsg({ id: 'm2', role: 'assistant', content: 'Let me analyze', agentId: 'supervisor' }),
      ];

      const result = filterMessagesForView(messages, supervisorView);

      expect(result).toHaveLength(2);
      expect(result[0].id).toBe('m1');
      expect(result[1].id).toBe('m2');
    });

    it('показывает supervisor tool calls', () => {
      const messages: MessageViewModel[] = [
        makeMsg({
          id: 'm1',
          role: 'assistant',
          content: 'Calling tool',
          agentId: 'supervisor',
          toolCall: { callId: 'c1', toolName: 'spawn_code_agent', arguments: {} },
        }),
      ];

      const result = filterMessagesForView(messages, supervisorView);

      expect(result).toHaveLength(1);
      expect(result[0].id).toBe('m1');
      expect(result[0].toolCall?.toolName).toBe('spawn_code_agent');
    });

    it('показывает lifecycle events', () => {
      const messages: MessageViewModel[] = [
        makeMsg({ id: 'm1', role: 'system', content: '⊕ Code Agent [abc] spawned: Fix imports' }),
        makeMsg({ id: 'm2', role: 'system', content: '✓ Code Agent [abc] completed' }),
        makeMsg({ id: 'm3', role: 'system', content: '✗ Code Agent [abc] failed' }),
        makeMsg({ id: 'm4', role: 'system', content: '↻ Code Agent [abc] retrying' }),
      ];

      const result = filterMessagesForView(messages, supervisorView);

      expect(result).toHaveLength(4);
      expect(result[0].id).toBe('m1');
      expect(result[1].id).toBe('m2');
      expect(result[2].id).toBe('m3');
      expect(result[3].id).toBe('m4');
    });

    it('скрывает separator messages', () => {
      const messages: MessageViewModel[] = [
        makeMsg({ id: 'm1', role: 'system', content: '─── Code Agent [abc]: Fix imports ───' }),
        makeMsg({ id: 'm2', role: 'system', content: '─────────────────────────' }),
      ];

      const result = filterMessagesForView(messages, supervisorView);

      // Separators hidden in supervisor view — lifecycle events provide agent boundaries
      expect(result).toHaveLength(0);
    });

    it('скрывает agent TEXT messages', () => {
      // Agent text is hidden in supervisor view to prevent duplication.
      // Supervisor sees agent results via lifecycle events and spawn_code_agent tool results.
      const messages: MessageViewModel[] = [
        makeMsg({
          id: 'm1',
          role: 'assistant',
          content: 'I fixed the imports',
          agentId: 'code-agent-abc',
        }),
        makeMsg({
          id: 'm2',
          role: 'assistant',
          content: 'Done',
          agentId: 'code-agent-abc',
        }),
      ];

      const result = filterMessagesForView(messages, supervisorView);

      expect(result).toHaveLength(0);
    });

    it('скрывает agent TOOL messages', () => {
      const messages: MessageViewModel[] = [
        makeMsg({
          id: 'm1',
          role: 'assistant',
          content: 'Calling tool',
          agentId: 'code-agent-abc',
          toolCall: { callId: 'c1', toolName: 'read_file', arguments: {} },
        }),
        makeMsg({
          id: 'm2',
          role: 'assistant',
          content: 'Tool call',
          agentId: 'code-agent-abc',
          toolCall: { callId: 'c2', toolName: 'edit_file', arguments: {} },
        }),
      ];

      const result = filterMessagesForView(messages, supervisorView);

      expect(result).toHaveLength(0);
    });

    it('скрывает agent tool RESULT messages', () => {
      const messages: MessageViewModel[] = [
        makeMsg({
          id: 'm1',
          role: 'tool',
          content: 'File content',
          agentId: 'code-agent-abc',
          toolResult: { callId: 'c1', toolName: 'read_file', result: 'file content' },
        }),
      ];

      const result = filterMessagesForView(messages, supervisorView);

      expect(result).toHaveLength(0);
    });

    it('скрывает agent reasoning messages', () => {
      const messages: MessageViewModel[] = [
        makeMsg({
          id: 'm1',
          role: 'assistant',
          content: 'Thinking...',
          agentId: 'code-agent-abc',
          reasoning: { thinking: 'Analyzing the code...', isComplete: true },
        }),
      ];

      const result = filterMessagesForView(messages, supervisorView);

      expect(result).toHaveLength(0);
    });

    it('комплексный тест: mix всех типов', () => {
      const messages: MessageViewModel[] = [
        makeMsg({ id: 'm1', role: 'user', content: 'Fix bug', agentId: undefined }), // показать
        makeMsg({ id: 'm2', role: 'assistant', content: 'I will spawn agent', agentId: 'supervisor' }), // показать
        makeMsg({
          id: 'm3',
          role: 'assistant',
          content: 'Spawning',
          agentId: 'supervisor',
          toolCall: { callId: 'c1', toolName: 'spawn_code_agent', arguments: {} },
        }), // показать
        makeMsg({ id: 'm4', role: 'system', content: '⊕ Code Agent [abc] spawned' }), // показать
        makeMsg({
          id: 'm5',
          role: 'assistant',
          content: 'Reading file',
          agentId: 'code-agent-abc',
          toolCall: { callId: 'c2', toolName: 'read_file', arguments: {} },
        }), // скрыть
        makeMsg({
          id: 'm6',
          role: 'tool',
          content: 'File content',
          agentId: 'code-agent-abc',
          toolResult: { callId: 'c2', toolName: 'read_file', result: 'content' },
        }), // скрыть
        makeMsg({
          id: 'm7',
          role: 'assistant',
          content: 'Thinking',
          agentId: 'code-agent-abc',
          reasoning: { thinking: 'Analyzing...', isComplete: true },
        }), // скрыть
        makeMsg({ id: 'm8', role: 'assistant', content: 'I fixed it', agentId: 'code-agent-abc' }), // скрыть (agent text)
        makeMsg({ id: 'm9', role: 'system', content: '✓ Code Agent [abc] completed' }), // показать
      ];

      const result = filterMessagesForView(messages, supervisorView);

      expect(result).toHaveLength(5);
      expect(result.map((m) => m.id)).toEqual(['m1', 'm2', 'm3', 'm4', 'm9']);
    });
  });

  describe('Agent view', () => {
    const agentView: ViewMode = { type: 'agent', agentId: 'code-agent-abc' };

    it('показывает все messages этого агента', () => {
      const messages: MessageViewModel[] = [
        makeMsg({ id: 'm1', role: 'assistant', content: 'Text', agentId: 'code-agent-abc' }), // text
        makeMsg({
          id: 'm2',
          role: 'assistant',
          content: 'Tool call',
          agentId: 'code-agent-abc',
          toolCall: { callId: 'c1', toolName: 'read_file', arguments: {} },
        }), // tool call
        makeMsg({
          id: 'm3',
          role: 'tool',
          content: 'Result',
          agentId: 'code-agent-abc',
          toolResult: { callId: 'c1', toolName: 'read_file', result: 'content' },
        }), // tool result
        makeMsg({
          id: 'm4',
          role: 'assistant',
          content: 'Thinking',
          agentId: 'code-agent-abc',
          reasoning: { thinking: 'Analyzing...', isComplete: true },
        }), // reasoning
      ];

      const result = filterMessagesForView(messages, agentView);

      expect(result).toHaveLength(4);
      expect(result.map((m) => m.id)).toEqual(['m1', 'm2', 'm3', 'm4']);
    });

    it('скрывает user messages (agent view = isolated workspace)', () => {
      // User messages belong to supervisor conversation.
      // Agent receives its task via [Task] lifecycle message.
      const messages: MessageViewModel[] = [
        makeMsg({ id: 'm1', role: 'user', content: 'Fix bug', agentId: undefined }),
        makeMsg({ id: 'm2', role: 'user', content: 'Add tests', agentId: undefined }),
      ];

      const result = filterMessagesForView(messages, agentView);

      expect(result).toHaveLength(0);
    });

    it('скрывает separators (agent view = isolated workspace)', () => {
      // Agent view is already scoped to one agent; separators are noise.
      const messages: MessageViewModel[] = [
        makeMsg({ id: 'm1', role: 'system', content: '─── Code Agent [abc]: Fix imports ───' }),
        makeMsg({ id: 'm2', role: 'system', content: '─── Code Agent [xyz]: Other task ───' }),
      ];

      const result = filterMessagesForView(messages, agentView);

      expect(result).toHaveLength(0);
    });

    it('показывает lifecycle для этого агента', () => {
      const messages: MessageViewModel[] = [
        makeMsg({ id: 'm1', role: 'system', content: '⊕ Code Agent [abc] spawned' }),
        makeMsg({ id: 'm2', role: 'system', content: '✓ Code Agent [abc] completed' }),
        makeMsg({ id: 'm3', role: 'system', content: '✗ Code Agent [xyz] failed' }),
      ];

      const result = filterMessagesForView(messages, agentView);

      // Только lifecycle содержащие 'abc'
      expect(result).toHaveLength(2);
      expect(result.map((m) => m.id)).toEqual(['m1', 'm2']);
    });

    it('скрывает supervisor messages', () => {
      const messages: MessageViewModel[] = [
        makeMsg({ id: 'm1', role: 'assistant', content: 'I will help', agentId: 'supervisor' }),
        makeMsg({
          id: 'm2',
          role: 'assistant',
          content: 'Tool call',
          agentId: 'supervisor',
          toolCall: { callId: 'c1', toolName: 'spawn_code_agent', arguments: {} },
        }),
      ];

      const result = filterMessagesForView(messages, agentView);

      expect(result).toHaveLength(0);
    });

    it('скрывает messages другого агента', () => {
      const messages: MessageViewModel[] = [
        makeMsg({ id: 'm1', role: 'assistant', content: 'Text', agentId: 'code-agent-xyz' }),
        makeMsg({
          id: 'm2',
          role: 'assistant',
          content: 'Tool',
          agentId: 'code-agent-xyz',
          toolCall: { callId: 'c1', toolName: 'read_file', arguments: {} },
        }),
      ];

      const result = filterMessagesForView(messages, agentView);

      expect(result).toHaveLength(0);
    });

    it('скрывает separator для другого агента', () => {
      const messages: MessageViewModel[] = [
        makeMsg({ id: 'm1', role: 'system', content: '─── Code Agent [xyz]: Other task ───' }),
      ];

      const result = filterMessagesForView(messages, agentView);

      expect(result).toHaveLength(0);
    });

    it('скрывает lifecycle для другого агента', () => {
      const messages: MessageViewModel[] = [
        makeMsg({ id: 'm1', role: 'system', content: '⊕ Code Agent [xyz] spawned' }),
        makeMsg({ id: 'm2', role: 'system', content: '✗ Code Agent [xyz] failed' }),
      ];

      const result = filterMessagesForView(messages, agentView);

      expect(result).toHaveLength(0);
    });
  });

  describe('Edge cases', () => {
    it('пустой массив messages → пустой результат', () => {
      const messages: MessageViewModel[] = [];
      const supervisorView: ViewMode = { type: 'supervisor' };

      const result = filterMessagesForView(messages, supervisorView);

      expect(result).toHaveLength(0);
    });

    it('один supervisor — без изменений', () => {
      const messages: MessageViewModel[] = [
        makeMsg({ id: 'm1', role: 'assistant', content: 'Hello', agentId: 'supervisor' }),
      ];
      const supervisorView: ViewMode = { type: 'supervisor' };

      const result = filterMessagesForView(messages, supervisorView);

      expect(result).toHaveLength(1);
      expect(result[0].id).toBe('m1');
    });

    it('нет agents — supervisor view возвращает всё кроме agent tools', () => {
      const messages: MessageViewModel[] = [
        makeMsg({ id: 'm1', role: 'user', content: 'Hello' }),
        makeMsg({ id: 'm2', role: 'assistant', content: 'Hi', agentId: 'supervisor' }),
        makeMsg({ id: 'm3', role: 'system', content: '⊕ System message' }),
      ];
      const supervisorView: ViewMode = { type: 'supervisor' };

      const result = filterMessagesForView(messages, supervisorView);

      expect(result).toHaveLength(3);
    });
  });
});
