/**
 * Debug script: tests TWO consecutive messages through the full chain.
 * Bridge → CLI → gRPC Server → CLI → Bridge
 *
 * Usage: bun run tests/e2e/debug-two-messages.ts
 */

import { BridgeHelper } from '../../src/test-utils/BridgeHelper.js';
import { WsMobileSimulator } from './WsMobileSimulator.js';

// Import test server
import { spawn, ChildProcess } from 'child_process';
import path from 'path';

const SERVER_DIR = path.resolve(import.meta.dir, '../../../bytebrew/engine');
const SERVER_BIN = path.join(SERVER_DIR, process.platform === 'win32' ? 'server.exe' : 'server');

// --- Container setup (minimal CLI with bridge) ---

async function main() {
  const log = (msg: string) => console.log(`[${new Date().toISOString().slice(11, 23)}] ${msg}`);

  log('=== Two-message debug test ===');

  // 1. Start bridge
  log('Starting bridge...');
  const bridge = new BridgeHelper();
  await bridge.start();
  log(`Bridge started on port ${bridge.port}`);

  // 2. Create container with bridge
  log('Creating CLI container...');
  const { Container } = await import('../../src/config/container.js');
  const container = new Container({
    serverAddress: 'localhost:60401', // real server
    sessionId: `debug-${Date.now()}`,
    projectKey: 'test-project',
    userId: 'debug-user',
    projectRoot: path.resolve(import.meta.dir, '../../../test-project'),
    bridgeUrl: bridge.url,
    bridgeAuthToken: bridge.authToken,
    dbPath: ':memory:',
  });

  log('Initializing container...');
  await container.initialize();

  // 3. Initialize bridge
  log('Initializing bridge connection...');
  await container.initializeBridge();
  log('Bridge connected');

  // 4. Connect mobile simulator
  const mobile = new WsMobileSimulator();
  const serverId = container.getServerId();
  log(`Connecting mobile to bridge (serverId=${serverId})...`);
  await mobile.connect(bridge.url, serverId);
  log('Mobile connected');

  // 5. Pair
  log('Generating pairing token...');
  const pairingToken = container.generatePairingToken();
  log(`Pairing (token=${pairingToken.slice(0, 8)}...)...`);
  const pairResult = await mobile.pair(pairingToken);
  log(`Paired! deviceId=${pairResult.deviceId}, encrypted=${mobile.isEncrypted}`);

  // 6. Subscribe to session
  const sessionId = container.getSessionId();
  log(`Subscribing to session ${sessionId}...`);
  mobile.subscribeFireAndForget(sessionId);
  await new Promise((r) => setTimeout(r, 200));
  log('Subscribed');

  // 7. Send FIRST message
  log('--- SENDING MESSAGE 1: "привет" ---');
  const ack1 = await mobile.sendNewTask('привет', sessionId);
  log(`Message 1 ack: type=${ack1.type}`);

  // Wait for response events
  log('Waiting for MessageCompleted(assistant)...');
  try {
    const assistantMsg = await mobile.waitForEvent(
      (e) => e.type === 'MessageCompleted' && e.role === 'assistant',
      30000,
    );
    log(`✓ Message 1 response: "${String(assistantMsg.content).slice(0, 50)}..."`);
  } catch (err) {
    log(`✗ Message 1 TIMEOUT: ${(err as Error).message}`);
    const events = mobile.drainEvents();
    log(`  Collected events: ${events.map((e) => e.type).join(', ')}`);
  }

  // Wait for ProcessingStopped
  log('Waiting for ProcessingStopped...');
  try {
    await mobile.waitForEvent((e) => e.type === 'ProcessingStopped', 10000);
    log('✓ ProcessingStopped received');
  } catch {
    log('⚠ ProcessingStopped timeout (continuing anyway)');
    mobile.drainEvents();
  }

  // Small delay between messages
  await new Promise((r) => setTimeout(r, 1000));

  // 8. Send SECOND message
  log('--- SENDING MESSAGE 2: "как дела" ---');
  const ack2 = await mobile.sendNewTask('как дела', sessionId);
  log(`Message 2 ack: type=${ack2.type}`);

  // Wait for response events
  log('Waiting for MessageCompleted(assistant) for message 2...');
  try {
    const assistantMsg2 = await mobile.waitForEvent(
      (e) => e.type === 'MessageCompleted' && e.role === 'assistant',
      30000,
    );
    log(`✓ Message 2 response: "${String(assistantMsg2.content).slice(0, 50)}..."`);
  } catch (err) {
    log(`✗ Message 2 TIMEOUT: ${(err as Error).message}`);
    const events = mobile.drainEvents();
    log(`  Collected events: ${events.map((e) => e.type).join(', ')}`);
  }

  // Cleanup
  log('Cleaning up...');
  mobile.disconnect();
  container.dispose();
  await bridge.stop();
  log('=== DONE ===');
  process.exit(0);
}

main().catch((err) => {
  console.error('FATAL:', err);
  process.exit(1);
});
