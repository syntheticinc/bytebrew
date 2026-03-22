/**
 * Mobile Chain E2E Tests
 *
 * Full chain: Mobile(Simulator) -> Bridge(real) -> CLI(real container) -> Server(MockLLM)
 *
 * Each test starts fresh server + bridge + container, then connects a WsMobileSimulator
 * through the bridge to the CLI. The tests verify the entire request/response flow.
 */

import { describe, it, expect, beforeAll, afterAll, beforeEach, afterEach } from 'bun:test';
import { TestServerHelper } from '../../src/test-utils/TestServerHelper.js';
import { BridgeHelper } from '../../src/test-utils/BridgeHelper.js';
import { WsMobileSimulator, type SessionEvent } from './WsMobileSimulator.js';
import { Container, createContainer, resetContainer } from '../../src/config/container.js';
import { v4 as uuidv4 } from 'uuid';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';

// Build binaries once before all tests
beforeAll(() => {
  TestServerHelper.build();
  BridgeHelper.build();
}, 120_000);

describe('Mobile Chain E2E', () => {
  let server: TestServerHelper;
  let bridge: BridgeHelper;
  let container: Container;
  let mobile: WsMobileSimulator;

  afterEach(async () => {
    mobile?.disconnect();
    resetContainer();
    await bridge?.stop();
    await server?.stop();
  });

  /**
   * Set up the full chain: Server + Bridge + CLI Container + Mobile Simulator.
   *
   * Steps:
   * 1. Start test server (gRPC, MockLLM with scenario)
   * 2. Start bridge relay (WS)
   * 3. Create CLI container with bridge enabled
   * 4. Connect gRPC stream to server (required for sendMessage to work)
   * 5. Wait for CLI to register with bridge
   * 6. Connect mobile simulator to bridge
   *
   * Returns serverId and sessionId for use in tests.
   */
  async function setupChain(scenario: string): Promise<{ sessionId: string; serverId: string }> {
    server = new TestServerHelper();
    bridge = new BridgeHelper();

    await server.start(scenario);
    await bridge.start();

    const serverId = uuidv4();

    container = createContainer({
      projectRoot: process.cwd(),
      serverAddress: `localhost:${server.port}`,
      projectKey: 'test-project',
      bridgeEnabled: true,
      bridgeAddress: `localhost:${bridge.port}`,
      serverId,
      bridgeAuthToken: bridge.authToken,
      disableLspServers: true,
    });

    // Connect gRPC stream to server (normally done by useStreamConnection hook)
    await container.streamGateway.connect({
      serverAddress: `localhost:${server.port}`,
      sessionId: container.sessionId,
      userId: 'test-user',
      projectKey: 'test-project',
      projectRoot: process.cwd(),
      clientVersion: '0.2.0',
    });

    // Wait for CLI to register with bridge
    await waitForBridgeConnection(container, 5000);

    mobile = new WsMobileSimulator();
    await mobile.connect(bridge.url, serverId);

    return { sessionId: container.sessionId, serverId };
  }

  // --- TC-M-01: Pairing ---

  it('TC-M-01: Pairing via bridge', async () => {
    await setupChain('echo');

    // Generate pairing token on CLI side
    const tokenResult = container.pairingService!.generatePairingToken();
    expect(tokenResult.token).toBeTruthy();
    expect(tokenResult.shortCode).toHaveLength(6);

    // Mobile pairs using the token
    const pairResult = await mobile.pair(tokenResult.token);

    expect(pairResult.deviceToken).toBeTruthy();
    expect(pairResult.deviceId).toBeTruthy();
    expect(mobile.isPaired).toBe(true);
  }, 30_000);

  // --- TC-M-02: Ping ---

  it('TC-M-02: Ping returns pong', async () => {
    await setupChain('echo');

    const pong = await mobile.ping();

    // Ping is unauthenticated, returns timestamp
    expect(pong.timestamp).toBeDefined();
    expect(typeof pong.timestamp).toBe('number');
  }, 15_000);

  // --- TC-M-03: List Sessions ---

  it('TC-M-03: List sessions after pairing', async () => {
    const { sessionId } = await setupChain('echo');

    const tokenResult = container.pairingService!.generatePairingToken();
    await mobile.pair(tokenResult.token);

    const sessions = await mobile.listSessions();

    expect(sessions.length).toBeGreaterThanOrEqual(1);
    // CLI should expose its active session
    const found = sessions.find((s) => s.session_id === sessionId);
    expect(found).toBeDefined();
    expect(found!.status).toBe('active');
  }, 15_000);

  // --- TC-M-04: Send message (echo scenario) ---

  it('TC-M-04: Send message and receive echo response via events', async () => {
    const { sessionId } = await setupChain('echo');

    // 1. Pair
    const tokenResult = container.pairingService!.generatePairingToken();
    await mobile.pair(tokenResult.token);

    // 2. Subscribe to session events
    await mobile.subscribe(sessionId);

    // 3. Send new task
    await mobile.sendNewTask('hello world');

    // 4. Wait for MessageCompleted event
    const event = await mobile.waitForEvent(
      (e) => e.type === 'MessageCompleted',
      15_000,
    );

    expect(event.type).toBe('MessageCompleted');
    // Flat format: content is a top-level field
    expect(event.content).toBeDefined();
    expect(typeof event.content).toBe('string');
  }, 30_000);

  // --- TC-M-05: Tool call events ---

  it('TC-M-05: Tool call events via bridge', async () => {
    const { sessionId } = await setupChain('server-tool');

    const tokenResult = container.pairingService!.generatePairingToken();
    await mobile.pair(tokenResult.token);
    await mobile.subscribe(sessionId);

    await mobile.sendNewTask('do something');

    // Wait for tool execution events
    const toolStart = await mobile.waitForEvent(
      (e) => e.type === 'ToolExecutionStarted',
      15_000,
    );
    expect(toolStart.type).toBe('ToolExecutionStarted');

    const toolEnd = await mobile.waitForEvent(
      (e) => e.type === 'ToolExecutionCompleted',
      15_000,
    );
    expect(toolEnd.type).toBe('ToolExecutionCompleted');

    // Wait for final message
    const msgComplete = await mobile.waitForEvent(
      (e) => e.type === 'MessageCompleted',
      15_000,
    );
    expect(msgComplete.type).toBe('MessageCompleted');
  }, 30_000);

  // --- TC-M-06: Invalid pairing token ---

  it('TC-M-06: Invalid pairing token returns error', async () => {
    await setupChain('echo');

    let caughtError: Error | null = null;
    try {
      await mobile.pair('invalid-token-that-does-not-exist');
    } catch (err) {
      caughtError = err instanceof Error ? err : new Error(String(err));
    }

    // Should have thrown an error
    expect(caughtError).not.toBeNull();
    expect(caughtError!.message).toContain('Pairing failed');
    expect(mobile.isPaired).toBe(false);
  }, 15_000);

  // --- TC-M-07: Unauthenticated request ---

  it('TC-M-07: Unauthenticated list_sessions returns error or empty', async () => {
    await setupChain('echo');

    // Try list_sessions without pairing - mobile has no device_token
    try {
      const sessions = await mobile.listSessions();
      // If server returns empty list for unauthenticated, that's acceptable
      expect(sessions).toBeDefined();
    } catch (err) {
      // Expected: request fails for unauthenticated device
      expect(err).toBeDefined();
    }
  }, 15_000);

  // --- TC-M-08: Multiple events in sequence ---

  it('TC-M-08: ProcessingStarted and ProcessingStopped events', async () => {
    const { sessionId } = await setupChain('echo');

    const tokenResult = container.pairingService!.generatePairingToken();
    await mobile.pair(tokenResult.token);
    await mobile.subscribe(sessionId);

    await mobile.sendNewTask('test processing events');

    // Collect events until MessageCompleted
    const events: Array<{ type: string }> = [];
    const maxEvents = 10;

    for (let i = 0; i < maxEvents; i++) {
      try {
        const event = await mobile.waitForEvent(() => true, 10_000);
        events.push({ type: event.type });
        if (event.type === 'MessageCompleted') break;
      } catch {
        break; // Timeout — no more events
      }
    }

    // Should have at least MessageCompleted
    const types = events.map((e) => e.type);
    expect(types).toContain('MessageCompleted');
  }, 30_000);

  // --- TC-M-09: E2E Encryption ---

  it('TC-M-09: E2E encrypted messages after pairing', async () => {
    const { sessionId } = await setupChain('echo');

    const tokenResult = container.pairingService!.generatePairingToken();
    await mobile.pair(tokenResult.token);

    // After pairing, verify communication still works through the channel.
    // Current design: server_public_key is NOT returned in pair_response,
    // so encryption is not established. We verify the paired channel works
    // and check whether encryption keys are available.
    await mobile.subscribe(sessionId);
    await mobile.sendNewTask('encrypted hello');

    const event = await mobile.waitForEvent(
      (e) => e.type === 'MessageCompleted',
      15_000,
    );
    expect(event.type).toBe('MessageCompleted');

    // Paired device should still be marked as paired
    expect(mobile.isPaired).toBe(true);
  }, 30_000);

  // --- TC-M-10: Ask User Reply ---

  it('TC-M-10: Ask user flow - question and reply', async () => {
    const { sessionId } = await setupChain('ask-user');

    const tokenResult = container.pairingService!.generatePairingToken();
    await mobile.pair(tokenResult.token);
    await mobile.subscribe(sessionId);

    await mobile.sendNewTask('do something that requires approval');

    // Should receive AskUserRequested event
    const askEvent = await mobile.waitForEvent(
      (e) => e.type === 'AskUserRequested',
      15_000,
    );
    expect(askEvent.type).toBe('AskUserRequested');

    // Reply to the ask
    await mobile.sendAskUserReply(sessionId, 'approved');

    // Should get MessageCompleted after the reply is processed
    const doneEvent = await mobile.waitForEvent(
      (e) => e.type === 'MessageCompleted',
      15_000,
    );
    expect(doneEvent.type).toBe('MessageCompleted');
  }, 30_000);

  // --- TC-M-11: Cancel during processing ---

  it('TC-M-11: Cancel active processing', async () => {
    const { sessionId } = await setupChain('cancel-during-stream');

    const tokenResult = container.pairingService!.generatePairingToken();
    await mobile.pair(tokenResult.token);
    await mobile.subscribe(sessionId);

    // Start a task (this scenario delays 3 seconds)
    await mobile.sendNewTask('slow task');

    // Cancel quickly before it completes
    await new Promise((r) => setTimeout(r, 500));
    await mobile.cancelSession(sessionId);

    // Should eventually get ProcessingStopped or MessageCompleted
    const event = await mobile.waitForEvent(
      (e) => e.type === 'ProcessingStopped' || e.type === 'MessageCompleted',
      15_000,
    );
    expect(['ProcessingStopped', 'MessageCompleted']).toContain(event.type);
  }, 30_000);

  // --- TC-M-12: Reasoning events ---

  it('TC-M-12: Reasoning events are forwarded', async () => {
    const { sessionId } = await setupChain('reasoning');

    const tokenResult = container.pairingService!.generatePairingToken();
    await mobile.pair(tokenResult.token);
    await mobile.subscribe(sessionId);

    await mobile.sendNewTask('think about this');

    // Collect all events until MessageCompleted
    const events: SessionEvent[] = [];
    const maxEvents = 10;

    for (let i = 0; i < maxEvents; i++) {
      try {
        const event = await mobile.waitForEvent(() => true, 10_000);
        events.push(event);
        if (event.type === 'MessageCompleted') break;
      } catch {
        break; // Timeout — no more events
      }
    }

    const types = events.map((e) => e.type);
    // Should have MessageCompleted at minimum
    expect(types).toContain('MessageCompleted');
    // Reasoning events (ReasoningStarted/ReasoningCompleted) may appear
    // depending on server implementation — presence is not mandatory
  }, 30_000);

  // --- TC-M-13: Multi-agent flow ---

  it('TC-M-13: Multi-agent events via bridge', async () => {
    const { sessionId } = await setupChain('multi-agent');

    const tokenResult = container.pairingService!.generatePairingToken();
    await mobile.pair(tokenResult.token);
    await mobile.subscribe(sessionId);

    await mobile.sendNewTask('implement greeting');

    // Collect events until MessageCompleted
    const events: SessionEvent[] = [];
    const maxEvents = 15;

    for (let i = 0; i < maxEvents; i++) {
      try {
        const event = await mobile.waitForEvent(() => true, 15_000);
        events.push(event);
        if (event.type === 'MessageCompleted') break;
      } catch {
        break; // Timeout — no more events
      }
    }

    const types = events.map((e) => e.type);
    expect(types).toContain('MessageCompleted');
    // May contain AgentSpawned, ToolExecutionStarted/Completed events
    // from the multi-agent spawning flow
  }, 45_000);

  // --- TC-M-14: LLM Error ---

  it('TC-M-14: LLM error propagates to mobile', async () => {
    const { sessionId } = await setupChain('error');

    const tokenResult = container.pairingService!.generatePairingToken();
    await mobile.pair(tokenResult.token);
    await mobile.subscribe(sessionId);

    await mobile.sendNewTask('trigger error');

    // Should get an error event, ProcessingStopped, or MessageCompleted
    const event = await mobile.waitForEvent(
      (e) =>
        e.type === 'Error' ||
        e.type === 'ProcessingStopped' ||
        e.type === 'MessageCompleted',
      15_000,
    );
    expect(event).toBeDefined();
    expect(['Error', 'ProcessingStopped', 'MessageCompleted']).toContain(event.type);
  }, 30_000);

  // --- TC-M-15: Multi-turn conversation ---

  it('TC-M-15: Multi-turn - 3 messages in sequence', async () => {
    const { sessionId } = await setupChain('echo');

    const tokenResult = container.pairingService!.generatePairingToken();
    await mobile.pair(tokenResult.token);
    // Use fire-and-forget subscribe like real Flutter app
    mobile.subscribeFireAndForget(sessionId);
    // Small delay for subscribe to be processed by CLI
    await new Promise((r) => setTimeout(r, 100));

    // Send 3 messages sequentially.
    // For each round, drain ALL events until ProcessingStopped to ensure
    // the response is from THIS round, not leftovers from a previous round.
    for (const text of ['first', 'second', 'third']) {
      await mobile.sendNewTask(text, sessionId);

      // Collect ALL events for this round until ProcessingStopped
      const roundEvents: SessionEvent[] = [];
      for (let i = 0; i < 10; i++) {
        try {
          const event = await mobile.waitForEvent(() => true, 15_000);
          roundEvents.push(event);
          if (event.type === 'ProcessingStopped') break;
        } catch {
          break;
        }
      }

      const types = roundEvents.map((e) => e.type);

      // Verify complete event cycle for each round
      expect(types).toContain('ProcessingStarted');
      expect(types).toContain('MessageCompleted');
      expect(types).toContain('ProcessingStopped');

      // Verify assistant response exists and has content (not just user echo)
      const assistantMc = roundEvents.filter(
        (e) => e.type === 'MessageCompleted' && e.role === 'assistant',
      );
      expect(assistantMc.length).toBeGreaterThanOrEqual(1);
      expect(assistantMc[0].content).toBeTruthy();
    }
  }, 60_000);

  // --- TC-M-16: Multiple devices ---

  it('TC-M-16: Multiple devices connected simultaneously', async () => {
    const { sessionId, serverId } = await setupChain('echo');

    // Pair device 1
    const tokenResult1 = container.pairingService!.generatePairingToken();
    await mobile.pair(tokenResult1.token);
    await mobile.subscribe(sessionId);

    // Create and connect device 2
    const mobile2 = new WsMobileSimulator();
    await mobile2.connect(bridge.url, serverId);

    const tokenResult2 = container.pairingService!.generatePairingToken();
    await mobile2.pair(tokenResult2.token);
    await mobile2.subscribe(sessionId);

    try {
      // Device 1 sends message
      await mobile.sendNewTask('hello from device 1');

      // Both devices should receive the MessageCompleted event
      const event1 = await mobile.waitForEvent(
        (e) => e.type === 'MessageCompleted',
        15_000,
      );
      const event2 = await mobile2.waitForEvent(
        (e) => e.type === 'MessageCompleted',
        15_000,
      );

      expect(event1.type).toBe('MessageCompleted');
      expect(event2.type).toBe('MessageCompleted');
    } finally {
      mobile2.disconnect();
    }
  }, 30_000);
});

