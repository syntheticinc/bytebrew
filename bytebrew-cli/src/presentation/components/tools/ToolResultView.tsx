// Tool result display component - memoized to prevent re-renders
import React, { useMemo } from 'react';
import { Box, Text } from 'ink';
import { ChatMessage } from '../../../domain/message.js';
import { formatResultSummary } from './formatToolDisplay.js';

interface ToolResultViewProps {
  message: ChatMessage;
}

export const ToolResultView: React.FC<ToolResultViewProps> = React.memo(({ message }) => {
  const toolResult = message.toolResult;
  if (!toolResult) return null;

  const hasError = !!toolResult.error;

  // Memoize to prevent re-computation on every render
  const summary = useMemo(() =>
    formatResultSummary(
      toolResult.toolName,
      toolResult.result,
      toolResult.error,
      toolResult.summary
    ),
    [toolResult.toolName, toolResult.result, toolResult.error, toolResult.summary]
  );

  return (
    <Box marginLeft={1} marginBottom={1}>
      <Text color={hasError ? 'red' : 'gray'}>└ {summary}</Text>
    </Box>
  );
});
