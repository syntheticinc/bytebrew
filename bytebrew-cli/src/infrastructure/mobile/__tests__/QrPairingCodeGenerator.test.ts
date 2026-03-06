import { describe, test, expect } from 'bun:test';
import { QrPairingCodeGenerator } from '../QrPairingCodeGenerator';
import type { LocalPairingInfo } from '../QrPairingCodeGenerator';

function mockLocalInfo(overrides?: Partial<LocalPairingInfo>): LocalPairingInfo {
  return {
    serverId: 'test-server-id',
    serverPublicKey: new Uint8Array([1, 2, 3, 4, 5]),
    token: 'abc123hex',
    shortCode: '482791',
    ...overrides,
  };
}

describe('QrPairingCodeGenerator', () => {
  test('composeLocalPayload() returns valid JSON with required fields', () => {
    const gen = new QrPairingCodeGenerator();
    const json = gen.composeLocalPayload({
      info: mockLocalInfo(),
      bridgeUrl: 'wss://bridge.example.com',
    });
    const payload = JSON.parse(json);

    expect(payload.sid).toBe('test-server-id');
    expect(payload.token).toBe('abc123hex');
    expect(typeof payload.spk).toBe('string');
    expect(payload.spk.length).toBeGreaterThan(0);
    expect(payload.bridge).toBe('wss://bridge.example.com');
  });

  test('composeLocalPayload() does not include lan field', () => {
    const gen = new QrPairingCodeGenerator();
    const payload = JSON.parse(
      gen.composeLocalPayload({
        info: mockLocalInfo(),
        bridgeUrl: 'wss://bridge.example.com',
      }),
    );

    expect(payload.lan).toBeUndefined();
  });

  test('composeLocalPayload() handles empty serverPublicKey', () => {
    const gen = new QrPairingCodeGenerator();
    const info = mockLocalInfo({ serverPublicKey: new Uint8Array(0) });
    const payload = JSON.parse(
      gen.composeLocalPayload({ info, bridgeUrl: 'wss://bridge.example.com' }),
    );

    expect(payload.spk).toBe('');
  });

  test('composeLocalPayload() encodes serverPublicKey as base64', () => {
    const gen = new QrPairingCodeGenerator();
    const key = new Uint8Array([72, 101, 108, 108, 111]); // "Hello"
    const info = mockLocalInfo({ serverPublicKey: key });
    const payload = JSON.parse(
      gen.composeLocalPayload({ info, bridgeUrl: 'wss://bridge.example.com' }),
    );

    expect(payload.spk).toBe(Buffer.from(key).toString('base64'));
  });

  test('composeLocalPayload() always includes bridge field', () => {
    const gen = new QrPairingCodeGenerator();
    const payload = JSON.parse(
      gen.composeLocalPayload({
        info: mockLocalInfo(),
        bridgeUrl: 'wss://custom-bridge.io',
      }),
    );

    expect(payload.bridge).toBe('wss://custom-bridge.io');
  });
});
