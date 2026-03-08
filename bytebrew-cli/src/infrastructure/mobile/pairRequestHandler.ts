/**
 * Shared handler for pair_request messages.
 *
 * Extracts pairing payload, validates, calls PairService.pair(),
 * and builds a pair_response (or error) MobileMessage.
 *
 * Used by:
 * - MobileRequestHandler (Container runtime — returns response, caller sends)
 * - MobilePairingBlock (onboarding — calls this, then sends + registers alias)
 */

import type { MobileMessage } from '../bridge/BridgeMessageRouter.js';

// --- Consumer-side interfaces ---

export interface PairService {
  pair(token: string, devicePublicKey: Uint8Array, deviceName: string): {
    deviceId: string;
    deviceToken: string;
    serverPublicKey: Uint8Array;
  };
}

// --- Handler ---

/**
 * Process a pair_request message and return a pair_response or error MobileMessage.
 *
 * Does NOT send the response or register crypto alias — caller is responsible for that.
 */
export function buildPairResponse(
  deviceId: string,
  message: MobileMessage,
  pairService: PairService,
): MobileMessage {
  const { token, device_public_key, device_name } = message.payload as {
    token?: string;
    device_public_key?: string;
    device_name?: string;
  };

  if (!token) {
    return errorResponse(deviceId, message.request_id, 'token is required');
  }
  if (!device_name) {
    return errorResponse(deviceId, message.request_id, 'device_name is required');
  }

  const publicKeyBytes = device_public_key
    ? Uint8Array.from(Buffer.from(device_public_key, 'base64'))
    : new Uint8Array(0);

  const result = pairService.pair(token, publicKeyBytes, device_name);

  const responsePayload: Record<string, unknown> = {
    device_id: result.deviceId,
    device_token: result.deviceToken,
  };

  if (result.serverPublicKey.length > 0) {
    responsePayload.server_public_key = Buffer.from(result.serverPublicKey).toString('base64');
  }

  return {
    type: 'pair_response',
    request_id: message.request_id,
    device_id: deviceId,
    payload: responsePayload,
  };
}

function errorResponse(deviceId: string, requestId: string, errorMsg: string): MobileMessage {
  return {
    type: 'error',
    request_id: requestId,
    device_id: deviceId,
    payload: { message: errorMsg },
  };
}
