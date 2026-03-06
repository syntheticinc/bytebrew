#!/usr/bin/env bun
/**
 * Headless bridge server for Flutter E2E tests.
 *
 * Starts a CLI container connected to testserver via gRPC and bridge via WS,
 * generates a pairing token, and stays alive to handle mobile requests.
 *
 * Protocol:
 *   stdout: READY:{serverId}:{sessionId}:{pairingToken}
 *   Stays alive until SIGTERM/SIGINT.
 *
 * Usage:
 *   bun src/test-utils/headless-bridge-server.ts \
 *     --server-port 12345 \
 *     --bridge-port 54321 \
 *     --bridge-auth-token xxx \
 *     --server-id yyy
 */

import { Container, createContainer, resetContainer } from '../config/container.js';

// ---------------------------------------------------------------------------
// Parse CLI args
// ---------------------------------------------------------------------------

const args = process.argv.slice(2);

function getArg(name: string): string {
  const idx = args.indexOf(`--${name}`);
  if (idx === -1 || idx + 1 >= args.length) {
    throw new Error(`Missing required argument: --${name}`);
  }
  return args[idx + 1];
}

const serverPort = parseInt(getArg('server-port'), 10);
const bridgePort = parseInt(getArg('bridge-port'), 10);
const bridgeAuthToken = getArg('bridge-auth-token');
const serverId = getArg('server-id');

if (isNaN(serverPort) || serverPort <= 0) {
  throw new Error(`Invalid server-port: ${getArg('server-port')}`);
}

if (isNaN(bridgePort) || bridgePort <= 0) {
  throw new Error(`Invalid bridge-port: ${getArg('bridge-port')}`);
}

// ---------------------------------------------------------------------------
// Create and wire up the container
// ---------------------------------------------------------------------------

const container: Container = createContainer({
  projectRoot: process.cwd(),
  serverAddress: `localhost:${serverPort}`,
  projectKey: 'flutter-e2e',
  bridgeEnabled: true,
  bridgeAddress: `localhost:${bridgePort}`,
  serverId,
  bridgeAuthToken,
  disableLspServers: true,
  headlessMode: true,
});

// Connect gRPC stream to testserver (required for sendMessage to work)
await container.streamGateway.connect({
  serverAddress: `localhost:${serverPort}`,
  sessionId: container.sessionId,
  userId: 'flutter-e2e-user',
  projectKey: 'flutter-e2e',
  projectRoot: process.cwd(),
  clientVersion: '0.2.0',
});

// ---------------------------------------------------------------------------
// Wait for bridge connection
// ---------------------------------------------------------------------------

const BRIDGE_CONNECT_TIMEOUT_MS = 10_000;
const deadline = Date.now() + BRIDGE_CONNECT_TIMEOUT_MS;

while (Date.now() < deadline) {
  if (container.bridgeConnector?.isConnected()) {
    break;
  }
  await new Promise((r) => setTimeout(r, 100));
}

if (!container.bridgeConnector?.isConnected()) {
  console.error('Bridge connection timeout');
  resetContainer();
  process.exit(1);
}

// ---------------------------------------------------------------------------
// Generate pairing token
// ---------------------------------------------------------------------------

const pairingService = container.pairingService;
if (!pairingService) {
  console.error('PairingService not initialized (bridge not enabled?)');
  resetContainer();
  process.exit(1);
}

const tokenResult = pairingService.generatePairingToken();

// ---------------------------------------------------------------------------
// Signal readiness to the parent process
// ---------------------------------------------------------------------------

console.log(`READY:${serverId}:${container.sessionId}:${tokenResult.token}`);

// ---------------------------------------------------------------------------
// Stay alive and handle graceful shutdown
// ---------------------------------------------------------------------------

function shutdown(): void {
  resetContainer();
  process.exit(0);
}

process.on('SIGTERM', shutdown);
process.on('SIGINT', shutdown);

// Keep the event loop alive
setInterval(() => {
  // no-op heartbeat
}, 60_000);
