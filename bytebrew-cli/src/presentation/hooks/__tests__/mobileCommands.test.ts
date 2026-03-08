import { describe, it, expect } from 'bun:test';
import { handleMobileCommand } from '../mobileCommands.js';
import type { Container } from '../../../config/container.js';
import { MobileDevice } from '../../../domain/entities/MobileDevice.js';

// --- Helpers ---

function createMockContainer(overrides: {
  bridgeConnected?: boolean;
  bridgeAddress?: string;
  serverId?: string;
  devices?: MobileDevice[];
  hasPairingService?: boolean;
  revokeResult?: boolean;
  waitForPairingResult?: { deviceId: string; deviceName: string };
  waitForPairingError?: Error;
} = {}): Container {
  const hasPairing = overrides.hasPairingService !== false;
  const devices = overrides.devices ?? [];

  const pairingService = hasPairing
    ? {
        listDevices: () => devices,
        revokeDevice: (id: string) => {
          if (overrides.revokeResult !== undefined) return overrides.revokeResult;
          return devices.some((d) => d.id === id);
        },
        generatePairingToken: () => ({
          token: 'abcdef1234567890',
          shortCode: '123456',
          serverPublicKey: new Uint8Array(32),
        }),
        waitForPairing: () => {
          if (overrides.waitForPairingError) {
            return Promise.reject(overrides.waitForPairingError);
          }
          if (overrides.waitForPairingResult) {
            return Promise.resolve(overrides.waitForPairingResult);
          }
          // Never resolves by default (simulates waiting)
          return new Promise(() => {});
        },
      }
    : null;

  const bridgeConnector =
    overrides.bridgeConnected !== undefined
      ? { isConnected: () => overrides.bridgeConnected! }
      : null;

  const cliIdentity = overrides.serverId
    ? {
        getServerId: () => overrides.serverId!,
        getKeyPair: () => ({
          publicKey: new Uint8Array(32),
          privateKey: new Uint8Array(32),
        }),
      }
    : null;

  return {
    bridgeConnector,
    pairingService,
    cliIdentity,
    config: {
      bridgeAddress: overrides.bridgeAddress ?? 'bridge.test.io:443',
    },
  } as unknown as Container;
}

function makeDevice(
  id: string,
  name: string,
  pairedAt?: Date,
  lastSeenAt?: Date,
): MobileDevice {
  return MobileDevice.fromProps({
    id,
    name,
    deviceToken: 'tok-' + id,
    publicKey: new Uint8Array(0),
    sharedSecret: new Uint8Array(0),
    pairedAt: pairedAt ?? new Date('2026-01-15T10:30:00Z'),
    lastSeenAt: lastSeenAt ?? new Date('2026-03-01T14:00:00Z'),
  });
}

async function runCommand(
  args: string,
  container: Container,
): Promise<string[]> {
  const outputs: string[] = [];
  await handleMobileCommand(args, container, (text) => outputs.push(text));
  return outputs;
}

// --- Tests ---

