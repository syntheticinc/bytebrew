/**
 * Debug script: Full mobile flow test with detailed logging.
 *
 * Starts real server + bridge + CLI container, connects WsMobileSimulator,
 * and traces every step of the multi-turn message flow.
 *
 * Usage: bun run tests/e2e/debug-mobile-flow.ts
 */

import { TestServerHelper } from '../../src/test-utils/TestServerHelper.js';
import { BridgeHelper } from '../../src/test-utils/BridgeHelper.js';
import { WsMobileSimulator, type SessionEvent } from './WsMobileSimulator.js';
import { Container, createContainer, resetContainer } from '../../src/config/container.js';
import { v4 as uuidv4 } from 'uuid';

function log(step: string, detail?: unknown) {
  const ts = new Date().toISOString().slice(11, 23);
  console.log(`[${ts}] ${step}`);
  if (detail !== undefined) {
    console.log(`         `, typeof detail === 'string' ? detail : JSON.stringify(detail, null, 2));
  }
}

async function main() {
  log('=== BUILDING ===');
  TestServerHelper.build();
  BridgeHelper.build();

  log('=== STARTING SERVER (echo scenario) ===');
  const server = new TestServerHelper();
  await server.start('echo');
  log('Server started', { port: server.port });

  log('=== STARTING BRIDGE ===');
  const bridge = new BridgeHelper();
  await bridge.start();
  log('Bridge started', { port: bridge.port, url: bridge.url });

  const serverId = uuidv4();
  log('=== CREATING CONTAINER ===', { serverId });

  const container = createContainer({
    projectRoot: process.cwd(),
    serverAddress: `localhost:${server.port}`,
    projectKey: 'test-project',
    bridgeEnabled: true,
    bridgeAddress: `localhost:${bridge.port}`,
    serverId,
    bridgeAuthToken: bridge.authToken,
    disableLspServers: true,
  });

  log('=== CONNECTING gRPC ===');
  await container.streamGateway.connect({
    serverAddress: `localhost:${server.port}`,
    sessionId: container.sessionId,
    userId: 'test-user',
    projectKey: 'test-project',
    projectRoot: process.cwd(),
    clientVersion: '0.2.0',
  });
  log('gRPC connected', { sessionId: container.sessionId });

  log('=== WAITING FOR BRIDGE CONNECTION ===');
  const deadline = Date.now() + 5000;
  while (Date.now() < deadline) {
    if (container.bridgeConnector?.isConnected()) break;
    await new Promise((r) => setTimeout(r, 100));
  }
  log('Bridge connected', { isConnected: container.bridgeConnector?.isConnected() });

  log('=== CONNECTING MOBILE SIMULATOR ===');
  const mobile = new WsMobileSimulator();
  await mobile.connect(bridge.url, serverId);
  log('Mobile WS connected', { deviceId: mobile.deviceId });

  log('=== PAIRING ===');
  const tokenResult = container.pairingService!.generatePairingToken();
  log('Token generated', { token: tokenResult.token, shortCode: tokenResult.shortCode });

  const pairResult = await mobile.pair(tokenResult.token);
  log('Paired', {
    deviceId: pairResult.deviceId,
    deviceToken: pairResult.deviceToken,
    isPaired: mobile.isPaired,
    isEncrypted: mobile.isEncrypted,
  });

  log('=== SUBSCRIBING (fire-and-forget, like Flutter) ===');
  mobile.subscribeFireAndForget(container.sessionId);
  await new Promise((r) => setTimeout(r, 200));
  log('Subscribe sent');

  // --- MESSAGE 1 ---
  log('');
  log('========================================');
  log('=== MESSAGE 1: "hello world" ===');
  log('========================================');

  const ack1 = await mobile.sendNewTask('hello world', container.sessionId);
  log('new_task_ack received', { type: ack1.type, payload: ack1.payload });

  log('--- Collecting events for message 1 ---');
  const msg1Events: SessionEvent[] = [];
  for (let i = 0; i < 15; i++) {
    try {
      const event = await mobile.waitForEvent(() => true, 10_000);
      log(`  Event[${i}]`, { type: event.type, role: event.role, content: event.content ? String(event.content).slice(0, 80) : undefined });
      msg1Events.push(event);
      if (event.type === 'ProcessingStopped') break;
    } catch (err) {
      log(`  Event[${i}] TIMEOUT`, (err as Error).message);
      break;
    }
  }

  const msg1Types = msg1Events.map((e) => e.type);
  log('Message 1 event types', msg1Types);
  log('Message 1 checks', {
    hasProcessingStarted: msg1Types.includes('ProcessingStarted'),
    hasMessageCompleted: msg1Types.includes('MessageCompleted'),
    hasProcessingStopped: msg1Types.includes('ProcessingStopped'),
    assistantMcCount: msg1Events.filter((e) => e.type === 'MessageCompleted' && e.role === 'assistant').length,
    userMcCount: msg1Events.filter((e) => e.type === 'MessageCompleted' && e.role === 'user').length,
  });

  // Check leftover events
  const leftover1 = mobile.drainEvents();
  if (leftover1.length > 0) {
    log('LEFTOVER events after message 1', leftover1.map((e) => e.type));
  }

  // --- MESSAGE 2 ---
  log('');
  log('========================================');
  log('=== MESSAGE 2: "second message" ===');
  log('========================================');

  log('Stream connected before msg2?', container.streamGateway.isConnected());
  log('isProcessing before msg2?', container.streamProcessor.getIsProcessing());

  const ack2 = await mobile.sendNewTask('second message', container.sessionId);
  log('new_task_ack received', { type: ack2.type, payload: ack2.payload });

  log('--- Collecting events for message 2 ---');
  const msg2Events: SessionEvent[] = [];
  for (let i = 0; i < 15; i++) {
    try {
      const event = await mobile.waitForEvent(() => true, 10_000);
      log(`  Event[${i}]`, { type: event.type, role: event.role, content: event.content ? String(event.content).slice(0, 80) : undefined });
      msg2Events.push(event);
      if (event.type === 'ProcessingStopped') break;
    } catch (err) {
      log(`  Event[${i}] TIMEOUT`, (err as Error).message);
      break;
    }
  }

  const msg2Types = msg2Events.map((e) => e.type);
  log('Message 2 event types', msg2Types);
  log('Message 2 checks', {
    hasProcessingStarted: msg2Types.includes('ProcessingStarted'),
    hasMessageCompleted: msg2Types.includes('MessageCompleted'),
    hasProcessingStopped: msg2Types.includes('ProcessingStopped'),
    assistantMcCount: msg2Events.filter((e) => e.type === 'MessageCompleted' && e.role === 'assistant').length,
    userMcCount: msg2Events.filter((e) => e.type === 'MessageCompleted' && e.role === 'user').length,
  });

  // --- MESSAGE 3 ---
  log('');
  log('========================================');
  log('=== MESSAGE 3: "third message" ===');
  log('========================================');

  log('Stream connected before msg3?', container.streamGateway.isConnected());
  log('isProcessing before msg3?', container.streamProcessor.getIsProcessing());

  const ack3 = await mobile.sendNewTask('third message', container.sessionId);
  log('new_task_ack received', { type: ack3.type, payload: ack3.payload });

  log('--- Collecting events for message 3 ---');
  const msg3Events: SessionEvent[] = [];
  for (let i = 0; i < 15; i++) {
    try {
      const event = await mobile.waitForEvent(() => true, 10_000);
      log(`  Event[${i}]`, { type: event.type, role: event.role, content: event.content ? String(event.content).slice(0, 80) : undefined });
      msg3Events.push(event);
      if (event.type === 'ProcessingStopped') break;
    } catch (err) {
      log(`  Event[${i}] TIMEOUT`, (err as Error).message);
      break;
    }
  }

  const msg3Types = msg3Events.map((e) => e.type);
  log('Message 3 event types', msg3Types);
  log('Message 3 checks', {
    hasProcessingStarted: msg3Types.includes('ProcessingStarted'),
    hasMessageCompleted: msg3Types.includes('MessageCompleted'),
    hasProcessingStopped: msg3Types.includes('ProcessingStopped'),
    assistantMcCount: msg3Events.filter((e) => e.type === 'MessageCompleted' && e.role === 'assistant').length,
    userMcCount: msg3Events.filter((e) => e.type === 'MessageCompleted' && e.role === 'user').length,
  });

  // --- CLEANUP ---
  log('');
  log('=== SUMMARY ===');
  log(`Message 1: ${msg1Types.includes('ProcessingStopped') ? 'OK' : 'FAILED'} — ${msg1Types.join(' → ')}`);
  log(`Message 2: ${msg2Types.includes('ProcessingStopped') ? 'OK' : 'FAILED'} — ${msg2Types.join(' → ')}`);
  log(`Message 3: ${msg3Types.includes('ProcessingStopped') ? 'OK' : 'FAILED'} — ${msg3Types.join(' → ')}`);

  mobile.disconnect();
  resetContainer();
  await bridge.stop();
  await server.stop();

  log('=== DONE ===');
  process.exit(0);
}

main().catch((err) => {
  console.error('FATAL:', err);
  process.exit(1);
});
