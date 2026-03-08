/**
 * Debug script: Tests full chain with REAL Flutter app on a physical device.
 *
 * Uses cloud bridge (49.12.226.216:8443) and real gRPC server.
 * Generates QR payload for Flutter app pairing, then waits for messages.
 *
 * Usage: bun run tests/e2e/debug-real-device.ts
 */

import path from 'path';
import fs from 'fs';
import { createContainer, resetContainer } from '../../src/config/container.js';

const BRIDGE_URL = 'bridge.bytebrew.ai:443';
const BRIDGE_AUTH_TOKEN = '7a4617deef1f3cd0152ddc59dd8ec4d94cc673d13c7d09beaa9044ed0e5a0e31';
const SERVER_ADDRESS = 'localhost:60466';
const PROJECT_ROOT = path.resolve(import.meta.dir, '../../../test-project');
const LOG_FILE = path.resolve(import.meta.dir, 'debug-real-device.log');

// Clear log file
fs.writeFileSync(LOG_FILE, '');

function log(msg: string, detail?: unknown) {
  const ts = new Date().toISOString().slice(11, 23);
  let line = `[${ts}] ${msg}`;
  if (detail !== undefined) {
    line += '\n          ' + (typeof detail === 'string' ? detail : JSON.stringify(detail, null, 2));
  }
  console.log(line);
  fs.appendFileSync(LOG_FILE, line + '\n');
}

async function main() {
  log('=== REAL DEVICE DEBUG TEST (v2 - verbose) ===');
  log('Bridge:', BRIDGE_URL);
  log('Server:', SERVER_ADDRESS);
  log('Log file:', LOG_FILE);

  // Create container with cloud bridge
  log('Creating container...');
  const container = createContainer({
    projectRoot: PROJECT_ROOT,
    serverAddress: SERVER_ADDRESS,
    projectKey: 'test-project',
    bridgeEnabled: true,
    bridgeAddress: BRIDGE_URL,
    bridgeAuthToken: BRIDGE_AUTH_TOKEN,
    disableLspServers: true,
  });

  log('Container created', {
    sessionId: container.sessionId,
    serverId: container.cliIdentity?.getServerId(),
  });

  // List paired devices from SQLite
  const devices = container.pairingService?.listDevices() ?? [];
  log(`Paired devices in SQLite: ${devices.length}`);
  for (const d of devices) {
    log(`  Device: ${d.id} name="${d.name}" token=${d.deviceToken.slice(0, 8)}... secret=${d.sharedSecret.length}B`);
  }

  // Connect gRPC to real server
  log('Connecting gRPC...');
  await container.streamGateway.connect({
    serverAddress: SERVER_ADDRESS,
    sessionId: container.sessionId,
    userId: 'test-user',
    projectKey: 'test-project',
    projectRoot: PROJECT_ROOT,
    clientVersion: '0.2.0',
  });
  log('gRPC connected');

  // Wait for bridge connection
  log('Waiting for bridge connection...');
  const deadline = Date.now() + 10000;
  while (Date.now() < deadline) {
    if (container.bridgeConnector?.isConnected()) break;
    await new Promise((r) => setTimeout(r, 200));
  }

  if (!container.bridgeConnector?.isConnected()) {
    log('ERROR: Bridge connection failed!');
    process.exit(1);
  }
  log('Bridge connected!');

  // Generate pairing token + QR data
  const tokenResult = container.pairingService!.generatePairingToken();
  const keyPair = container.cliIdentity!.getKeyPair();
  const serverId = container.cliIdentity!.getServerId();

  const qrPayload = JSON.stringify({
    server_id: serverId,
    server_public_key: Buffer.from(keyPair.publicKey).toString('base64'),
    bridge_url: BRIDGE_URL,
    token: tokenResult.token,
  });

  log('');
  log('============================================================');
  log('PAIRING INFO (for Flutter app):');
  log('============================================================');
  log('Server ID:', serverId);
  log('Short Code:', tokenResult.shortCode);
  log('Token:', tokenResult.token);
  log('Session ID:', container.sessionId);
  log('');
  log('QR Payload (paste into AddServerScreen manual input if available):');
  log(qrPayload);
  log('============================================================');
  log('');

  // Hook into bridge connector for raw data logging
  if (container.bridgeConnector) {
    container.bridgeConnector.onData((deviceId, payload) => {
      const payloadStr = typeof payload === 'string'
        ? `string(${payload.length} chars): ${payload.slice(0, 100)}...`
        : JSON.stringify(payload).slice(0, 200);
      log(`[BRIDGE RAW] deviceId=${deviceId}`, payloadStr);
    });
    container.bridgeConnector.onDeviceConnect((deviceId) => {
      log(`[BRIDGE] Device connected: ${deviceId}`);
      // Check if device is known
      const known = devices.find(d => d.id === deviceId);
      if (known) {
        log(`  -> KNOWN device: ${known.name} (paired)`);
      } else {
        log(`  -> UNKNOWN device (not in SQLite - needs pairing)`);
      }
    });
    container.bridgeConnector.onDeviceDisconnect((deviceId) => {
      log(`[BRIDGE] Device disconnected: ${deviceId}`);
    });
  }

  // Listen to EventBus for all events
  let eventCount = 0;
  container.eventBus.subscribeAll((event) => {
    eventCount++;
    log(`[EventBus] #${eventCount} ${event.type}`, JSON.stringify(event).slice(0, 500));
  });

  log('Waiting for Flutter app to connect and send messages...');
  log('(This process will stay alive for 10 minutes)');
  log('Press Ctrl+C to stop.');
  log('');

  // Keep alive for 10 minutes
  const keepAliveMs = 10 * 60 * 1000;
  const startTime = Date.now();

  const interval = setInterval(() => {
    const elapsed = Math.round((Date.now() - startTime) / 1000);
    const bridgeOk = container.bridgeConnector?.isConnected() ?? false;
    const grpcOk = container.streamGateway.isConnected();
    const currentDevices = container.pairingService?.listDevices() ?? [];
    log(`[heartbeat] ${elapsed}s elapsed | bridge=${bridgeOk} | grpc=${grpcOk} | events=${eventCount} | devices=${currentDevices.length}`);
  }, 15000);

  // Wait
  await new Promise((r) => setTimeout(r, keepAliveMs));

  clearInterval(interval);
  log('=== TIMEOUT, shutting down ===');
  resetContainer();
  process.exit(0);
}

// Graceful shutdown on Ctrl+C
process.on('SIGINT', () => {
  log('SIGINT received, shutting down...');
  resetContainer();
  process.exit(0);
});

main().catch((err) => {
  console.error('FATAL:', err);
  process.exit(1);
});
