/**
 * MobilePairingBlock — onboarding step for configuring mobile app via Bridge.
 *
 * Creates a lightweight, temporary bridge stack for initial pairing:
 * 1. User chooses bridge endpoint (cloud / self-hosted / skip)
 * 2. Connects to bridge, generates pairing token, shows QR
 * 3. Waits for mobile device to complete pairing (up to 5 minutes)
 * 4. Saves bridge config; Container.initializeBridge() re-creates its own stack later
 */

import { OnboardingBlock, OnboardingCheckResult, OnboardingContext, OnboardingResult } from '../OnboardingBlock.js';
import { OnboardingStateStore } from '../../../infrastructure/onboarding/OnboardingStateStore.js';
import { ByteBrewConfig } from '../../../infrastructure/config/ByteBrewConfig.js';
import { ByteBrewDatabase } from '../../../infrastructure/persistence/ByteBrewDatabase.js';
import { CliIdentity } from '../../../infrastructure/config/CliIdentity.js';
import { CryptoService } from '../../../infrastructure/mobile/CryptoService.js';
import { SqliteDeviceStore } from '../../../infrastructure/mobile/stores/SqliteDeviceStore.js';
import { InMemoryPairingTokenStore } from '../../../infrastructure/mobile/stores/InMemoryPairingTokenStore.js';
import { PairingWaiter } from '../../../infrastructure/mobile/PairingWaiter.js';
import { DeviceCryptoAdapter } from '../../../infrastructure/mobile/DeviceCryptoAdapter.js';
import { BridgeConnector } from '../../../infrastructure/bridge/BridgeConnector.js';
import { BridgeMessageRouter } from '../../../infrastructure/bridge/BridgeMessageRouter.js';
import { PairingService } from '../../../application/services/PairingService.js';
import { QrPairingCodeGenerator } from '../../../infrastructure/mobile/QrPairingCodeGenerator.js';
import { buildPairResponse } from '../../../infrastructure/mobile/pairRequestHandler.js';
import { prompt } from '../../../infrastructure/auth/prompt.js';

const DEFAULT_BRIDGE_URL = 'bridge.bytebrew.ai:443';
const PAIRING_TIMEOUT_MS = 5 * 60 * 1000; // 5 minutes

export class MobilePairingBlock implements OnboardingBlock {
  readonly id = 'mobile-pairing';
  readonly displayName = 'Mobile App';
  readonly description = 'Set up ByteBrew mobile app for remote access';
  readonly priority = 10;
  readonly skippable = true;

  private stateStore: OnboardingStateStore;

  constructor(stateStore?: OnboardingStateStore) {
    this.stateStore = stateStore ?? new OnboardingStateStore();
  }

  check(): OnboardingCheckResult {
    const state = this.stateStore.getBlockState(this.id);

    if (state?.completedAt) {
      return { needsOnboarding: false, skipCount: 0 };
    }

    return {
      needsOnboarding: true,
      lastSkippedAt: state?.lastSkippedAt ? new Date(state.lastSkippedAt) : undefined,
      skipCount: state?.skipCount ?? 0,
    };
  }

  async run(_context: OnboardingContext): Promise<OnboardingResult> {
    // Step 1: Choose bridge
    const bridgeChoice = await this.chooseBridge();

    if (!bridgeChoice) {
      // User chose "Skip"
      return { status: 'skipped' };
    }

    // Save bridge config
    const config = new ByteBrewConfig();
    config.setBridgeUrl(bridgeChoice.url);
    if (bridgeChoice.authToken) {
      config.setBridgeAuthToken(bridgeChoice.authToken);
    }

    // Step 2: Pairing via temporary bridge stack
    try {
      return await this.runPairing(bridgeChoice.url, bridgeChoice.authToken ?? '');
    } catch (err) {
      console.log(`\nPairing error: ${(err as Error).message}`);
      console.log('Bridge is configured. You can pair later with /mobile.');
      return { status: 'completed' };
    }
  }

  // --- Step 1: Choose bridge ---

  private async chooseBridge(): Promise<{ url: string; authToken?: string } | null> {
    console.log('');
    console.log('Mobile app for remote access:');
    console.log('  1) ByteBrew Cloud \u2014 bridge.bytebrew.ai (recommended)');
    console.log('  2) Self-hosted \u2014 your own bridge in a private network');
    console.log('  3) Skip \u2014 don\'t use mobile app');
    console.log('');

    const choice = (await prompt('Choose (1/2/3): ')).trim();

    if (choice === '3') {
      return null;
    }

    if (choice === '2') {
      return this.promptSelfHostedBridge();
    }

    // Default: cloud bridge (option 1 or any other input)
    return { url: DEFAULT_BRIDGE_URL };
  }

