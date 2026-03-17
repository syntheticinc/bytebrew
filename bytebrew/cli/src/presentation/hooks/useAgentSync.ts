// useAgentSync hook - manages multi-agent UI state synchronization
import { useState, useEffect } from 'react';
import { AgentStateManager, AgentState } from '../../infrastructure/state/AgentStateManager.js';
import { IEventBus } from '../../domain/ports/IEventBus.js';

export interface UseAgentSyncOptions {
  agentStateManager: AgentStateManager;
  eventBus: IEventBus;
  isProcessing: boolean;
  isBlocked: boolean; // true when askUserQuestion shown - blocks polling
}

export interface UseAgentSyncResult {
  agents: AgentState[];
  activeAgentId: string;
}

/**
 * Hook that manages multi-agent UI state synchronization.
 * Polls agent state during processing (when not blocked) and
 * subscribes to AgentLifecycle events for immediate UI refresh.
 *
 * Message creation (lifecycle + task messages) is handled by
 * StreamProcessorService.handleLifecycleEvent() — not here.
 */
export function useAgentSync(options: UseAgentSyncOptions): UseAgentSyncResult {
  const { agentStateManager, eventBus, isProcessing, isBlocked } = options;

  const [agents, setAgents] = useState<AgentState[]>([]);
  const [activeAgentId, setActiveAgentId] = useState('supervisor');

  // Subscribe to AgentLifecycle events for immediate UI refresh.
  // Messages are created by StreamProcessorService.handleLifecycleEvent().
  // State update (updateLifecycle) is idempotent — safe to call from both places.
  useEffect(() => {
    const unsubLifecycle = eventBus.subscribe('AgentLifecycle', (event) => {
      agentStateManager.updateLifecycle(event.agentId, event.lifecycleType, event.description);

      const allAgents = agentStateManager.getAllAgents();
      if (allAgents.length > 0) {
        setAgents([...allAgents]);
        setActiveAgentId(agentStateManager.activeAgentId);
      }
    });

    return () => {
      unsubLifecycle();
    };
  }, [eventBus, agentStateManager]);

  // Sync agent state periodically during processing
  useEffect(() => {
    if (!isProcessing) return;

    const interval = setInterval(() => {
      // Skip agent state updates when blocked (e.g., QuestionnairePrompt is shown) -
      // setAgents creates new array each time, triggering re-render
      // which causes Ink to redraw and prevent terminal scrolling
      if (!isBlocked) {
        const allAgents = agentStateManager.getAllAgents();
        if (allAgents.length > 0) {
          setAgents([...allAgents]);
          setActiveAgentId(agentStateManager.activeAgentId);
        }
      }
    }, 500);

    return () => clearInterval(interval);
  }, [isProcessing, agentStateManager, isBlocked]);

  return {
    agents,
    activeAgentId,
  };
}
