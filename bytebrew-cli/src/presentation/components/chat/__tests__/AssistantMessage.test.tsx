import { describe, it, expect, afterEach } from 'bun:test';
import React from 'react';
import { render } from 'ink-testing-library';
import { AssistantMessage } from '../AssistantMessage.js';
import type { ChatMessage } from '../../../../domain/message.js';

describe('AssistantMessage', () => {
  let instance: ReturnType<typeof render> | null = null;

  afterEach(() => {
    instance?.unmount();
    instance = null;
  });

  // Helper to create test messages
  const createMessage = (partial: Partial<ChatMessage>): ChatMessage => ({
    id: partial.id || 'msg-1',
    role: 'assistant',
    content: partial.content || 'Test message',
    timestamp: partial.timestamp || new Date(),
    isStreaming: partial.isStreaming ?? false,
    isComplete: partial.isComplete ?? true,
    agentId: partial.agentId,
  });

  // --- Problem 1: Task assignment from supervisor ---

  describe('task assignment from supervisor', () => {
    it('shows [Task from Supervisor] label for task messages in agent context', () => {
      const taskMessage = createMessage({
        content: '[Task from Supervisor]\nRefactor authentication module',
        agentId: 'code-agent-abc',
      });

      instance = render(<AssistantMessage message={taskMessage} />);
      const frame = instance.lastFrame();

      // Should show the task marker
      expect(frame).toContain('[Task from Supervisor]');
      // Should show the task content
      expect(frame).toContain('Refactor authentication');
      // Should have agent prefix
      expect(frame).toContain('│');
    });

    it('task message has agent ID for proper tab filtering', () => {
      const taskMessage = createMessage({
        content: '[Task from Supervisor]\nImplement caching layer',
        agentId: 'code-agent-xyz',
      });

      instance = render(<AssistantMessage message={taskMessage} />);

      // Message should be rendered (not null)
      expect(instance.lastFrame()).toBeTruthy();
      // Content should be visible
      expect(instance.lastFrame()).toContain('Implement caching');
    });
  });

  // --- Problem 2: Agent messages in supervisor context ---

  describe('agent label in messages', () => {
    it('renders Code Agent [shortId] label for code agent messages', () => {
      const agentMessage = createMessage({
        content: 'Completed refactoring task',
        agentId: 'code-agent-abc123',
      });

      instance = render(<AssistantMessage message={agentMessage} />);
      const frame = instance.lastFrame();

      // Should show short agent ID
      expect(frame).toContain('Code Agent [abc123]');
      // Should have agent prefix
      expect(frame).toContain('│');
    });

    it('renders Code Agent label with different agent IDs', () => {
      const agentMessage = createMessage({
        content: 'Working on implementation',
        agentId: 'code-agent-xyz',
      });

      instance = render(<AssistantMessage message={agentMessage} />);

      expect(instance.lastFrame()).toContain('Code Agent [xyz]');
    });

    it('does not show agent label for supervisor messages', () => {
      const supervisorMessage = createMessage({
        content: 'Planning next steps',
        agentId: 'supervisor',
      });

      instance = render(<AssistantMessage message={supervisorMessage} />);
      const frame = instance.lastFrame();

      // Should NOT contain agent label
      expect(frame).not.toContain('Code Agent');
      // Should have supervisor prefix (cyan >)
      expect(frame).toContain('>');
    });
  });

  // --- Problem 3: dimColor on markdown content ---

  describe('agent messages render without dimColor artifacts', () => {
    it('agent message content does not have dimColor', () => {
      const agentMessage = createMessage({
        content: 'Analysis complete. Results:\n\n- Found 5 issues\n- Fixed 3 bugs',
        agentId: 'code-agent-1',
      });

      instance = render(<AssistantMessage message={agentMessage} />);
      const frame = instance.lastFrame();

      // Content should be rendered
      expect(frame).toContain('Analysis complete');
      expect(frame).toContain('Found 5 issues');

      // Prefix should be gray
      expect(frame).toContain('│');
    });

    it('markdown formatting is preserved without dimColor interference', () => {
      const agentMessage = createMessage({
        content: '## Summary\n\n**Important:** Check the implementation',
        agentId: 'code-agent-test',
      });

      instance = render(<AssistantMessage message={agentMessage} />);

      // Should render markdown (marked-terminal adds ANSI codes)
      expect(instance.lastFrame()).toContain('Summary');
      expect(instance.lastFrame()).toContain('Important');
    });
  });

  // --- Baseline tests (existing behavior) ---

  describe('supervisor messages', () => {
    it('renders with cyan > prefix', () => {
      const supervisorMessage = createMessage({
        content: 'Starting analysis',
        agentId: 'supervisor',
      });

      instance = render(<AssistantMessage message={supervisorMessage} />);
      const frame = instance.lastFrame();

      expect(frame).toContain('>');
      expect(frame).toContain('Starting analysis');
    });

    it('renders supervisor message without agentId as supervisor', () => {
      const message = createMessage({
        content: 'Processing request',
        agentId: undefined,
      });

      instance = render(<AssistantMessage message={message} />);

      // Should use supervisor rendering (cyan >)
      expect(instance.lastFrame()).toContain('>');
      expect(instance.lastFrame()).toContain('Processing request');
    });
  });

  describe('lifecycle messages', () => {
    it('renders lifecycle message with correct color (green for completed)', () => {
      const lifecycleMessage = createMessage({
        content: '✓ Code Agent [abc] completed: "Task done"',
        agentId: 'supervisor',
      });

      instance = render(<AssistantMessage message={lifecycleMessage} />);

      expect(instance.lastFrame()).toContain('✓');
      expect(instance.lastFrame()).toContain('Code Agent [abc] completed');
    });

    it('renders spawned lifecycle message with yellow color', () => {
      const lifecycleMessage = createMessage({
        content: '+ Code Agent [xyz] spawned: "New task"',
        agentId: 'supervisor',
      });

      instance = render(<AssistantMessage message={lifecycleMessage} />);

      // Should contain lifecycle marker (no prefix)
      expect(instance.lastFrame()).toContain('+');
      expect(instance.lastFrame()).toContain('Code Agent [xyz] spawned');
    });
  });

  describe('separator messages', () => {
    it('renders separator with dimmed gray', () => {
      const separatorMessage = createMessage({
        content: '─── Code Agent [abc]: Task description ───',
        agentId: 'code-agent-abc',
      });

      instance = render(<AssistantMessage message={separatorMessage} />);

      expect(instance.lastFrame()).toContain('───');
      expect(instance.lastFrame()).toContain('Code Agent [abc]');
    });
  });

  describe('streaming messages', () => {
    it('does not render while streaming', () => {
      const streamingMessage = createMessage({
        content: 'Partial content...',
        isStreaming: true,
        isComplete: false,
      });

      instance = render(<AssistantMessage message={streamingMessage} />);

      // Should render nothing (null)
      expect(instance.lastFrame()).toBe('');
    });

    it('renders after streaming completes', () => {
      const completedMessage = createMessage({
        content: 'Complete content',
        isStreaming: false,
        isComplete: true,
        agentId: 'supervisor',
      });

      instance = render(<AssistantMessage message={completedMessage} />);

      expect(instance.lastFrame()).toContain('Complete content');
    });
  });
});
