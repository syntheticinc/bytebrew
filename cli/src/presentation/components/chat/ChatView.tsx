// ChatView - chat view with only complete messages (static rendering)
import React, { useMemo, useRef } from 'react';
import { Box, Text, Static } from 'ink';
import { MessageViewModel } from '../../mappers/MessageViewMapper.js';
import { DisplayItem, mapMessagesToDisplayItems } from '../../mappers/DisplayItemMapper.js';
import { IToolRenderingService } from '../../../domain/ports/IToolRenderingService.js';
import { Message } from './Message.js';
import { ToolGroupView } from '../tools/ToolGroupView.js';

interface ChatViewProps {
  messages: MessageViewModel[];
  renderingService?: IToolRenderingService;
  maxDisplayMessages?: number;
}

/**
 * ChatView that ONLY renders complete messages.
 * Since all messages are complete, we use Ink's Static component
 * for all of them - they're rendered once and never updated.
 *
 * This eliminates the complexity of determining static vs dynamic
 * items and guarantees no re-renders during streaming.
 */
export const ChatView: React.FC<ChatViewProps> = ({
  messages,
  renderingService,
  maxDisplayMessages = 50,
}) => {
  // Only display last N messages to prevent terminal overflow
  const displayMessages = useMemo(
    () => messages.slice(-maxDisplayMessages),
    [messages, maxDisplayMessages]
  );

  // Map messages to display items (grouping logic extracted to mapper)
  const displayItems = useMemo(
    () => mapMessagesToDisplayItems(displayMessages, renderingService),
    [displayMessages, renderingService]
  );

  // Append-only tracking: Ink's Static uses an index-based watermark internally
  // (items.slice(index)). When a tool message completes AFTER lifecycle messages
  // that were created later in orderedIds, findComplete() inserts it BEFORE them.
  // Static's watermark has already advanced past that position, so the tool is
  // never rendered. By only appending new items to the end, we guarantee Static
  // always sees a growing array and renders every item exactly once.
  // Tab switch resets refs via key={currentViewAgentId} in ChatApp.
  const renderedItemsRef = useRef<DisplayItem[]>([]);
  const renderedKeysRef = useRef(new Set<string>());

  for (const item of displayItems) {
    if (!renderedKeysRef.current.has(item.key)) {
      renderedItemsRef.current.push(item);
      renderedKeysRef.current.add(item.key);
    }
  }

  // Welcome message is always first - Static needs consistent item positions
  const welcomeItem: DisplayItem = {
    type: 'message',
    message: { id: 'welcome', role: 'system', content: '', timestamp: new Date(), isStreaming: false, isComplete: true },
    key: 'welcome'
  };

  // Always include welcome first, then append-only display items
  const allItems = [welcomeItem, ...renderedItemsRef.current];

  // All messages are complete, so we use Static for everything
  // This means they're rendered once and never re-rendered
  return (
    <Box flexDirection="column">
      <Static items={allItems}>
        {(item, index) => {
          // Welcome message
          if (item.key === 'welcome') {
            return (
              <Box key="welcome" flexDirection="column" marginY={1}>
                <Text color="gray">
                  Welcome to ByteBrew CLI! Type a message to start chatting.
                </Text>
                <Text color="gray" dimColor>
                  Press Ctrl+C to exit.
                </Text>
              </Box>
            );
          }

          // Use unique key combining type and index to avoid duplicates
          const uniqueKey = `${item.type}-${index}-${item.key}`;

          if (item.type === 'customTool') {
            // Render with custom tool renderer
            const renderer = renderingService?.getRenderer(item.toolName);
            if (renderer && item.message.toolCall) {
              const rendered = renderer({
                toolName: item.toolName,
                arguments: item.message.toolCall.arguments,
                result: item.message.toolResult?.result,
                error: item.message.toolResult?.error,
                isExecuting: !item.message.isComplete,
              });
              if (rendered) {
                return <Box key={uniqueKey}>{rendered}</Box>;
              }
            }
            // Fallback to regular message display if renderer fails
            const chatMessage = {
              id: item.message.id,
              role: item.message.role,
              content: item.message.content,
              timestamp: item.message.timestamp,
              isStreaming: item.message.isStreaming,
              isComplete: item.message.isComplete,
              toolCall: item.message.toolCall,
              toolResult: item.message.toolResult,
              reasoning: item.message.reasoning,
              agentId: item.message.agentId,
            };
            return <Message key={uniqueKey} message={chatMessage} />;
          }

          if (item.type === 'toolGroup') {
            // Cast to ChatMessage for compatibility with existing ToolGroupView
            const chatMessages = item.messages.map(m => ({
              id: m.id,
              role: m.role,
              content: m.content,
              timestamp: m.timestamp,
              isStreaming: m.isStreaming,
              isComplete: m.isComplete,
              toolCall: m.toolCall,
              toolResult: m.toolResult,
              reasoning: m.reasoning,
              agentId: m.agentId,
            }));
            return (
              <ToolGroupView
                key={uniqueKey}
                messages={chatMessages}
              />
            );
          }

          // Regular message
          const chatMessage = {
            id: item.message.id,
            role: item.message.role,
            content: item.message.content,
            timestamp: item.message.timestamp,
            isStreaming: item.message.isStreaming,
            isComplete: item.message.isComplete,
            toolCall: item.message.toolCall,
            toolResult: item.message.toolResult,
            reasoning: item.message.reasoning,
            agentId: item.message.agentId,
          };
          return <Message key={uniqueKey} message={chatMessage} />;
        }}
      </Static>
    </Box>
  );
};
