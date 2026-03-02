import fs from 'fs';
import path from 'path';
import net from 'net';
import { ByteBrewHome } from '../config/ByteBrewHome.js';

export interface PortInfo {
  pid: number;
  port: number;
  host: string;
  startedAt: string;
}

/**
 * Reads the server port file to discover a running server.
 * The server writes this file on startup and removes it on shutdown.
 */
export class PortFileReader {
  private readonly filePath: string;

  constructor() {
    this.filePath = path.join(ByteBrewHome.dataDir(), 'bytebrew', 'server.port');
  }

  /**
   * Read port info from the port file.
   * Returns null if file doesn't exist or process is dead (stale file).
   */
  read(): PortInfo | null {
    if (!fs.existsSync(this.filePath)) return null;

    let info: PortInfo;
    try {
      const content = fs.readFileSync(this.filePath, 'utf-8');
      info = JSON.parse(content);
    } catch {
      return null;
    }

    if (!this.isProcessAlive(info.pid)) return null;

    return info;
  }

  /**
   * Check if a process is alive by sending signal 0.
   */
  private isProcessAlive(pid: number): boolean {
    try {
      process.kill(pid, 0);
      return true;
    } catch {
      return false;
    }
  }
}

/**
 * Check if a server is reachable at the given address.
 * Uses a TCP connect with timeout.
 */
export function isServerReachable(
  host: string,
  port: number,
  timeoutMs: number = 2000,
): Promise<boolean> {
  return new Promise((resolve) => {
    const socket = new net.Socket();
    socket.setTimeout(timeoutMs);

    socket.on('connect', () => {
      socket.destroy();
      resolve(true);
    });

    socket.on('timeout', () => {
      socket.destroy();
      resolve(false);
    });

    socket.on('error', () => {
      socket.destroy();
      resolve(false);
    });

    socket.connect(port, host);
  });
}