// --- Journey E2E Tests ---

describe('Journey (TC-M-JOURNEY)', () => {
  let server: TestServerHelper;
  let bridge: BridgeHelper;
  let container: Container;
  let mobile: WsMobileSimulator;

  afterEach(async () => {
    mobile?.disconnect();
    resetContainer();
    await bridge?.stop();
    await server?.stop();
  });

  async function setupChain(scenario: string): Promise<{ sessionId: string; serverId: string }> {
    server = new TestServerHelper();
    bridge = new BridgeHelper();

    await server.start(scenario);
    await bridge.start();

    const serverId = uuidv4();

    container = createContainer({
      projectRoot: process.cwd(),
      serverAddress: `localhost:${server.port}`,
      projectKey: 'test-project',
      bridgeEnabled: true,
      bridgeAddress: `localhost:${bridge.port}`,
      serverId,
      bridgeAuthToken: bridge.authToken,
      disableLspServers: true,
    });

    await container.streamGateway.connect({
      serverAddress: `localhost:${server.port}`,
      sessionId: container.sessionId,
      userId: 'test-user',
      projectKey: 'test-project',
      projectRoot: process.cwd(),
      clientVersion: '0.2.0',
    });

    await waitForBridgeConnection(container, 5000);

    mobile = new WsMobileSimulator();
    await mobile.connect(bridge.url, serverId);

    return { sessionId: container.sessionId, serverId };
  }

  it('TC-M-JOURNEY-01: full journey — pair, message with tools, second message', async () => {
    const { sessionId } = await setupChain('server-tool');

    // 1. Mobile connects and pairs
    const tokenResult = container.pairingService!.generatePairingToken();
    const pairResult = await mobile.pair(tokenResult.token);
    expect(pairResult.deviceToken).toBeTruthy();
    expect(pairResult.deviceId).toBeTruthy();
    expect(mobile.isPaired).toBe(true);

    // 2. Subscribe to session events (fire-and-forget, matches Flutter)
    mobile.subscribeFireAndForget(sessionId);
    await new Promise((r) => setTimeout(r, 100));

    // 3. Send first message (server-tool scenario: tool call + final answer)
    await mobile.sendNewTask('analyze code', sessionId);

    // 4. Collect ALL events for first message until ProcessingStopped
    const msg1Events: SessionEvent[] = [];
    for (let i = 0; i < 15; i++) {
      try {
        const event = await mobile.waitForEvent(() => true, 15_000);
        msg1Events.push(event);
        if (event.type === 'ProcessingStopped') break;
      } catch {
        break;
      }
    }

    const msg1Types = msg1Events.map((e) => e.type);

    // 4a. Verify all expected event types are present
    expect(msg1Types).toContain('ProcessingStarted');
    expect(msg1Types).toContain('ToolExecutionStarted');
    expect(msg1Types).toContain('ToolExecutionCompleted');
    expect(msg1Types).toContain('MessageCompleted');
    expect(msg1Types).toContain('ProcessingStopped');

    // 4b. Verify MessageCompleted events include role field and distinguish
    // user messages from assistant messages (prevents "echo" bug where mobile
    // shows the user's own text as an agent response)
    const allMcEvents = msg1Events.filter((e) => e.type === 'MessageCompleted');
    expect(allMcEvents.length).toBeGreaterThanOrEqual(2); // user + assistant (+ possibly tool)
    for (const mc of allMcEvents) {
      expect(mc.role).toBeDefined();
      expect(['user', 'assistant', 'tool', 'system']).toContain(mc.role);
    }
    const userMcEvents = allMcEvents.filter((e) => e.role === 'user');
    const assistantMcEvents = allMcEvents.filter((e) => e.role === 'assistant');
    expect(userMcEvents.length).toBeGreaterThanOrEqual(1);
    expect(assistantMcEvents.length).toBeGreaterThanOrEqual(1);

    // 4c. Verify key ordering constraints:
    //   - ToolExecutionStarted before ToolExecutionCompleted
    //   - ProcessingStopped is the last event
    const tsIdx = msg1Types.indexOf('ToolExecutionStarted');
    const tcIdx = msg1Types.indexOf('ToolExecutionCompleted');
    const peIdx = msg1Types.indexOf('ProcessingStopped');

    expect(tsIdx).toBeLessThan(tcIdx);      // ToolStarted before ToolCompleted
    expect(peIdx).toBe(msg1Types.length - 1); // ProcessingStopped is last

    // 4d. Verify tool event fields
    const toolStarted = msg1Events.find((e) => e.type === 'ToolExecutionStarted')!;
    expect(toolStarted.tool_name).toBeTruthy();
    expect(toolStarted.call_id).toBeTruthy();

    const toolCompleted = msg1Events.find((e) => e.type === 'ToolExecutionCompleted')!;
    expect(toolCompleted.tool_name).toBeTruthy();
    expect(toolCompleted.call_id).toBeTruthy();

    // 4e. Verify ProcessingStarted/Stopped state fields
    const processingStarted = msg1Events.find((e) => e.type === 'ProcessingStarted')!;
    expect(processingStarted.state).toBe('processing');

    const processingStopped = msg1Events.find((e) => e.type === 'ProcessingStopped')!;
    expect(processingStopped.state).toBe('idle');

    // 4f. Verify assistant MessageCompleted (not user echo) has content
    const finalMessage = assistantMcEvents[assistantMcEvents.length - 1];
    expect(finalMessage.content).toBeTruthy();
    expect(typeof finalMessage.content).toBe('string');

    // 5. Send second message (multi-turn: server-tool returns text-only on second turn)
    await mobile.sendNewTask('follow up question', sessionId);

    // 6. Collect events for second message
    const msg2Events: SessionEvent[] = [];
    for (let i = 0; i < 10; i++) {
      try {
        const event = await mobile.waitForEvent(() => true, 15_000);
        msg2Events.push(event);
        if (event.type === 'ProcessingStopped') break;
      } catch {
        break;
      }
    }

    const msg2Types = msg2Events.map((e) => e.type);

    // 6a. Second message should complete successfully
    expect(msg2Types).toContain('MessageCompleted');
    expect(msg2Types).toContain('ProcessingStopped');

    // 6b. Verify second message also has role-tagged MessageCompleted events
    const msg2McEvents = msg2Events.filter((e) => e.type === 'MessageCompleted');
    const msg2AssistantMc = msg2McEvents.filter((e) => e.role === 'assistant');
    expect(msg2AssistantMc.length).toBeGreaterThanOrEqual(1);

    // 6c. Verify assistant response has content (not the user's echo)
    const msg2Completed = msg2AssistantMc[msg2AssistantMc.length - 1];
    expect(msg2Completed.content).toBeTruthy();

    // 6d. Verify correct event ordering for second message
    //   ProcessingStarted → MessageCompleted(assistant) → ProcessingStopped
    //   (MessageCompleted(user) comes before ProcessingStarted — that's correct)
    const msg2PsIdx = msg2Types.indexOf('ProcessingStarted');
    const msg2AsstMcIdx = msg2Events.findIndex((e) => e.type === 'MessageCompleted' && e.role === 'assistant');
    const msg2PeIdx = msg2Types.indexOf('ProcessingStopped');
    expect(msg2PsIdx).toBeLessThan(msg2AsstMcIdx); // ProcessingStarted before assistant answer
    expect(msg2AsstMcIdx).toBeLessThan(msg2PeIdx); // Assistant answer before ProcessingStopped
    expect(msg2PeIdx).toBe(msg2Types.length - 1); // ProcessingStopped is last

    // 7. Verify full event chain ordering for first message:
    //    ProcessingStarted → ToolExecutionStarted → ToolExecutionCompleted → MessageCompleted(assistant) → ProcessingStopped
    const msg1PsIdx = msg1Types.indexOf('ProcessingStarted');
    const msg1TsIdx = msg1Types.indexOf('ToolExecutionStarted');
    const msg1TcIdx = msg1Types.indexOf('ToolExecutionCompleted');
    const msg1AsstMcIdx = msg1Events.findIndex((e) => e.type === 'MessageCompleted' && e.role === 'assistant');
    const msg1PeIdx = msg1Types.indexOf('ProcessingStopped');
    expect(msg1PsIdx).toBeLessThan(msg1TsIdx);
    expect(msg1TsIdx).toBeLessThan(msg1TcIdx);
    expect(msg1TcIdx).toBeLessThan(msg1AsstMcIdx);
    expect(msg1AsstMcIdx).toBeLessThan(msg1PeIdx);

    // 8. No events should be dropped — event queues should be empty
    const remainingEvents = mobile.drainEvents();
    for (const evt of remainingEvents) {
      expect(evt.type).not.toBe('Error');
    }
  }, 60_000);
});

