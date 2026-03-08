/**
 * Handles /mobile slash command for managing mobile device pairing.
 *
 * Sub-commands:
 *   /mobile pair     - Generate QR code and wait for device pairing
 *   /mobile devices  - List all paired devices
 *   /mobile revoke   - Remove a paired device by ID
 *   /mobile status   - Show bridge connection status
 *   /mobile help     - Show usage
 */

import { renderQrForTerminal } from '../../infrastructure/mobile/QrPairingCodeGenerator.js';
import type { Container } from '../../config/container.js';
import { writeFileSync, mkdirSync } from 'fs';
import { homedir } from 'os';
import { join } from 'path';

type OutputFn = (text: string) => void;

function formatDate(date: Date): string {
  return date.toISOString().replace('T', ' ').slice(0, 19);
}

// --- Sub-command handlers ---

async function handlePair(container: Container, output: OutputFn): Promise<void> {
  const bridgeConnector = container.bridgeConnector;
  if (!bridgeConnector || !bridgeConnector.isConnected()) {
    output('Bridge not connected. Start with --bridge flag or run onboarding first.');
    return;
  }

  const pairingService = container.pairingService;
  if (!pairingService) {
    output('Pairing service not available. Bridge may not be configured.');
    return;
  }

  const bridgeAddress = container.config.bridgeAddress;
  if (!bridgeAddress) {
    output('Bridge address not configured.');
    return;
  }

  // Generate pairing token
  const tokenResult = pairingService.generatePairingToken();

  // Register short code on bridge for CLI display purposes
  const serverPublicKeyB64 = Buffer.from(tokenResult.serverPublicKey).toString('base64');
  bridgeConnector.sendRegisterCode(tokenResult.shortCode, serverPublicKeyB64);

  // Build QR payload (includes server_public_key so mobile can verify out-of-band)
  const serverId = container.cliIdentity?.getServerId() ?? 'unknown';
  const payloadJson = JSON.stringify({
    s: serverId,
    t: tokenResult.token,
    b: bridgeAddress,
    k: serverPublicKeyB64,
  });

  // Write QR + text as a single stdout block.
  // CRITICAL: do NOT mix process.stdout.write with Ink's output() here —
  // Ink redraws after output() and overwrites the bottom lines of the QR.
  // Save token to file so it survives Ink's terminal redraws
  const tokenFile = join(homedir(), '.bytebrew', 'pairing-token.txt');
  mkdirSync(join(homedir(), '.bytebrew'), { recursive: true });
  writeFileSync(tokenFile, tokenResult.token, 'utf8');

  const qrText = renderQrForTerminal(payloadJson);
  process.stdout.write(
    '\n' + qrText + '\n' +
    `Code: ${tokenResult.shortCode}\n` +
    `Token saved to: ${tokenFile}\n` +
    'Scan the QR code with the ByteBrew mobile app, or enter the code manually.\n' +
    'Waiting for device to pair (up to 5 minutes)...\n'
  );

  // Wait for pairing completion
  try {
    const result = await pairingService.waitForPairing(tokenResult.token, 5 * 60 * 1000);
    output(`Device "${result.deviceName}" paired successfully!`);
  } catch {
    output('Pairing timed out. Try again with /mobile pair');
  }
}

function handleDevices(container: Container, output: OutputFn): void {
  const pairingService = container.pairingService;
  if (!pairingService) {
    output('Pairing service not available. Bridge may not be configured.');
    return;
  }

  const devices = pairingService.listDevices();
  if (devices.length === 0) {
    output('No paired devices. Use /mobile pair to add one.');
    return;
  }

  const lines: string[] = ['Paired devices:'];
  for (const device of devices) {
    const paired = formatDate(device.pairedAt);
    const lastSeen = formatDate(device.lastSeenAt);
    lines.push(`  ${device.name} (${device.id.slice(0, 8)}...)`);
    lines.push(`    Paired: ${paired} | Last seen: ${lastSeen}`);
  }

  output(lines.join('\n'));
}

function handleRevoke(container: Container, output: OutputFn, deviceId: string): void {
  if (!deviceId) {
    output('Usage: /mobile revoke <device-id>');
    return;
  }

  const pairingService = container.pairingService;
  if (!pairingService) {
    output('Pairing service not available. Bridge may not be configured.');
    return;
  }

  const removed = pairingService.revokeDevice(deviceId);
  if (removed) {
    output('Device revoked.');
  } else {
    output('Device not found.');
  }
}

function handleStatus(container: Container, output: OutputFn): void {
  const lines: string[] = ['Mobile bridge status:'];

  const bridgeAddress = container.config.bridgeAddress;
  lines.push(`  Bridge URL: ${bridgeAddress ?? 'not configured'}`);

  const bridgeConnector = container.bridgeConnector;
  const connected = bridgeConnector?.isConnected() ?? false;
  lines.push(`  Connection: ${connected ? 'connected' : 'disconnected'}`);

  const serverId = container.cliIdentity?.getServerId();
  if (serverId) {
    lines.push(`  Server ID: ${serverId}`);
  }

  const pairingService = container.pairingService;
  if (pairingService) {
    const devices = pairingService.listDevices();
    lines.push(`  Paired devices: ${devices.length}`);
  }

  output(lines.join('\n'));
}

function showHelp(output: OutputFn): void {
  const help = [
    'Mobile commands:',
    '  /mobile pair          - Generate QR code for device pairing',
    '  /mobile devices       - List all paired devices',
    '  /mobile revoke <id>   - Remove a paired device',
    '  /mobile status        - Show bridge connection status',
    '  /mobile help          - Show this help',
  ].join('\n');

  output(help);
}

// --- Main handler ---

export async function handleMobileCommand(
  args: string,
  container: Container,
  output: OutputFn,
): Promise<void> {
  const parts = args.split(/\s+/).filter(Boolean);
  const subCommand = parts[0] ?? 'pair';

  switch (subCommand) {
    case 'pair':
      await handlePair(container, output);
      break;
    case 'devices':
      handleDevices(container, output);
      break;
    case 'revoke':
      handleRevoke(container, output, parts.slice(1).join(' '));
      break;
    case 'status':
      handleStatus(container, output);
      break;
    case 'help':
      showHelp(output);
      break;
    default:
      output(`Unknown sub-command: ${subCommand}`);
      showHelp(output);
      break;
  }
}
