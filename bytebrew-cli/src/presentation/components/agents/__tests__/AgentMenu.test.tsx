import React from 'react';
import { describe, it, expect } from 'bun:test';
import { render } from 'ink-testing-library';
import { AgentMenu } from '../AgentMenu.js';
import { AgentState } from '../../../../infrastructure/state/AgentStateManager.js';

// Helper to create AgentState
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

// Helper to wait for async state updates
const tick = () => new Promise(r => setTimeout(r, 10));

describe('AgentMenu', () => {
  describe('closed state', () => {
    it('returns null when agents <= 1', () => {
      const agents = [makeAgent({ agentId: 'supervisor', role: 'supervisor' })];
      const { lastFrame } = render(
        <AgentMenu
          agents={agents}
          currentViewAgentId="supervisor"
          isOpen={false}
          onSelect={() => {}}
          onClose={() => {}}
        />
      );
      expect(lastFrame()).toBe('');
    });

    it('shows compact hint when agents > 1', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code' }),
      ];
      const { lastFrame } = render(
        <AgentMenu
          agents={agents}
          currentViewAgentId="supervisor"
          isOpen={false}
          onSelect={() => {}}
          onClose={() => {}}
        />
      );
      expect(lastFrame()).toContain('2 chats');
      expect(lastFrame()).toContain('Shift+Tab');
    });

    it('shows correct count for 3 agents', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-aaa', role: 'code' }),
        makeAgent({ agentId: 'code-agent-bbb', role: 'code' }),
      ];
      const { lastFrame } = render(
        <AgentMenu
          agents={agents}
          currentViewAgentId="supervisor"
          isOpen={false}
          onSelect={() => {}}
          onClose={() => {}}
        />
      );
      expect(lastFrame()).toContain('3 chats');
    });
  });

  describe('open state', () => {
    it('renders vertical list when open', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code', taskDescription: 'Fix imports' }),
      ];
      const { lastFrame } = render(
        <AgentMenu
          agents={agents}
          currentViewAgentId="supervisor"
          isOpen={true}
          onSelect={() => {}}
          onClose={() => {}}
        />
      );
      expect(lastFrame()).toContain('Supervisor');
      expect(lastFrame()).toContain('abc: Fix imports');
    });

    it('sorts by lastMessageAt DESC (most recent first)', () => {
      const now = new Date();
      const oneMinAgo = new Date(now.getTime() - 60000);
      const twoMinAgo = new Date(now.getTime() - 120000);

      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor', lastMessageAt: twoMinAgo }),
        makeAgent({ agentId: 'code-agent-aaa', role: 'code', taskDescription: 'Old', lastMessageAt: oneMinAgo }),
        makeAgent({ agentId: 'code-agent-bbb', role: 'code', taskDescription: 'New', lastMessageAt: now }),
      ];
      const { lastFrame } = render(
        <AgentMenu
          agents={agents}
          currentViewAgentId="supervisor"
          isOpen={true}
          onSelect={() => {}}
          onClose={() => {}}
        />
      );
      const output = lastFrame() || '';
      const newPos = output.indexOf('New');
      const oldPos = output.indexOf('Old');
      const supPos = output.indexOf('Supervisor');
      // Order should be: New, Old, Supervisor
      expect(newPos).toBeLessThan(oldPos);
      expect(oldPos).toBeLessThan(supPos);
    });

    it('limits to max 10 agents', () => {
      const agents = Array.from({ length: 15 }, (_, i) =>
        makeAgent({ agentId: `code-agent-${i}`, role: 'code', taskDescription: `Task ${i}` })
      );
      const { lastFrame } = render(
        <AgentMenu
          agents={agents}
          currentViewAgentId="supervisor"
          isOpen={true}
          onSelect={() => {}}
          onClose={() => {}}
        />
      );
      const output = lastFrame() || '';
      // First 10 should be visible
      expect(output).toContain('Task 0');
      expect(output).toContain('Task 9');
      // 11th+ should NOT be visible
      expect(output).not.toContain('Task 10');
      expect(output).not.toContain('Task 14');
    });
  });

  describe('status icons', () => {
    it('shows ★ for supervisor', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code' }),
      ];
      const { lastFrame } = render(
        <AgentMenu
          agents={agents}
          currentViewAgentId="supervisor"
          isOpen={true}
          onSelect={() => {}}
          onClose={() => {}}
        />
      );
      expect(lastFrame()).toContain('★');
    });

    it('shows ✓ for completed code agent', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code', completionStatus: 'completed' }),
      ];
      const { lastFrame } = render(
        <AgentMenu
          agents={agents}
          currentViewAgentId="supervisor"
          isOpen={true}
          onSelect={() => {}}
          onClose={() => {}}
        />
      );
      expect(lastFrame()).toContain('✓');
    });

    it('shows ✗ for failed code agent', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code', completionStatus: 'failed' }),
      ];
      const { lastFrame } = render(
        <AgentMenu
          agents={agents}
          currentViewAgentId="supervisor"
          isOpen={true}
          onSelect={() => {}}
          onClose={() => {}}
        />
      );
      expect(lastFrame()).toContain('✗');
    });

    it('shows ● for running code agent', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code', completionStatus: 'running' }),
      ];
      const { lastFrame } = render(
        <AgentMenu
          agents={agents}
          currentViewAgentId="supervisor"
          isOpen={true}
          onSelect={() => {}}
          onClose={() => {}}
        />
      );
      expect(lastFrame()).toContain('●');
    });

    it('shows ○ for idle code agent', () => {
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code', completionStatus: 'idle' }),
      ];
      const { lastFrame } = render(
        <AgentMenu
          agents={agents}
          currentViewAgentId="supervisor"
          isOpen={true}
          onSelect={() => {}}
          onClose={() => {}}
        />
      );
      expect(lastFrame()).toContain('○');
    });
  });

  describe('relative time', () => {
    it('shows "just now" for < 60s', () => {
      const now = new Date();
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor', lastMessageAt: now }),
      ];
      const { lastFrame } = render(
        <AgentMenu
          agents={[agents[0]]}
          currentViewAgentId="supervisor"
          isOpen={true}
          onSelect={() => {}}
          onClose={() => {}}
        />
      );
      expect(lastFrame()).toContain('just now');
    });

    it('shows "Nm ago" for < 60m', () => {
      const now = new Date();
      const twoMinAgo = new Date(now.getTime() - 2 * 60 * 1000);
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor', lastMessageAt: twoMinAgo }),
      ];
      const { lastFrame } = render(
        <AgentMenu
          agents={[agents[0]]}
          currentViewAgentId="supervisor"
          isOpen={true}
          onSelect={() => {}}
          onClose={() => {}}
        />
      );
      expect(lastFrame()).toContain('2m ago');
    });

    it('shows "Nh ago" for < 24h', () => {
      const now = new Date();
      const threeHoursAgo = new Date(now.getTime() - 3 * 60 * 60 * 1000);
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor', lastMessageAt: threeHoursAgo }),
      ];
      const { lastFrame } = render(
        <AgentMenu
          agents={[agents[0]]}
          currentViewAgentId="supervisor"
          isOpen={true}
          onSelect={() => {}}
          onClose={() => {}}
        />
      );
      expect(lastFrame()).toContain('3h ago');
    });

    it('shows "Nd ago" for >= 24h', () => {
      const now = new Date();
      const twoDaysAgo = new Date(now.getTime() - 2 * 24 * 60 * 60 * 1000);
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor', lastMessageAt: twoDaysAgo }),
      ];
      const { lastFrame } = render(
        <AgentMenu
          agents={[agents[0]]}
          currentViewAgentId="supervisor"
          isOpen={true}
          onSelect={() => {}}
          onClose={() => {}}
        />
      );
      expect(lastFrame()).toContain('2d ago');
    });
  });

  describe('keyboard interaction', () => {
    it('calls onClose when Esc pressed', async () => {
      let closeCalled = false;
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code' }),
      ];
      const instance = render(
        <AgentMenu
          agents={agents}
          currentViewAgentId="supervisor"
          isOpen={true}
          onSelect={() => {}}
          onClose={() => { closeCalled = true; }}
        />
      );
      instance.stdin.write('\x1b'); // Esc
      await tick();
      expect(closeCalled).toBe(true);
      instance.unmount();
    });

    it('calls onSelect and onClose when Enter pressed', async () => {
      let selectedAgentId = '';
      let closeCalled = false;
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code' }),
      ];
      const instance = render(
        <AgentMenu
          agents={agents}
          currentViewAgentId="supervisor"
          isOpen={true}
          onSelect={(id) => { selectedAgentId = id; }}
          onClose={() => { closeCalled = true; }}
        />
      );
      instance.stdin.write('\r'); // Enter
      await tick();
      expect(selectedAgentId).toBe('supervisor'); // first item selected by default
      expect(closeCalled).toBe(true);
      instance.unmount();
    });

    it('navigates down with Arrow Down', async () => {
      let selectedAgentId = '';
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code' }),
      ];
      const instance = render(
        <AgentMenu
          agents={agents}
          currentViewAgentId="supervisor"
          isOpen={true}
          onSelect={(id) => { selectedAgentId = id; }}
          onClose={() => {}}
        />
      );
      instance.stdin.write('\x1b[B'); // Arrow Down
      await tick();
      instance.stdin.write('\r'); // Enter
      await tick();
      expect(selectedAgentId).toBe('code-agent-abc'); // second item
      instance.unmount();
    });

    it('navigates up with Arrow Up', async () => {
      let selectedAgentId = '';
      const agents = [
        makeAgent({ agentId: 'supervisor', role: 'supervisor' }),
        makeAgent({ agentId: 'code-agent-abc', role: 'code' }),
      ];
      const instance = render(
        <AgentMenu
          agents={agents}
          currentViewAgentId="supervisor"
          isOpen={true}
          onSelect={(id) => { selectedAgentId = id; }}
          onClose={() => {}}
        />
      );
      instance.stdin.write('\x1b[B'); // Down
      await tick();
      instance.stdin.write('\x1b[A'); // Up
      await tick();
      instance.stdin.write('\r'); // Enter
      await tick();
      expect(selectedAgentId).toBe('supervisor'); // back to first
      instance.unmount();
    });
  });
});
