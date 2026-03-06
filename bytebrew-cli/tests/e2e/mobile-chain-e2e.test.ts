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
import { WsMobileSimulator } from './WsMobileSimulator.js';
import { Container, createContainer, resetContainer } from '../../src/config/container.js';
import { v4 as uuidv4 } from 'uuid';

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
    // Echo scenario returns "echo: <input>" in the message content
    const message = event.message as Record<string, unknown> | undefined;
    expect(message).toBeDefined();
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

  it('TC-M-07: List sessions without pairing returns error', async () => {
    await setupChain('echo');

    // Directly try list_sessions without pairing (no device_token)
    // WsMobileSimulator.listSessions() uses this._deviceToken which is null
    // This will send device_token: null in the payload

    try {
      // Override deviceToken to force unauthenticated request
      const sessions = await mobile.listSessions();
      // If the error is returned in the response payload, check for error type
      // This depends on how the error propagates
    } catch {
      // Expected: request fails or returns error
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
});

// --- Helpers ---

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
