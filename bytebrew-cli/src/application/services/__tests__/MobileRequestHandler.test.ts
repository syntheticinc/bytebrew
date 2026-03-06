import { describe, test, expect } from 'bun:test';
import { MobileRequestHandler } from '../MobileRequestHandler';
import type { MobileMessage } from '../../../infrastructure/bridge/BridgeMessageRouter';

// --- Mock factories ---

function mockPairing() {
  const paired: string[] = [];
  return {
    handler: {
      pair: (token: string, _pk: Uint8Array, name: string) => {
        paired.push(name);
        return {
          deviceId: 'new-dev-id',
          deviceToken: 'new-dev-token',
          serverPublicKey: new Uint8Array([1, 2, 3]),
        };
      },
    },
    paired,
  };
}

function mockCommand() {
  const tasks: string[] = [];
  const replies: string[] = [];
  const cancels: string[] = [];
  return {
    handler: {
      handleNewTask: (_devId: string, msg: string) => tasks.push(msg),
      handleAskUserReply: (_sid: string, reply: string) => replies.push(reply),
      handleCancel: (_sid: string) => cancels.push('cancelled'),
    },
    tasks,
    replies,
    cancels,
  };
}

function mockAuth(validToken = 'valid-token') {
  return {
    authenticateDevice: (token: string) =>
      token === validToken
        ? { id: 'auth-dev-id', name: 'AuthDevice' }
        : undefined,
  };
}

function mockSessions() {
  return {
    listSessions: () => [
      {
        sessionId: 's1',
        projectName: 'proj',
        status: 'active' as const,
        startedAt: new Date('2026-01-01'),
      },
    ],
  };
}

function mockBroadcaster() {
  const subs: string[] = [];
  const unsubs: string[] = [];
  return {
    broadcaster: {
      subscribe: (devId: string) => subs.push(devId),
      unsubscribe: (devId: string) => unsubs.push(devId),
    },
    subs,
    unsubs,
  };
}

function mockDeviceLister() {
  return {
    listDevices: () => [
      {
        id: 'd1',
        name: 'Phone',
        pairedAt: new Date('2026-01-01'),
        lastSeenAt: new Date('2026-01-02'),
      },
    ],
  };
}

function msg(type: string, payload: Record<string, unknown> = {}): MobileMessage {
  return { type, request_id: 'req-1', device_id: 'dev-1', payload };
}

function createHandler() {
  const { handler: pairing, paired } = mockPairing();
  const { handler: command, tasks, replies, cancels } = mockCommand();
  const sessions = mockSessions();
  const { broadcaster, subs, unsubs } = mockBroadcaster();
  const auth = mockAuth();
  const deviceLister = mockDeviceLister();

  const handler = new MobileRequestHandler(
    pairing, command, sessions, broadcaster, auth, deviceLister,
  );

  return { handler, paired, tasks, replies, cancels, subs, unsubs };
}

describe('MobileRequestHandler', () => {
  test('routes pair_request to PairingService', async () => {
    const { handler, paired } = createHandler();

    const resp = await handler.handleMessage('dev-1', msg('pair_request', {
      token: 'tok-1', device_name: 'iPhone', device_public_key: '',
    }));

    expect(resp?.type).toBe('pair_response');
    expect(resp?.payload?.device_id).toBe('new-dev-id');
    expect(paired).toEqual(['iPhone']);
  });

  test('routes new_task to CommandHandler after auth', async () => {
    const { handler, tasks } = createHandler();

    const resp = await handler.handleMessage('dev-1', msg('new_task', {
      device_token: 'valid-token', text: 'do something',
    }));

    expect(resp?.type).toBe('new_task_ack');
    expect(tasks).toEqual(['do something']);
  });

  test('routes cancel to CommandHandler after auth', async () => {
    const { handler, cancels } = createHandler();

    const resp = await handler.handleMessage('dev-1', msg('cancel', {
      device_token: 'valid-token', session_id: 's1',
    }));

    expect(resp?.type).toBe('cancel_ack');
    expect(cancels).toHaveLength(1);
  });

  test('unauthenticated request returns error', async () => {
    const { handler } = createHandler();

    const resp = await handler.handleMessage('dev-1', msg('new_task', {
      device_token: 'bad-token', text: 'hack',
    }));

    expect(resp?.type).toBe('error');
    expect(resp?.payload?.message).toContain('authentication failed');
  });

  test('missing device_token returns error', async () => {
    const { handler } = createHandler();

    const resp = await handler.handleMessage('dev-1', msg('new_task', { text: 'no token' }));

    expect(resp?.type).toBe('error');
    expect(resp?.payload?.message).toContain('device_token is required');
  });

  test('unknown type returns error', async () => {
    const { handler } = createHandler();

    const resp = await handler.handleMessage('dev-1', msg('unknown_cmd', {
      device_token: 'valid-token',
    }));

    expect(resp?.type).toBe('error');
    expect(resp?.payload?.message).toContain('Unknown message type');
  });

  test('ping returns pong without auth', async () => {
    const { handler } = createHandler();

    const resp = await handler.handleMessage('dev-1', msg('ping'));

    expect(resp?.type).toBe('pong');
    expect(resp?.payload?.timestamp).toBeDefined();
  });

  test('list_sessions returns sessions after auth', async () => {
    const { handler } = createHandler();

    const resp = await handler.handleMessage('dev-1', msg('list_sessions', {
      device_token: 'valid-token',
    }));

    expect(resp?.type).toBe('list_sessions_response');
    const sessions = resp?.payload?.sessions as unknown[];
    expect(sessions).toHaveLength(1);
  });

  test('subscribe routes to broadcaster after auth', async () => {
    const { handler, subs } = createHandler();

    await handler.handleMessage('dev-1', msg('subscribe', {
      device_token: 'valid-token',
    }));

    // subscribe uses bridge-level deviceId (not authenticated deviceId)
    // so that EventBroadcaster sends events to the correct bridge device ID
    expect(subs).toEqual(['dev-1']);
  });
});
