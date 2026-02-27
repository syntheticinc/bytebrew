import { OutputBuffer } from './OutputBuffer.js';
import { Shell } from './shell.js';
import { getLogger } from '../../lib/logger.js';
import type { Subprocess } from 'bun';

export interface BackgroundProcess {
  id: string; // "bg-1", "bg-2", ...
  command: string; // original command
  pid: number; // OS process ID
  startTime: Date;
  status: 'running' | 'exited';
  exitCode?: number;
}

const KILL_GRACE_PERIOD = 5000; // 5s before SIGKILL
const AUTO_CLEANUP_MS = 30 * 60 * 1000; // 30 minutes

export class BackgroundProcessManager {
  private processes: Map<
    string,
    {
      info: BackgroundProcess;
      process: Subprocess;
      outputBuffer: OutputBuffer;
      cleanupTimer?: ReturnType<typeof setTimeout>;
    }
  > = new Map();
  private nextId = 1;

  /**
   * Spawn a new background process.
   * Returns the BackgroundProcess info immediately.
   */
  spawn(command: string, cwd: string, env?: Record<string, string>): BackgroundProcess {
    const logger = getLogger();
    const id = `bg-${this.nextId++}`;

    const shellEnv: Record<string, string> = {
      ...(process.env as Record<string, string>),
      LANG: 'en_US.UTF-8',
      LC_ALL: 'en_US.UTF-8',
      PYTHONIOENCODING: 'utf-8',
      ...(env || {}),
    };

    const shell = Shell.acceptable();

    const proc = Bun.spawn([shell, '-c', command], {
      cwd,
      env: shellEnv,
      stdin: 'ignore',
      stdout: 'pipe',
      stderr: 'pipe',
    });

    const outputBuffer = new OutputBuffer();

    const info: BackgroundProcess = {
      id,
      command,
      pid: proc.pid,
      startTime: new Date(),
      status: 'running',
    };

    const entry = { info, process: proc, outputBuffer };
    this.processes.set(id, entry);

    // Pipe stdout and stderr to buffer
    this.pipeToBuffer(proc.stdout, outputBuffer);
    this.pipeToBuffer(proc.stderr, outputBuffer);

    // Handle exit
    proc.exited.then((exitCode: number) => {
      const e = this.processes.get(id);
      if (e) {
        e.info.status = 'exited';
        e.info.exitCode = exitCode;

        // Auto-cleanup after 30 minutes
        e.cleanupTimer = setTimeout(() => {
          this.processes.delete(id);
        }, AUTO_CLEANUP_MS);
      }
    });

    logger.info('Background process spawned', { id, command, pid: proc.pid });
    return info;
  }

  /**
   * List all tracked background processes.
   */
  list(): BackgroundProcess[] {
    return Array.from(this.processes.values()).map((e) => ({ ...e.info }));
  }

  /**
   * Read output from a background process.
   */
  readOutput(id: string): string | null {
    const entry = this.processes.get(id);
    if (!entry) return null;
    return entry.outputBuffer.getOutput();
  }

  /**
   * Kill a background process.
   * Sends SIGTERM first, then SIGKILL after grace period.
   */
  async kill(id: string): Promise<boolean> {
    const entry = this.processes.get(id);
    if (!entry) return false;

    const logger = getLogger();
    logger.info('Killing background process', { id, pid: entry.info.pid });

    try {
      entry.process.kill();
    } catch {
      // Already dead
    }

    // Wait for graceful exit or force kill
    const exitPromise = entry.process.exited;
    const timeout = new Promise<void>((resolve) => setTimeout(resolve, KILL_GRACE_PERIOD));

    await Promise.race([exitPromise, timeout]);

    // If still running after grace period, force kill
    if (entry.info.status === 'running') {
      try {
        entry.process.kill(9); // SIGKILL
      } catch {
        // Already dead
      }
    }

    return true;
  }

  /**
   * Get a specific background process.
   */
  get(id: string): BackgroundProcess | undefined {
    const entry = this.processes.get(id);
    return entry ? { ...entry.info } : undefined;
  }

  /**
   * Kill all background processes and clean up.
   */
  async disposeAll(): Promise<void> {
    const logger = getLogger();
    logger.debug('Disposing all background processes');

    const killPromises: Promise<boolean>[] = [];
    for (const [id, entry] of this.processes) {
      if (entry.info.status === 'running') {
        killPromises.push(this.kill(id));
      }
      if (entry.cleanupTimer) {
        clearTimeout(entry.cleanupTimer);
      }
    }
    await Promise.all(killPromises);
    this.processes.clear();
  }

  /**
   * Pipe a ReadableStream to an OutputBuffer.
   */
  private async pipeToBuffer(
    stream: ReadableStream<Uint8Array>,
    buffer: OutputBuffer
  ): Promise<void> {
    const reader = stream.getReader();
    const decoder = new TextDecoder();
    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer.append(decoder.decode(value, { stream: true }));
      }
    } catch {
      // Stream closed
    }
  }
}
