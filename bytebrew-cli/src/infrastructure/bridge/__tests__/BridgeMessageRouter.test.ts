import { describe, test, expect } from 'bun:test';
import {
  BridgeMessageRouter,
  type MobileMessage,
  type IMessageCrypto,
} from '../BridgeMessageRouter';
import type { IBridgeConnector } from '../BridgeConnector';

// --- Mock BridgeConnector ---

type DataHandler = (deviceId: string, payload: unknown) => void;
type DeviceHandler = (deviceId: string) => void;

function createMockConnector() {
  let dataHandler: DataHandler | null = null;
  let connectHandler: DeviceHandler | null = null;
  let disconnectHandler: DeviceHandler | null = null;
  const sentData: Array<{ deviceId: string; payload: unknown }> = [];

  const connector: IBridgeConnector = {
    connect: async () => {},
    disconnect: () => {},
    sendData: (deviceId, payload) => sentData.push({ deviceId, payload }),
    onData: (h) => { dataHandler = h; },
    onDeviceConnect: (h) => { connectHandler = h; },
    onDeviceDisconnect: (h) => { disconnectHandler = h; },
    isConnected: () => true,
  };

  return {
    connector,
    sentData,
    simulateData: (deviceId: string, payload: unknown) => {
      dataHandler?.(deviceId, payload);
    },
    simulateConnect: (deviceId: string) => connectHandler?.(deviceId),
    simulateDisconnect: (deviceId: string) => disconnectHandler?.(deviceId),
  };
}

function sampleMessage(overrides?: Partial<MobileMessage>): MobileMessage {
  return {
    type: 'new_task',
    request_id: 'req-1',
    device_id: 'dev-1',
    payload: { text: 'hello' },
    ...overrides,
  };
}

