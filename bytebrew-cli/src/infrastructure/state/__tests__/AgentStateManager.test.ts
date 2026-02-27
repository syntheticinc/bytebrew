import { describe, it, expect, beforeEach } from 'bun:test';
import { AgentStateManager } from '../AgentStateManager.js';
import { MessageId } from '../../../domain/value-objects/MessageId.js';

describe('AgentStateManager', () => {
  let manager: AgentStateManager;

  beforeEach(() => {
    manager = new AgentStateManager();
  });

  describe('getOrCreateAgent', () => {
    it('creates supervisor with role=supervisor, name=Supervisor', () => {
      const agent = manager.getOrCreateAgent('supervisor');
      expect(agent.agentId).toBe('supervisor');
      expect(agent.role).toBe('supervisor');
      expect(agent.name).toBe('Supervisor');
    });

    it('creates code agent with role=code, name="Code Agent {shortId}"', () => {
      const agent = manager.getOrCreateAgent('code-agent-abc123');
      expect(agent.agentId).toBe('code-agent-abc123');
      expect(agent.role).toBe('code');
      expect(agent.name).toBe('Code Agent abc123');
    });

    it('strips "code-agent-" prefix: "code-agent-abc123" → "Code Agent abc123"', () => {
      const agent1 = manager.getOrCreateAgent('code-agent-xyz789');
      expect(agent1.name).toBe('Code Agent xyz789');

      const agent2 = manager.getOrCreateAgent('code-agent-test-id');
      expect(agent2.name).toBe('Code Agent test-id');
    });

    it('returns same instance on repeated calls', () => {
      const agent1 = manager.getOrCreateAgent('supervisor');
      const agent2 = manager.getOrCreateAgent('supervisor');
      expect(agent1).toBe(agent2);

      const codeAgent1 = manager.getOrCreateAgent('code-agent-123');
      const codeAgent2 = manager.getOrCreateAgent('code-agent-123');
      expect(codeAgent1).toBe(codeAgent2);
    });

    it('creates distinct agents for different IDs', () => {
      const supervisor = manager.getOrCreateAgent('supervisor');
      const codeAgent = manager.getOrCreateAgent('code-agent-123');
      expect(supervisor).not.toBe(codeAgent);
      expect(supervisor.agentId).toBe('supervisor');
      expect(codeAgent.agentId).toBe('code-agent-123');
    });

    it('initial status is idle, messageId/reasoningId are null', () => {
      const agent = manager.getOrCreateAgent('code-agent-test');
      expect(agent.status).toBe('idle');
      expect(agent.currentMessageId).toBeNull();
      expect(agent.currentReasoningId).toBeNull();
    });
  });

  describe('getActiveAgent', () => {
    it('returns supervisor by default', () => {
      const active = manager.getActiveAgent();
      expect(active.agentId).toBe('supervisor');
      expect(active.role).toBe('supervisor');
    });

    it('returns agent set by setActiveAgent', () => {
      manager.setActiveAgent('code-agent-999');
      const active = manager.getActiveAgent();
      expect(active.agentId).toBe('code-agent-999');
      expect(active.role).toBe('code');
    });
  });

  describe('setActiveAgent', () => {
    it('switches active agent', () => {
      expect(manager.activeAgentId).toBe('supervisor');
      manager.setActiveAgent('code-agent-abc');
      expect(manager.activeAgentId).toBe('code-agent-abc');
    });

    it('creates agent if not in map', () => {
      expect(manager.getAllAgents().length).toBe(0);
      manager.setActiveAgent('code-agent-new');
      expect(manager.getAllAgents().length).toBe(1);
      const agent = manager.getActiveAgent();
      expect(agent.agentId).toBe('code-agent-new');
    });

    it('activeAgentId getter reflects change', () => {
      manager.setActiveAgent('test-agent');
      expect(manager.activeAgentId).toBe('test-agent');
      manager.setActiveAgent('another-agent');
      expect(manager.activeAgentId).toBe('another-agent');
    });
  });

  describe('cycleNextAgent', () => {
    it('no-op with single agent (activeAgentId stays the same)', () => {
      manager.getOrCreateAgent('supervisor');
      expect(manager.activeAgentId).toBe('supervisor');
      manager.cycleNextAgent();
      expect(manager.activeAgentId).toBe('supervisor');
    });

    it('cycles supervisor → code-agent → supervisor', () => {
      manager.getOrCreateAgent('supervisor');
      manager.getOrCreateAgent('code-agent-1');

      expect(manager.activeAgentId).toBe('supervisor');
      manager.cycleNextAgent();
      expect(manager.activeAgentId).toBe('code-agent-1');
      manager.cycleNextAgent();
      expect(manager.activeAgentId).toBe('supervisor');
    });

    it('cycles through 3 agents wrapping around', () => {
      manager.getOrCreateAgent('supervisor');
      manager.getOrCreateAgent('code-agent-1');
      manager.getOrCreateAgent('code-agent-2');

      expect(manager.activeAgentId).toBe('supervisor');
      manager.cycleNextAgent();
      expect(manager.activeAgentId).toBe('code-agent-1');
      manager.cycleNextAgent();
      expect(manager.activeAgentId).toBe('code-agent-2');
      manager.cycleNextAgent();
      expect(manager.activeAgentId).toBe('supervisor');
    });

    it('resets to first if active not in map', () => {
      manager.getOrCreateAgent('supervisor');
      manager.getOrCreateAgent('code-agent-1');
      manager['_activeAgentId'] = 'non-existent';

      manager.cycleNextAgent();
      // First agent in insertion order is 'supervisor'
      expect(manager.activeAgentId).toBe('supervisor');
    });
  });

  describe('getAllAgents', () => {
    it('empty array initially', () => {
      expect(manager.getAllAgents()).toEqual([]);
    });

    it('returns all created agents', () => {
      manager.getOrCreateAgent('supervisor');
      manager.getOrCreateAgent('code-agent-1');
      manager.getOrCreateAgent('code-agent-2');

      const agents = manager.getAllAgents();
      expect(agents.length).toBe(3);
      expect(agents.map(a => a.agentId)).toContain('supervisor');
      expect(agents.map(a => a.agentId)).toContain('code-agent-1');
      expect(agents.map(a => a.agentId)).toContain('code-agent-2');
    });

    it('preserves insertion order', () => {
      manager.getOrCreateAgent('supervisor');
      manager.getOrCreateAgent('code-agent-1');
      manager.getOrCreateAgent('code-agent-2');

      const agents = manager.getAllAgents();
      expect(agents[0].agentId).toBe('supervisor');
      expect(agents[1].agentId).toBe('code-agent-1');
      expect(agents[2].agentId).toBe('code-agent-2');
    });
  });

  describe('hasMultipleAgents', () => {
    it('false with 0 agents', () => {
      expect(manager.hasMultipleAgents()).toBe(false);
    });

    it('false with 1 agent', () => {
      manager.getOrCreateAgent('supervisor');
      expect(manager.hasMultipleAgents()).toBe(false);
    });

    it('true with 2+ agents', () => {
      manager.getOrCreateAgent('supervisor');
      manager.getOrCreateAgent('code-agent-1');
      expect(manager.hasMultipleAgents()).toBe(true);

      manager.getOrCreateAgent('code-agent-2');
      expect(manager.hasMultipleAgents()).toBe(true);
    });
  });

  describe('per-agent message tracking', () => {
    it('null initially for any agent', () => {
      const msgId = manager.getCurrentMessageId('supervisor');
      expect(msgId).toBeNull();
    });

    it('stores per-agent message ID', () => {
      const id = MessageId.create();
      manager.setCurrentMessageId('supervisor', id);
      expect(manager.getCurrentMessageId('supervisor')).toBe(id);
    });

    it('different agents independent', () => {
      const id1 = MessageId.create();
      const id2 = MessageId.create();

      manager.setCurrentMessageId('supervisor', id1);
      manager.setCurrentMessageId('code-agent-1', id2);

      expect(manager.getCurrentMessageId('supervisor')).toBe(id1);
      expect(manager.getCurrentMessageId('code-agent-1')).toBe(id2);
    });

    it('null clears ID', () => {
      const id = MessageId.create();
      manager.setCurrentMessageId('supervisor', id);
      expect(manager.getCurrentMessageId('supervisor')).toBe(id);

      manager.setCurrentMessageId('supervisor', null);
      expect(manager.getCurrentMessageId('supervisor')).toBeNull();
    });

    it('auto-creates agent on access', () => {
      expect(manager.getAllAgents().length).toBe(0);
      manager.getCurrentMessageId('code-agent-new');
      expect(manager.getAllAgents().length).toBe(1);
    });
  });

  describe('per-agent reasoning tracking', () => {
    it('null initially', () => {
      const reasoningId = manager.getCurrentReasoningId('supervisor');
      expect(reasoningId).toBeNull();
    });

    it('stores per-agent reasoning ID', () => {
      const id = MessageId.create();
      manager.setCurrentReasoningId('supervisor', id);
      expect(manager.getCurrentReasoningId('supervisor')).toBe(id);
    });

    it('independent from message ID', () => {
      const msgId = MessageId.create();
      const reasoningId = MessageId.create();

      manager.setCurrentMessageId('supervisor', msgId);
      manager.setCurrentReasoningId('supervisor', reasoningId);

      expect(manager.getCurrentMessageId('supervisor')).toBe(msgId);
      expect(manager.getCurrentReasoningId('supervisor')).toBe(reasoningId);
      expect(msgId).not.toBe(reasoningId);
    });
  });

  describe('resetAll', () => {
    it('clears all message IDs', () => {
      const id1 = MessageId.create();
      const id2 = MessageId.create();

      manager.setCurrentMessageId('supervisor', id1);
      manager.setCurrentMessageId('code-agent-1', id2);

      manager.resetAll();

      expect(manager.getCurrentMessageId('supervisor')).toBeNull();
      expect(manager.getCurrentMessageId('code-agent-1')).toBeNull();
    });

    it('clears all reasoning IDs', () => {
      const id1 = MessageId.create();
      const id2 = MessageId.create();

      manager.setCurrentReasoningId('supervisor', id1);
      manager.setCurrentReasoningId('code-agent-1', id2);

      manager.resetAll();

      expect(manager.getCurrentReasoningId('supervisor')).toBeNull();
      expect(manager.getCurrentReasoningId('code-agent-1')).toBeNull();
    });

    it('preserves agents in map', () => {
      manager.getOrCreateAgent('supervisor');
      manager.getOrCreateAgent('code-agent-1');

      expect(manager.getAllAgents().length).toBe(2);
      manager.resetAll();
      expect(manager.getAllAgents().length).toBe(2);
    });

    it('preserves active agent', () => {
      manager.setActiveAgent('code-agent-1');
      expect(manager.activeAgentId).toBe('code-agent-1');

      manager.resetAll();
      expect(manager.activeAgentId).toBe('code-agent-1');
    });

    it('preserves role and name', () => {
      manager.getOrCreateAgent('supervisor');
      manager.getOrCreateAgent('code-agent-1');

      const supervisorBefore = manager.getAllAgents().find(a => a.agentId === 'supervisor')!;
      const codeBefore = manager.getAllAgents().find(a => a.agentId === 'code-agent-1')!;

      manager.resetAll();

      const supervisorAfter = manager.getAllAgents().find(a => a.agentId === 'supervisor')!;
      const codeAfter = manager.getAllAgents().find(a => a.agentId === 'code-agent-1')!;

      expect(supervisorAfter.role).toBe(supervisorBefore.role);
      expect(supervisorAfter.name).toBe(supervisorBefore.name);
      expect(codeAfter.role).toBe(codeBefore.role);
      expect(codeAfter.name).toBe(codeBefore.name);
    });
  });

  describe('updateLifecycle', () => {
    it('creates agent with default values (taskDescription="", completionStatus="idle")', () => {
      const agent = manager.getOrCreateAgent('code-agent-test');
      expect(agent.taskDescription).toBe('');
      expect(agent.completionStatus).toBe('idle');
    });

    it('agent_spawned → completionStatus="running", taskDescription filled', () => {
      manager.updateLifecycle('code-agent-1', 'agent_spawned', 'Implement feature X');

      const agent = manager.getOrCreateAgent('code-agent-1');
      expect(agent.completionStatus).toBe('running');
      expect(agent.taskDescription).toBe('Implement feature X');
    });

    it('agent_completed → completionStatus="completed"', () => {
      manager.updateLifecycle('code-agent-1', 'agent_spawned', 'Task A');
      manager.updateLifecycle('code-agent-1', 'agent_completed', '');

      const agent = manager.getOrCreateAgent('code-agent-1');
      expect(agent.completionStatus).toBe('completed');
      expect(agent.taskDescription).toBe('Task A'); // Preserved
    });

    it('agent_failed → completionStatus="failed"', () => {
      manager.updateLifecycle('code-agent-2', 'agent_spawned', 'Task B');
      manager.updateLifecycle('code-agent-2', 'agent_failed', '');

      const agent = manager.getOrCreateAgent('code-agent-2');
      expect(agent.completionStatus).toBe('failed');
      expect(agent.taskDescription).toBe('Task B'); // Preserved
    });

    it('agent_restarted → completionStatus="running", taskDescription updated if not empty', () => {
      manager.updateLifecycle('code-agent-3', 'agent_spawned', 'Initial task');
      manager.updateLifecycle('code-agent-3', 'agent_restarted', 'Restarted task');

      const agent = manager.getOrCreateAgent('code-agent-3');
      expect(agent.completionStatus).toBe('running');
      expect(agent.taskDescription).toBe('Restarted task');
    });

    it('agent_restarted with empty description → completionStatus="running", taskDescription unchanged', () => {
      manager.updateLifecycle('code-agent-4', 'agent_spawned', 'Original task');
      manager.updateLifecycle('code-agent-4', 'agent_restarted', '');

      const agent = manager.getOrCreateAgent('code-agent-4');
      expect(agent.completionStatus).toBe('running');
      expect(agent.taskDescription).toBe('Original task'); // Not overwritten by empty string
    });

    it('multiple lifecycle transitions preserve agent identity', () => {
      manager.updateLifecycle('code-agent-5', 'agent_spawned', 'Task 1');
      const agent1 = manager.getOrCreateAgent('code-agent-5');

      manager.updateLifecycle('code-agent-5', 'agent_completed', '');
      const agent2 = manager.getOrCreateAgent('code-agent-5');

      expect(agent1).toBe(agent2); // Same instance
      expect(agent2.completionStatus).toBe('completed');
    });

    it('lifecycle updates do not affect other agents', () => {
      manager.updateLifecycle('supervisor', 'agent_spawned', 'Supervisor task');
      manager.updateLifecycle('code-agent-1', 'agent_spawned', 'Code task 1');
      manager.updateLifecycle('code-agent-2', 'agent_spawned', 'Code task 2');

      manager.updateLifecycle('code-agent-1', 'agent_completed', '');

      const supervisor = manager.getOrCreateAgent('supervisor');
      const code1 = manager.getOrCreateAgent('code-agent-1');
      const code2 = manager.getOrCreateAgent('code-agent-2');

      expect(supervisor.completionStatus).toBe('running');
      expect(code1.completionStatus).toBe('completed');
      expect(code2.completionStatus).toBe('running');
    });
  });
});
