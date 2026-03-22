// Tool group display component - shows multiple tool calls of same type grouped
import React from 'react';
import { Box, Text } from 'ink';
import { ChatMessage } from '../../../domain/message.js';
import { getToolPrefix, getKeyArgument, formatResultSummary } from './formatToolDisplay.js';

interface ToolGroupViewProps {
  messages: ChatMessage[];
}

/**
 * Check if keyArg is redundant given the summary.
 * Handles cases where both are truncated at different lengths.
 */
function isKeyArgRedundant(summary: string, keyArg: string): boolean {
  const s = summary.toLowerCase();
  const k = keyArg.toLowerCase();
  // Direct: summary contains keyArg
  if (s.includes(k)) return true;
  // Reverse: keyArg contains summary content (strip action prefix like "created: ")
  const stripped = s.replace(/^(created|updated|deleted|approved|started|completed|failed|spawned|stopped|restarted):\s*/i, '');
  if (stripped.length > 5 && k.includes(stripped)) return true;
  // Truncation: both start with the same text (common prefix > 10 chars)
  if (stripped.length > 10 && k.length > 10 && k.startsWith(stripped.slice(0, 10))) return true;
  return false;
}

/**
 * Deduplicate messages by keyArg for action-based tools.
 * For each unique keyArg, keeps only the LAST message.
 * Non action-based tools are returned unchanged.
 */
function deduplicateByKeyArg(messages: ChatMessage[]): ChatMessage[] {
  if (messages.length === 0) return messages;

  const firstTool = messages[0].toolCall;
  if (!firstTool) return messages;

  const toolName = firstTool.toolName.toLowerCase();

  // List of action-based tools that need deduplication
  const actionBasedTools = ['spawn_code_agent', 'manage_subtasks', 'manage_tasks'];
  if (!actionBasedTools.includes(toolName)) {
    return messages; // No deduplication for other tools
  }

  // Build map: keyArg → last message with that keyArg
  const keyArgMap = new Map<string, ChatMessage>();
  const keyArgOrder: string[] = []; // Track first appearance order

  for (const msg of messages) {
    if (!msg.toolCall) continue;

    const keyArg = getKeyArgument(msg.toolCall.toolName, msg.toolCall.arguments);
    const key = keyArg || '__no_key__'; // Messages without keyArg grouped separately

    if (!keyArgMap.has(key)) {
      keyArgOrder.push(key);
    }
    keyArgMap.set(key, msg); // Overwrites with LAST message
  }

  // Return deduplicated messages in order of first appearance
  return keyArgOrder.map(key => keyArgMap.get(key)!);
}

const ToolGroupViewComponent: React.FC<ToolGroupViewProps> = ({ messages }) => {
  if (messages.length === 0) return null;

  // Deduplicate before rendering
  const dedupedMessages = deduplicateByKeyArg(messages);

  if (dedupedMessages.length === 0) return null;

  const firstTool = dedupedMessages[0].toolCall;
  if (!firstTool) return null;

  const prefix = getToolPrefix(firstTool.toolName);
  const displayName = prefix.charAt(0).toUpperCase() + prefix.slice(1);

  // Check if any tool is still executing
  const isAnyExecuting = dedupedMessages.some(m => !m.isComplete);

  // Single completed tool call → compact inline format
  if (dedupedMessages.length === 1 && !isAnyExecuting && dedupedMessages[0].isComplete) {
    const msg = dedupedMessages[0];
    const toolCall = msg.toolCall!;
    const toolResult = msg.toolResult;
    const keyArg = getKeyArgument(toolCall.toolName, toolCall.arguments);
    const hasError = !!toolResult?.error;
    const summary = toolResult
      ? formatResultSummary(toolResult.toolName, toolResult.result, toolResult.error, toolResult.summary)
      : 'done';
    const shouldShowKeyArg = keyArg && !isKeyArgRedundant(summary, keyArg);

    return (
      <Box flexDirection="column" marginBottom={0}>
        {/* Inline summary */}
        <Box>
          <Text color="green">●</Text>
          <Text color="white" bold> {displayName}</Text>
          <Text color="gray"> → </Text>
          <Text color={hasError ? 'red' : 'white'}>{summary}</Text>
          {shouldShowKeyArg && <Text color="gray"> ({keyArg})</Text>}
        </Box>
        {/* Diff lines if available */}
        {toolResult?.diffLines && toolResult.diffLines.length > 0 && (
          <Box flexDirection="column" marginLeft={2}>
            {toolResult.diffLines.map((line, i) => (
              <Text key={i} color={line.type === '+' ? 'green' : line.type === '-' ? 'red' : 'gray'}>
                {line.type === ' ' ? '  ' : line.type + ' '}{line.content}
              </Text>
            ))}
          </Box>
        )}
      </Box>
    );
  }

  return (
    <Box flexDirection="column" marginBottom={1}>
      {/* Header: tool name */}
      <Box>
        <Text color={isAnyExecuting ? 'gray' : 'green'}>●</Text>
        <Text color="white" bold> {displayName}</Text>
      </Box>

      {/* Results: one line per tool call */}
      {dedupedMessages.map((message) => {
        const toolCall = message.toolCall;
        const toolResult = message.toolResult;
        if (!toolCall) return null;

        const keyArg = getKeyArgument(toolCall.toolName, toolCall.arguments);
        const isExecuting = !message.isComplete;

        if (isExecuting) {
          // Still executing - show spinner-like state
          return (
            <Box key={message.id} marginLeft={1}>
              <Text color="gray">└ </Text>
              <Text color="gray">{keyArg || '...'}</Text>
            </Box>
          );
        }

        // Completed - show result
        const hasError = !!toolResult?.error;
        const summary = toolResult
          ? formatResultSummary(toolResult.toolName, toolResult.result, toolResult.error, toolResult.summary)
          : 'done';
        const shouldShowKeyArg = keyArg && !isKeyArgRedundant(summary, keyArg);

        return (
          <Box key={message.id} flexDirection="column" marginLeft={1}>
            {/* Summary line */}
            <Box>
              <Text color={hasError ? 'red' : 'gray'}>└ </Text>
              <Text color={hasError ? 'red' : 'white'}>{summary}</Text>
              {shouldShowKeyArg && <Text color="gray"> ({keyArg})</Text>}
            </Box>
            {/* Diff lines if available */}
            {toolResult?.diffLines && toolResult.diffLines.length > 0 && (
              <Box flexDirection="column" marginLeft={3}>
                {toolResult.diffLines.map((line, i) => (
                  <Text key={i} color={line.type === '+' ? 'green' : line.type === '-' ? 'red' : 'gray'}>
                    {line.type === ' ' ? '  ' : line.type + ' '}{line.content}
                  </Text>
                ))}
              </Box>
            )}
          </Box>
        );
      })}
    </Box>
  );
};

// Memoize to prevent re-renders when messages haven't changed
export const ToolGroupView = React.memo(ToolGroupViewComponent);
