// Message component - renders the appropriate message type
import React from 'react';
import { ChatMessage } from '../../../domain/message.js';
import { UserMessage } from './UserMessage.js';
import { AssistantMessage } from './AssistantMessage.js';
import { ReasoningBlock } from './ReasoningBlock.js';
import { ToolCallView } from '../tools/ToolCallView.js';
import { ToolResultView } from '../tools/ToolResultView.js';

interface MessageProps {
  message: ChatMessage;
}

const MessageComponent: React.FC<MessageProps> = ({ message }) => {
  // Reasoning message
  if (message.reasoning) {
    return <ReasoningBlock message={message} />;
  }

  // Tool message
  if (message.role === 'tool') {
    return (
      <>
        <ToolCallView message={message} />
        {message.toolResult && <ToolResultView message={message} />}
      </>
    );
  }

  // User message
  if (message.role === 'user') {
    return <UserMessage message={message} />;
  }

  // Assistant message
  if (message.role === 'assistant') {
    return <AssistantMessage message={message} />;
  }

  return null;
};

// Memoize to prevent re-renders when message hasn't changed
export const Message = React.memo(MessageComponent);
