/**
 * Diagnostic script: full Mobile→Bridge→CLI→gRPC chain test.
 *
 * Uses an ALREADY PAIRED device from SQLite (~/.bytebrew/bytebrew.db).
 * No manual pairing needed — just run CLI in another terminal, then:
 *
 *   cd bytebrew-cli && bun tests/e2e/diagnose-chain.ts
 *
 * Prerequisites:
 *   - bytebrew CLI running in another terminal (with bridge enabled)
 *   - gRPC server running
 *   - At least one paired device in SQLite (run /mobile pair once beforehand)
 */

process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';

import path from 'path';
import { Database } from 'bun:sqlite';
import { WsMobileSimulator } from './WsMobileSimulator.js';

// --- Config ---
const BRIDGE_WS_URL = 'wss://bridge.bytebrew.ai';
const DB_PATH = path.join(process.env.USERPROFILE ?? process.env.HOME ?? '~', '.bytebrew', 'bytebrew.db');

// --- Logging ---
function ts(): string {
  const now = new Date();
  return now.toTimeString().slice(0, 8) + '.' + String(now.getMilliseconds()).padStart(3, '0');
}

function log(msg: string, detail?: unknown): void {
  const suffix = detail !== undefined
    ? ': ' + (typeof detail === 'string' ? detail : JSON.stringify(detail))
    : '';
  console.log(`[${ts()}] ${msg}${suffix}`);
}

// --- Step runner ---
interface StepResult {
  step: number;
  name: string;
  passed: boolean;
  durationMs: number;
  detail?: string;
}

const results: StepResult[] = [];
let currentStep = 0;

async function step<T>(num: number, name: string, fn: () => Promise<T>): Promise<T | null> {
  currentStep = num;
  const start = Date.now();
  process.stdout.write(`[${ts()}] [STEP ${num}] ${name}... `);
  try {
    const result = await fn();
    const ms = Date.now() - start;
    console.log(`PASS (${ms}ms)`);
    results.push({ step: num, name, passed: true, durationMs: ms });
    return result;
  } catch (err) {
    const ms = Date.now() - start;
    const msg = err instanceof Error ? err.message : String(err);
    console.log(`FAIL: ${msg}`);
    results.push({ step: num, name, passed: false, durationMs: ms, detail: msg });
    return null;
  }
}

// --- Load from SQLite ---

function loadServerId(): string {
  const db = new Database(DB_PATH, { readonly: true });
  const row = db.query('SELECT value FROM config WHERE key = ?').get('server_id') as { value: string } | null;
  db.close();
  if (!row?.value) throw new Error('server_id not found in config table');
  return row.value;
}

interface DeviceRow {
  id: string;
  name: string;
  device_token: string;
  shared_secret: Uint8Array | null;
}

function loadFirstDevice(): DeviceRow | null {
  const db = new Database(DB_PATH, { readonly: true });
  const row = db.query(
    'SELECT id, name, device_token, shared_secret FROM paired_devices LIMIT 1',
  ).get() as DeviceRow | null;
  db.close();
  return row;
}

