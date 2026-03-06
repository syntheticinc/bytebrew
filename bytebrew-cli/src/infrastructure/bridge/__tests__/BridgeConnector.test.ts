import { describe, test, expect, afterEach } from 'bun:test';
import { BridgeConnector } from '../BridgeConnector';

// --- Mock WS Server ---

type WsServerHandle = {
  port: number;
  close: () => void;
  connections: WebSocket[];
  lastReceivedMessages: string[];
  sendToAll: (msg: object) => void;
};

function createMockBridgeServer(): Promise<WsServerHandle> {
  return new Promise((resolve) => {
    const connections: WebSocket[] = [];
    const lastReceivedMessages: string[] = [];

    const server = Bun.serve({
      port: 0, // random port
      fetch(req, server) {
        const url = new URL(req.url);
        if (url.pathname === '/register') {
          const upgraded = server.upgrade(req);
          if (!upgraded) {
            return new Response('WebSocket upgrade failed', { status: 400 });
          }
          return undefined;
        }
        return new Response('Not found', { status: 404 });
      },
      websocket: {
        open(ws) {
          connections.push(ws as unknown as WebSocket);
        },
        message(ws, message) {
          const msgStr = typeof message === 'string' ? message : new TextDecoder().decode(message as unknown as ArrayBuffer);
          lastReceivedMessages.push(msgStr);

          const parsed = JSON.parse(msgStr);
          if (parsed.type === 'register') {
            ws.send(JSON.stringify({ type: 'registered' }));
          }
        },
        close(ws) {
          const idx = connections.indexOf(ws as unknown as WebSocket);
          if (idx !== -1) connections.splice(idx, 1);
        },
      },
    });

    resolve({
      port: server.port!,
      close: () => server.stop(true),
      connections,
      lastReceivedMessages,
      sendToAll: (msg: object) => {
        const data = JSON.stringify(msg);
        for (const ws of connections) {
          (ws as unknown as { send: (d: string) => void }).send(data);
        }
      },
    });
  });
}

// Track servers for cleanup
const servers: WsServerHandle[] = [];

afterEach(() => {
  for (const s of servers) {
    s.close();
  }
  servers.length = 0;
});

describe('BridgeConnector', () => {
  test('connects and sends register message', async () => {
    const server = await createMockBridgeServer();
    servers.push(server);

    const connector = new BridgeConnector();
    await connector.connect(`ws://localhost:${server.port}`, 'srv-1', 'TestPC', 'token-abc');

    expect(connector.isConnected()).toBe(true);

    // Verify register message was sent
    expect(server.lastReceivedMessages.length).toBeGreaterThanOrEqual(1);
    const registerMsg = JSON.parse(server.lastReceivedMessages[0]);
    expect(registerMsg.type).toBe('register');
    expect(registerMsg.server_id).toBe('srv-1');
    expect(registerMsg.server_name).toBe('TestPC');
    expect(registerMsg.auth_token).toBe('token-abc');

    connector.disconnect();
  });

  test('receives device_connected event', async () => {
    const server = await createMockBridgeServer();
    servers.push(server);

    const connector = new BridgeConnector();

    const connectedDevices: string[] = [];
    connector.onDeviceConnect((id) => connectedDevices.push(id));

    await connector.connect(`ws://localhost:${server.port}`, 'srv-1', 'TestPC', 'token');

    // Simulate device_connected from bridge
    server.sendToAll({ type: 'device_connected', device_id: 'mobile-1' });

    // Wait for message to be received
    await new Promise((r) => setTimeout(r, 50));

    expect(connectedDevices).toEqual(['mobile-1']);

    connector.disconnect();
  });

  test('receives device_disconnected event', async () => {
    const server = await createMockBridgeServer();
    servers.push(server);

    const connector = new BridgeConnector();

    const disconnectedDevices: string[] = [];
    connector.onDeviceDisconnect((id) => disconnectedDevices.push(id));

    await connector.connect(`ws://localhost:${server.port}`, 'srv-1', 'TestPC', 'token');

    server.sendToAll({ type: 'device_disconnected', device_id: 'mobile-2' });

    await new Promise((r) => setTimeout(r, 50));

    expect(disconnectedDevices).toEqual(['mobile-2']);

    connector.disconnect();
  });

  test('receives data and forwards to onData handlers', async () => {
    const server = await createMockBridgeServer();
    servers.push(server);

    const connector = new BridgeConnector();

    const received: Array<{ deviceId: string; payload: unknown }> = [];
    connector.onData((deviceId, payload) => received.push({ deviceId, payload }));

    await connector.connect(`ws://localhost:${server.port}`, 'srv-1', 'TestPC', 'token');

    server.sendToAll({
      type: 'data',
      device_id: 'mobile-3',
      payload: { type: 'new_task', text: 'hello' },
    });

    await new Promise((r) => setTimeout(r, 50));

    expect(received).toHaveLength(1);
    expect(received[0].deviceId).toBe('mobile-3');
    expect(received[0].payload).toEqual({ type: 'new_task', text: 'hello' });

    connector.disconnect();
  });

  test('sendData sends data message to bridge', async () => {
    const server = await createMockBridgeServer();
    servers.push(server);

    const connector = new BridgeConnector();
    await connector.connect(`ws://localhost:${server.port}`, 'srv-1', 'TestPC', 'token');

    // Clear register message
    server.lastReceivedMessages.length = 0;

    connector.sendData('mobile-4', { type: 'pair_response', data: 'test' });

    await new Promise((r) => setTimeout(r, 50));

    expect(server.lastReceivedMessages).toHaveLength(1);
    const sent = JSON.parse(server.lastReceivedMessages[0]);
    expect(sent.type).toBe('data');
    expect(sent.device_id).toBe('mobile-4');
    expect(sent.payload).toEqual({ type: 'pair_response', data: 'test' });

    connector.disconnect();
  });

  test('disconnect sets isConnected to false', async () => {
    const server = await createMockBridgeServer();
    servers.push(server);

    const connector = new BridgeConnector();
    await connector.connect(`ws://localhost:${server.port}`, 'srv-1', 'TestPC', 'token');

    expect(connector.isConnected()).toBe(true);

    connector.disconnect();

    expect(connector.isConnected()).toBe(false);
  });

  test('connect rejects if server is not reachable', async () => {
    const connector = new BridgeConnector();

    // Port 1 is unlikely to have a WS server
    await expect(
      connector.connect('ws://localhost:1', 'srv-1', 'TestPC', 'token')
    ).rejects.toThrow();
  });

  test('sendData is no-op when not connected', () => {
    const connector = new BridgeConnector();

    // Should not throw
    connector.sendData('dev-1', { type: 'test' });

    expect(connector.isConnected()).toBe(false);
  });

  test('buildWsUrl handles different address formats', async () => {
    // This test verifies via the connect call that URLs are built correctly.
    // We test the "host:port" format by connecting to a mock server.
    const server = await createMockBridgeServer();
    servers.push(server);

    const connector = new BridgeConnector();
    // Using host:port format (no ws:// prefix)
    await connector.connect(`localhost:${server.port}`, 'srv-1', 'TestPC', 'token');

    expect(connector.isConnected()).toBe(true);

    connector.disconnect();
  });
});
