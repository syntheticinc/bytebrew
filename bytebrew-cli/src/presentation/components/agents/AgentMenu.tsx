// AgentMenu - vertical menu of agents (opened with Shift+Tab)
import React, { useState, useMemo } from 'react';
import { Box, Text, useInput } from 'ink';
import { AgentState } from '../../../infrastructure/state/AgentStateManager.js';

interface AgentMenuProps {
  agents: AgentState[];
  currentViewAgentId: string;
  isOpen: boolean;
  onSelect: (agentId: string) => void;
  onClose: () => void;
}

/**
 * Get status icon and color for agent state.
 */
function getStatusIcon(agent: AgentState): { icon: string; color: string } {
  if (agent.agentId === 'supervisor') return { icon: '★', color: 'yellow' };
  if (agent.completionStatus === 'completed') return { icon: '✓', color: 'green' };
  if (agent.completionStatus === 'failed') return { icon: '✗', color: 'red' };
  if (agent.completionStatus === 'running' || agent.status === 'processing') {
    return { icon: '●', color: 'yellow' };
  }
  return { icon: '○', color: 'gray' };
}

/**
 * Get display label for agent.
 */
function getLabel(agent: AgentState): string {
  if (agent.role === 'supervisor') return 'Supervisor';

  const shortId = agent.agentId.replace('code-agent-', '');
  if (agent.taskDescription) {
    const truncated = agent.taskDescription.length > 40
      ? agent.taskDescription.slice(0, 40) + '…'
      : agent.taskDescription;
    return `${shortId}: ${truncated}`;
  }
  return shortId;
}

/**
 * Format relative time (just now, Nm ago, Nh ago, Nd ago)
 */
function formatRelativeTime(date: Date): string {
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);

  if (diffSec < 60) return 'just now';

  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;

  const diffHours = Math.floor(diffMin / 60);
  if (diffHours < 24) return `${diffHours}h ago`;

  const diffDays = Math.floor(diffHours / 24);
  return `${diffDays}d ago`;
}

/**
 * Vertical agent menu (opened with Shift+Tab).
 *
 * When closed: shows compact hint "▸ N chats (Shift+Tab)" if agents > 1
 * When open: shows vertical list of agents sorted by lastMessageAt (DESC)
 *
 * Keyboard:
 * - Arrow Up/Down: navigate
 * - Enter: select agent
 * - Esc: close menu
 */
export const AgentMenu: React.FC<AgentMenuProps> = ({
  agents,
  currentViewAgentId,
  isOpen,
  onSelect,
  onClose,
}) => {
  // Sort agents by lastMessageAt DESC (most recent first), memoized
  const displayAgents = useMemo(() => {
    const sorted = [...agents].sort((a, b) =>
      b.lastMessageAt.getTime() - a.lastMessageAt.getTime()
    );
    return sorted.slice(0, 10);
  }, [agents]);

  // Find current selection index (default to current view agent)
  const initialIndex = displayAgents.findIndex(a => a.agentId === currentViewAgentId);
  const [selectedIndex, setSelectedIndex] = useState(initialIndex >= 0 ? initialIndex : 0);

  // Keyboard input (only when menu is open)
  useInput((input, key) => {
    if (key.escape) {
      onClose();
      return;
    }
    if (key.return) {
      onSelect(displayAgents[selectedIndex].agentId);
      onClose();
      return;
    }
    if (key.upArrow) {
      setSelectedIndex(prev => Math.max(0, prev - 1));
      return;
    }
    if (key.downArrow) {
      setSelectedIndex(prev => Math.min(displayAgents.length - 1, prev + 1));
      return;
    }
  }, { isActive: isOpen });

  // Closed state: compact hint (only if multiple agents)
  if (!isOpen) {
    if (agents.length <= 1) return null;
    return (
      <Box paddingX={1} marginBottom={1}>
        <Text color="gray" dimColor>▸ {agents.length} chats (Shift+Tab)</Text>
      </Box>
    );
  }

  // Open state: vertical menu
  return (
    <Box flexDirection="column" paddingX={1} marginBottom={1} borderStyle="single" borderColor="cyan">
      {displayAgents.map((agent, index) => {
        const isSelected = index === selectedIndex;
        const statusIcon = getStatusIcon(agent);
        const label = getLabel(agent);
        const relativeTime = formatRelativeTime(agent.lastMessageAt);

        return (
          <Box key={agent.agentId} justifyContent="space-between">
            <Box>
              <Text color={statusIcon.color} inverse={isSelected}>{statusIcon.icon} </Text>
              <Text color={isSelected ? 'cyan' : 'white'} inverse={isSelected} bold={isSelected}>
                {label}
              </Text>
            </Box>
            <Text color="gray" inverse={isSelected} dimColor>  {relativeTime}</Text>
          </Box>
        );
      })}
    </Box>
  );
};