describe('handleMobileCommand', () => {
  // ----- /mobile help -----

  describe('help', () => {
    it('outputs help text listing all sub-commands', async () => {
      const container = createMockContainer();
      const outputs = await runCommand('help', container);
      const text = outputs.join('\n');

      expect(text).toContain('/mobile pair');
      expect(text).toContain('/mobile devices');
      expect(text).toContain('/mobile revoke');
      expect(text).toContain('/mobile status');
      expect(text).toContain('/mobile help');
    });
  });

  // ----- /mobile status -----

  describe('status', () => {
    it('shows bridge URL', async () => {
      const container = createMockContainer({
        bridgeAddress: 'bridge.example.com:8443',
      });
      const outputs = await runCommand('status', container);
      const text = outputs.join('\n');

      expect(text).toContain('bridge.example.com:8443');
    });

    it('shows "connected" when bridge is connected', async () => {
      const container = createMockContainer({
        bridgeConnected: true,
        serverId: 'srv-123',
      });
      const outputs = await runCommand('status', container);
      const text = outputs.join('\n');

      expect(text).toContain('connected');
      expect(text).not.toContain('disconnected');
    });

    it('shows "disconnected" when bridge is not connected', async () => {
      const container = createMockContainer({
        bridgeConnected: false,
      });
      const outputs = await runCommand('status', container);
      const text = outputs.join('\n');

      expect(text).toContain('disconnected');
    });

    it('shows "disconnected" when bridgeConnector is null', async () => {
      const container = createMockContainer();
      const outputs = await runCommand('status', container);
      const text = outputs.join('\n');

      expect(text).toContain('disconnected');
    });

    it('shows server ID when cliIdentity is available', async () => {
      const container = createMockContainer({
        bridgeConnected: true,
        serverId: 'abc-def-ghi',
      });
      const outputs = await runCommand('status', container);
      const text = outputs.join('\n');

      expect(text).toContain('abc-def-ghi');
    });

    it('shows paired device count', async () => {
      const devices = [
        makeDevice('d1', 'iPhone'),
        makeDevice('d2', 'Pixel'),
      ];
      const container = createMockContainer({
        bridgeConnected: true,
        devices,
      });
      const outputs = await runCommand('status', container);
      const text = outputs.join('\n');

      expect(text).toContain('2');
    });

    it('shows "not configured" when bridgeAddress is undefined', async () => {
      const container = {
        bridgeConnector: null,
        pairingService: { listDevices: () => [] },
        cliIdentity: null,
        config: { bridgeAddress: undefined },
      } as unknown as Container;

      const outputs = await runCommand('status', container);
      const text = outputs.join('\n');

      expect(text).toContain('not configured');
    });
  });

  // ----- /mobile devices -----

  describe('devices', () => {
    it('shows "No paired devices" when list is empty', async () => {
      const container = createMockContainer({ devices: [] });
      const outputs = await runCommand('devices', container);
      const text = outputs.join('\n');

      expect(text).toContain('No paired devices');
    });

    it('shows device names and truncated IDs', async () => {
      const devices = [
        makeDevice('aaaabbbb-cccc-dddd-eeee-ffffffffffff', 'iPhone 15 Pro'),
      ];
      const container = createMockContainer({ devices });
      const outputs = await runCommand('devices', container);
      const text = outputs.join('\n');

      expect(text).toContain('iPhone 15 Pro');
      expect(text).toContain('aaaabbbb...');
    });

    it('shows paired and last seen dates', async () => {
      const devices = [
        makeDevice(
          'd1-long-id-here-1234',
          'Pixel 8',
          new Date('2026-02-20T08:15:30Z'),
          new Date('2026-03-05T12:45:00Z'),
        ),
      ];
      const container = createMockContainer({ devices });
      const outputs = await runCommand('devices', container);
      const text = outputs.join('\n');

      expect(text).toContain('2026-02-20 08:15:30');
      expect(text).toContain('2026-03-05 12:45:00');
    });

    it('lists multiple devices', async () => {
      const devices = [
        makeDevice('d1-xxxxxxxx', 'iPhone'),
        makeDevice('d2-yyyyyyyy', 'Pixel'),
        makeDevice('d3-zzzzzzzz', 'Galaxy'),
      ];
      const container = createMockContainer({ devices });
      const outputs = await runCommand('devices', container);
      const text = outputs.join('\n');

      expect(text).toContain('iPhone');
      expect(text).toContain('Pixel');
      expect(text).toContain('Galaxy');
    });

    it('shows error when pairingService is null', async () => {
      const container = createMockContainer({ hasPairingService: false });
      const outputs = await runCommand('devices', container);
      const text = outputs.join('\n');

      expect(text).toContain('not available');
    });
  });

  // ----- /mobile revoke -----

  describe('revoke', () => {
    it('shows success when device is found and revoked', async () => {
      const container = createMockContainer({ revokeResult: true });
      const outputs = await runCommand('revoke some-device-id', container);
      const text = outputs.join('\n');

      expect(text).toContain('Device revoked');
    });

    it('shows "Device not found" when ID does not match', async () => {
      const container = createMockContainer({ revokeResult: false });
      const outputs = await runCommand('revoke unknown-id', container);
      const text = outputs.join('\n');

      expect(text).toContain('Device not found');
    });

    it('shows usage when no device ID is provided', async () => {
      const container = createMockContainer();
      const outputs = await runCommand('revoke', container);
      const text = outputs.join('\n');

      expect(text).toContain('Usage');
      expect(text).toContain('/mobile revoke');
    });

    it('shows error when pairingService is null', async () => {
      const container = createMockContainer({ hasPairingService: false });
      const outputs = await runCommand('revoke some-id', container);
      const text = outputs.join('\n');

      expect(text).toContain('not available');
    });
  });

  // ----- /mobile pair -----

  describe('pair', () => {
    it('shows error when bridge is not connected', async () => {
      const container = createMockContainer({ bridgeConnected: false });
      const outputs = await runCommand('pair', container);
      const text = outputs.join('\n');

      expect(text).toContain('Bridge not connected');
    });

    it('shows error when bridgeConnector is null', async () => {
      // bridgeConnector defaults to null when bridgeConnected is not set
      const container = createMockContainer();
      const outputs = await runCommand('pair', container);
      const text = outputs.join('\n');

      expect(text).toContain('Bridge not connected');
    });

    it('shows error when pairingService is null', async () => {
      const container = createMockContainer({
        bridgeConnected: true,
        hasPairingService: false,
      });
      const outputs = await runCommand('pair', container);
      const text = outputs.join('\n');

      expect(text).toContain('not available');
    });

    it('shows error when bridgeAddress is not configured', async () => {
      const container = {
        bridgeConnector: { isConnected: () => true },
        pairingService: {
          generatePairingToken: () => ({
            token: 'tok',
            shortCode: '111111',
            serverPublicKey: new Uint8Array(32),
          }),
        },
        cliIdentity: null,
        config: { bridgeAddress: undefined },
      } as unknown as Container;

      const outputs = await runCommand('pair', container);
      const text = outputs.join('\n');

      expect(text).toContain('Bridge address not configured');
    });

    it('shows short code and wait message on successful start', async () => {
      const container = createMockContainer({
        bridgeConnected: true,
        bridgeAddress: 'bridge.test.io:443',
        serverId: 'srv-1',
      });

      // handlePair never resolves (waitForPairing), so we race with a timeout
      const outputs: string[] = [];
      const pairPromise = handleMobileCommand('pair', container, (text) =>
        outputs.push(text),
      );

      // Give it a tick to print initial output before waitForPairing blocks
      await new Promise((r) => setTimeout(r, 50));

      const text = outputs.join('\n');
      expect(text).toContain('123456');
      expect(text).toContain('Waiting for device to pair');

      // Note: pairPromise never resolves in this test — that is expected.
      // We only test the synchronous output before waitForPairing.
      void pairPromise;
    });

    it('shows success message when pairing completes', async () => {
      const container = createMockContainer({
        bridgeConnected: true,
        bridgeAddress: 'bridge.test.io:443',
        serverId: 'srv-1',
        waitForPairingResult: {
          deviceId: 'new-device-id',
          deviceName: 'My Phone',
        },
      });

      const outputs = await runCommand('pair', container);
      const text = outputs.join('\n');

      expect(text).toContain('My Phone');
      expect(text).toContain('paired successfully');
    });

    it('shows timeout message when pairing times out', async () => {
      const container = createMockContainer({
        bridgeConnected: true,
        bridgeAddress: 'bridge.test.io:443',
        serverId: 'srv-1',
        waitForPairingError: new Error('pairing timeout'),
      });

      const outputs = await runCommand('pair', container);
      const text = outputs.join('\n');

      expect(text).toContain('Pairing timed out');
    });
  });

  // ----- default (empty args) -----

  describe('default sub-command', () => {
    it('defaults to pair when args is empty', async () => {
      const container = createMockContainer({ bridgeConnected: false });
      const outputs = await runCommand('', container);
      const text = outputs.join('\n');

      // Empty args → defaults to 'pair' → bridge not connected error
      expect(text).toContain('Bridge not connected');
    });
  });

  // ----- unknown sub-command -----

  describe('unknown sub-command', () => {
    it('shows unknown command error and help text', async () => {
      const container = createMockContainer();
      const outputs = await runCommand('foobar', container);
      const text = outputs.join('\n');

      expect(text).toContain('Unknown sub-command: foobar');
      expect(text).toContain('/mobile pair');
      expect(text).toContain('/mobile help');
    });
  });
});
