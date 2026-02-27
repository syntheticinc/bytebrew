// useConversation hook - bridge between React and application layer
import { useState, useEffect, useCallback, useRef } from 'react';
import { Message } from '../../domain/entities/Message.js';
import { IMessageRepository } from '../../domain/ports/IMessageRepository.js';
import { IEventBus } from '../../domain/ports/IEventBus.js';
import { StreamProcessorService } from '../../application/services/StreamProcessorService.js';
import { MessageAccumulatorService } from '../../application/services/MessageAccumulatorService.js';
import { MessageViewModel, toMessageViewModels } from '../mappers/MessageViewMapper.js';
import { useViewStore } from '../store/viewStore.js';

export interface UseConversationOptions {
  streamProcessor: StreamProcessorService;
  messageRepository: IMessageRepository;
  accumulator: MessageAccumulatorService;
  eventBus: IEventBus;
}

export interface UseConversationResult {
  messages: MessageViewModel[];
  isProcessing: boolean;
  sendMessage: (content: string) => void;
  cancel: () => void;
  clearMessages: () => void;
}

// Throttle interval for token updates (ms)
const TOKEN_UPDATE_INTERVAL = 150;

/**
 * Hook that connects React components to the application layer.
 * Subscribes to events and repository changes, providing a clean API
 * for the UI components.
 */
export function useConversation(options: UseConversationOptions): UseConversationResult {
  const { streamProcessor, messageRepository, accumulator, eventBus } = options;

  const [messages, setMessages] = useState<MessageViewModel[]>([]);
  const lastTokenUpdateRef = useRef<number>(0);
  const tokenUpdateTimerRef = useRef<NodeJS.Timeout | null>(null);

  // View store actions
  const setProcessing = useViewStore((state) => state.setProcessing);
  const setStreaming = useViewStore((state) => state.setStreaming);
  const setTokenCounts = useViewStore((state) => state.setTokenCounts);
  const addActiveToolCall = useViewStore((state) => state.addActiveToolCall);
  const removeActiveToolCall = useViewStore((state) => state.removeActiveToolCall);
  const clearActiveToolCalls = useViewStore((state) => state.clearActiveToolCalls);
  const isProcessing = useViewStore((state) => state.isProcessing);

  // Update messages from repository
  const updateMessages = useCallback(() => {
    const completeMessages = messageRepository.findComplete();
    setMessages(toMessageViewModels(completeMessages));
  }, [messageRepository]);

  // Throttled token update
  const updateTokensThrottled = useCallback((tokens: { input: number; output: number }) => {
    const now = Date.now();
    const timeSinceLastUpdate = now - lastTokenUpdateRef.current;

    if (timeSinceLastUpdate >= TOKEN_UPDATE_INTERVAL) {
      // Update immediately
      setTokenCounts(tokens);
      lastTokenUpdateRef.current = now;
    } else if (!tokenUpdateTimerRef.current) {
      // Schedule update
      tokenUpdateTimerRef.current = setTimeout(() => {
        setTokenCounts(accumulator.getTokenCounts());
        lastTokenUpdateRef.current = Date.now();
        tokenUpdateTimerRef.current = null;
      }, TOKEN_UPDATE_INTERVAL - timeSinceLastUpdate);
    }
  }, [setTokenCounts, accumulator]);

  // Subscribe to events
  useEffect(() => {
    const unsubscribers: (() => void)[] = [];

    // Message completed - update messages list
    unsubscribers.push(
      eventBus.subscribe('MessageCompleted', () => {
        updateMessages();
      })
    );

    // Processing state
    unsubscribers.push(
      eventBus.subscribe('ProcessingStarted', () => {
        setProcessing(true);
        setStreaming(true);
      })
    );

    unsubscribers.push(
      eventBus.subscribe('ProcessingStopped', () => {
        setProcessing(false);
        setStreaming(false);
        // Flush final token counts
        setTokenCounts(accumulator.getTokenCounts());
        // Clear any stale tool calls (e.g. server-side tools whose TOOL_RESULT never arrived)
        clearActiveToolCalls();
      })
    );

    // Streaming progress - throttled token updates
    unsubscribers.push(
      eventBus.subscribe('StreamingProgress', (event) => {
        updateTokensThrottled(event.totalTokens);
      })
    );

    // Tool execution
    unsubscribers.push(
      eventBus.subscribe('ToolExecutionStarted', (event) => {
        const toolCall = {
          callId: event.execution.callId,
          toolName: event.execution.toolName,
          arguments: event.execution.arguments,
        };
        addActiveToolCall(toolCall);
      })
    );

    unsubscribers.push(
      eventBus.subscribe('ToolExecutionCompleted', (event) => {
        removeActiveToolCall(event.execution.callId);
      })
    );

    // Error handling - update messages when error occurs (error is saved as a message)
    unsubscribers.push(
      eventBus.subscribe('ErrorOccurred', () => {
        // Error message was already added to repository by ErrorHandler
        // Just update the messages to show it
        updateMessages();
        // Also stop processing indicator
        setProcessing(false);
        setStreaming(false);
      })
    );

    return () => {
      unsubscribers.forEach((unsub) => unsub());
      if (tokenUpdateTimerRef.current) {
        clearTimeout(tokenUpdateTimerRef.current);
      }
    };
  }, [
    eventBus,
    updateMessages,
    setProcessing,
    setStreaming,
    setTokenCounts,
    addActiveToolCall,
    removeActiveToolCall,
    clearActiveToolCalls,
    updateTokensThrottled,
    accumulator,
  ]);

  // Initial load of messages
  useEffect(() => {
    updateMessages();
  }, [updateMessages]);

  // Send message
  const sendMessage = useCallback(
    (content: string) => {
      if (!content.trim()) return;
      streamProcessor.sendMessage(content);
    },
    [streamProcessor]
  );

  // Cancel
  const cancel = useCallback(() => {
    streamProcessor.cancel();
  }, [streamProcessor]);

  // Clear messages
  const clearMessages = useCallback(() => {
    messageRepository.clear();
    accumulator.clear();
    setTokenCounts({ input: 0, output: 0 });
  }, [messageRepository, accumulator, setTokenCounts]);

  return {
    messages,
    isProcessing,
    sendMessage,
    cancel,
    clearMessages,
  };
}
