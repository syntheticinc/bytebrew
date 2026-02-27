// Main chat application component using clean architecture
import React, { useEffect, useRef, useMemo, useState, useCallback } from 'react';
import { Box, useApp, useStdout, useStdin } from 'ink';
import { ChatView } from '../components/chat/ChatView.js';
import { InputField } from '../components/input/InputField.js';
import { StatusBar } from '../components/status/StatusBar.js';
import { AgentMenu } from '../components/agents/AgentMenu.js';
import { useBackgroundIndexer } from '../hooks/useBackgroundIndexer.js';
import { usePermissionApproval } from '../hooks/usePermissionApproval.js';
import { PermissionApprovalPrompt } from '../components/PermissionApprovalPrompt.js';
import { QuestionnairePrompt } from '../components/QuestionnairePrompt.js';
import { Container } from '../../config/container.js';
import { useConversation } from '../hooks/useConversation.js';
import { useStreamConnection } from '../hooks/useStreamConnection.js';
import { useAgentSync } from '../hooks/useAgentSync.js';
import { useAskUser } from '../hooks/useAskUser.js';
import { useKeyboardShortcuts } from '../hooks/useKeyboardShortcuts.js';
import { useChatCommands } from '../hooks/useChatCommands.js';
import { useLicenseInfo } from '../hooks/useLicenseInfo.js';
import { useViewStore, selectActionLabel, selectActionColor, selectCurrentViewAgentId } from '../store/viewStore.js';
import { filterMessagesForView, ViewMode } from '../mappers/MessageViewFilter.js';
import { Message } from '../../domain/entities/Message.js';
import { readProviderConfig } from '../../infrastructure/config/ProviderConfig.js';
import { readTestingStrategy } from '../../lib/testingStrategy.js';

interface ChatAppProps {
  container: Container;
  initialQuestion?: string;
}

/**
 * Build provider badge from provider config and license info.
 */
function buildProviderBadge(
  licenseInfo: { proxyStepsRemaining?: number; proxyStepsLimit?: number; byokEnabled?: boolean } | null,
): { label: string; color: string } | undefined {
  const config = readProviderConfig();

  if (config.mode === 'proxy') {
    if (licenseInfo?.proxyStepsRemaining !== undefined) {
      const limit = licenseInfo.proxyStepsLimit ?? '?';
      return {
        label: `proxy ${licenseInfo.proxyStepsRemaining}/${limit}`,
        color: licenseInfo.proxyStepsRemaining > 50 ? 'green' : 'yellow',
      };
    }
    return { label: 'proxy', color: 'green' };
  }

  if (config.mode === 'byok') {
    return { label: 'byok', color: 'cyan' };
  }

  // auto mode
  if (licenseInfo?.proxyStepsRemaining !== undefined) {
    const limit = licenseInfo.proxyStepsLimit ?? '?';
    return {
      label: `auto ${licenseInfo.proxyStepsRemaining}/${limit}`,
      color: 'blue',
    };
  }

  return { label: 'auto', color: 'blue' };
}

/**
 * Main chat application component using clean architecture.
 * Uses the DI container to wire up all dependencies.
 */
