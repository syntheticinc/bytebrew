// User message component - minimal style
import React from 'react';
import { Box, Text } from 'ink';
import { ChatMessage } from '../../../domain/message.js';

interface UserMessageProps {
  message: ChatMessage;
}

export const UserMessage: React.FC<UserMessageProps> = ({ message }) => {
  return (
    <Box marginBottom={1} flexDirection="row">
      <Text color="green" bold>{'> '}</Text>
      <Text color="green">{message.content}</Text>
    </Box>
  );
};
