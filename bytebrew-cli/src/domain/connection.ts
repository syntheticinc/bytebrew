// Connection state types

export type ConnectionStatus =
  | 'disconnected'
  | 'connecting'
  | 'connected'
  | 'reconnecting'
  | 'error';

export interface ConnectionState {
  status: ConnectionStatus;
  sessionId: string;
  projectKey: string;
  userId: string;
  serverAddress: string;
  reconnectAttempts: number;
  lastError?: string;
  lastPingTime?: Date;
  lastPongTime?: Date;
}

export const DEFAULT_SERVER_ADDRESS = 'localhost:60401';
export const PING_INTERVAL_MS = 20000; // 20 seconds
export const MAX_RECONNECT_ATTEMPTS = 10;
export const INITIAL_RECONNECT_DELAY_MS = 1000;
export const MAX_RECONNECT_DELAY_MS = 30000;
