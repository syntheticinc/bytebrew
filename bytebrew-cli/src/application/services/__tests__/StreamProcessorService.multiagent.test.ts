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
import type { AgentLifecycleEvent } from '../../../domain/ports/IEventBus.js';

describe('StreamProcessorService - Multi-Agent', () => {
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

  describe('agent registration', () => {
    it('registers supervisor when no agentId in response', () => {
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Hello',
        isFinal: false,
      });

      const agents = agentStateManager.getAllAgents();
      expect(agents.length).toBe(1);
      expect(agents[0].agentId).toBe('supervisor');
      expect(agents[0].role).toBe('supervisor');
    });

    it('registers supervisor when agentId="supervisor"', () => {
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Hello',
        agentId: 'supervisor',
        isFinal: false,
      });

      const agents = agentStateManager.getAllAgents();
      expect(agents.length).toBe(1);
      expect(agents[0].agentId).toBe('supervisor');
      expect(agents[0].role).toBe('supervisor');
    });

    it('registers code agent on agentId="code-agent-abc"', () => {
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Working...',
        agentId: 'code-agent-abc',
        isFinal: false,
      });

      const agents = agentStateManager.getAllAgents();
      expect(agents.length).toBe(1);
      expect(agents[0].agentId).toBe('code-agent-abc');
      expect(agents[0].role).toBe('code');
    });

    it('no duplicate registration on repeated responses from same agent', () => {
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'First chunk',
        agentId: 'code-agent-xyz',
        isFinal: false,
      });

      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Second chunk',
        agentId: 'code-agent-xyz',
        isFinal: false,
      });

      const agents = agentStateManager.getAllAgents();
      expect(agents.length).toBe(1);
      expect(agents[0].agentId).toBe('code-agent-xyz');
    });
  });

  describe('agent routing - per-agent context', () => {
    it('supervisor chunk accumulates into supervisor message', () => {
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Supervisor says hello',
        agentId: 'supervisor',
        isFinal: false,
      });

      gateway.simulateResponse({
        type: 'ANSWER',
        content: '',
        isFinal: true,
        agentId: 'supervisor',
      });

      const messages = repository.findAll();
      expect(messages.length).toBe(1);
      expect(messages[0].content.value).toBe('Supervisor says hello');
      expect(messages[0].agentId).toBe('supervisor');
    });

    it('code-agent chunk accumulates into code-agent message', () => {
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Code agent working',
        agentId: 'code-agent-1',
        isFinal: false,
      });

      gateway.simulateResponse({
        type: 'ANSWER',
        content: '',
        isFinal: true,
        agentId: 'code-agent-1',
      });

      const messages = repository.findAll();
      expect(messages.length).toBe(1);
      expect(messages[0].content.value).toBe('Code agent working');
      expect(messages[0].agentId).toBe('code-agent-1');
    });

    it('interleaved chunks: 2 agents accumulate independently', () => {
      // Supervisor first chunk
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Supervisor part 1',
        agentId: 'supervisor',
        isFinal: false,
      });

      // Code agent chunk
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Code agent output',
        agentId: 'code-agent-1',
        isFinal: false,
      });

      // Supervisor second chunk
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: ' part 2',
        agentId: 'supervisor',
        isFinal: false,
      });

      // Complete code agent
      gateway.simulateResponse({
        type: 'ANSWER',
        content: '',
        isFinal: true,
        agentId: 'code-agent-1',
      });

      // Complete supervisor
      gateway.simulateResponse({
        type: 'ANSWER',
        content: '',
        isFinal: true,
        agentId: 'supervisor',
      });

      const allMessages = repository.findAll();
      // Filter out agent separator messages (contain ───)
      const messages = allMessages.filter(m => !m.content.value.includes('───'));
      expect(messages.length).toBe(2);

      const codeMessage = messages.find(m => m.content.value.includes('Code agent'));
      const supervisorMessage = messages.find(m => m.content.value.includes('Supervisor'));

      expect(codeMessage?.content.value).toBe('Code agent output');
      expect(codeMessage?.agentId).toBe('code-agent-1');
      expect(supervisorMessage?.content.value).toBe('Supervisor part 1 part 2');
      expect(supervisorMessage?.agentId).toBe('supervisor');
    });

    it('REASONING from different agents creates separate reasoning messages', () => {
      // Supervisor reasoning
      gateway.simulateResponse({
        type: 'REASONING',
        content: '',
        isFinal: false,
        reasoning: {
          thinking: 'Supervisor thinking',
          isComplete: true,
        },
        agentId: 'supervisor',
      });

      // Code agent reasoning
      gateway.simulateResponse({
        type: 'REASONING',
        content: '',
        isFinal: false,
        reasoning: {
          thinking: 'Code agent thinking',
          isComplete: true,
        },
        agentId: 'code-agent-1',
      });

      const allMessages = repository.findAll();
      // Filter out agent separator messages (contain ───)
      const messages = allMessages.filter(m => !m.content.value.includes('───'));
      expect(messages.length).toBe(2);

      expect(messages[0].content.value).toBe('Supervisor thinking');
      expect(messages[0].hasReasoning).toBe(true);
      expect(messages[0].agentId).toBe('supervisor');

      expect(messages[1].content.value).toBe('Code agent thinking');
      expect(messages[1].hasReasoning).toBe(true);
      expect(messages[1].agentId).toBe('code-agent-1');
    });
  });

  describe('isFinal semantics', () => {
    it('isFinal from supervisor → ProcessingStopped event', () => {
      // Start processing first
      processor.sendMessage('test');
      eventBus.publishedEvents = []; // Clear events from sendMessage

      gateway.simulateResponse({
        type: 'ANSWER',
        content: 'Done',
        isFinal: true,
        agentId: 'supervisor',
      });

      const stoppedEvents = eventBus.getEventsOfType('ProcessingStopped');
      expect(stoppedEvents.length).toBe(1);
      expect(processor.getIsProcessing()).toBe(false);
    });

    it('isFinal from code-agent → message completed, NO ProcessingStopped', () => {
      gateway.simulateResponse({
        type: 'ANSWER',
        content: 'Code done',
        isFinal: true,
        agentId: 'code-agent-1',
      });

      const stoppedEvents = eventBus.getEventsOfType('ProcessingStopped');
      expect(stoppedEvents.length).toBe(0);
      expect(processor.getIsProcessing()).toBe(false); // Initial state

      const completedEvents = eventBus.getEventsOfType('MessageCompleted');
      expect(completedEvents.length).toBe(1);
      expect((completedEvents[0] as Extract<typeof completedEvents[0], { type: 'MessageCompleted' }>).message.agentId).toBe('code-agent-1');
    });

    it('isFinal code-agent then isFinal supervisor → ProcessingStopped after supervisor', () => {
      // Start processing first
      processor.sendMessage('test');
      eventBus.publishedEvents = []; // Clear events from sendMessage

      // Code agent finishes
      gateway.simulateResponse({
        type: 'ANSWER',
        content: 'Code done',
        isFinal: true,
        agentId: 'code-agent-1',
      });

      let stoppedEvents = eventBus.getEventsOfType('ProcessingStopped');
      expect(stoppedEvents.length).toBe(0);

      // Supervisor finishes
      gateway.simulateResponse({
        type: 'ANSWER',
        content: 'All done',
        isFinal: true,
        agentId: 'supervisor',
      });

      stoppedEvents = eventBus.getEventsOfType('ProcessingStopped');
      expect(stoppedEvents.length).toBe(1);
    });

    it('empty isFinal from supervisor → stops processing', () => {
      // Start processing first
      processor.sendMessage('test');
      eventBus.publishedEvents = []; // Clear events from sendMessage

      gateway.simulateResponse({
        type: 'ANSWER',
        content: '',
        isFinal: true,
        agentId: 'supervisor',
      });

      const stoppedEvents = eventBus.getEventsOfType('ProcessingStopped');
      expect(stoppedEvents.length).toBe(1);
    });

    it('empty isFinal from code-agent → completes only that agent\'s message', () => {
      // Start accumulating
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Working',
        agentId: 'code-agent-1',
        isFinal: false,
      });

      // Empty isFinal
      gateway.simulateResponse({
        type: 'ANSWER',
        content: '',
        isFinal: true,
        agentId: 'code-agent-1',
      });

      const messages = repository.findAll();
      expect(messages.length).toBe(1);
      expect(messages[0].content.value).toBe('Working');
      expect(messages[0].agentId).toBe('code-agent-1');

      const stoppedEvents = eventBus.getEventsOfType('ProcessingStopped');
      expect(stoppedEvents.length).toBe(0);
    });
  });

  describe('lifecycle event parsing', () => {
    it('[agent_spawned] code-agent-abc: Starting work → AgentLifecycle event', () => {
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: '[agent_spawned] code-agent-abc: Starting work on task',
        agentId: 'supervisor',
        isFinal: false,
      });

      const lifecycleEvents = eventBus.getEventsOfType('AgentLifecycle');
      expect(lifecycleEvents.length).toBe(1);

      const event = lifecycleEvents[0] as AgentLifecycleEvent;
      expect(event.lifecycleType).toBe('agent_spawned');
      expect(event.agentId).toBe('code-agent-abc');
      expect(event.description).toBe('Starting work on task');
    });

    it('[agent_completed] code-agent-abc: Done → AgentLifecycle event', () => {
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: '[agent_completed] code-agent-abc: Task completed successfully',
        agentId: 'supervisor',
        isFinal: false,
      });

      const lifecycleEvents = eventBus.getEventsOfType('AgentLifecycle');
      expect(lifecycleEvents.length).toBe(1);

      const event = lifecycleEvents[0] as AgentLifecycleEvent;
      expect(event.lifecycleType).toBe('agent_completed');
      expect(event.agentId).toBe('code-agent-abc');
      expect(event.description).toBe('Task completed successfully');
    });

    it('[agent_failed] code-agent-abc: Error occurred → AgentLifecycle event', () => {
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: '[agent_failed] code-agent-abc: Connection timeout',
        agentId: 'supervisor',
        isFinal: false,
      });

      const lifecycleEvents = eventBus.getEventsOfType('AgentLifecycle');
      expect(lifecycleEvents.length).toBe(1);

      const event = lifecycleEvents[0] as AgentLifecycleEvent;
      expect(event.lifecycleType).toBe('agent_failed');
      expect(event.agentId).toBe('code-agent-abc');
      expect(event.description).toBe('Connection timeout');
    });

    it('[agent_restarted] code-agent-abc: Retry → AgentLifecycle event', () => {
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: '[agent_restarted] code-agent-abc: Retrying after failure',
        agentId: 'supervisor',
        isFinal: false,
      });

      const lifecycleEvents = eventBus.getEventsOfType('AgentLifecycle');
      expect(lifecycleEvents.length).toBe(1);

      const event = lifecycleEvents[0] as AgentLifecycleEvent;
      expect(event.lifecycleType).toBe('agent_restarted');
      expect(event.agentId).toBe('code-agent-abc');
      expect(event.description).toBe('Retrying after failure');
    });

    it('registers agent in state manager on lifecycle event', () => {
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: '[agent_spawned] code-agent-xyz: New agent',
        agentId: 'supervisor',
        isFinal: false,
      });

      const agents = agentStateManager.getAllAgents();
      const codeAgent = agents.find(a => a.agentId === 'code-agent-xyz');

      expect(codeAgent).toBeDefined();
      expect(codeAgent?.role).toBe('code');
    });

    it('normal ANSWER_CHUNK content is NOT treated as lifecycle', () => {
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Normal message without lifecycle marker',
        agentId: 'supervisor',
        isFinal: false,
      });

      const lifecycleEvents = eventBus.getEventsOfType('AgentLifecycle');
      expect(lifecycleEvents.length).toBe(0);

      // Should be treated as normal chunk
      const startedEvents = eventBus.getEventsOfType('MessageStarted');
      expect(startedEvents.length).toBe(1);
    });

    it('null/empty content → no lifecycle event (normal chunk handling)', () => {
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: '',
        agentId: 'supervisor',
        isFinal: false,
      });

      const lifecycleEvents = eventBus.getEventsOfType('AgentLifecycle');
      expect(lifecycleEvents.length).toBe(0);
    });

    it('malformed "[agent_spawned]" without agentId/colon → no lifecycle event', () => {
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: '[agent_spawned] No colon here',
        agentId: 'supervisor',
        isFinal: false,
      });

      const lifecycleEvents = eventBus.getEventsOfType('AgentLifecycle');
      expect(lifecycleEvents.length).toBe(0);

      // Should be treated as normal chunk
      const startedEvents = eventBus.getEventsOfType('MessageStarted');
      expect(startedEvents.length).toBe(1);
    });
  });

  describe('cancel with multiple agents', () => {
    it('start supervisor and code-agent chunks, then cancel → ProcessingStopped', () => {
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Supervisor working',
        agentId: 'supervisor',
        isFinal: false,
      });

      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Code agent working',
        agentId: 'code-agent-1',
        isFinal: false,
      });

      processor.cancel();

      const stoppedEvents = eventBus.getEventsOfType('ProcessingStopped');
      expect(stoppedEvents.length).toBe(1);
      expect(processor.getIsProcessing()).toBe(false);
    });

    it('cancel resets all agent message/reasoning IDs', () => {
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Supervisor msg',
        agentId: 'supervisor',
        isFinal: false,
      });

      gateway.simulateResponse({
        type: 'REASONING',
        content: '',
        isFinal: false,
        reasoning: { thinking: 'Thinking...', isComplete: false },
        agentId: 'code-agent-1',
      });

      const supervisorAgent = agentStateManager.getOrCreateAgent('supervisor');
      const codeAgent = agentStateManager.getOrCreateAgent('code-agent-1');

      expect(supervisorAgent.currentMessageId).not.toBeNull();
      expect(codeAgent.currentReasoningId).not.toBeNull();

      processor.cancel();

      expect(supervisorAgent.currentMessageId).toBeNull();
      expect(codeAgent.currentReasoningId).toBeNull();
    });

    it('cancel aborts accumulating messages for all agents', () => {
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Supervisor chunk',
        agentId: 'supervisor',
        isFinal: false,
      });

      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Code chunk',
        agentId: 'code-agent-1',
        isFinal: false,
      });

      expect(accumulator.getAccumulatingCount()).toBe(2);

      processor.cancel();

      expect(accumulator.getAccumulatingCount()).toBe(0);
    });
  });

  describe('sendMessage resets agents', () => {
    it('sendMessage clears per-agent message/reasoning IDs', () => {
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Message',
        agentId: 'supervisor',
        isFinal: false,
      });

      gateway.simulateResponse({
        type: 'REASONING',
        content: '',
        isFinal: false,
        reasoning: { thinking: 'Think', isComplete: false },
        agentId: 'code-agent-1',
      });

      const supervisorAgent = agentStateManager.getOrCreateAgent('supervisor');
      const codeAgent = agentStateManager.getOrCreateAgent('code-agent-1');

      expect(supervisorAgent.currentMessageId).not.toBeNull();
      expect(codeAgent.currentReasoningId).not.toBeNull();

      processor.sendMessage('New message');

      expect(supervisorAgent.currentMessageId).toBeNull();
      expect(codeAgent.currentReasoningId).toBeNull();
    });

    it('sendMessage preserves agent registration (agents still in map)', () => {
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Hello',
        agentId: 'code-agent-1',
        isFinal: false,
      });

      const beforeCount = agentStateManager.getAllAgents().length;
      expect(beforeCount).toBe(1);

      processor.sendMessage('New message');

      const afterCount = agentStateManager.getAllAgents().length;
      expect(afterCount).toBe(1);

      const agent = agentStateManager.getOrCreateAgent('code-agent-1');
      expect(agent.agentId).toBe('code-agent-1');
    });
  });

  describe('full multi-agent scenario', () => {
    it('supervisor spawns agent → agent works → agent finishes → supervisor answers → stops', () => {
      // Start processing first
      processor.sendMessage('test');
      eventBus.publishedEvents = []; // Clear events from sendMessage

      // 1. Supervisor spawns agent
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: '[agent_spawned] code-agent-1: Working on task',
        agentId: 'supervisor',
        isFinal: false,
      });

      let lifecycleEvents = eventBus.getEventsOfType('AgentLifecycle');
      expect(lifecycleEvents.length).toBe(1);
      expect((lifecycleEvents[0] as AgentLifecycleEvent).lifecycleType).toBe('agent_spawned');

      // 2. Code agent works
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Analyzing code...',
        agentId: 'code-agent-1',
        isFinal: false,
      });

      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: ' Done analyzing.',
        agentId: 'code-agent-1',
        isFinal: false,
      });

      // 3. Code agent finishes (isFinal, non-supervisor)
      gateway.simulateResponse({
        type: 'ANSWER',
        content: '',
        isFinal: true,
        agentId: 'code-agent-1',
      });

      // Filter out non-content messages (separator, lifecycle, task) from completed events
      const isContentMsg = (e: any) => {
        const v: string = e.message.content.value;
        return !v.includes('───') && !v.startsWith('+') && !v.startsWith('⊕') && !v.startsWith('✓') && !v.startsWith('✗') && !v.startsWith('↻') && !v.startsWith('[Task');
      };
      let completedEvents = eventBus.getEventsOfType('MessageCompleted').filter(isContentMsg);
      expect(completedEvents.length).toBe(1);
      expect((completedEvents[0] as any).message.content.value).toBe('Analyzing code... Done analyzing.');
      expect((completedEvents[0] as any).message.agentId).toBe('code-agent-1');

      let stoppedEvents = eventBus.getEventsOfType('ProcessingStopped');
      expect(stoppedEvents.length).toBe(0); // No stop yet

      // 4. Supervisor reports completion
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: '[agent_completed] code-agent-1: Task completed',
        agentId: 'supervisor',
        isFinal: false,
      });

      lifecycleEvents = eventBus.getEventsOfType('AgentLifecycle');
      expect(lifecycleEvents.length).toBe(2);
      expect((lifecycleEvents[1] as AgentLifecycleEvent).lifecycleType).toBe('agent_completed');

      // 5. Supervisor provides final answer
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Based on the agent\'s work, here are the results...',
        agentId: 'supervisor',
        isFinal: false,
      });

      // 6. Supervisor finishes (isFinal=true)
      gateway.simulateResponse({
        type: 'ANSWER',
        content: '',
        isFinal: true,
        agentId: 'supervisor',
      });

      completedEvents = eventBus.getEventsOfType('MessageCompleted').filter(isContentMsg);
      expect(completedEvents.length).toBe(2);

      const supervisorMessage = (completedEvents[1] as any).message;
      expect(supervisorMessage.content.value).toContain('Based on the agent\'s work');
      expect(supervisorMessage.agentId).toBe('supervisor');

      stoppedEvents = eventBus.getEventsOfType('ProcessingStopped');
      expect(stoppedEvents.length).toBe(1);

      expect(processor.getIsProcessing()).toBe(false);
    });
  });

  describe('edge cases', () => {
    it('lifecycle event with empty description is handled', () => {
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: '[agent_spawned] code-agent-empty: ',
        agentId: 'supervisor',
        isFinal: false,
      });

      const lifecycleEvents = eventBus.getEventsOfType('AgentLifecycle');
      expect(lifecycleEvents.length).toBe(1);

      const event = lifecycleEvents[0] as AgentLifecycleEvent;
      expect(event.description).toBe('');
    });

    it('multiple agents with same prefix do not conflict', () => {
      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Agent 1 message',
        agentId: 'code-agent-1',
        isFinal: false,
      });

      gateway.simulateResponse({
        type: 'ANSWER_CHUNK',
        content: 'Agent 11 message',
        agentId: 'code-agent-11',
        isFinal: false,
      });

      const agents = agentStateManager.getAllAgents();
      expect(agents.length).toBe(2);
      expect(agents.some(a => a.agentId === 'code-agent-1')).toBe(true);
      expect(agents.some(a => a.agentId === 'code-agent-11')).toBe(true);
    });

    it('empty isFinal response without prior chunks completes gracefully', () => {
      gateway.simulateResponse({
        type: 'ANSWER',
        content: '',
        isFinal: true,
        agentId: 'code-agent-1',
      });

      // Should not crash, no message accumulated
      const messages = repository.findAll();
      expect(messages.length).toBe(0);
    });
  });
});
