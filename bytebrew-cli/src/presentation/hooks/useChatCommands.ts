// useChatCommands hook - manages chat input submission and commands
import { useEffect, useCallback, useRef, type MutableRefObject } from 'react';
import { useInputHistory } from './useInputHistory.js';
import {
  readProviderConfig,
  writeProviderConfig,
  isValidProviderMode,
  readModelsConfig,
  writeModelOverride,
  resetModelOverrides,
} from '../../infrastructure/config/ProviderConfig.js';
import type { LicenseBadgeInfo } from './useLicenseInfo.js';
import { handleLogoutCommand, handleStatusCommand, handleLoginCommand, handleActivateCommand } from './authCommands.js';
import { handleMobileCommand } from './mobileCommands.js';
import type { Container } from '../../config/container.js';

export interface UseChatCommandsOptions {
  isConnected: boolean;
  isProcessing: boolean;
  sendMessage: (content: string) => void;
  clearMessages: () => void;
  disconnect: () => void;
  exit: () => void;
  initialQuestion?: string;
  isExitingRef: MutableRefObject<boolean>;
  /** Callback for commands that produce local output (not sent to server) */
  onCommandOutput?: (output: string) => void;
  /** Called when provider mode changes via /provider command */
  onProviderChange?: () => void;
  /** Called after /login or /activate to refresh license badge */
  onLicenseChange?: () => void;
  /** License info for /provider status display */
  licenseInfo?: LicenseBadgeInfo | null;
  /** DI container for commands that need access to services (e.g. /mobile) */
  container?: Container;
}

export interface UseChatCommandsResult {
  history: string[];
  handleSubmit: (value: string) => void;
}

function handleProviderCommand(args: string, licenseInfo?: LicenseBadgeInfo | null): string {
  if (!args) {
    // Show current provider status
    const config = readProviderConfig();
    const parts: string[] = [`Provider: ${config.mode}`];

    if (licenseInfo?.proxyStepsRemaining !== undefined) {
      const limit = licenseInfo.proxyStepsLimit ?? '?';
      parts.push(`Proxy steps: ${licenseInfo.proxyStepsRemaining}/${limit}`);
    }

    if (licenseInfo?.byokEnabled !== undefined) {
      parts.push(`BYOK: ${licenseInfo.byokEnabled ? 'enabled' : 'disabled'}`);
    }

    return parts.join(' | ');
  }

  if (!isValidProviderMode(args)) {
    return `Invalid provider mode: "${args}". Valid modes: proxy, byok, auto`;
  }

  writeProviderConfig({ mode: args });
  return `Provider mode set to: ${args}`;
}

function handleModelCommand(args: string): string {
  if (!args) {
    // Show current model config
    const models = readModelsConfig();
    const entries = Object.entries(models.overrides);

    if (entries.length === 0) {
      return 'No model overrides configured. Using defaults.';
    }

    const lines = entries.map(([role, model]) => `  ${role} -> ${model}`);
    return `Model overrides:\n${lines.join('\n')}`;
  }

  if (args === 'reset') {
    resetModelOverrides();
    return 'Model overrides reset to defaults';
  }

  // Parse: /model <role> <model>
  const spaceIdx = args.indexOf(' ');
  if (spaceIdx === -1) {
    return `Usage: /model <role> <model> or /model reset`;
  }

  const role = args.slice(0, spaceIdx).trim();
  const model = args.slice(spaceIdx + 1).trim();

  if (!role || !model) {
    return `Usage: /model <role> <model> or /model reset`;
  }

  writeModelOverride(role, model);
  return `Model override set: ${role} -> ${model}`;
}

/**
 * Hook that manages chat input submission, commands, and history.
 * - Handles special commands (/quit, /exit, /clear, /help, /provider, /model)
 * - Manages input history (encapsulates useInputHistory)
 * - Sends initial question when connected
 */
