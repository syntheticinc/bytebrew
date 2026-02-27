// Connection status indicator
import React from 'react';
import { Text, Box } from 'ink';
import { ConnectionStatus } from '../../../domain/connection.js';

interface ConnectionIndicatorProps {
  status: ConnectionStatus;
  reconnectAttempts?: number;
}

const STATUS_CONFIG: Record<ConnectionStatus, { symbol: string; color: string; label: string }> = {
  connected: { symbol: '●', color: 'green', label: 'Connected' },
  connecting: { symbol: '◐', color: 'yellow', label: 'Connecting' },
  disconnected: { symbol: '○', color: 'red', label: 'Disconnected' },
  reconnecting: { symbol: '◐', color: 'yellow', label: 'Reconnecting' },
  error: { symbol: '✕', color: 'red', label: 'Error' },
};

export const ConnectionIndicator: React.FC<ConnectionIndicatorProps> = ({
  status,
  reconnectAttempts,
}) => {
  const config = STATUS_CONFIG[status];

  return (
    <Box>
      <Text color={config.color}>{config.symbol}</Text>
      <Text color="gray"> {config.label}</Text>
      {status === 'reconnecting' && reconnectAttempts !== undefined && (
        <Text color="gray"> ({reconnectAttempts})</Text>
      )}
    </Box>
  );
};