// --- Main ---
async function main(): Promise<void> {
  console.log('');
  console.log('=== ByteBrew Chain Diagnostics ===');
  console.log(`Bridge: ${BRIDGE_WS_URL}`);
  console.log(`DB:     ${DB_PATH}`);
  console.log('');

  // Load identity
  let serverId: string;
  try {
    serverId = loadServerId();
  } catch (err) {
    console.error('ERROR: Cannot read server_id from SQLite. Is CLI initialized?');
    console.error(`  ${err instanceof Error ? err.message : err}`);
    process.exit(1);
  }
  log(`server_id: ${serverId}`);

  // Load paired device
  const device = loadFirstDevice();
  if (!device || !device.shared_secret || device.shared_secret.length === 0) {
    console.error('ERROR: No paired device with shared_secret found in SQLite.');
    console.error('  Run /mobile pair in CLI first, then re-run this script.');
    process.exit(1);
  }
  log(`device: id=${device.id} name="${device.name}" token=${device.device_token.slice(0, 8)}...`);
  log(`         shared_secret=${device.shared_secret.length} bytes`);
  console.log('');

  // Create simulator with existing device credentials (bypass pairing)
  const sim = new WsMobileSimulator();
  const simAny = sim as any;
  simAny._deviceId = device.id;
  simAny._deviceToken = device.device_token;
  simAny.sharedSecret = new Uint8Array(device.shared_secret);
  simAny.encryptCounter = 0;

  // Collect ALL events in background for diagnostics
  const allReceivedEvents: Array<{ type: string; ts: string; [k: string]: unknown }> = [];

  // ---- STEP 1: Connect to bridge ----
  const connected = await step(1, 'Connect to bridge', async () => {
    await sim.connect(BRIDGE_WS_URL, serverId);
    return true;
  });
  if (!connected) {
    printSummary();
    process.exit(1);
  }

  await new Promise((r) => setTimeout(r, 300));

  // ---- STEP 2: Ping (E2E encrypted) ----
  const pong = await step(2, 'Ping (E2E encrypted)', async () => {
    const result = await sim.ping();
    log(`  pong`, result);
    return result;
  });
  if (!pong) {
    log('Ping failed — shared_secret may be stale. Try: /mobile revoke + /mobile pair');
    sim.disconnect();
    printSummary();
    process.exit(1);
  }

  // ---- STEP 3: List sessions ----
  let sessionId: string | undefined;
  const sessions = await step(3, 'List sessions', async () => {
    const list = await sim.listSessions();
    log(`  ${list.length} session(s)`);
    for (const s of list) {
      log(`    session_id=${s.session_id} status=${s.status}`);
    }
    return list;
  });

  if (sessions && sessions.length > 0) {
    sessionId = (sessions[0].session_id as string) ?? (sessions[0].id as string);
  }

  if (!sessionId) {
    log('No active session — CLI may not have started a session yet.');
    sim.disconnect();
    printSummary();
    process.exit(0);
  }

  // ---- STEP 4: Subscribe ----
  await step(4, `Subscribe to session ${sessionId.slice(0, 8)}...`, async () => {
    await sim.subscribe(sessionId!);
  });

  // ---- STEP 5: Send new_task ----
  const ack = await step(5, 'Send new_task', async () => {
    const response = await sim.sendNewTask(
      'Ответь одним словом: какой сегодня день недели?',
      sessionId!,
    );
    log(`  ack type=${response.type}`);
    return response;
  });

  if (!ack) {
    // Try to collect any events anyway
    log('new_task_ack failed. Collecting events for 10s...');
    await collectAndLogEvents(sim, 10_000, allReceivedEvents);
    sim.disconnect();
    printSummary();
    process.exit(1);
  }

  // ---- STEP 6: Wait for ProcessingStarted ----
  await step(6, 'ProcessingStarted', async () => {
    const ev = await sim.waitForEvent((e) => e.type === 'ProcessingStarted', 30_000);
    allReceivedEvents.push({ type: ev.type, ts: ts() });
  });

  // ---- STEP 7: Wait for MessageCompleted (assistant) ----
  // This is the critical step that has been failing.
  // Collect ALL events during the wait to see what arrives instead.
  const msgCompleted = await step(7, 'MessageCompleted (assistant)', async () => {
    // Use a manual loop so we can log every event we see
    const deadline = Date.now() + 120_000;
    while (Date.now() < deadline) {
      try {
        const ev = await sim.waitForEvent(() => true, Math.min(5000, deadline - Date.now()));
        allReceivedEvents.push({ type: ev.type, ts: ts(), role: ev.role, content: ev.content ? String(ev.content).slice(0, 80) : undefined });
        log(`  [event] type=${ev.type} role=${ev.role ?? '-'} content=${ev.content ? String(ev.content).slice(0, 50) : '-'}`);

        if (ev.type === 'MessageCompleted' && ev.role === 'assistant') {
          return ev;
        }
      } catch {
        // timeout on individual wait, continue
      }
    }
    throw new Error(`Timed out (120s). Events received: ${allReceivedEvents.map(e => e.type).join(', ')}`);
  });

  // ---- STEP 8: Wait for ProcessingStopped ----
  await step(8, 'ProcessingStopped', async () => {
    const ev = await sim.waitForEvent((e) => e.type === 'ProcessingStopped', 30_000);
    allReceivedEvents.push({ type: ev.type, ts: ts() });
  });

  sim.disconnect();
  printSummary();
  process.exit(results.every((r) => r.passed) ? 0 : 1);
}

