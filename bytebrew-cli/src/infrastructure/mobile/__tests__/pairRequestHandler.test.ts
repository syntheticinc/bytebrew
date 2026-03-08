import { describe, it, expect } from 'bun:test';
import { buildPairResponse, type PairService } from '../pairRequestHandler.js';
import type { MobileMessage } from '../../bridge/BridgeMessageRouter.js';

function createPairRequest(payload: Record<string, unknown>): MobileMessage {
  return {
    type: 'pair_request',
    request_id: 'req-1',
    device_id: 'bridge-device-1',
    payload,
  };
}

function createMockPairService(overrides?: Partial<ReturnType<PairService['pair']>>): PairService {
  return {
    pair: (_token: string, _publicKey: Uint8Array, _deviceName: string) => ({
      deviceId: 'authenticated-device-1',
      deviceToken: 'tok-abc',
      serverPublicKey: new Uint8Array([1, 2, 3]),
      ...overrides,
    }),
  };
}

describe('buildPairResponse', () => {
  it('returns error when token is missing', () => {
    const message = createPairRequest({ device_name: 'iPhone' });
    const result = buildPairResponse('bridge-device-1', message, createMockPairService());

    expect(result.type).toBe('error');
    expect(result.payload.message).toBe('token is required');
    expect(result.request_id).toBe('req-1');
    expect(result.device_id).toBe('bridge-device-1');
  });

  it('returns error when device_name is missing', () => {
    const message = createPairRequest({ token: 'tok-123' });
    const result = buildPairResponse('bridge-device-1', message, createMockPairService());

    expect(result.type).toBe('error');
    expect(result.payload.message).toBe('device_name is required');
  });

  it('returns pair_response with device_id, device_token, server_public_key', () => {
    const publicKeyBase64 = Buffer.from(new Uint8Array([10, 20, 30])).toString('base64');
    const message = createPairRequest({
      token: 'tok-123',
      device_name: 'iPhone',
      device_public_key: publicKeyBase64,
    });

    const pairService = createMockPairService();
    const result = buildPairResponse('bridge-device-1', message, pairService);

    expect(result.type).toBe('pair_response');
    expect(result.request_id).toBe('req-1');
    expect(result.device_id).toBe('bridge-device-1');
    expect(result.payload.device_id).toBe('authenticated-device-1');
    expect(result.payload.device_token).toBe('tok-abc');
    expect(result.payload.server_public_key).toBe(Buffer.from(new Uint8Array([1, 2, 3])).toString('base64'));
  });

  it('omits server_public_key when empty', () => {
    const message = createPairRequest({
      token: 'tok-123',
      device_name: 'Android',
    });

    const pairService = createMockPairService({ serverPublicKey: new Uint8Array(0) });
    const result = buildPairResponse('bridge-device-1', message, pairService);

    expect(result.type).toBe('pair_response');
    expect(result.payload.server_public_key).toBeUndefined();
  });

  it('handles missing device_public_key (passes empty Uint8Array)', () => {
    const message = createPairRequest({
      token: 'tok-123',
      device_name: 'Pixel',
    });

    let capturedPublicKey: Uint8Array | undefined;
    const pairService: PairService = {
      pair: (_token, publicKey, _name) => {
        capturedPublicKey = publicKey;
        return {
          deviceId: 'dev-1',
          deviceToken: 'tok-1',
          serverPublicKey: new Uint8Array(0),
        };
      },
    };

    buildPairResponse('bridge-device-1', message, pairService);

    expect(capturedPublicKey).toEqual(new Uint8Array(0));
  });

  it('propagates pair service errors as thrown exceptions', () => {
    const message = createPairRequest({
      token: 'invalid-token',
      device_name: 'iPhone',
    });

    const pairService: PairService = {
      pair: () => { throw new Error('invalid pairing token'); },
    };

    // buildPairResponse does NOT catch exceptions — caller handles them
    expect(() => buildPairResponse('bridge-device-1', message, pairService))
      .toThrow('invalid pairing token');
  });
});
