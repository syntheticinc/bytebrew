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

    expect(payload.s).toBe('test-server-id');
    expect(payload.t).toBe('abc123hex');
    expect(payload.b).toBe('wss://bridge.example.com');
    // server_public_key is NOT in QR — exchanged via pair_response
    expect(payload.k).toBeUndefined();
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

  test('composeLocalPayload() does not include server_public_key (exchanged via handshake)', () => {
    const gen = new QrPairingCodeGenerator();
    const info = mockLocalInfo({ serverPublicKey: new Uint8Array([72, 101, 108, 108, 111]) });
    const payload = JSON.parse(
      gen.composeLocalPayload({ info, bridgeUrl: 'wss://bridge.example.com' }),
    );

    expect(payload.k).toBeUndefined();
    expect(payload.server_public_key).toBeUndefined();
  });

  test('composeLocalPayload() always includes bridge url field', () => {
    const gen = new QrPairingCodeGenerator();
    const payload = JSON.parse(
      gen.composeLocalPayload({
        info: mockLocalInfo(),
        bridgeUrl: 'wss://custom-bridge.io',
      }),
    );

    expect(payload.b).toBe('wss://custom-bridge.io');
  });
});
