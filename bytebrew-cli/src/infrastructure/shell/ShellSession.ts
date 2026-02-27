import { OutputBuffer } from './OutputBuffer.js';
import { Shell } from './shell.js';
import { getLogger } from '../../lib/logger.js';
import { debugLog } from '../../lib/debugLog.js';
import type { Subprocess } from 'bun';

export interface ShellResult {
  stdout: string; // full or partial output
  exitCode: number | null; // null if not completed
  completed: boolean; // true if marker was found
}

export interface ShellSessionConfig {
  cwd: string;
  env?: Record<string, string>;
  shell?: string; // path to bash, auto-detected if not provided
  maxOutputSize?: number; // ring buffer size, default 1MB
}

export class ShellSession {
  private process: Subprocess | null = null;
  private outputBuffer: OutputBuffer;
  private config: ShellSessionConfig;
  private _isExecuting = false;
  private shell: string;

  constructor(config: ShellSessionConfig) {
    this.config = config;
    this.outputBuffer = new OutputBuffer(config.maxOutputSize);
    this.shell = config.shell || Shell.acceptable();
  }

  /**
   * Execute a command in the persistent shell session.
   * Spawns the shell process lazily on first call.
   * Returns output when marker detected or timeout.
   */
  async execute(command: string, timeoutMs: number): Promise<ShellResult> {
    debugLog('SHELL', 'execute start', { command: command?.slice(0, 80), timeoutMs });

    if (this._isExecuting) {
      throw new Error('Shell session is busy — another command is executing');
    }

    this.ensureAlive();
    debugLog('SHELL', 'ensureAlive done', { isAlive: this.isAlive(), pid: this.process?.pid });
    this._isExecuting = true;

    try {
      // Wrap command with marker
      const { markerId, wrappedCommand } = OutputBuffer.wrapCommand(command);

      // Reset buffer for fresh output
      this.outputBuffer.reset();

      // Write command to stdin (stdin is FileSink when spawned with stdin: 'pipe')
      const stdin = this.process!.stdin as import('bun').FileSink;
      stdin.write(wrappedCommand + '\n');
      stdin.flush();
      debugLog('SHELL', 'stdin written', { markerId, cmdLen: wrappedCommand.length });

      // Wait for marker or timeout
      try {
        const result = await this.outputBuffer.waitForMarker(markerId, timeoutMs);
        debugLog('SHELL', 'marker found', { markerId, exitCode: result.exitCode, outputLen: result.output?.length });
        return {
          stdout: result.output,
          exitCode: result.exitCode,
          completed: true,
        };
      } catch (err: any) {
        if (err.message.includes('timeout') || err.message.includes('Marker timeout')) {
          // Timeout — return partial output
          debugLog('SHELL', 'TIMEOUT', { markerId, timeoutMs, bufferLen: this.outputBuffer.getOutput()?.length });
          return {
            stdout: this.outputBuffer.getOutput(),
            exitCode: null,
            completed: false,
          };
        }
        debugLog('SHELL', 'ERROR', { markerId, error: err.message });
        throw err;
      }
    } finally {
      this._isExecuting = false;
    }
  }

  /**
   * Send interrupt signal (Ctrl+C).
   * On Unix: sends '\x03' to stdin (which bash translates to SIGINT).
   * Same approach works on Windows Git Bash.
   */
  async interrupt(): Promise<void> {
    if (!this.process) return;
    try {
      const stdin = this.process.stdin as import('bun').FileSink;
      stdin.write('\x03');
      stdin.flush();
    } catch {
      // Process may have died
    }
  }

  /**
   * Check if a command is currently executing.
   */
  isExecuting(): boolean {
    return this._isExecuting;
  }

  /**
   * Check if the shell process is alive.
   */
  isAlive(): boolean {
    return this.process !== null && !this.process.killed && this.process.exitCode === null;
  }

  /**
   * Kill the shell process.
   */
  destroy(): void {
    this.outputBuffer.cancelPending();
    if (this.process) {
      try {
        this.process.kill();
      } catch {
        // Already dead
      }
      this.process = null;
    }
  }

  /**
   * Ensure the shell process is alive, spawn if needed.
   */
  private ensureAlive(): void {
    if (this.isAlive()) return;

    const logger = getLogger();
    logger.debug('Spawning shell session', {
      shell: this.shell,
      cwd: this.config.cwd,
    });

    // Reset state
    this.process = null;
    this.outputBuffer.reset();

    const env: Record<string, string> = {
      ...(process.env as Record<string, string>),
      TERM: 'dumb',
      PS1: '',
      PS2: '',
      PROMPT_COMMAND: '',
      LANG: 'en_US.UTF-8',
      LC_ALL: 'en_US.UTF-8',
      PYTHONIOENCODING: 'utf-8',
      ...(this.config.env || {}),
    };

    this.process = Bun.spawn([this.shell, '--norc', '--noprofile'], {
      cwd: this.config.cwd,
      env,
      stdin: 'pipe',
      stdout: 'pipe',
      stderr: 'pipe',
    });

    // Pipe stdout to OutputBuffer (stdout/stderr are ReadableStream when spawned with 'pipe')
    this.pipeToBuffer(this.process.stdout as ReadableStream<Uint8Array>);
    // Also pipe stderr (though we merge in command with 2>&1, this catches shell-level errors)
    this.pipeToBuffer(this.process.stderr as ReadableStream<Uint8Array>);

    // Handle process exit
    this.process.exited.then(() => {
      logger.debug('Shell session process exited', { exitCode: this.process?.exitCode });
      // If a command was waiting, the next check in ensureAlive will respawn
    });
  }

  /**
   * Pipe a ReadableStream to the OutputBuffer.
   */
  private async pipeToBuffer(stream: ReadableStream<Uint8Array>): Promise<void> {
    const reader = stream.getReader();
    const decoder = new TextDecoder();
    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        this.outputBuffer.append(decoder.decode(value, { stream: true }));
      }
    } catch {
      // Stream closed, that's ok
    }
  }

}
