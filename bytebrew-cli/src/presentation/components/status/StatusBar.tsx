// Status bar component - compact inline view with coffee spinner
import React from 'react';
import { Box, Text } from 'ink';
import { ConnectionIndicator } from './ConnectionIndicator.js';
import { CoffeeSpinner } from '../common/CoffeeSpinner.js';
import { ConnectionStatus } from '../../../domain/connection.js';
import { IndexingStatus } from '../../../indexing/backgroundIndexer.js';

interface StatusBarProps {
  connectionStatus: ConnectionStatus;
  reconnectAttempts?: number;
  projectKey?: string;
  isProcessing?: boolean;
  indexingStatus?: IndexingStatus;
  streamingTokens?: { input: number; output: number };
  actionLabel?: string;
  actionColor?: string;
  tierBadge?: { label: string; color: string };
  providerBadge?: { label: string; color: string };
}

const IndexingIndicator: React.FC<{ status: IndexingStatus }> = ({ status }) => {
  switch (status.phase) {
    case 'syncing': {
      // Don't show "Syncing 0..." — only show count when there's progress
      if (!status.filesUpdated && !status.filesTotal) {
        return <Text color="yellow">Syncing...</Text>;
      }
      const progress = status.filesTotal
        ? `${status.filesUpdated}/${status.filesTotal}`
        : String(status.filesUpdated);
      return <Text color="yellow">Syncing {progress}...</Text>;
    }
    case 'embedding': {
      const embProgress = status.chunksTotal
        ? `${status.chunksEmbedded}/${status.chunksTotal}`
        : '';
      return <Text color="yellow">Embedding {embProgress}...</Text>;
    }
    case 'watching':
      return status.ollamaAvailable === false
        ? <Text color="yellow">No embeddings</Text>
        : <Text color="green">Indexed</Text>;
    case 'error':
      return <Text color="red">Index error</Text>;
    case 'idle':
    default:
      return null;
  }
};

const StatusBarComponent: React.FC<StatusBarProps> = ({
  connectionStatus,
  reconnectAttempts,
  projectKey,
  isProcessing,
  indexingStatus,
  streamingTokens,
  actionLabel,
  actionColor,
  tierBadge,
  providerBadge,
}) => {
  return (
    <Box paddingX={1}>
      <ConnectionIndicator
        status={connectionStatus}
        reconnectAttempts={reconnectAttempts}
      />

      {tierBadge && (
        <>
          <Text color="gray"> · </Text>
          <Text color={tierBadge.color}>{tierBadge.label}</Text>
        </>
      )}

      {providerBadge && (
        <>
          <Text color="gray"> · </Text>
          <Text color={providerBadge.color}>{providerBadge.label}</Text>
        </>
      )}

      {projectKey && (
        <>
          <Text color="gray"> · </Text>
          <Text color="cyan">{projectKey}</Text>
        </>
      )}

      {indexingStatus && indexingStatus.phase !== 'idle' && (
        <>
          <Text color="gray"> · </Text>
          <IndexingIndicator status={indexingStatus} />
        </>
      )}

      {isProcessing && (
        <>
          <Text color="gray"> · </Text>
          <CoffeeSpinner variant="aesthetic" actionLabel={actionLabel} color={actionColor} />
          {streamingTokens && (streamingTokens.input > 0 || streamingTokens.output > 0) && (
            <Text color="gray" dimColor>
              {' '}in:{streamingTokens.input} out:{streamingTokens.output}
            </Text>
          )}
        </>
      )}
    </Box>
  );
};

// Memoize to prevent unnecessary re-renders
export const StatusBar = React.memo(StatusBarComponent);