  private async promptSelfHostedBridge(): Promise<{ url: string; authToken?: string }> {
    const url = (await prompt('Bridge URL (e.g. bridge.example.com:8443): ')).trim();
    if (!url) {
      console.log('No URL entered, using cloud bridge.');
      return { url: DEFAULT_BRIDGE_URL };
    }

    const authToken = (await prompt('Auth token (leave empty if none): ')).trim();
    return { url, authToken: authToken || undefined };
  }

  // --- Step 2: Pairing ---

  private async runPairing(bridgeUrl: string, authToken: string): Promise<OnboardingResult> {
    // Create lightweight bridge stack
    const database = new ByteBrewDatabase();
    const cryptoService = new CryptoService();
    const cliIdentity = new CliIdentity(database, cryptoService);

    const serverId = cliIdentity.getServerId();
    const serverKeyPair = cliIdentity.getKeyPair();

    const deviceStore = new SqliteDeviceStore(database);
    const pairingTokenStore = new InMemoryPairingTokenStore();
    const pairingWaiter = new PairingWaiter();
    const cryptoAdapter = new DeviceCryptoAdapter(cryptoService, deviceStore);

    const bridgeConnector = new BridgeConnector();
    const messageRouter = new BridgeMessageRouter(cryptoAdapter);

    const pairingService = new PairingService(
      deviceStore,
      pairingTokenStore,
      cryptoService,
      pairingWaiter,
      serverKeyPair,
    );

    const qrGenerator = new QrPairingCodeGenerator();

    // Wire pair_request handler (only need pairing for onboarding)
    messageRouter.onMessage((deviceId, message) => {
      if (message.type !== 'pair_request') {
        return;
      }

      try {
        const response = buildPairResponse(deviceId, message, pairingService);

        // Send pair_response BEFORE registering alias (plaintext first —
        // mobile needs server_public_key to compute shared secret)
        messageRouter.sendMessage(deviceId, response);

        // Register alias so future encrypted messages resolve correctly
        if (response.type === 'pair_response' && response.payload?.device_id) {
          cryptoAdapter.registerAlias(deviceId, response.payload.device_id as string);
        }
      } catch (err) {
        messageRouter.sendMessage(deviceId, {
          type: 'error',
          request_id: message.request_id,
          device_id: deviceId,
          payload: { message: (err as Error).message },
        });
      }
    });

    // Start router and connect to bridge
    messageRouter.start(bridgeConnector);

    try {
      console.log('\nConnecting to bridge...');
      await bridgeConnector.connect(
        bridgeUrl,
        serverId,
        'ByteBrew CLI (onboarding)',
        authToken,
      );
      console.log('Connected.');
    } catch (err) {
      // Clean up on connection failure
      messageRouter.stop();
      database.close();
      throw err;
    }

    // Generate pairing token, register short code on bridge for manual entry
    const tokenResult = pairingService.generatePairingToken();
    const serverPublicKeyB64 = Buffer.from(tokenResult.serverPublicKey).toString('base64');
    bridgeConnector.sendRegisterCode(tokenResult.shortCode, serverPublicKeyB64);

    console.log('');
    qrGenerator.displayLocalPairingInfo({
      info: {
        serverId,
        serverPublicKey: tokenResult.serverPublicKey,
        token: tokenResult.token,
        shortCode: tokenResult.shortCode,
      },
      bridgeUrl,
    });

    console.log('\nWaiting for mobile device to scan (5 minutes)...');

    // Wait for pairing or timeout
    try {
      const pairedDevice = await pairingService.waitForPairing(tokenResult.token, PAIRING_TIMEOUT_MS);
      console.log(`\nDevice "${pairedDevice.deviceName}" paired successfully!`);
      return { status: 'completed' };
    } catch {
      // Timeout or error
      console.log('\nPairing timed out. You can pair later with /mobile.');
      return { status: 'completed' };
    } finally {
      // Cleanup: disconnect temporary bridge stack
      bridgeConnector.disconnect();
      messageRouter.stop();
      database.close();
    }
  }
}
