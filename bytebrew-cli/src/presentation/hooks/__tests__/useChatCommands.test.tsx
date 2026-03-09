import { describe, it, expect, afterEach, beforeEach, beforeAll } from 'bun:test';
import React, { useRef } from 'react';
import { Text } from 'ink';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import { useChatCommands, type UseChatCommandsOptions } from '../useChatCommands.js';
import { AuthStorage } from '../../../infrastructure/auth/AuthStorage.js';
import { LicenseStorage } from '../../../infrastructure/license/LicenseStorage.js';

let render: typeof import('ink-testing-library').render;

beforeAll(async () => {
  const inkTesting = await import('ink-testing-library');
  render = inkTesting.render;
});

const tick = () => new Promise(r => setTimeout(r, 30));

// Track outputs from the hook
let lastCommandOutput: string | null = null;
let lastSendMessage: string | null = null;
let capturedHandleSubmit: ((value: string) => void) | null = null;

interface TestProps {
  onCommandOutput?: (output: string) => void;
  onProviderChange?: () => void;
  onLicenseChange?: () => void;
  licenseInfo?: UseChatCommandsOptions['licenseInfo'];
}

function TestComponent({ onCommandOutput, onProviderChange, onLicenseChange, licenseInfo }: TestProps) {
  const isExitingRef = useRef(false);

  const { handleSubmit } = useChatCommands({
    isConnected: true,
    isProcessing: false,
    sendMessage: (content: string) => { lastSendMessage = content; },
    clearMessages: () => {},
    disconnect: () => {},
    exit: () => {},
    isExitingRef,
    onCommandOutput: onCommandOutput ?? ((output: string) => { lastCommandOutput = output; }),
    onProviderChange,
    onLicenseChange,
    licenseInfo,
  });

  capturedHandleSubmit = handleSubmit;

  return <Text>ready</Text>;
}

/** Type-safe fetch mock — avoids Bun's `preconnect` property mismatch */
function mockFetch(handler: (...args: Parameters<typeof fetch>) => ReturnType<typeof fetch>): void {
  globalThis.fetch = handler as unknown as typeof fetch;
}

