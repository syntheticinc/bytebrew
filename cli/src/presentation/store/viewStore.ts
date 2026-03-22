// viewStore - UI-only state management
import { create } from 'zustand';
import { ConnectionStatus } from '../../domain/ports/IStreamGateway.js';
import { ToolCallInfo } from '../../domain/entities/Message.js';
import { colors } from '../theme/colors.js';

/**
 * UI-only state for the presentation layer.
 * Does NOT contain message content - that's in the repository.
 * Only contains transient UI state like connection status,
 * streaming progress, and active tool calls.
 */
interface ViewState {
  // Connection
  connectionStatus: ConnectionStatus;
  reconnectAttempts: number;

  // Processing
  isProcessing: boolean;
  isStreaming: boolean;

  // Token counts (for StatusBar)
  tokenCounts: { input: number; output: number };

  // Active tool calls (currently executing)
  activeToolCalls: ToolCallInfo[];

  // Current view agent (for filtering messages)
  currentViewAgentId: string; // 'supervisor' | 'code-agent-xxx'

  // Actions
  setConnectionStatus: (status: ConnectionStatus) => void;
  setReconnectAttempts: (attempts: number) => void;
  incrementReconnectAttempts: () => void;
  resetReconnectAttempts: () => void;
  setProcessing: (isProcessing: boolean) => void;
  setStreaming: (isStreaming: boolean) => void;
  setTokenCounts: (counts: { input: number; output: number }) => void;
  addActiveToolCall: (toolCall: ToolCallInfo) => void;
  removeActiveToolCall: (callId: string) => void;
  clearActiveToolCalls: () => void;
  setCurrentViewAgentId: (id: string) => void;
  reset: () => void;
}

const initialState = {
  connectionStatus: 'disconnected' as ConnectionStatus,
  reconnectAttempts: 0,
  isProcessing: false,
  isStreaming: false,
  tokenCounts: { input: 0, output: 0 },
  activeToolCalls: [],
  currentViewAgentId: 'supervisor',
};

export const useViewStore = create<ViewState>((set, get) => ({
  ...initialState,

  setConnectionStatus: (status: ConnectionStatus) =>
    set({ connectionStatus: status }),

  setReconnectAttempts: (attempts: number) =>
    set({ reconnectAttempts: attempts }),

  incrementReconnectAttempts: () =>
    set((state) => ({ reconnectAttempts: state.reconnectAttempts + 1 })),

  resetReconnectAttempts: () =>
    set({ reconnectAttempts: 0 }),

  setProcessing: (isProcessing: boolean) =>
    set({ isProcessing }),

  setStreaming: (isStreaming: boolean) =>
    set({ isStreaming }),

  setTokenCounts: (counts: { input: number; output: number }) =>
    set({ tokenCounts: counts }),

  addActiveToolCall: (toolCall: ToolCallInfo) =>
    set((state) => ({
      activeToolCalls: [...state.activeToolCalls, toolCall],
    })),

  removeActiveToolCall: (callId: string) =>
    set((state) => ({
      activeToolCalls: state.activeToolCalls.filter((tc) => tc.callId !== callId),
    })),

  clearActiveToolCalls: () =>
    set({ activeToolCalls: [] }),

  setCurrentViewAgentId: (id: string) => {
    const current = get().currentViewAgentId;
    if (id !== current && process.stdout.isTTY) {
      // Clear screen + scrollback BEFORE state update
      // Guard: isTTY prevents escape codes in tests (piped stdout)
      process.stdout.write('\x1B[2J\x1B[3J\x1B[H');
    }
    set({ currentViewAgentId: id });
  },

  reset: () =>
    set(initialState),
}));

// Selectors for optimized subscriptions
export const selectConnectionStatus = (state: ViewState) => state.connectionStatus;
export const selectReconnectAttempts = (state: ViewState) => state.reconnectAttempts;
export const selectIsProcessing = (state: ViewState) => state.isProcessing;
export const selectIsStreaming = (state: ViewState) => state.isStreaming;
export const selectTokenCounts = (state: ViewState) => state.tokenCounts;
export const selectActiveToolCalls = (state: ViewState) => state.activeToolCalls;
export const selectIsConnected = (state: ViewState) => state.connectionStatus === 'connected';
export const selectCurrentViewAgentId = (state: ViewState) => state.currentViewAgentId;
// Primitive selectors — no object allocation, stable for zustand Object.is comparison
export const selectActionLabel = (state: ViewState): string | undefined => {
  if (!state.isProcessing) return undefined;
  if (state.activeToolCalls.length > 0) {
    return `Executing ${state.activeToolCalls[0].toolName}`;
  }
  return undefined; // thinking → CoffeeSpinner uses its own phrase rotation
};

export const selectActionColor = (state: ViewState): string | undefined => {
  if (!state.isProcessing) return undefined;
  return colors.processing;
};
