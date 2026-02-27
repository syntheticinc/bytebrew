// Reasoning/thinking block component - gray border, no label
import React from 'react';
import { Box, Text } from 'ink';
import { ChatMessage } from '../../../domain/message.js';

interface ReasoningBlockProps {
  message: ChatMessage;
}

const MAX_THINKING_LINES = 4;

export const ReasoningBlock: React.FC<ReasoningBlockProps> = ({ message }) => {
  const reasoning = message.reasoning;
  if (!reasoning) return null;

  const isStreaming = !reasoning.isComplete;

  // During streaming, don't render (progress shown in StatusBar)
  if (isStreaming) {
    return null;
  }

  // After streaming is complete, show the full content
  const lines = reasoning.thinking.split('\n');
  const truncated = lines.length > MAX_THINKING_LINES;
  const displayText = truncated
    ? lines.slice(0, MAX_THINKING_LINES).join('\n') + '...'
    : reasoning.thinking;

  return (
    <Box marginBottom={1} borderStyle="round" borderColor="gray" paddingX={1}>
      <Text color="white">{displayText}</Text>
    </Box>
  );
};
