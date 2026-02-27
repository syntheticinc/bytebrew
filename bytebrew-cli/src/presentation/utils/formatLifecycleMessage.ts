/**
 * Format agent lifecycle event into human-readable message.
 * Used to display lifecycle events in the unified message stream.
 */
export function formatLifecycleMessage(lifecycleType: string, agentId: string, description: string): string {
  const shortId = agentId.replace('code-agent-', '');
  const label = agentId === 'supervisor' ? 'Supervisor' : `Code Agent [${shortId}]`;

  switch (lifecycleType) {
    case 'agent_spawned': {
      // First line only — full description is in [Task] message
      const title = description.split('\n')[0];
      return `+ ${label} spawned: "${title}"`;
    }
    case 'agent_completed': {
      const title = description.startsWith('Completed: ')
        ? description.split('\n')[0].replace('Completed: ', '')
        : description;
      return `✓ ${label} completed: "${title}"`;
    }
    case 'agent_failed': {
      const failTitle = description.startsWith('Failed: ')
        ? description.split('\n')[0].replace('Failed: ', '')
        : description;
      return `✗ ${label} failed: "${failTitle}"`;
    }
    case 'agent_restarted':
      return `↻ ${label} restarted: "${description}"`;
    default:
      return `[${lifecycleType}] ${label}: ${description}`;
  }
}
