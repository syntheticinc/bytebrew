// useStreamConnection hook - manages stream gateway connection
import { useEffect, useRef, useCallback } from 'react';
import { IStreamGateway, ConnectionStatus } from '../../domain/ports/IStreamGateway.js';
import { useViewStore } from '../store/viewStore.js';
import { VERSION } from '../../version.js';

export interface UseStreamConnectionOptions {
  streamGateway: IStreamGateway;
  serverAddress: string;
  sessionId: string;
  projectKey: string;
  projectRoot: string;
  testingStrategy?: string;
  agentName?: string;
}

export interface UseStreamConnectionResult {
  status: ConnectionStatus;
  reconnectAttempts: number;
  isConnected: boolean;
  connect: () => Promise<void>;
  disconnect: () => void;
}

/**
 * Hook for managing stream gateway connection state.
 * Subscribes to gateway status changes and updates the view store.
 */
export function useStreamConnection(options: UseStreamConnectionOptions): UseStreamConnectionResult {
  const { streamGateway, serverAddress, sessionId, projectKey, projectRoot, testingStrategy, agentName } = options;

  const hasConnectedRef = useRef(false);

  // View store state and actions
  const status = useViewStore((state) => state.connectionStatus);
  const reconnectAttempts = useViewStore((state) => state.reconnectAttempts);
  const setConnectionStatus = useViewStore((state) => state.setConnectionStatus);
  const incrementReconnectAttempts = useViewStore((state) => state.incrementReconnectAttempts);
  const resetReconnectAttempts = useViewStore((state) => state.resetReconnectAttempts);

  // Subscribe to status changes
  useEffect(() => {
    const unsubscribe = streamGateway.onStatusChange((newStatus) => {
      setConnectionStatus(newStatus);
      if (newStatus === 'connected') {
        resetReconnectAttempts();
      } else if (newStatus === 'reconnecting') {
        incrementReconnectAttempts();
      }
    });

    return unsubscribe;
  }, [streamGateway, setConnectionStatus, resetReconnectAttempts, incrementReconnectAttempts]);

  // Connect function
  const connect = useCallback(async () => {
    if (hasConnectedRef.current) return;
    hasConnectedRef.current = true;

    try {
      await streamGateway.connect({
        serverAddress,
        sessionId,
        userId: 'cli-user',
        projectKey,
        projectRoot,
        clientVersion: VERSION,
        testingStrategy,
        agentName,
      });
    } catch (error) {
      // Log error but don't throw - just stay disconnected
      // The status will already be set to 'disconnected' by the gateway
      hasConnectedRef.current = false; // Allow retry
    }
  }, [streamGateway, serverAddress, sessionId, projectKey, projectRoot, testingStrategy, agentName]);

  // Disconnect function
  const disconnect = useCallback(() => {
    streamGateway.disconnect();
  }, [streamGateway]);

  return {
    status,
    reconnectAttempts,
    isConnected: status === 'connected',
    connect,
    disconnect,
  };
}
