// AgentTabs - informational display of agent states (Supervisor, Code Agents)
// Only visible when there are multiple agents active
import React from 'react';
import { Box, Text } from 'ink';
import { AgentState } from '../../../infrastructure/state/AgentStateManager.js';

interface AgentTabsProps {
  agents: AgentState[];
  activeAgentId: string;
}

/**
 * Get status icon and color for agent state.
 * Returns null if agent is idle (no icon shown).
 */
function getStatusIcon(agent: AgentState): { icon: string; color: string } | null {
  if (agent.completionStatus === 'completed') return { icon: '✓', color: 'green' };
  if (agent.completionStatus === 'failed') return { icon: '✗', color: 'red' };
  if (agent.completionStatus === 'running' || agent.status === 'processing') {
    return { icon: '●', color: 'yellow' };
  }
  return null;
}

/**
 * Get display label for agent.
 * Supervisor: "Supervisor"
 * Code Agent: "abc: Task description (truncated)"
 */
function getLabel(agent: AgentState): string {
  if (agent.role === 'supervisor') return 'Supervisor';

  const shortId = agent.agentId.replace('code-agent-', '');
  if (agent.taskDescription) {
    const truncated = agent.taskDescription.length > 25
      ? agent.taskDescription.slice(0, 25) + '…'
      : agent.taskDescription;
    return `${shortId}: ${truncated}`;
  }
  return shortId;
}

/**
 * Compact agent tab bar for multi-agent mode.
 * Shows: [Supervisor ●] [abc: Fix imports ✓] [def: Refactor ●]
 *
 * Only rendered when agents.length > 1.
 * Informational display only (no Shift+Tab interaction).
 */
export const AgentTabs: React.FC<AgentTabsProps> = ({ agents, activeAgentId }) => {
  if (agents.length <= 1) return null;

  return (
    <Box marginBottom={1} paddingX={1}>
      {agents.map((agent) => {
        const isActive = agent.agentId === activeAgentId;
        const statusIcon = getStatusIcon(agent);
        const label = getLabel(agent);

        return (
          <Box key={agent.agentId} marginRight={1}>
            <Text color={isActive ? 'cyan' : 'gray'} bold={isActive} inverse={isActive}>
              [{label}
              {statusIcon ? ' ' : ''}
            </Text>
            {statusIcon && (
              <Text color={isActive ? statusIcon.color : statusIcon.color} inverse={isActive}>
                {statusIcon.icon}
              </Text>
            )}
            <Text color={isActive ? 'cyan' : 'gray'} bold={isActive} inverse={isActive}>
              ]
            </Text>
          </Box>
        );
      })}
      <Text color="gray" dimColor> Shift+Tab</Text>
    </Box>
  );
};
