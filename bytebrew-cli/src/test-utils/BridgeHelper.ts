import { spawn, ChildProcess, execSync } from 'child_process';
import path from 'path';
import fs from 'fs';
import net from 'net';
import { v4 as uuidv4 } from 'uuid';

const BRIDGE_DIR = path.resolve(import.meta.dir, '../../../bytebrew-bridge');
const BINARY_NAME = process.platform === 'win32' ? 'bridge.exe' : 'bridge';
const BINARY_PATH = path.join(BRIDGE_DIR, 'bin', BINARY_NAME);

/**
 * Helper for starting/stopping the Go bridge relay server in tests.
 *
 * Usage:
 * ```typescript
 * beforeAll(() => { BridgeHelper.build(); }, 60000);
 * beforeEach(() => { bridge = new BridgeHelper(); });
 * afterEach(async () => { await bridge.stop(); });
 *
 * it('test', async () => {
 *   await bridge.start();
 *   // bridge.port, bridge.url, bridge.authToken
 * });
 * ```
 *
 * Bridge config is passed via environment variables:
 * - BRIDGE_PORT — listening port
 * - BRIDGE_AUTH_TOKEN — auth token for CLI registration
 *
 * Since Bridge validates port >= 1 (port 0 is rejected), we find a free
 * ephemeral port first, release it, then pass it to the bridge process.
 * We confirm startup by polling /health until it responds.
 */
export class BridgeHelper {
  private process: ChildProcess | null = null;
  private _port: number = 0;
  private _authToken: string;

  constructor() {
    this._authToken = uuidv4();
  }

  get port(): number {
    return this._port;
  }

  get authToken(): string {
    return this._authToken;
  }

  get url(): string {
    return `ws://localhost:${this._port}`;
  }

  get httpUrl(): string {
    return `http://localhost:${this._port}`;
  }

  /**
   * Build bridge binary once (call from beforeAll).
   * Compiles bytebrew-bridge/cmd/bridge to bytebrew-bridge/bin/bridge[.exe]
   */
  static build(): void {
    const output = process.platform === 'win32' ? 'bin/bridge.exe' : 'bin/bridge';

    const binDir = path.join(BRIDGE_DIR, 'bin');
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }

    execSync(`go build -o ${output} ./cmd/bridge`, {
      cwd: BRIDGE_DIR,
      stdio: 'pipe',
    });
  }

  /**
   * Start bridge server, wait until /health responds 200.
   *
   * Strategy:
   * 1. Find free ephemeral port (bind, get port, close)
   * 2. Spawn bridge with BRIDGE_PORT=<port>
   * 3. Poll /health until 200 or timeout
   */
  async start(timeoutMs = 15000): Promise<void> {
    if (!fs.existsSync(BINARY_PATH)) {
      BridgeHelper.build();
    }

    // Find a free port
    this._port = await findFreePort();

    this.process = spawn(BINARY_PATH, [], {
      env: {
        ...process.env,
        BRIDGE_PORT: String(this._port),
        BRIDGE_AUTH_TOKEN: this._authToken,
      },
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    // Collect stderr for error reporting
    let stderr = '';
    this.process.stderr!.on('data', (data: Buffer) => {
      stderr += data.toString();
    });

    // Wait for process to not exit immediately
    const earlyExit = new Promise<never>((_, reject) => {
      this.process!.on('error', (err: Error) => {
        reject(err);
      });
      this.process!.on('exit', (code: number | null) => {
        if (code !== null && code !== 0) {
          reject(new Error(`Bridge exited with code ${code}: ${stderr}`));
        }
      });
    });

    // Poll /health until it responds
    const healthReady = this.waitForHealth(timeoutMs);

    await Promise.race([healthReady, earlyExit]);
  }

  /**
   * Stop bridge process.
   */
  async stop(): Promise<void> {
    if (!this.process) return;

    const proc = this.process;
    this.process = null;

    proc.kill();

    await new Promise<void>((resolve) => {
      const killTimeout = setTimeout(() => {
        try {
          proc.kill('SIGKILL');
        } catch {
          // Ignore errors (process might already be dead)
        }
        resolve();
      }, 3000);

      proc.on('exit', () => {
        clearTimeout(killTimeout);
        resolve();
      });
    });
  }

  // --- Private ---

  private async waitForHealth(timeoutMs: number): Promise<void> {
    const deadline = Date.now() + timeoutMs;
    const healthUrl = `${this.httpUrl}/health`;

    while (Date.now() < deadline) {
      try {
        const response = await fetch(healthUrl);
        if (response.ok) {
          return;
        }
      } catch {
        // Connection refused — server not ready yet
      }

      await new Promise((r) => setTimeout(r, 100));
    }

    throw new Error(`Bridge health check timeout (${timeoutMs}ms) on port ${this._port}`);
  }
}

/**
 * Find a free ephemeral port by binding to port 0, reading the assigned port,
 * then closing the listener. There is a small race window, but for tests
 * this is acceptable.
 */
function findFreePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.listen(0, '127.0.0.1', () => {
      const addr = server.address();
      if (addr && typeof addr === 'object') {
        const port = addr.port;
        server.close(() => resolve(port));
      } else {
        server.close(() => reject(new Error('Failed to get port from server address')));
      }
    });
    server.on('error', reject);
  });
}
