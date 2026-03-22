// Status bar component - compact inline view with coffee spinner
import React from 'react';
import { Box, Text } from 'ink';
import { ConnectionIndicator } from './ConnectionIndicator.js';
import { CoffeeSpinner } from '../common/CoffeeSpinner.js';
import { ConnectionStatus } from '../../../domain/connection.js';

interface StatusBarProps {
  connectionStatus: ConnectionStatus;
  reconnectAttempts?: number;
  projectKey?: string;
  isProcessing?: boolean;
  streamingTokens?: { input: number; output: number };
  actionLabel?: string;
  actionColor?: string;
  tierBadge?: { label: string; color: string };
  providerBadge?: { label: string; color: string };
}

const StatusBarComponent: React.FC<StatusBarProps> = ({
  connectionStatus,
  reconnectAttempts,
  projectKey,
  isProcessing,
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
