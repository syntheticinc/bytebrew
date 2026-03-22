// Presentation layer exports

// Store
export {
  useViewStore,
  selectConnectionStatus,
  selectReconnectAttempts,
  selectIsProcessing,
  selectIsStreaming,
  selectTokenCounts,
  selectActiveToolCalls,
  selectIsConnected,
} from './store/viewStore.js';

// Hooks
export {
  useConversation,
  type UseConversationOptions,
  type UseConversationResult,
} from './hooks/useConversation.js';

export {
  useStreamConnection,
  type UseStreamConnectionOptions,
  type UseStreamConnectionResult,
} from './hooks/useStreamConnection.js';

// Mappers
export {
  toMessageViewModel,
  toMessageViewModels,
  createMessageViewModel,
  type MessageViewModel,
} from './mappers/MessageViewMapper.js';
