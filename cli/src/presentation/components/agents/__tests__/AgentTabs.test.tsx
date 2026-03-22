import React from 'react';
import { describe, it, expect } from 'bun:test';
import { render } from 'ink-testing-library';
import { AgentTabs } from '../AgentTabs.js';
import { AgentState } from '../../../../infrastructure/state/AgentStateManager.js';

// Helper function to create AgentState with defaults
function makeAgent(
  overrides: Partial<AgentState> & { agentId: string }
): AgentState {
  return {
    role: 'code',
    name: '',
    status: 'idle',
    currentMessageId: null,
    currentReasoningId: null,
    taskDescription: '',
    completionStatus: 'idle',
    lastMessageAt: new Date(),
    ...overrides,
  };
}

describe('AgentTabs', () => {
  describe('visibility', () => {
    it('returns null with empty agents array', () => {
      const { lastFrame } = render(
        <AgentTabs agents={[]} activeAgentId="any" />
      );
      expect(lastFrame()).toBe('');
    });

    it('returns null with single agent', () => {
      const agents = [makeAgent({ agentId: 'supervisor', role: 'supervisor' })];
      const { lastFrame } = render(
        <AgentTabs agents={agents} activeAgentId="supervisor" />
      );
      expect(lastFrame()).toBe('');
    });

    it('renders with 2+ agents', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code' }),
      ];
      const { lastFrame } = render(
        <AgentTabs agents={agents} activeAgentId="supervisor" />
      );
      expect(lastFrame()).not.toBe('');
      expect(lastFrame()).toContain('Supervisor');
    });
  });

  describe('tab display', () => {
    it('shows Supervisor label for supervisor agent', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code' }),
      ];
      const { lastFrame } = render(
        <AgentTabs agents={agents} activeAgentId="supervisor" />
      );
      expect(lastFrame()).toContain('Supervisor');
    });

    it('shows short ID for code agent without task description', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc123', role: 'code' }),
      ];
      const { lastFrame } = render(
        <AgentTabs agents={agents} activeAgentId="supervisor" />
      );
      expect(lastFrame()).toContain('abc123');
    });

    it('shows task description next to short ID', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code', taskDescription: 'Fix imports' }),
      ];
      const { lastFrame } = render(
        <AgentTabs agents={agents} activeAgentId="supervisor" />
      );
      expect(lastFrame()).toContain('abc: Fix imports');
    });

    it('truncates long task description to 25 chars', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({
          agentId: 'code-agent-abc',
          role: 'code',
          taskDescription: 'This is a very long task description that should be truncated',
        }),
      ];
      const { lastFrame } = render(
        <AgentTabs agents={agents} activeAgentId="supervisor" />
      );
      const output = lastFrame() || '';
      // Should contain truncated text with ellipsis
      expect(output).toContain('abc: This is a very long task …');
      // Should NOT contain full description
      expect(output).not.toContain('should be truncated');
    });

    it('shows all agents in order', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-aaa', role: 'code', taskDescription: 'First task' }),
        makeAgent({ agentId: 'code-agent-bbb', role: 'code', taskDescription: 'Second task' }),
      ];
      const { lastFrame } = render(
        <AgentTabs agents={agents} activeAgentId="supervisor" />
      );
      const output = lastFrame() || '';
      const supPos = output.indexOf('Supervisor');
      const firstPos = output.indexOf('First task');
      const secondPos = output.indexOf('Second task');
      expect(supPos).toBeLessThan(firstPos);
      expect(firstPos).toBeLessThan(secondPos);
    });
  });

  describe('status icons', () => {
    it('shows ✓ for completed agent', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code', completionStatus: 'completed' }),
      ];
      const { lastFrame } = render(
        <AgentTabs agents={agents} activeAgentId="supervisor" />
      );
      expect(lastFrame()).toContain('✓');
    });

    it('shows ✗ for failed agent', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code', completionStatus: 'failed' }),
      ];
      const { lastFrame } = render(
        <AgentTabs agents={agents} activeAgentId="supervisor" />
      );
      expect(lastFrame()).toContain('✗');
    });

    it('shows ● for running agent', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code', completionStatus: 'running' }),
      ];
      const { lastFrame } = render(
        <AgentTabs agents={agents} activeAgentId="supervisor" />
      );
      expect(lastFrame()).toContain('●');
    });

    it('shows ● for processing agent', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code', status: 'processing' }),
      ];
      const { lastFrame } = render(
        <AgentTabs agents={agents} activeAgentId="supervisor" />
      );
      expect(lastFrame()).toContain('●');
    });

    it('no icon for idle agent', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor', status: 'idle', completionStatus: 'idle' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code', status: 'idle', completionStatus: 'idle' }),
      ];
      const { lastFrame } = render(
        <AgentTabs agents={agents} activeAgentId="supervisor" />
      );
      const output = lastFrame() || '';
      expect(output).not.toContain('●');
      expect(output).not.toContain('✓');
      expect(output).not.toContain('✗');
    });
  });

  describe('active highlighting', () => {
    it('renders both active and inactive agents', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code', taskDescription: 'Fix bug' }),
      ];
      const { lastFrame } = render(
        <AgentTabs agents={agents} activeAgentId="supervisor" />
      );
      const output = lastFrame() || '';
      expect(output).toContain('Supervisor');
      expect(output).toContain('abc: Fix bug');
    });

    it('switching activeAgentId works without errors', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code' }),
      ];
      const { lastFrame, rerender } = render(
        <AgentTabs agents={agents} activeAgentId="supervisor" />
      );
      expect(lastFrame()).toContain('Supervisor');

      rerender(<AgentTabs agents={agents} activeAgentId="code-agent-abc" />);
      expect(lastFrame()).toContain('Supervisor');
      expect(lastFrame()).toContain('abc');
    });
  });

  describe('combined states', () => {
    it('supervisor idle + code agent running', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor', status: 'idle' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code', completionStatus: 'running', taskDescription: 'Fix imports' }),
      ];
      const { lastFrame } = render(
        <AgentTabs agents={agents} activeAgentId="supervisor" />
      );
      const output = lastFrame() || '';
      expect(output).toContain('Supervisor');
      expect(output).toContain('abc: Fix imports');
      expect(output).toContain('●');
    });

    it('3 agents: supervisor + 2 code agents with different statuses', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor', status: 'idle' }),
        makeAgent({ agentId: 'code-agent-aaa', role: 'code', completionStatus: 'completed', taskDescription: 'Fix imports' }),
        makeAgent({ agentId: 'code-agent-bbb', role: 'code', completionStatus: 'running', taskDescription: 'Add tests' }),
      ];
      const { lastFrame } = render(
        <AgentTabs agents={agents} activeAgentId="supervisor" />
      );
      const output = lastFrame() || '';
      expect(output).toContain('Supervisor');
      expect(output).toContain('aaa: Fix imports');
      expect(output).toContain('bbb: Add tests');
      expect(output).toContain('✓');
      expect(output).toContain('●');
    });
  });

  describe('hint display', () => {
    it('shows Shift+Tab hint when multiple agents present', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code', taskDescription: 'Fix bug' }),
      ];
      const { lastFrame } = render(
        <AgentTabs agents={agents} activeAgentId="supervisor" />
      );
      const output = lastFrame() || '';
      expect(output).toContain('Shift+Tab');
    });
  });
});
