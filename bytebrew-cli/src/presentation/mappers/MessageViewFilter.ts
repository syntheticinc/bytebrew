import { MessageViewModel } from './MessageViewMapper.js';
import { isLifecycleMessage, isSeparatorMessage } from '../utils/messageClassifiers.js';

export type ViewMode =
  | { type: 'supervisor' }
  | { type: 'agent'; agentId: string };

// Debug logging (enabled via BYTEBREW_DEBUG_FILTER=1)
const DEBUG_FILTER = process.env.BYTEBREW_DEBUG_FILTER === '1';
function debugLog(msg: string): void {
  if (DEBUG_FILTER) {
    process.stderr.write(`[FILTER] ${msg}\n`);
  }
}

// Helper functions

function isForAgent(content: string, agentId: string): boolean {
  const shortId = agentId.replace('code-agent-', '');
  return content.includes(shortId);
}

function isAgentToolOrReasoning(msg: MessageViewModel): boolean {
  const isAgent = msg.agentId !== undefined && msg.agentId !== 'supervisor';
  return isAgent && (!!msg.toolCall || !!msg.toolResult || !!msg.reasoning);
}

/**
 * Filters messages based on view mode.
 *
 * Supervisor view: shows user messages, supervisor messages (all), lifecycle events.
 * Hides agent tools/results/reasoning, separators, and agent text messages.
 *
 * Agent view: shows messages for specific agent (all), lifecycle events for this agent.
 * Hides separators, user messages, and other agents' messages.
 */
export function filterMessagesForView(
  messages: MessageViewModel[],
  view: ViewMode
): MessageViewModel[] {
  if (view.type === 'supervisor') {
    if (DEBUG_FILTER) {
      debugLog(`=== Supervisor filter: ${messages.length} messages ===`);
      for (const m of messages) {
        const hasTool = m.toolCall ? `tool=${m.toolCall.toolName}` : '';
        const hasResult = m.toolResult ? `result=${m.toolResult.toolName}` : '';
        const hasReason = m.reasoning ? 'reasoning' : '';
        debugLog(`  id=${m.id.slice(0,8)} role=${m.role} agentId=${m.agentId ?? 'undefined'} ${hasTool} ${hasResult} ${hasReason} content=${m.content.slice(0,60)}`);
      }
    }
    return messages.filter((msg) => {
      // Separators → HIDE in supervisor view (check FIRST, before agentId)
      // Lifecycle events already indicate agent boundaries.
      // Separators precede agent tool calls which are hidden, so they'd be orphaned.
      // "─── Supervisor ───" has agentId='supervisor' and would pass the next check.
      if (isSeparatorMessage(msg.content)) return false;

      // User messages
      if (msg.agentId === undefined && msg.role === 'user') return true;

      // Supervisor messages (all)
      if (msg.agentId === 'supervisor') return true;

      // Lifecycle events (all) — mark agent spawn/complete/fail
      if (isLifecycleMessage(msg.content)) return true;

      // Agent tool/result/reasoning → HIDE
      if (isAgentToolOrReasoning(msg)) {
        debugLog(`  HIDDEN: agentId=${msg.agentId} tool=${msg.toolCall?.toolName || msg.toolResult?.toolName || 'reasoning'}`);
        return false;
      }

      // Agent text messages → HIDE in supervisor view.
      // Agent work is visible through lifecycle messages and spawn_code_agent tool results.
      const isAgentMessage = msg.agentId !== undefined && msg.agentId !== 'supervisor';
      if (isAgentMessage) {
        debugLog(`  HIDDEN: agent text agentId=${msg.agentId} content=${msg.content.slice(0,40)}`);
        return false;
      }

      // Unknown message types without agentId → show by default (fail-safe)
      debugLog(`  PASS (unknown): role=${msg.role} agentId=${msg.agentId ?? 'undefined'} content=${msg.content.slice(0,40)}`);
      return true;
    });
  }

  // Agent view — isolated workspace: only this agent's messages + its lifecycle events.
  // User messages belong to supervisor conversation and are NOT shown here.
  const targetAgentId = view.agentId;
  return messages.filter((msg) => {
    // Separators: never show in agent view — the entire tab is this agent's workspace
    if (isSeparatorMessage(msg.content)) return false;

    // Messages from this agent (tools, reasoning, answers)
    if (msg.agentId === targetAgentId) return true;

    // Lifecycle for this agent (spawn, complete, fail)
    if (isLifecycleMessage(msg.content) && isForAgent(msg.content, targetAgentId)) {
      return true;
    }

    return false;
  });
}