describe('BridgeMessageRouter', () => {
  test('onMessage callback receives parsed MobileMessage from JSON object payload', () => {
    const { connector, simulateData } = createMockConnector();
    const router = new BridgeMessageRouter();
    router.start(connector);

    const received: Array<{ deviceId: string; message: MobileMessage }> = [];
    router.onMessage((deviceId, message) => received.push({ deviceId, message }));

    const msg = sampleMessage();
    // Plaintext: payload is a JSON object
    simulateData('dev-1', msg);

    expect(received).toHaveLength(1);
    expect(received[0].deviceId).toBe('dev-1');
    expect(received[0].message.type).toBe('new_task');
    expect(received[0].message.payload).toEqual({ text: 'hello' });
  });

  test('sendMessage sends JSON object via connector for plaintext', () => {
    const { connector, sentData } = createMockConnector();
    const router = new BridgeMessageRouter();
    router.start(connector);

    const msg = sampleMessage();
    router.sendMessage('dev-1', msg);

    expect(sentData).toHaveLength(1);
    expect(sentData[0].deviceId).toBe('dev-1');

    // Plaintext: payload should be a JSON object, not bytes
    const payload = sentData[0].payload as MobileMessage;
    expect(payload.type).toBe('new_task');
    expect(payload.request_id).toBe('req-1');
  });

  test('invalid payload does not crash, handler not called', () => {
    const { connector, simulateData } = createMockConnector();
    const router = new BridgeMessageRouter();
    router.start(connector);

    const received: MobileMessage[] = [];
    router.onMessage((_, msg) => received.push(msg));

    // Unexpected payload type
    simulateData('dev-1', 12345);

    expect(received).toHaveLength(0);
  });

  test('onDeviceConnect callback fires', () => {
    const { connector, simulateConnect } = createMockConnector();
    const router = new BridgeMessageRouter();
    router.start(connector);

    const connected: string[] = [];
    router.onDeviceConnect((id) => connected.push(id));

    simulateConnect('dev-42');
    expect(connected).toEqual(['dev-42']);
  });

  test('onDeviceDisconnect callback fires', () => {
    const { connector, simulateDisconnect } = createMockConnector();
    const router = new BridgeMessageRouter();
    router.start(connector);

    const disconnected: string[] = [];
    router.onDeviceDisconnect((id) => disconnected.push(id));

    simulateDisconnect('dev-99');
    expect(disconnected).toEqual(['dev-99']);
  });

  test('stop() clears handlers — no callbacks after stop', () => {
    const { connector, simulateData } = createMockConnector();
    const router = new BridgeMessageRouter();
    router.start(connector);

    const received: MobileMessage[] = [];
    router.onMessage((_, msg) => received.push(msg));

    router.stop();

    // After stop, router.running=false so handleData returns early
    simulateData('dev-1', sampleMessage());
    expect(received).toHaveLength(0);
  });

  test('unsubscribe function removes handler', () => {
    const { connector, simulateData } = createMockConnector();
    const router = new BridgeMessageRouter();
    router.start(connector);

    const received: MobileMessage[] = [];
    const unsub = router.onMessage((_, msg) => received.push(msg));

    simulateData('dev-1', sampleMessage());
    expect(received).toHaveLength(1);

    unsub();

    simulateData('dev-1', sampleMessage());
    expect(received).toHaveLength(1); // no new messages
  });

  test('sendMessage mirrors device encryption mode (encrypt if device sent encrypted)', () => {
    const crypto: IMessageCrypto = {
      hasSharedSecret: (id) => id === 'dev-paired',
      encrypt: (_id, plain) => new Uint8Array([0xff, ...plain]),
      decrypt: (_id, cipher) => cipher.slice(1),
    };

    const { connector, sentData, simulateData } = createMockConnector();
    const router = new BridgeMessageRouter(crypto);
    router.start(connector);

    // Simulate an incoming encrypted message from dev-paired (activates encryption mode)
    const msg = sampleMessage();
    const jsonBytes = new TextEncoder().encode(JSON.stringify(msg));
    const encrypted = new Uint8Array([0xff, ...jsonBytes]);
    const b64 = Buffer.from(encrypted).toString('base64');
    simulateData('dev-paired', b64);

    // Response to a paired device that sent encrypted → should be encrypted
    router.sendMessage('dev-paired', sampleMessage());
    expect(typeof sentData[0].payload).toBe('string');
    // Verify the base64 decodes to 0xff prefix + JSON
    const decoded = Buffer.from(sentData[0].payload as string, 'base64');
    expect(decoded[0]).toBe(0xff);

    // Device that sent plaintext → plaintext response even if it has shared secret
    // (simulate plaintext message first)
    simulateData('dev-paired', sampleMessage()); // object, not string
    router.sendMessage('dev-paired', sampleMessage());
    const plain = sentData[1].payload as MobileMessage;
    expect(plain.type).toBe('new_task');

    // Unpaired device: always plaintext
    router.sendMessage('dev-unpaired', sampleMessage());
    const plain2 = sentData[2].payload as MobileMessage;
    expect(plain2.type).toBe('new_task');
  });

  test('receiving encrypted data (base64 string) is decrypted', () => {
    const crypto: IMessageCrypto = {
      hasSharedSecret: (id) => id === 'dev-paired',
      encrypt: (_id, plain) => new Uint8Array([0xff, ...plain]),
      decrypt: (_id, cipher) => cipher.slice(1),
    };

    const { connector, simulateData } = createMockConnector();
    const router = new BridgeMessageRouter(crypto);
    router.start(connector);

    const received: Array<{ deviceId: string; message: MobileMessage }> = [];
    router.onMessage((deviceId, message) => received.push({ deviceId, message }));

    // Simulate encrypted payload: 0xff prefix + JSON bytes, base64 encoded
    const msg = sampleMessage();
    const jsonBytes = new TextEncoder().encode(JSON.stringify(msg));
    const encrypted = new Uint8Array([0xff, ...jsonBytes]);
    const b64 = Buffer.from(encrypted).toString('base64');

    simulateData('dev-paired', b64);

    expect(received).toHaveLength(1);
    expect(received[0].deviceId).toBe('dev-paired');
    expect(received[0].message.type).toBe('new_task');
    expect(received[0].message.payload).toEqual({ text: 'hello' });
  });
});