describe('useChatCommands - /provider and /model', () => {
  let instance: ReturnType<typeof render> | null = null;
  let tempDir: string;
  let originalHome: string | undefined;
  let originalUserProfile: string | undefined;

  beforeEach(() => {
    // Use temp dir as HOME so config files are isolated
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'chatcmd-test-'));
    originalHome = process.env.HOME;
    originalUserProfile = process.env.USERPROFILE;
    process.env.HOME = tempDir;
    process.env.USERPROFILE = tempDir;

    lastCommandOutput = null;
    lastSendMessage = null;
    capturedHandleSubmit = null;
  });

  afterEach(() => {
    instance?.unmount();
    instance = null;

    if (originalHome !== undefined) {
      process.env.HOME = originalHome;
    } else {
      delete process.env.HOME;
    }
    if (originalUserProfile !== undefined) {
      process.env.USERPROFILE = originalUserProfile;
    } else {
      delete process.env.USERPROFILE;
    }
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true });
    }
  });

  describe('/provider', () => {
    it('shows current provider mode (default auto)', async () => {
      instance = render(<TestComponent />);
      await tick();

      capturedHandleSubmit!('/provider');
      await tick();

      expect(lastCommandOutput).toContain('Provider: auto');
      expect(lastSendMessage).toBeNull();
    });

    it('shows proxy steps when license info available', async () => {
      instance = render(
        <TestComponent
          licenseInfo={{
            tier: 'personal',
            status: 'active',
            daysRemaining: 30,
            label: '[Personal]',
            color: 'green',
            proxyStepsRemaining: 253,
            proxyStepsLimit: 300,
            byokEnabled: true,
          }}
        />
      );
      await tick();

      capturedHandleSubmit!('/provider');
      await tick();

      expect(lastCommandOutput).toContain('Provider: auto');
      expect(lastCommandOutput).toContain('Proxy steps: 253/300');
      expect(lastCommandOutput).toContain('BYOK: enabled');
    });

    it('sets provider mode to proxy', async () => {
      instance = render(<TestComponent />);
      await tick();

      capturedHandleSubmit!('/provider proxy');
      await tick();

      expect(lastCommandOutput).toBe('Provider mode set to: proxy');

      // Verify persisted
      lastCommandOutput = null;
      capturedHandleSubmit!('/provider');
      await tick();

      expect(lastCommandOutput).not.toBeNull();
      expect(lastCommandOutput!).toContain('Provider: proxy');
    });

    it('sets provider mode to byok', async () => {
      instance = render(<TestComponent />);
      await tick();

      capturedHandleSubmit!('/provider byok');
      await tick();

      expect(lastCommandOutput).toBe('Provider mode set to: byok');
    });

    it('rejects invalid mode', async () => {
      instance = render(<TestComponent />);
      await tick();

      capturedHandleSubmit!('/provider invalid');
      await tick();

      expect(lastCommandOutput).toContain('Invalid provider mode');
      expect(lastCommandOutput).toContain('proxy, byok, auto');
    });

    it('does not send to server', async () => {
      instance = render(<TestComponent />);
      await tick();

      capturedHandleSubmit!('/provider');
      await tick();

      expect(lastSendMessage).toBeNull();
    });

    it('calls onProviderChange when mode is set', async () => {
      let providerChanged = false;
      instance = render(<TestComponent onProviderChange={() => { providerChanged = true; }} />);
      await tick();

      capturedHandleSubmit!('/provider proxy');
      await tick();

      expect(providerChanged).toBe(true);
    });

    it('does not call onProviderChange for status query', async () => {
      let providerChanged = false;
      instance = render(<TestComponent onProviderChange={() => { providerChanged = true; }} />);
      await tick();

      capturedHandleSubmit!('/provider');
      await tick();

      expect(providerChanged).toBe(false);
    });

    it('does not call onProviderChange for invalid mode', async () => {
      let providerChanged = false;
      instance = render(<TestComponent onProviderChange={() => { providerChanged = true; }} />);
      await tick();

      capturedHandleSubmit!('/provider invalid');
      await tick();

      expect(providerChanged).toBe(false);
    });
  });

  describe('/model', () => {
    it('shows no overrides message when none set', async () => {
      instance = render(<TestComponent />);
      await tick();

      capturedHandleSubmit!('/model');
      await tick();

      expect(lastCommandOutput).toContain('No model overrides configured');
    });

    it('sets a model override', async () => {
      instance = render(<TestComponent />);
      await tick();

      capturedHandleSubmit!('/model reviewer glm-5');
      await tick();

      expect(lastCommandOutput).toBe('Model override set: reviewer -> glm-5');

      // Verify persisted
      lastCommandOutput = null;
      capturedHandleSubmit!('/model');
      await tick();

      expect(lastCommandOutput).not.toBeNull();
      expect(lastCommandOutput!).toContain('reviewer -> glm-5');
    });

    it('resets model overrides', async () => {
      instance = render(<TestComponent />);
      await tick();

      capturedHandleSubmit!('/model reviewer glm-5');
      await tick();

      capturedHandleSubmit!('/model reset');
      await tick();

      expect(lastCommandOutput).toBe('Model overrides reset to defaults');

      lastCommandOutput = null;
      capturedHandleSubmit!('/model');
      await tick();

      expect(lastCommandOutput).not.toBeNull();
      expect(lastCommandOutput!).toContain('No model overrides configured');
    });

    it('shows usage for incomplete command', async () => {
      instance = render(<TestComponent />);
      await tick();

      capturedHandleSubmit!('/model reviewer');
      await tick();

      expect(lastCommandOutput).toContain('Usage:');
    });

    it('does not send to server', async () => {
      instance = render(<TestComponent />);
      await tick();

      capturedHandleSubmit!('/model reviewer glm-5');
      await tick();

      expect(lastSendMessage).toBeNull();
    });
  });

  describe('/help', () => {
    it('lists available commands including /provider and /model', async () => {
      instance = render(<TestComponent />);
      await tick();

      capturedHandleSubmit!('/help');
      await tick();

      expect(lastCommandOutput).toContain('/provider');
      expect(lastCommandOutput).toContain('/model');
      expect(lastCommandOutput).toContain('/clear');
      expect(lastCommandOutput).toContain('/quit');
    });
  });

  describe('regular messages', () => {
    it('sends non-command messages to server', async () => {
      instance = render(<TestComponent />);
      await tick();

      capturedHandleSubmit!('hello world');
      await tick();

      expect(lastSendMessage).toBe('hello world');
      expect(lastCommandOutput).toBeNull();
    });
  });

  describe('/auth commands', () => {
    const originalFetch = globalThis.fetch;

    afterEach(() => {
      globalThis.fetch = originalFetch;
    });

    it('/help contains /login, /logout, /status, /activate', async () => {
      instance = render(<TestComponent />);
      await tick();

      capturedHandleSubmit!('/help');
      await tick();

      expect(lastCommandOutput).toContain('/login');
      expect(lastCommandOutput).toContain('/logout');
      expect(lastCommandOutput).toContain('/status');
      expect(lastCommandOutput).toContain('/activate');
    });

    it('/logout calls onCommandOutput with confirmation', async () => {
      instance = render(<TestComponent />);
      await tick();

      capturedHandleSubmit!('/logout');
      await tick();

      expect(lastCommandOutput).toContain('Logged out');
      expect(lastSendMessage).toBeNull();
    });

    it('/logout calls onLicenseChange', async () => {
      let licenseChanged = false;
      instance = render(<TestComponent onLicenseChange={() => { licenseChanged = true; }} />);
      await tick();

      capturedHandleSubmit!('/logout');
      await tick();

      expect(licenseChanged).toBe(true);
    });

    it('/status without auth shows not logged in', async () => {
      instance = render(<TestComponent />);
      await tick();

      capturedHandleSubmit!('/status');
      await tick();

      expect(lastCommandOutput).toContain('Not logged in');
      expect(lastSendMessage).toBeNull();
    });

    it('/status with auth shows email', async () => {
      new AuthStorage().save({
        accessToken: 'at',
        refreshToken: 'rt',
        email: 'user@test.com',
        userId: 'u1',
      });

      instance = render(<TestComponent />);
      await tick();

      capturedHandleSubmit!('/status');
      await tick();

      expect(lastCommandOutput).toContain('Email: user@test.com');
      expect(lastCommandOutput).toContain('License: not activated');
    });

    it('/login without args shows usage', async () => {
      instance = render(<TestComponent />);
      await tick();

      capturedHandleSubmit!('/login');
      await tick();

      expect(lastCommandOutput).toBe('Usage: /login <email> <password>');
      expect(lastSendMessage).toBeNull();
    });

    it('/login with args shows "Logging in..." first', async () => {
      // Mock fetch to never resolve (we only check the sync output)
      mockFetch(() => new Promise(() => {}));

      const outputs: string[] = [];
      instance = render(
        <TestComponent onCommandOutput={(output: string) => { outputs.push(output); }} />
      );
      await tick();

      capturedHandleSubmit!('/login user@test.com password');

      // First sync output should be "Logging in..."
      expect(outputs[0]).toBe('Logging in...');
      expect(lastSendMessage).toBeNull();
    });

    it('/activate shows "Activating license..." first', async () => {
      new AuthStorage().save({
        accessToken: 'at',
        refreshToken: 'rt',
        email: 'user@test.com',
        userId: 'u1',
      });

      // Mock fetch to never resolve
      mockFetch(() => new Promise(() => {}));

      const outputs: string[] = [];
      instance = render(
        <TestComponent onCommandOutput={(output: string) => { outputs.push(output); }} />
      );
      await tick();

      capturedHandleSubmit!('/activate');

      expect(outputs[0]).toBe('Activating license...');
      expect(lastSendMessage).toBeNull();
    });

    it('/login error is displayed via onCommandOutput', async () => {
      mockFetch(async () => {
        return new Response(
          JSON.stringify({ error: { code: 'INVALID_CREDENTIALS', message: 'Invalid email or password' } }),
          { status: 401, headers: { 'Content-Type': 'application/json' } },
        );
      });

      const outputs: string[] = [];
      instance = render(
        <TestComponent onCommandOutput={(output: string) => { outputs.push(output); }} />
      );
      await tick();

      capturedHandleSubmit!('/login bad@test.com wrongpw');
      // Wait for async handler
      await new Promise(r => setTimeout(r, 200));

      expect(outputs[0]).toBe('Logging in...');
      expect(outputs.some(o => o.includes('Login failed:'))).toBe(true);
    });

    it('/activate error is displayed via onCommandOutput', async () => {
      const outputs: string[] = [];
      instance = render(
        <TestComponent onCommandOutput={(output: string) => { outputs.push(output); }} />
      );
      await tick();

      capturedHandleSubmit!('/activate');
      await new Promise(r => setTimeout(r, 200));

      expect(outputs[0]).toBe('Activating license...');
      // Without auth, handleActivateCommand returns "Not logged in. Use /login first."
      expect(outputs.some(o => o.includes('Not logged in'))).toBe(true);
    });

    it('/activate without auth shows not logged in', async () => {
      const outputs: string[] = [];
      instance = render(
        <TestComponent onCommandOutput={(output: string) => { outputs.push(output); }} />
      );
      await tick();

      capturedHandleSubmit!('/activate');

      // First output is "Activating license..." (sync), then the async result
      expect(outputs[0]).toBe('Activating license...');
      await tick();

      // Second output from the async handler: "Not logged in..."
      expect(outputs[1]).toContain('Not logged in');
    });
  });
});
