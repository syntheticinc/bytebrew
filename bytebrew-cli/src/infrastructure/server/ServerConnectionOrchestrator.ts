import { ServerBinaryManager } from './ServerBinaryManager.js';
import { ServerProcessManager } from './ServerProcessManager.js';

export interface ServerConnection {
  /** host:port address to connect gRPC client to */
  address: string;
  /** Stop the managed server (no-op for external servers) */
  cleanup: () => Promise<void>;
}

/**
 * Decides how to connect to the server:
 * - If externalAddress is provided (--server flag or BYTEBREW_SERVER env) -> use it directly
 * - Otherwise -> find server binary and start it in managed mode
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
    // External server mode — user specified an address
    if (externalAddress) {
      return {
        address: externalAddress,
        cleanup: async () => {},
      };
    }

    // Managed mode — find binary and start server
    const binaryManager = new ServerBinaryManager();
    const binaryPath = binaryManager.findBinary();
    if (!binaryPath) {
      // Fallback: try connecting to default address (server may already be running)
      return {
        address: 'localhost:60401',
        cleanup: async () => {},
      };
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
}
