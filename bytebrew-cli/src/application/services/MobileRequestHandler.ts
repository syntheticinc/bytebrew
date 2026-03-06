/**
 * MobileRequestHandler routes incoming MobileMessages to the appropriate service.
 *
 * Incoming messages from BridgeMessageRouter are dispatched by type:
 * - pair_request -> PairingService
 * - new_task, ask_user_reply, cancel -> MobileCommandHandler (requires auth)
 * - subscribe, unsubscribe -> EventBroadcaster (requires auth)
 * - list_sessions, list_devices, ping -> read-only queries (requires auth)
 *
 * Consumer-side interfaces defined in this file (ISP).
 */

import { v4 as uuidv4 } from 'uuid';
import { getLogger } from '../../lib/logger.js';
import type { MobileMessage } from '../../infrastructure/bridge/BridgeMessageRouter.js';

const logger = getLogger();

// --- Consumer-side interfaces ---

interface PairingHandler {
  pair(token: string, devicePublicKey: Uint8Array, deviceName: string): {
    deviceId: string;
    deviceToken: string;
    serverPublicKey: Uint8Array;
  };
}

interface CommandHandler {
  handleNewTask(deviceId: string, message: string): void;
  handleAskUserReply(sessionId: string, reply: string): void;
  handleCancel(sessionId: string): void;
}

interface SessionProvider {
  listSessions(): Array<{
    sessionId: string;
    projectName: string;
    status: string;
    startedAt: Date;
  }>;
}

interface Broadcaster {
  subscribe(deviceId: string, sessionId?: string): void;
  unsubscribe(deviceId: string): void;
}

interface DeviceAuthenticator {
  authenticateDevice(deviceToken: string): { id: string; name: string } | undefined;
}

interface DeviceLister {
  listDevices(): Array<{ id: string; name: string; pairedAt: Date; lastSeenAt: Date }>;
}

// --- Service ---

export class MobileRequestHandler {
  private readonly pairingHandler: PairingHandler;
  private readonly commandHandler: CommandHandler;
  private readonly sessionProvider: SessionProvider;
  private readonly broadcaster: Broadcaster;
  private readonly authenticator: DeviceAuthenticator;
  private readonly deviceLister: DeviceLister;

  constructor(
    pairingHandler: PairingHandler,
    commandHandler: CommandHandler,
    sessionProvider: SessionProvider,
    broadcaster: Broadcaster,
    authenticator: DeviceAuthenticator,
    deviceLister: DeviceLister,
  ) {
    this.pairingHandler = pairingHandler;
    this.commandHandler = commandHandler;
    this.sessionProvider = sessionProvider;
    this.broadcaster = broadcaster;
    this.authenticator = authenticator;
    this.deviceLister = deviceLister;
  }

  /**
   * Routes an incoming message to the appropriate handler.
   * Returns a response message, or undefined if no response is needed.
   */
  async handleMessage(deviceId: string, message: MobileMessage): Promise<MobileMessage | undefined> {
    try {
      return this.dispatch(deviceId, message);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : String(err);
      logger.error('MobileRequestHandler error', {
        deviceId,
        type: message.type,
        error: errorMessage,
      });

      return this.errorResponse(deviceId, message.request_id, errorMessage);
    }
  }

  // --- Private ---

  private dispatch(deviceId: string, message: MobileMessage): MobileMessage | undefined {
    // Unauthenticated commands
    switch (message.type) {
      case 'pair_request':
        return this.handlePairRequest(deviceId, message);
      case 'ping':
        return this.handlePing(deviceId, message);
    }

    // All other commands require authentication
    const deviceToken = message.payload?.device_token as string | undefined;
    if (!deviceToken) {
      return this.errorResponse(deviceId, message.request_id, 'device_token is required');
    }

    const device = this.authenticator.authenticateDevice(deviceToken);
    if (!device) {
      return this.errorResponse(deviceId, message.request_id, 'authentication failed');
    }

    switch (message.type) {
      case 'new_task':
        return this.handleNewTask(device.id, message);
      case 'ask_user_reply':
        return this.handleAskUserReply(message);
      case 'cancel':
        return this.handleCancel(message);
      case 'subscribe':
        // Use bridge-level deviceId for subscribe/unsubscribe so that
        // EventBroadcaster sends events to the correct bridge device ID.
        return this.handleSubscribe(deviceId, message);
      case 'unsubscribe':
        return this.handleUnsubscribe(deviceId, message);
      case 'list_sessions':
        return this.handleListSessions(deviceId, message);
      case 'list_devices':
        return this.handleListDevices(deviceId, message);
      default:
        return this.errorResponse(
          deviceId,
          message.request_id,
          `Unknown message type: ${message.type}`,
        );
    }
  }

