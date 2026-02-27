// Assistant message component - renders markdown content
import React, { useMemo } from 'react';
import { Box, Text } from 'ink';
import { ChatMessage } from '../../../domain/message.js';
import { isLifecycleMessage, getLifecycleColor, isSeparatorMessage } from '../../utils/messageClassifiers.js';
import { renderMarkdown } from '../../utils/markdown.js';

interface AssistantMessageProps {
  message: ChatMessage;
}

export const AssistantMessage: React.FC<AssistantMessageProps> = ({ message }) => {
  const isStreaming = message.isStreaming && !message.isComplete;

  // Render markdown only when streaming is complete
  const renderedContent = useMemo(() => {
    return renderMarkdown(message.content);
  }, [message.content]);

  // During streaming, don't render (progress shown in StatusBar)
  if (isStreaming) {
    return null;
  }

  // Lifecycle events (spawned, completed, failed) - special rendering without prefix
  if (isLifecycleMessage(message.content)) {
    return (
      <Box marginBottom={1}>
        <Text color={getLifecycleColor(message.content)}>{message.content}</Text>
      </Box>
    );
  }

  // Separator messages — rendered without prefix (dimmed gray)
  if (isSeparatorMessage(message.content)) {
    return (
      <Box marginBottom={1}>
        <Text dimColor>{message.content}</Text>
      </Box>
    );
  }

  // Agent messages (not supervisor, not lifecycle) - with │ prefix and agent label
  const isAgentMessage = message.agentId !== undefined && message.agentId !== 'supervisor';
  if (isAgentMessage) {
    const shortId = message.agentId!.replace('code-agent-', '');
    return (
      <Box flexDirection="column" marginBottom={1}>
        <Text color="gray">{`│ Code Agent [${shortId}]`}</Text>
        <Box flexDirection="row">
          <Text color="gray">{'│ '}</Text>
          <Box flexDirection="column" flexGrow={1}>
            <Text>{renderedContent}</Text>
          </Box>
        </Box>
      </Box>
    );
  }

  // Supervisor messages - cyan > prefix (unchanged)
  return (
    <Box flexDirection="row" marginBottom={1}>
      <Text bold color="cyan">{'> '}</Text>
      <Box flexDirection="column" flexGrow={1}>
        <Text>{renderedContent}</Text>
      </Box>
    </Box>
  );
};
