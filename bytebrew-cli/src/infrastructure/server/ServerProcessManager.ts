import { spawn, ChildProcess } from 'child_process';

/**
 * Manages a Go server subprocess in managed mode.
 * Spawns the binary with --managed --port 0, waits for READY:{port} on stdout.
 *
 * Pattern from TestServerHelper.ts, adapted for production use.
 */
export class ServerProcessManager {
  private process: ChildProcess | null = null;
  private _port: number = 0;

  get port(): number {
    return this._port;
  }

  /**
   * Start server binary in managed mode.
   * Waits for READY:{port} on stdout before resolving.
   *
   * @param binaryPath - absolute path to vector-srv binary
   * @param timeoutMs - timeout waiting for READY (default 30000)
   * @returns the port the server is listening on
   */
  async start(binaryPath: string, timeoutMs = 30000): Promise<number> {
    this.process = spawn(binaryPath, ['--managed', '--port', '0'], {
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    return new Promise<number>((resolve, reject) => {
      const timeout = setTimeout(() => {
        this.stop();
        reject(new Error(`Server start timeout (${timeoutMs}ms)`));
      }, timeoutMs);

      let stdout = '';
      this.process!.stdout!.on('data', (data: Buffer) => {
        stdout += data.toString();
        const match = stdout.match(/READY:(\d+)/);
        if (match) {
          this._port = parseInt(match[1], 10);
          clearTimeout(timeout);
          resolve(this._port);
        }
      });

      this.process!.stderr!.on('data', (_data: Buffer) => {
        // Server stderr is intentionally not forwarded to CLI output.
        // Logs are written to user data dir by the server in managed mode.
      });

      this.process!.on('error', (err: Error) => {
        clearTimeout(timeout);
        reject(err);
      });

      this.process!.on('exit', (code: number | null) => {
        if (code !== null && code !== 0) {
          clearTimeout(timeout);
          reject(new Error(`Server exited with code ${code}`));
        }
      });
    });
  }

  /**
   * Stop server. SIGTERM -> wait 5s -> SIGKILL.
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
          // Already dead
        }
        resolve();
      }, 5000);

      proc.on('exit', () => {
        clearTimeout(killTimeout);
        resolve();
      });
    });
  }

  isRunning(): boolean {
    return this.process !== null && !this.process.killed;
  }
}
