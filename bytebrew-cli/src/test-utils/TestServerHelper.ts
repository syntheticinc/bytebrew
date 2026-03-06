import { spawn, ChildProcess, execSync } from 'child_process';
import path from 'path';
import fs from 'fs';

const SERVER_DIR = path.resolve(import.meta.dir, '../../../bytebrew-srv');
const BINARY_NAME = process.platform === 'win32' ? 'testserver.exe' : 'testserver';
const BINARY_PATH = path.join(SERVER_DIR, 'bin', BINARY_NAME);

/**
 * Helper for starting/stopping Go test server with mock LLM.
 *
 * Usage:
 * ```typescript
 * beforeAll(() => { TestServerHelper.build(); }, 60000);
 * beforeEach(() => { server = new TestServerHelper(); });
 * afterEach(async () => { await server.stop(); });
 *
 * it('test', async () => {
 *   await server.start('echo');
 *   const container = createTestContainer(server.port);
 *   // ...
 * });
 * ```
 */
export class TestServerHelper {
  private process: ChildProcess | null = null;
  private _port: number = 0;

  get port(): number {
    return this._port;
  }

  /**
   * Build test server binary once (call from beforeAll).
   * Compiles vector-srv/cmd/testserver to vector-srv/bin/testserver[.exe]
   */
  static build(): void {
    const output = process.platform === 'win32' ? 'bin/testserver.exe' : 'bin/testserver';

    // Create bin directory if it doesn't exist
    const binDir = path.join(SERVER_DIR, 'bin');
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }

    execSync(`go build -o ${output} ./cmd/testserver`, {
      cwd: SERVER_DIR,
      stdio: 'pipe',
    });
  }

  /**
   * Start server with scenario, wait for READY:{port} on stdout.
   *
   * @param scenario - Scenario name: "echo", "server-tool", "reasoning", "error"
   * @param options - Optional settings (license status, etc.)
   * @param timeoutMs - Timeout in milliseconds (default: 30000)
   * @throws If server doesn't emit READY:{port} within timeout
   */
  async start(
    scenario: string,
    optionsOrTimeout?: { license?: 'active' | 'grace' | 'blocked' } | number,
    timeoutMs = 30000,
  ): Promise<void> {
    // Support old signature: start(scenario, timeoutMs)
    let options: { license?: 'active' | 'grace' | 'blocked' } = {};
    if (typeof optionsOrTimeout === 'number') {
      timeoutMs = optionsOrTimeout;
    } else if (optionsOrTimeout) {
      options = optionsOrTimeout;
    }

    // Build if binary doesn't exist
    if (!fs.existsSync(BINARY_PATH)) {
      TestServerHelper.build();
    }

    const args = ['--scenario', scenario, '--port', '0'];
    if (options.license) {
      args.push('--license', options.license);
    }

    this.process = spawn(BINARY_PATH, args, {
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    return new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error(`Test server start timeout (${timeoutMs}ms)`));
      }, timeoutMs);

      let stdout = '';
      this.process!.stdout!.on('data', (data: Buffer) => {
        stdout += data.toString();

        // Parse READY:{port} from stdout
        const match = stdout.match(/READY:(\d+)/);
        if (match) {
          this._port = parseInt(match[1], 10);
          clearTimeout(timeout);
          resolve();
        }
      });

      this.process!.stderr!.on('data', (data: Buffer) => {
        // Log stderr for debugging
        process.stderr.write(`[testserver] ${data.toString()}`);
      });

      this.process!.on('error', (err: Error) => {
        clearTimeout(timeout);
        reject(err);
      });

      this.process!.on('exit', (code: number | null) => {
        if (code !== null && code !== 0) {
          clearTimeout(timeout);
          reject(new Error(`Test server exited with code ${code}`));
        }
      });
    });
  }

  /**
   * Stop server process.
   * Sends SIGTERM on Unix, terminates on Windows.
   * Waits up to 3 seconds, then force-kills.
   */
  async stop(): Promise<void> {
    if (!this.process) return;

    const proc = this.process;
    this.process = null;

    proc.kill(); // SIGTERM on Unix, terminates on Windows

    await new Promise<void>((resolve) => {
      const killTimeout = setTimeout(() => {
        try {
          proc.kill('SIGKILL');
        } catch {
          // Ignore errors (process might be already dead)
        }
        resolve();
      }, 3000);

      proc.on('exit', () => {
        clearTimeout(killTimeout);
        resolve();
      });
    });
  }
}
