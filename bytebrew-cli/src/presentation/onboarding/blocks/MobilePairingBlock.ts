import { OnboardingBlock, OnboardingCheckResult, OnboardingContext, OnboardingResult } from '../OnboardingBlock.js';
import { OnboardingStateStore } from '../../../infrastructure/onboarding/OnboardingStateStore.js';
import { ByteBrewConfig } from '../../../infrastructure/config/ByteBrewConfig.js';
import { QrPairingCodeGenerator } from '../../../infrastructure/mobile/QrPairingCodeGenerator.js';
import { prompt } from '../../../infrastructure/auth/prompt.js';
import type { GeneratePairingTokenResponse, ListDevicesResponse } from '../../../infrastructure/grpc/mobile_client.js';

const DEFAULT_BRIDGE_URL = 'bridge.bytebrew.io:8443';

/** Consumer-side interface for mobile service operations needed by pairing. */
interface MobilePairingClient {
  generatePairingToken(): Promise<GeneratePairingTokenResponse>;
  listDevices(): Promise<ListDevicesResponse>;
  close(): void;
}

/** Factory for creating MobilePairingClient instances. */
type ClientFactory = (address: string) => MobilePairingClient;

export class MobilePairingBlock implements OnboardingBlock {
  readonly id = 'mobile-pairing';
  readonly displayName = 'Mobile App';
  readonly description = 'Set up ByteBrew mobile app for remote monitoring';
  readonly priority = 10;
  readonly skippable = true;

  private stateStore: OnboardingStateStore;
  private createClient: ClientFactory;

  constructor(stateStore?: OnboardingStateStore, createClient?: ClientFactory) {
    this.stateStore = stateStore ?? new OnboardingStateStore();
    this.createClient = createClient ?? MobilePairingBlock.defaultClientFactory;
  }

  private static defaultClientFactory(address: string): MobilePairingClient {
    // Lazy import to avoid loading gRPC at module level
    const { MobileServiceClient } = require('../../../infrastructure/grpc/mobile_client.js');
    return new MobileServiceClient(address);
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

  async run(context: OnboardingContext): Promise<OnboardingResult> {
    if (!context.serverAddress) {
      return { status: 'skipped' };
    }

    // Step 0: Ask user
    const choice = await prompt('Connect mobile app for remote monitoring? (y/n/never): ');
    const answer = choice.trim().toLowerCase();

    if (answer === 'never') {
      return { status: 'completed' };
    }
    if (answer !== 'y' && answer !== 'yes') {
      return { status: 'skipped' };
    }

    // Step 1: Bridge endpoint selection
    const bridgeUrl = await this.selectBridgeEndpoint();
    const config = new ByteBrewConfig();
    if (bridgeUrl !== undefined) {
      config.setBridgeUrl(bridgeUrl);
    } else {
      config.clearBridgeUrl();
    }

    // Steps 2-4: Generate token, show QR, verify
    try {
      const client = this.createClient(context.serverAddress);
      try {
        return await this.pairDevice(client, bridgeUrl);
      } finally {
        client.close();
      }
    } catch (err) {
      console.log(`Could not connect to mobile service: ${(err as Error).message}`);
      console.log('You can set up mobile pairing later with "bytebrew mobile-pair".');
      return { status: 'skipped' };
    }
  }

  private async selectBridgeEndpoint(): Promise<string | undefined> {
    console.log('');
    console.log('Remote access when not on same network:');
    console.log('  1) ByteBrew Cloud \u2014 bridge.bytebrew.io (default)');
    console.log('  2) Self-hosted \u2014 enter your bridge URL');
    console.log('  3) Direct LAN \u2014 same network, no bridge');
    console.log('');

    const choice = (await prompt('Choose (1/2/3): ')).trim();

    if (choice === '3') {
      return undefined;
    }

    if (choice === '2') {
      const url = (await prompt('Bridge URL: ')).trim();
      if (!url) {
        console.log('No URL entered, using default cloud bridge.');
        return DEFAULT_BRIDGE_URL;
      }
      return url;
    }

    return DEFAULT_BRIDGE_URL;
  }

  private async pairDevice(
    client: MobilePairingClient,
    bridgeUrl?: string,
  ): Promise<OnboardingResult> {
    // Capture device count before pairing to detect NEW devices
    const devicesBefore = await client.listDevices();
    const countBefore = devicesBefore.devices.length;

    // Step 2: Generate pairing token + show QR
    const result = await client.generatePairingToken();
    const generator = new QrPairingCodeGenerator();
    generator.displayPairingInfo({ response: result, bridgeUrl });

    // Step 3: Wait for scan
    console.log('');
    const response = await prompt('Press Enter after scanning, or type "skip": ');
    if (response.trim().toLowerCase() === 'skip') {
      return { status: 'skipped' };
    }

    // Step 4: Verify — check for NEW device (not just any device)
    const devicesAfter = await client.listDevices();
    if (devicesAfter.devices.length > countBefore) {
      console.log('Mobile device paired successfully!');
      return { status: 'completed' };
    }

    console.log('No paired device detected. You can pair later with "bytebrew mobile-pair".');
    return { status: 'skipped' };
  }
}