  private handlePairRequest(deviceId: string, message: MobileMessage): MobileMessage {
    const { token, device_public_key, device_name } = message.payload as {
      token?: string;
      device_public_key?: string;
      device_name?: string;
    };

    if (!token) {
      return this.errorResponse(deviceId, message.request_id, 'token is required');
    }
    if (!device_name) {
      return this.errorResponse(deviceId, message.request_id, 'device_name is required');
    }

    const publicKeyBytes = device_public_key
      ? Uint8Array.from(Buffer.from(device_public_key, 'base64'))
      : new Uint8Array(0);

    const result = this.pairingHandler.pair(token, publicKeyBytes, device_name);

    logger.info('Device paired via bridge', {
      deviceId: result.deviceId,
      deviceName: device_name,
    });

    // Include server_public_key so mobile can compute sharedSecret for E2E encryption
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

  private handleNewTask(authenticatedDeviceId: string, message: MobileMessage): MobileMessage {
    const { text, session_id } = message.payload as {
      text?: string;
      session_id?: string;
    };

    if (!text) {
      return this.errorResponse(authenticatedDeviceId, message.request_id, 'text is required');
    }

    this.commandHandler.handleNewTask(authenticatedDeviceId, text);

    return {
      type: 'new_task_ack',
      request_id: message.request_id,
      device_id: authenticatedDeviceId,
      payload: { session_id: session_id ?? '' },
    };
  }

  private handleAskUserReply(message: MobileMessage): MobileMessage {
    const { session_id, reply } = message.payload as {
      session_id?: string;
      reply?: string;
    };

    if (!session_id) {
      return this.errorResponse(message.device_id, message.request_id, 'session_id is required');
    }
    if (!reply) {
      return this.errorResponse(message.device_id, message.request_id, 'reply is required');
    }

    this.commandHandler.handleAskUserReply(session_id, reply);

    return {
      type: 'ask_user_reply_ack',
      request_id: message.request_id,
      device_id: message.device_id,
      payload: {},
    };
  }

  private handleCancel(message: MobileMessage): MobileMessage {
    const { session_id } = message.payload as { session_id?: string };

    if (!session_id) {
      return this.errorResponse(message.device_id, message.request_id, 'session_id is required');
    }

    this.commandHandler.handleCancel(session_id);

    return {
      type: 'cancel_ack',
      request_id: message.request_id,
      device_id: message.device_id,
      payload: {},
    };
  }

  private handleSubscribe(authenticatedDeviceId: string, message: MobileMessage): MobileMessage {
    const { session_id } = message.payload as { session_id?: string };

    this.broadcaster.subscribe(authenticatedDeviceId, session_id);

    return {
      type: 'subscribe_ack',
      request_id: message.request_id,
      device_id: authenticatedDeviceId,
      payload: {},
    };
  }

  private handleUnsubscribe(authenticatedDeviceId: string, message: MobileMessage): MobileMessage {
    this.broadcaster.unsubscribe(authenticatedDeviceId);

    return {
      type: 'unsubscribe_ack',
      request_id: message.request_id,
      device_id: authenticatedDeviceId,
      payload: {},
    };
  }

  private handleListSessions(deviceId: string, message: MobileMessage): MobileMessage {
    const sessions = this.sessionProvider.listSessions();

    return {
      type: 'list_sessions_response',
      request_id: message.request_id,
      device_id: deviceId,
      payload: {
        sessions: sessions.map((s) => ({
          session_id: s.sessionId,
          project_name: s.projectName,
          status: s.status,
          started_at: s.startedAt.toISOString(),
        })),
      },
    };
  }

  private handleListDevices(deviceId: string, message: MobileMessage): MobileMessage {
    const devices = this.deviceLister.listDevices();

    return {
      type: 'list_devices_response',
      request_id: message.request_id,
      device_id: deviceId,
      payload: {
        devices: devices.map((d) => ({
          device_id: d.id,
          device_name: d.name,
          paired_at: d.pairedAt.toISOString(),
          last_seen_at: d.lastSeenAt.toISOString(),
        })),
      },
    };
  }

  private handlePing(deviceId: string, message: MobileMessage): MobileMessage {
    return {
      type: 'pong',
      request_id: message.request_id,
      device_id: deviceId,
      payload: { timestamp: Date.now() },
    };
  }

  private errorResponse(deviceId: string, requestId: string, errorMsg: string): MobileMessage {
    return {
      type: 'error',
      request_id: requestId || uuidv4(),
      device_id: deviceId,
      payload: { message: errorMsg },
    };
  }
}
