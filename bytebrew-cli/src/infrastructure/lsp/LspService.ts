import { pathToFileURL } from "url";
import path from "path";
import type { LspManager } from "./LspManager.js";
import { getLogger } from "../../lib/logger.js";

export interface LspLocation {
  uri: string;
  range: {
    start: { line: number; character: number };
    end: { line: number; character: number };
  };
}

type LspRawResult = LspLocation | LspLocation[] | null | undefined;

function normalizeToLocations(raw: LspRawResult): LspLocation[] {
  if (!raw) return [];
  if (Array.isArray(raw)) return raw.filter(Boolean) as LspLocation[];
  return [raw];
}

export class LspService {
  constructor(private manager: LspManager) {}

  async definition(filePath: string, line: number, character: number): Promise<LspLocation[]> {
    const logger = getLogger();
    const clients = await this.manager.getClients(filePath);
    if (clients.length === 0) {
      logger.warn("[LSP] no clients available for definition", { file: filePath });
      return [];
    }

    const uri = pathToFileURL(filePath).href;
    const results = await Promise.all(
      clients.map(async (client) => {
        await client.notify.open({ path: filePath });
        const raw = await client.connection
          .sendRequest("textDocument/definition", {
            textDocument: { uri },
            position: { line, character },
          })
          .catch((err) => {
            logger.error("[LSP] definition request failed", { error: err, file: filePath });
            return null;
          });
        return normalizeToLocations(raw as LspRawResult);
      }),
    );

    return results.flat();
  }

  async references(filePath: string, line: number, character: number): Promise<LspLocation[]> {
    const logger = getLogger();
    const clients = await this.manager.getClients(filePath);
    if (clients.length === 0) {
      logger.warn("[LSP] no clients available for references", { file: filePath });
      return [];
    }

    const uri = pathToFileURL(filePath).href;
    const results = await Promise.all(
      clients.map(async (client) => {
        await client.notify.open({ path: filePath });
        const raw = await client.connection
          .sendRequest("textDocument/references", {
            textDocument: { uri },
            position: { line, character },
            context: { includeDeclaration: true },
          })
          .catch((err) => {
            logger.error("[LSP] references request failed", { error: err, file: filePath });
            return null;
          });
        return normalizeToLocations(raw as LspRawResult);
      }),
    );

    return results.flat();
  }

  async waitForReady(timeoutMs = 15_000): Promise<void> {
    await this.manager.waitForReady(timeoutMs);
  }

  /**
   * Check if any LSP servers are actively running (without spawning new ones).
   */
  hasActiveClients(): boolean {
    return this.manager.hasActiveClients();
  }

  getHealthInfo(_filePath: string): string | null {
    const health = this.manager.getHealthStatus();

    if (health.brokenServers.length > 0) {
      return `LSP server failed to start: ${health.brokenServers.join(', ')}. Check that the server binary is installed.`;
    }

    if (!health.warmupComplete) {
      return 'LSP servers still warming up, try again in a few seconds.';
    }

    if (health.activeServers.length === 0) {
      return 'No LSP servers are running. Language servers may not be configured for this project.';
    }

    return null;
  }

  async implementation(filePath: string, line: number, character: number): Promise<LspLocation[]> {
    const logger = getLogger();
    const clients = await this.manager.getClients(filePath);
    if (clients.length === 0) {
      logger.warn("[LSP] no clients available for implementation", { file: filePath });
      return [];
    }

    const uri = pathToFileURL(filePath).href;
    const results = await Promise.all(
      clients.map(async (client) => {
        await client.notify.open({ path: filePath });
        const raw = await client.connection
          .sendRequest("textDocument/implementation", {
            textDocument: { uri },
            position: { line, character },
          })
          .catch((err) => {
            logger.error("[LSP] implementation request failed", { error: err, file: filePath });
            return null;
          });
        return normalizeToLocations(raw as LspRawResult);
      }),
    );

    return results.flat();
  }
}
