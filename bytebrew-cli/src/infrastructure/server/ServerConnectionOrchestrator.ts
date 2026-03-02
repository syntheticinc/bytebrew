import { ServerBinaryManager } from './ServerBinaryManager.js';
import { ServerProcessManager } from './ServerProcessManager.js';
import { PortFileReader, isServerReachable } from './PortFileReader.js';

export interface ServerConnection {
  /** host:port address to connect gRPC client to */
  address: string;
  /** Stop the managed server (no-op for external servers) */
  cleanup: () => Promise<void>;
}

/**
 * Decides how to connect to the server (chain of discovery):
 * 1. Explicit address (--server flag or BYTEBREW_SERVER env) -> use directly
 * 2. Port file discovery -> check if standalone server is already running
 * 3. Managed mode -> find binary and start server
 * 4. Error -> throw with actionable message
 */
export class ServerConnectionOrchestrator {
  private processManager: ServerProcessManager | null = null;

  /**
   * Establish a server connection.
   *
   * @param externalAddress - explicit server address (from --server or BYTEBREW_SERVER)
   * @returns connection with address and cleanup function
   */
  async connect(externalAddress?: string): Promise<ServerConnection> {
    // 1. Explicit address — user specified via --server or BYTEBREW_SERVER
    if (externalAddress) {
      return { address: externalAddress, cleanup: async () => {} };
    }

    // 2. Port file discovery — check if a standalone server is already running
    const portFileConnection = await this.tryPortFile();
    if (portFileConnection) return portFileConnection;

    // 3. Managed mode — find binary and start server
    const binaryManager = new ServerBinaryManager();
    const binaryPath = binaryManager.findBinary();
    if (!binaryPath) {
      throw new Error(
        'ByteBrew server not found. Either:\n' +
          '  - Start the server manually (go run ./cmd/server)\n' +
          '  - Use --server flag to specify address\n' +
          '  - Install bytebrew-srv binary',
      );
    }

    this.processManager = new ServerProcessManager();
    const port = await this.processManager.start(binaryPath);

    return {
      address: `localhost:${port}`,
      cleanup: async () => {
        await this.processManager?.stop();
      },
    };
  }

  private async tryPortFile(): Promise<ServerConnection | null> {
    const reader = new PortFileReader();
    const info = reader.read();
    if (!info) return null;

    // 0.0.0.0 means "all interfaces" on the server side, but clients must connect to localhost
    const host = (!info.host || info.host === '0.0.0.0') ? '127.0.0.1' : info.host;
    const reachable = await isServerReachable(host, info.port);
    if (!reachable) return null;

    return {
      address: `${host}:${info.port}`,
      cleanup: async () => {},
    };
  }
}