export function useChatCommands(options: UseChatCommandsOptions): UseChatCommandsResult {
  const {
    isConnected,
    isProcessing,
    sendMessage,
    clearMessages,
    disconnect,
    exit,
    initialQuestion,
    isExitingRef,
    onCommandOutput,
    onProviderChange,
    onLicenseChange,
    licenseInfo,
    container,
  } = options;

  const hasSentInitialQuestionRef = useRef(false);

  // Input history (encapsulated)
  const { history, addToHistory } = useInputHistory();

  // Send initial question after connection
  useEffect(() => {
    if (
      initialQuestion &&
      isConnected &&
      !hasSentInitialQuestionRef.current &&
      !isProcessing
    ) {
      hasSentInitialQuestionRef.current = true;
      addToHistory(initialQuestion);
      sendMessage(initialQuestion);
    }
  }, [initialQuestion, isConnected, isProcessing, sendMessage, addToHistory]);

  // Handle message submission
  const handleSubmit = useCallback(
    (value: string) => {
      if (!value.trim()) return;

      // Handle special commands
      if (value === '/quit' || value === '/exit') {
        isExitingRef.current = true;
        disconnect();
        exit();
        return;
      }

      if (value === '/clear') {
        clearMessages();
        return;
      }

      if (value === '/help') {
        const helpText = [
          'Available commands:',
          '  /provider          - Show current provider mode',
          '  /provider <mode>   - Set mode (proxy, byok, auto)',
          '  /model             - Show model overrides',
          '  /model <role> <m>  - Set model for role',
          '  /model reset       - Reset model overrides',
          '  /login <email> <pw> - Login to your account',
          '  /logout             - Logout and clear credentials',
          '  /status             - Show license and account info',
          '  /activate           - Activate or refresh license',
          '  /mobile             - Pair a mobile device (QR code)',
          '  /mobile devices     - List paired devices',
          '  /mobile status      - Bridge connection status',
          '  /clear             - Clear chat history',
          '  /quit, /exit       - Exit application',
        ].join('\n');
        onCommandOutput?.(helpText);
        return;
      }

      if (value === '/provider' || value.startsWith('/provider ')) {
        const args = value.slice('/provider'.length).trim();
        const output = handleProviderCommand(args, licenseInfo);
        onCommandOutput?.(output);
        if (args && isValidProviderMode(args)) {
          onProviderChange?.();
        }
        return;
      }

      if (value === '/model' || value.startsWith('/model ')) {
        const args = value.slice('/model'.length).trim();
        const output = handleModelCommand(args);
        onCommandOutput?.(output);
        return;
      }

      if (value === '/logout') {
        const output = handleLogoutCommand();
        onCommandOutput?.(output);
        onLicenseChange?.();
        return;
      }

      if (value === '/status') {
        const output = handleStatusCommand();
        onCommandOutput?.(output);
        return;
      }

      if (value === '/login' || value.startsWith('/login ')) {
        const args = value.slice('/login'.length).trim();
        if (!args) {
          onCommandOutput?.('Usage: /login <email> <password>');
          return;
        }
        onCommandOutput?.('Logging in...');
        void handleLoginCommand(args).then(result => {
          onCommandOutput?.(result);
          onLicenseChange?.();
        }).catch(err => {
          onCommandOutput?.(`Error: ${(err as Error).message}`);
        });
        return;
      }

      if (value === '/activate') {
        onCommandOutput?.('Activating license...');
        void handleActivateCommand().then(result => {
          onCommandOutput?.(result);
          onLicenseChange?.();
        }).catch(err => {
          onCommandOutput?.(`Error: ${(err as Error).message}`);
        });
        return;
      }

      if (value === '/mobile' || value.startsWith('/mobile ')) {
        const args = value.slice('/mobile'.length).trim();
        if (!container) {
          onCommandOutput?.('Mobile not available.');
          return;
        }
        void handleMobileCommand(args, container, (text) => onCommandOutput?.(text));
        return;
      }

      addToHistory(value);
      sendMessage(value);
    },
    [addToHistory, sendMessage, disconnect, exit, clearMessages, isExitingRef, onCommandOutput, onProviderChange, onLicenseChange, licenseInfo, container]
  );

  return {
    history,
    handleSubmit,
  };
}