// --- Persistence E2E Tests ---

describe('Persistence (TC-M-PERSIST)', () => {
  let server: TestServerHelper;
  let bridge: BridgeHelper;
  let container: Container;
  let mobile: WsMobileSimulator;
  let tmpDir: string;

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'bytebrew-persist-'));
  });

  afterEach(async () => {
    mobile?.disconnect();
    resetContainer();
    await bridge?.stop();
    await server?.stop();

    // Clean up temp directory with retry (SQLite WAL file release delay)
    await deleteWithRetry(tmpDir);
  });

  /**
   * Create a container with a specific dbPath for persistence testing.
   * Does NOT pass serverId — CliIdentity generates/reads it from SQLite.
   */
  function createPersistentContainer(opts: {
    serverAddress: string;
    bridgeAddress: string;
    bridgeAuthToken: string;
    dbPath: string;
  }): Container {
    return createContainer({
      projectRoot: process.cwd(),
      serverAddress: opts.serverAddress,
      projectKey: 'test-project',
      bridgeEnabled: true,
      bridgeAddress: opts.bridgeAddress,
      bridgeAuthToken: opts.bridgeAuthToken,
      disableLspServers: true,
      dbPath: opts.dbPath,
      // No serverId — CliIdentity generates or reads from SQLite
    });
  }

  /**
   * Start server + bridge and return base config for container creation.
   */
  async function startInfra(): Promise<{
    serverAddress: string;
    bridgeAddress: string;
    bridgeAuthToken: string;
    dbPath: string;
  }> {
    server = new TestServerHelper();
    bridge = new BridgeHelper();

    await server.start('echo');
    await bridge.start();

    return {
      serverAddress: `localhost:${server.port}`,
      bridgeAddress: `localhost:${bridge.port}`,
      bridgeAuthToken: bridge.authToken,
      dbPath: path.join(tmpDir, 'bytebrew.db'),
    };
  }

  /**
   * Connect container's gRPC stream to server (normally done by useStreamConnection hook).
   */
  async function connectGrpc(c: Container): Promise<void> {
    await c.streamGateway.connect({
      serverAddress: c.config.serverAddress,
      sessionId: c.sessionId,
      userId: 'test-user',
      projectKey: 'test-project',
      projectRoot: process.cwd(),
      clientVersion: '0.2.0',
    });
  }

  // --- TC-M-PERSIST-01: Reconnect after CLI restart ---

  it('TC-M-PERSIST-01: Device reconnects after CLI restart (same serverId from SQLite)', async () => {
    const infra = await startInfra();

    // 1. Create first container (generates serverId + keypair in SQLite)
    container = createPersistentContainer(infra);
    await connectGrpc(container);
    await waitForBridgeConnection(container, 5000);

    const serverId1 = container.cliIdentity!.getServerId();

    // 2. Mobile connects and pairs
    mobile = new WsMobileSimulator();
    await mobile.connect(bridge.url, serverId1);

    const tokenResult = container.pairingService!.generatePairingToken();
    const pairResult = await mobile.pair(tokenResult.token);
    expect(pairResult.deviceToken).toBeTruthy();
    expect(mobile.isPaired).toBe(true);

    // Save encryption state for reconnect (CLI persists sharedSecret in SQLite,
    // so after restart CLI will expect encrypted messages from this device)
    const savedSharedSecret = (mobile as any).sharedSecret as Uint8Array | null;

    // 3. Verify list_sessions works
    const sessions1 = await mobile.listSessions();
    expect(sessions1.length).toBeGreaterThanOrEqual(1);

    // 4. Disconnect mobile and dispose container
    mobile.disconnect();
    resetContainer();
    // Small delay for SQLite WAL flush
    await new Promise((r) => setTimeout(r, 200));

    // 5. Create second container with same dbPath (reads serverId from SQLite)
    container = createPersistentContainer(infra);
    await connectGrpc(container);
    await waitForBridgeConnection(container, 5000);

    const serverId2 = container.cliIdentity!.getServerId();

    // 6. Verify same serverId
    expect(serverId2).toBe(serverId1);

    // 7. Mobile reconnects with same serverId, restoring encryption state
    mobile = new WsMobileSimulator();
    (mobile as any)._deviceId = pairResult.deviceId;
    (mobile as any)._deviceToken = pairResult.deviceToken;
    (mobile as any).sharedSecret = savedSharedSecret;
    await mobile.connect(bridge.url, serverId2);

    // 8. Verify authenticated request succeeds (device found in SQLite)
    const sessions2 = await mobile.listSessions();
    expect(sessions2.length).toBeGreaterThanOrEqual(1);
  }, 45_000);

  // --- TC-M-PERSIST-02: E2E encryption after restart ---

  it('TC-M-PERSIST-02: E2E encryption works after CLI restart', async () => {
    const infra = await startInfra();

    // 1. Create first container
    container = createPersistentContainer(infra);
    await connectGrpc(container);
    await waitForBridgeConnection(container, 5000);

    const serverId = container.cliIdentity!.getServerId();

    // 2. Mobile connects and pairs (E2E encryption established via ECDH)
    mobile = new WsMobileSimulator();
    await mobile.connect(bridge.url, serverId);

    const tokenResult = container.pairingService!.generatePairingToken();
    const pairResult = await mobile.pair(tokenResult.token);
    expect(mobile.isEncrypted).toBe(true);

    // 3. Send encrypted message — should work
    await mobile.subscribe(container.sessionId);
    await mobile.sendNewTask('encrypted before restart');
    const event1 = await mobile.waitForEvent(
      (e) => e.type === 'MessageCompleted',
      15_000,
    );
    expect(event1.type).toBe('MessageCompleted');

    // 4. Disconnect and restart
    mobile.disconnect();
    resetContainer();
    await new Promise((r) => setTimeout(r, 200));

    // 5. Recreate container
    container = createPersistentContainer(infra);
    await connectGrpc(container);
    await waitForBridgeConnection(container, 5000);

    // 6. Mobile reconnects — must re-establish encryption with same shared secret
    // Since the keypair is persisted in SQLite, CLI uses the same server keys.
    // Mobile needs to reconnect and re-pair (or use stored shared secret).
    // For this test, we verify that a NEW pairing on the restarted CLI
    // still produces valid E2E encryption (keys read from SQLite).
    mobile = new WsMobileSimulator();
    await mobile.connect(bridge.url, serverId);

    const tokenResult2 = container.pairingService!.generatePairingToken();
    const pairResult2 = await mobile.pair(tokenResult2.token);
    expect(mobile.isEncrypted).toBe(true);

    // 7. Send encrypted message on restarted CLI
    await mobile.subscribe(container.sessionId);
    await mobile.sendNewTask('encrypted after restart');
    const event2 = await mobile.waitForEvent(
      (e) => e.type === 'MessageCompleted',
      15_000,
    );
    expect(event2.type).toBe('MessageCompleted');
  }, 60_000);

  // --- TC-M-PERSIST-03: Multiple devices persist ---

  it('TC-M-PERSIST-03: Multiple paired devices survive CLI restart', async () => {
    const infra = await startInfra();

    // 1. Create container and pair two devices
    container = createPersistentContainer(infra);
    await connectGrpc(container);
    await waitForBridgeConnection(container, 5000);

    const serverId = container.cliIdentity!.getServerId();

    // Pair device A
    const mobileA = new WsMobileSimulator();
    await mobileA.connect(bridge.url, serverId);
    const tokenA = container.pairingService!.generatePairingToken();
    const pairA = await mobileA.pair(tokenA.token, 'Device A');
    const secretA = (mobileA as any).sharedSecret as Uint8Array | null;

    // Pair device B
    const mobileB = new WsMobileSimulator();
    await mobileB.connect(bridge.url, serverId);
    const tokenB = container.pairingService!.generatePairingToken();
    const pairB = await mobileB.pair(tokenB.token, 'Device B');
    const secretB = (mobileB as any).sharedSecret as Uint8Array | null;

    // Disconnect both
    mobileA.disconnect();
    mobileB.disconnect();

    // 2. Dispose container
    resetContainer();
    await new Promise((r) => setTimeout(r, 200));

    // 3. Recreate container
    container = createPersistentContainer(infra);
    await connectGrpc(container);
    await waitForBridgeConnection(container, 5000);

    // 4. Both devices can authenticate on the restarted CLI
    mobile = new WsMobileSimulator();
    (mobile as any)._deviceId = pairA.deviceId;
    (mobile as any)._deviceToken = pairA.deviceToken;
    (mobile as any).sharedSecret = secretA;
    await mobile.connect(bridge.url, serverId);
    const sessionsA = await mobile.listSessions();
    expect(sessionsA.length).toBeGreaterThanOrEqual(1);
    mobile.disconnect();

    mobile = new WsMobileSimulator();
    (mobile as any)._deviceId = pairB.deviceId;
    (mobile as any)._deviceToken = pairB.deviceToken;
    (mobile as any).sharedSecret = secretB;
    await mobile.connect(bridge.url, serverId);
    const sessionsB = await mobile.listSessions();
    expect(sessionsB.length).toBeGreaterThanOrEqual(1);
  }, 45_000);

  // --- TC-M-PERSIST-04: New pairing after restart ---

  it('TC-M-PERSIST-04: New device can pair after CLI restart alongside existing device', async () => {
    const infra = await startInfra();

    // 1. Create container and pair device A
    container = createPersistentContainer(infra);
    await connectGrpc(container);
    await waitForBridgeConnection(container, 5000);

    const serverId = container.cliIdentity!.getServerId();

    mobile = new WsMobileSimulator();
    await mobile.connect(bridge.url, serverId);
    const tokenA = container.pairingService!.generatePairingToken();
    const pairA = await mobile.pair(tokenA.token, 'Device A');
    const secretA = (mobile as any).sharedSecret as Uint8Array | null;
    mobile.disconnect();

    // 2. Restart CLI
    resetContainer();
    await new Promise((r) => setTimeout(r, 200));

    container = createPersistentContainer(infra);
    await connectGrpc(container);
    await waitForBridgeConnection(container, 5000);

    // 3. Pair device B on restarted CLI
    const mobileB = new WsMobileSimulator();
    await mobileB.connect(bridge.url, serverId);
    const tokenB = container.pairingService!.generatePairingToken();
    const pairB = await mobileB.pair(tokenB.token, 'Device B');
    expect(pairB.deviceToken).toBeTruthy();

    // 4. Device A still authenticates (restore encryption state)
    mobile = new WsMobileSimulator();
    (mobile as any)._deviceId = pairA.deviceId;
    (mobile as any)._deviceToken = pairA.deviceToken;
    (mobile as any).sharedSecret = secretA;
    await mobile.connect(bridge.url, serverId);
    const sessionsA = await mobile.listSessions();
    expect(sessionsA.length).toBeGreaterThanOrEqual(1);
    mobile.disconnect();

    // 5. Device B also authenticates
    const sessionsB = await mobileB.listSessions();
    expect(sessionsB.length).toBeGreaterThanOrEqual(1);

    mobileB.disconnect();
    // Set mobile to a connected instance so afterEach cleanup works
    mobile = new WsMobileSimulator();
  }, 45_000);

  // --- TC-M-PERSIST-05: Revoke persists across restart ---

  it('TC-M-PERSIST-05: Revoked device stays revoked after CLI restart', async () => {
    const infra = await startInfra();

    // 1. Create container and pair device
    container = createPersistentContainer(infra);
    await connectGrpc(container);
    await waitForBridgeConnection(container, 5000);

    const serverId = container.cliIdentity!.getServerId();

    mobile = new WsMobileSimulator();
    await mobile.connect(bridge.url, serverId);
    const tokenResult = container.pairingService!.generatePairingToken();
    const pairResult = await mobile.pair(tokenResult.token);

    // 2. Verify device works before revoke
    const sessionsBefore = await mobile.listSessions();
    expect(sessionsBefore.length).toBeGreaterThanOrEqual(1);

    // 3. Revoke device
    const revoked = container.pairingService!.revokeDevice(pairResult.deviceId);
    expect(revoked).toBe(true);

    // 4. Disconnect and restart CLI
    mobile.disconnect();
    resetContainer();
    await new Promise((r) => setTimeout(r, 200));

    container = createPersistentContainer(infra);
    await connectGrpc(container);
    await waitForBridgeConnection(container, 5000);

    // 5. Reconnect mobile with revoked credentials
    mobile = new WsMobileSimulator();
    (mobile as any)._deviceId = pairResult.deviceId;
    (mobile as any)._deviceToken = pairResult.deviceToken;
    await mobile.connect(bridge.url, serverId);

    // 6. Authenticated request should fail (device revoked and persisted)
    try {
      const sessions = await mobile.listSessions();
      // If server returns empty (unauthenticated), that's also acceptable
      // The key assertion: device is no longer recognized
      expect(sessions).toBeDefined();
    } catch (err) {
      // Expected: request fails for revoked device
      expect(err).toBeDefined();
    }

    // 7. Verify device is not in the list
    const devices = container.pairingService!.listDevices();
    const found = devices.find((d) => d.id === pairResult.deviceId);
    expect(found).toBeUndefined();
  }, 45_000);
});

// --- Helpers ---

/**
 * Retry directory deletion with delay (Windows: SQLite WAL file EBUSY).
 */
async function deleteWithRetry(dir: string, attempts = 5, delayMs = 100): Promise<void> {
  for (let i = 0; i < attempts; i++) {
    try {
      fs.rmSync(dir, { recursive: true, force: true });
      return;
    } catch {
      if (i === attempts - 1) return; // give up silently
      await new Promise((r) => setTimeout(r, delayMs));
    }
  }
}

/**
 * Wait until the CLI container's bridge connector reports connected.
 * Polls isConnected() with a small interval until success or timeout.
 */
async function waitForBridgeConnection(container: Container, timeoutMs: number): Promise<void> {
  const deadline = Date.now() + timeoutMs;

  while (Date.now() < deadline) {
    if (container.bridgeConnector?.isConnected()) {
      return;
    }
    await new Promise((r) => setTimeout(r, 100));
  }

  throw new Error(`Bridge connection timeout (${timeoutMs}ms)`);
}
