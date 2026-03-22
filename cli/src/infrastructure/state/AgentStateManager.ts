// AgentStateManager - manages per-agent state for multi-agent support
import { MessageId } from '../../domain/value-objects/MessageId.js';

export type AgentRole = 'supervisor' | 'code';

export interface AgentState {
  agentId: string;
  role: AgentRole;
  name: string;
  status: 'idle' | 'processing';
  // Per-agent streaming state
  currentMessageId: MessageId | null;
  currentReasoningId: MessageId | null;
  // Multi-agent UI state
  taskDescription: string;
  completionStatus: 'idle' | 'running' | 'completed' | 'failed';
  lastMessageAt: Date;
}

/**
 * Manages per-agent state for multi-agent scenarios.
 * Each agent (supervisor, code-agent-xxx) has its own streaming state
 * so concurrent events from different agents don't interfere.
 */
export class AgentStateManager {
  private agents = new Map<string, AgentState>();
  private _activeAgentId = 'supervisor';

  /**
   * Get or create agent state for the given agentId
   */
  getOrCreateAgent(agentId: string): AgentState {
    let agent = this.agents.get(agentId);
    if (!agent) {
      agent = {
        agentId,
        role: agentId === 'supervisor' ? 'supervisor' : 'code',
        name: agentId === 'supervisor' ? 'Supervisor' : `Code Agent ${agentId.replace('code-agent-', '')}`,
        status: 'idle',
        currentMessageId: null,
        currentReasoningId: null,
        taskDescription: '',
        completionStatus: 'idle',
        lastMessageAt: new Date(),
      };
      this.agents.set(agentId, agent);
    }
    return agent;
  }

  /**
   * Get the currently active agent
   */
  getActiveAgent(): AgentState {
    return this.getOrCreateAgent(this._activeAgentId);
  }

  /**
   * Set active agent by ID
   */
  setActiveAgent(agentId: string): void {
    this._activeAgentId = agentId;
    this.getOrCreateAgent(agentId);
  }

  /**
   * Get active agent ID
   */
  get activeAgentId(): string {
    return this._activeAgentId;
  }

  /**
   * Cycle to next agent (for Shift+Tab).
   * Returns the new active agent ID.
   */
  cycleNextAgent(): string {
    const ids = Array.from(this.agents.keys());
    if (ids.length <= 1) return this._activeAgentId;

    const currentIndex = ids.indexOf(this._activeAgentId);
    // If current agent not found in map, start from first
    const nextIndex = currentIndex < 0 ? 0 : (currentIndex + 1) % ids.length;
    this._activeAgentId = ids[nextIndex];
    return this._activeAgentId;
  }

  /**
   * Get all agents
   */
  getAllAgents(): AgentState[] {
    return Array.from(this.agents.values());
  }

  /**
   * Get per-agent message tracking state
   */
  getCurrentMessageId(agentId: string): MessageId | null {
    return this.getOrCreateAgent(agentId).currentMessageId;
  }

  setCurrentMessageId(agentId: string, id: MessageId | null): void {
    this.getOrCreateAgent(agentId).currentMessageId = id;
  }

  getCurrentReasoningId(agentId: string): MessageId | null {
    return this.getOrCreateAgent(agentId).currentReasoningId;
  }

  setCurrentReasoningId(agentId: string, id: MessageId | null): void {
    this.getOrCreateAgent(agentId).currentReasoningId = id;
  }

  /**
   * Reset all agent states (for new message)
   */
  resetAll(): void {
    for (const agent of this.agents.values()) {
      agent.currentMessageId = null;
      agent.currentReasoningId = null;
    }
  }

  /**
   * Check if there are multiple agents
   */
  hasMultipleAgents(): boolean {
    return this.agents.size > 1;
  }

  /**
   * Update agent lifecycle state based on lifecycle event
   */
  updateLifecycle(agentId: string, lifecycleType: string, description: string): void {
    const agent = this.getOrCreateAgent(agentId);
    agent.lastMessageAt = new Date();
    switch (lifecycleType) {
      case 'agent_spawned':
        agent.completionStatus = 'running';
        // First line only — used for separator labels and short UI display
        agent.taskDescription = description.split('\n')[0];
        break;
      case 'agent_completed':
        agent.completionStatus = 'completed';
        break;
      case 'agent_failed':
        agent.completionStatus = 'failed';
        break;
      case 'agent_restarted':
        agent.completionStatus = 'running';
        if (description) agent.taskDescription = description;
        break;
    }
  }

  /**
   * Touch agent to update lastMessageAt (called when agent receives content)
   */
  touchAgent(agentId: string): void {
    this.getOrCreateAgent(agentId).lastMessageAt = new Date();
  }
}