export const ChatApp: React.FC<ChatAppProps> = ({
  container,
  initialQuestion,
}) => {
  const { exit } = useApp();
  const { stdout } = useStdout();
  const { isRawModeSupported } = useStdin();
  const isExitingRef = useRef(false);

  // Get terminal width
  const terminalWidth = stdout?.columns || 80;

  // Read testing strategy once from project root
  const testingStrategy = useMemo(
    () => readTestingStrategy(container.config.projectRoot),
    [container.config.projectRoot]
  );

  // Connection management
  const {
    status,
    reconnectAttempts,
    isConnected,
    connect,
    disconnect,
  } = useStreamConnection({
    streamGateway: container.streamGateway,
    serverAddress: container.config.serverAddress,
    sessionId: container.sessionId,
    projectKey: container.config.projectKey,
    projectRoot: container.config.projectRoot,
    testingStrategy,
  });

  // Conversation management
  const {
    messages,
    isProcessing,
    sendMessage,
    cancel,
    clearMessages,
  } = useConversation({
    streamProcessor: container.streamProcessor,
    messageRepository: container.messageRepository,
    accumulator: container.accumulator,
    eventBus: container.eventBus,
  });

  // Token counts from view store
  const streamingTokens = useViewStore((state) => state.tokenCounts);

  // Current action (primitive selectors — stable for zustand)
  const actionLabel = useViewStore(selectActionLabel);
  const actionColor = useViewStore(selectActionColor);

  // Current view agent (for filtering messages)
  const currentViewAgentId = useViewStore(selectCurrentViewAgentId);
  const setCurrentViewAgentId = useViewStore((s) => s.setCurrentViewAgentId);

  // Filter messages based on current view
  const viewMode: ViewMode = currentViewAgentId === 'supervisor'
    ? { type: 'supervisor' }
    : { type: 'agent', agentId: currentViewAgentId };

  const filteredMessages = useMemo(
    () => filterMessagesForView(messages, viewMode),
    [messages, currentViewAgentId]
  );

  // Background indexer - starts when connected
  const { status: indexingStatus } = useBackgroundIndexer({
    projectRoot: container.config.projectRoot,
    store: container.chunkStore,
    embeddingsClient: container.embeddingsClient,
    enabled: isConnected,
  });

  // Permission approval for tool operations
  const { pendingPermission, approve, reject } = usePermissionApproval();

  // Ask user questionnaire management
  const { questions, handleComplete } = useAskUser({
    eventBus: container.eventBus,
  });

  // Multi-agent state synchronization
  const { agents, activeAgentId } = useAgentSync({
    agentStateManager: container.agentStateManager,
    eventBus: container.eventBus,
    isProcessing,
    isBlocked: !!questions || !!pendingPermission,
  });

  // License version — incremented by /login, /logout, /activate to re-read JWT
  const [licenseVersion, setLicenseVersion] = useState(0);

  // License info for tier badge and provider info in status bar
  const licenseInfo = useLicenseInfo(licenseVersion);

  // Provider mode version — incremented by /provider command to invalidate badge
  const [providerVersion, setProviderVersion] = useState(0);

  // Provider badge for status bar
  const providerBadge = useMemo(
    () => buildProviderBadge(licenseInfo),
    [licenseInfo, providerVersion]
  );

  // Agent menu state (opened with Shift+Tab)
  const [isMenuOpen, setIsMenuOpen] = useState(false);

  // Auto-switch to agent's tab when permission requested from different agent
  useEffect(() => {
    if (pendingPermission && agents.length > 1 && activeAgentId !== currentViewAgentId) {
      setCurrentViewAgentId(activeAgentId);
    }
  }, [pendingPermission, activeAgentId, agents.length, currentViewAgentId, setCurrentViewAgentId]);

  // Command output callback: adds a local-only system message to the chat
  const onCommandOutput = useCallback((output: string) => {
    const msg = Message.createAssistantWithContent(output);
    container.messageRepository.save(msg);
    container.eventBus.publish({ type: 'MessageCompleted', message: msg });
  }, [container.messageRepository, container.eventBus]);

  // Chat commands and input history
  const { history, handleSubmit } = useChatCommands({
    isConnected,
    isProcessing,
    sendMessage,
    clearMessages,
    disconnect,
    exit,
    initialQuestion,
    isExitingRef,
    onCommandOutput,
    onProviderChange: useCallback(() => setProviderVersion(v => v + 1), []),
    onLicenseChange: useCallback(() => setLicenseVersion(v => v + 1), []),
    licenseInfo,
  });

  // Connect on mount
  useEffect(() => {
    connect();

    return () => {
      if (!isExitingRef.current) {
        disconnect();
      }
    };
  }, [connect, disconnect]);

  // Keyboard shortcuts (Shift+Tab, Ctrl+C)
  useKeyboardShortcuts({
    agents,
    agentStateManager: container.agentStateManager,
    isProcessing,
    isRawModeSupported,
    cancel,
    disconnect,
    exit,
    setCurrentViewAgentId,
    isExitingRef,
    isMenuOpen,
    toggleMenu: () => setIsMenuOpen(prev => !prev),
  });

  return (
    <Box flexDirection="column" width={terminalWidth}>
      {/* Chat messages — filtered by current view agent */}
      <Box flexDirection="column" paddingX={1}>
        <ChatView
          key={currentViewAgentId}
          messages={filteredMessages}
          renderingService={container.toolRenderingService}
        />
      </Box>

      {/* Agent menu — compact hint when closed, vertical menu when open */}
      <AgentMenu
        agents={agents}
        currentViewAgentId={currentViewAgentId}
        isOpen={isMenuOpen}
        onSelect={(id) => setCurrentViewAgentId(id)}
        onClose={() => setIsMenuOpen(false)}
      />

      {/* Dynamic section: prompts or input+status */}
      {pendingPermission ? (
        <PermissionApprovalPrompt
          request={pendingPermission.request}
          onApprove={approve}
          onReject={reject}
          agentId={agents.length > 1 ? activeAgentId : undefined}
        />
      ) : questions ? (
        <QuestionnairePrompt
          questions={questions}
          onComplete={handleComplete}
        />
      ) : (
        <>
          <Box width={terminalWidth}>
            <InputField
              onSubmit={handleSubmit}
              disabled={!isConnected || currentViewAgentId !== 'supervisor' || isMenuOpen}
              placeholder={
                !isConnected
                ? 'Connecting...'
                : isMenuOpen
                ? 'Menu is open (Esc to close)'
                : currentViewAgentId !== 'supervisor'
                ? 'Switch to Supervisor tab to send messages (Shift+Tab)'
                : isProcessing
                ? 'Type to send a follow-up message...'
                : 'Type a message... (Ctrl+C to cancel/exit)'
              }
              history={history}
            />
          </Box>
          <Box width={terminalWidth}>
            <StatusBar
              connectionStatus={status}
              reconnectAttempts={reconnectAttempts}
              projectKey={container.config.projectKey}
              isProcessing={isProcessing}
              indexingStatus={indexingStatus}
              streamingTokens={streamingTokens}
              actionLabel={actionLabel}
              actionColor={actionColor}
              tierBadge={licenseInfo ? { label: licenseInfo.label, color: licenseInfo.color } : undefined}
              providerBadge={providerBadge}
            />
          </Box>
        </>
      )}
    </Box>
  );
};
