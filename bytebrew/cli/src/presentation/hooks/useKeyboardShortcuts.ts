// useKeyboardShortcuts hook - manages global keyboard shortcuts
import type { MutableRefObject } from 'react';
import { useInput } from 'ink';
import { AgentState } from '../../infrastructure/state/AgentStateManager.js';
import { AgentStateManager } from '../../infrastructure/state/AgentStateManager.js';

export interface UseKeyboardShortcutsOptions {
  agents: AgentState[];
  agentStateManager: AgentStateManager;
  isProcessing: boolean;
  isRawModeSupported: boolean;
  cancel: () => void;
  disconnect: () => void;
  exit: () => void;
  setCurrentViewAgentId: (id: string) => void;
  isExitingRef: MutableRefObject<boolean>;
  isMenuOpen: boolean;
  toggleMenu: () => void;
}

/**
 * Hook that manages global keyboard shortcuts.
 * - Shift+Tab: cycle view between agents
 * - Ctrl+C: cancel processing or exit
 *
 * Note: useInput is an Ink hook, works when called from custom hook
 * invoked inside Ink Provider component.
 */
export function useKeyboardShortcuts(options: UseKeyboardShortcutsOptions): void {
  const {
    agents,
    isProcessing,
    isRawModeSupported,
    cancel,
    disconnect,
    exit,
    isExitingRef,
    toggleMenu,
  } = options;

  // Handle keyboard shortcuts
  // Only active when raw mode is supported (interactive terminal)
  useInput(
    (input, key) => {
      // Shift+Tab: toggle agent menu
      if (key.tab && key.shift) {
        if (agents.length > 1) {
          toggleMenu();
        }
        return;
      }

      // Escape: Cancel processing
      if (key.escape && isProcessing) {
        cancel();
        return;
      }

      // Ctrl+C: Cancel or exit
      if (key.ctrl && input === 'c') {
        if (isProcessing) {
          cancel();
        } else {
          isExitingRef.current = true;
          disconnect();
          exit();
        }
      }
    },
    { isActive: isRawModeSupported }
  );
}