async function collectAndLogEvents(
  sim: WsMobileSimulator,
  timeoutMs: number,
  allEvents: Array<{ type: string; ts: string; [k: string]: unknown }>,
): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      const ev = await sim.waitForEvent(() => true, Math.min(2000, deadline - Date.now()));
      allEvents.push({ type: ev.type, ts: ts(), role: ev.role });
      log(`  [late event] type=${ev.type} role=${ev.role ?? '-'}`);
    } catch {
      break;
    }
  }
}

function printSummary(): void {
  console.log('');
  console.log('=== SUMMARY ===');
  let passed = 0;
  for (const r of results) {
    const icon = r.passed ? '✓' : '✗';
    const detail = r.detail ? ` — ${r.detail}` : '';
    console.log(`  [${icon}] Step ${r.step}: ${r.name} (${r.durationMs}ms)${detail}`);
    if (r.passed) passed++;
  }
  const total = 8;
  for (let i = results.length + 1; i <= total; i++) {
    console.log(`  [-] Step ${i}: skipped`);
  }
  console.log('');

  const firstFail = results.find((r) => !r.passed);
  if (!firstFail) {
    console.log(`Result: ${passed}/${results.length} steps passed ✓ — full chain works!`);
    return;
  }

  console.log(`Result: ${passed}/${results.length} passed ✗ — broken at step ${firstFail.step}`);
  console.log('');
  console.log('Diagnosis:');

  if (firstFail.step <= 2) {
    console.log('  Bridge or CLI connection broken.');
    console.log('  - Is CLI running with bridge enabled?');
    console.log('  - Check: curl https://bridge.bytebrew.ai/health');
    console.log('  - If ping failed with decrypt error: /mobile revoke + /mobile pair');
  } else if (firstFail.step <= 4) {
    console.log('  Session listing or subscribe failed.');
    console.log('  - CLI may not have active session');
    console.log('  - device_token may be rejected (re-pair)');
  } else if (firstFail.step === 5) {
    console.log('  new_task_ack not received.');
    console.log('  - CLI is not forwarding task to gRPC');
    console.log('  - Check CLI terminal for errors after receiving new_task');
    console.log('  - gRPC server may not be running');
  } else if (firstFail.step === 6) {
    console.log('  ProcessingStarted not received.');
    console.log('  - gRPC server received task but EventBroadcaster is not broadcasting');
    console.log('  - Check: EventBroadcaster.subscriptions has this deviceId');
  } else if (firstFail.step === 7) {
    console.log('  MessageCompleted (assistant) not received — THE KEY BUG.');
    console.log('  - Check CLI terminal for: "Broadcasting event" with MessageCompleted');
    console.log('  - Check CLI terminal for: "sendData OK" or "sendData DROPPED"');
    console.log('  - If sendData OK but Flutter never gets it: bridge drops the message');
    console.log('  - If no "Broadcasting event MessageCompleted": gRPC never sent isFinal=true');
  } else if (firstFail.step === 8) {
    console.log('  ProcessingStopped not received.');
    console.log('  - isFinal=true from gRPC may not trigger stopProcessing');
  }
}

process.on('SIGINT', () => {
  log('SIGINT — printing results so far');
  printSummary();
  process.exit(1);
});

main().catch((err) => {
  console.error('FATAL:', err);
  process.exit(1);
});
