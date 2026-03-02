import { describe, test, expect } from 'bun:test';
import { QrPairingCodeGenerator } from '../QrPairingCodeGenerator';
import type { GeneratePairingTokenResponse } from '../../grpc/mobile_client';

class TestableGenerator extends QrPairingCodeGenerator {
  constructor(private fixedLanIp?: string) {
    super();
  }
  override detectLanIp(): string | undefined {
    return this.fixedLanIp;
  }
}

function mockResponse(overrides?: Partial<GeneratePairingTokenResponse>): GeneratePairingTokenResponse {
  return {
    shortCode: '482791',
    token: 'abc123hex',
    expiresAt: '1709312400',
    serverName: 'TestServer',
    serverId: 'test-server-id',
    serverPort: 60401,
    serverPublicKey: Buffer.from('test-public-key-bytes'),
    ...overrides,
  };
}

describe('QrPairingCodeGenerator', () => {
  test('composePayload() returns valid JSON with required fields', () => {
    const gen = new TestableGenerator('192.168.1.10');
    const json = gen.composePayload({ response: mockResponse() });
    const payload = JSON.parse(json);

    expect(payload.sid).toBe('test-server-id');
    expect(payload.token).toBe('abc123hex');
    expect(typeof payload.spk).toBe('string');
    expect(payload.spk.length).toBeGreaterThan(0);
  });

  test('composePayload() includes lan field when LAN IP is available', () => {
    const gen = new TestableGenerator('10.0.0.5');
    const payload = JSON.parse(gen.composePayload({ response: mockResponse() }));

    expect(payload.lan).toBe('10.0.0.5:60401');
  });

  test('composePayload() omits lan field when no LAN IP', () => {
    const gen = new TestableGenerator(undefined);
    const payload = JSON.parse(gen.composePayload({ response: mockResponse() }));

    expect(payload.lan).toBeUndefined();
  });

  test('composePayload() includes bridge field when bridgeUrl is provided', () => {
    const gen = new TestableGenerator('192.168.1.10');
    const payload = JSON.parse(
      gen.composePayload({ response: mockResponse(), bridgeUrl: 'wss://bridge.example.com' }),
    );

    expect(payload.bridge).toBe('wss://bridge.example.com');
  });

  test('composePayload() omits bridge field when bridgeUrl is undefined', () => {
    const gen = new TestableGenerator('192.168.1.10');
    const payload = JSON.parse(gen.composePayload({ response: mockResponse() }));

    expect(payload.bridge).toBeUndefined();
  });

  test('composePayload() handles empty serverPublicKey', () => {
    const gen = new TestableGenerator('192.168.1.10');
    const resp = mockResponse({ serverPublicKey: Buffer.alloc(0) });
    const payload = JSON.parse(gen.composePayload({ response: resp }));

    expect(payload.spk).toBe('');
  });

  test('detectLanIp() returns string or undefined', () => {
    const gen = new QrPairingCodeGenerator();
    const result = gen.detectLanIp();

    if (result !== undefined) {
      expect(typeof result).toBe('string');
      // Should be a valid IPv4 address (not link-local)
      expect(result).not.toMatch(/^169\.254\./);
    }
  });
});
