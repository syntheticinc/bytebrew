// Tool call display component - Claude CLI style
import React from 'react';
import { Box, Text } from 'ink';
import { ChatMessage } from '../../../domain/message.js';
import { getToolPrefix, getKeyArgument } from './formatToolDisplay.js';

interface ToolCallViewProps {
  message: ChatMessage;
}

export const ToolCallView: React.FC<ToolCallViewProps> = ({ message }) => {
  const toolCall = message.toolCall;
  if (!toolCall) return null;

  const isExecuting = !message.isComplete;
  const prefix = getToolPrefix(toolCall.toolName);
  const keyArg = getKeyArgument(toolCall.toolName, toolCall.arguments);

  // Capitalize first letter
  const displayName = prefix.charAt(0).toUpperCase() + prefix.slice(1);

  return (
    <Box marginBottom={isExecuting ? 1 : 0}>
      {/* Gray dot while executing, green dot when complete */}
      <Text color={isExecuting ? 'gray' : 'green'}>●</Text>
      <Text color="white" bold> {displayName}</Text>
      {keyArg && (
        <Text color="gray">({keyArg})</Text>
      )}
    </Box>
  );
};
