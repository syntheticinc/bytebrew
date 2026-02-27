import path from "path";
import fs from "fs/promises";
import type { Diagnostic } from "vscode-languageserver-types";
import { createLspClient, type LspClientInfo } from "./LspClient.js";
import {
  ALL_SERVERS,
  setManagedBinDir,
  type LspServerConfig,
} from "./LspServerConfigs.js";
import { LspInstaller } from "./install/LspInstaller.js";
import { getLogger } from "../../lib/logger.js";

export interface LspHealthStatus {
  applicableServers: string[];
  activeServers: string[];
  brokenServers: string[];
  warmupComplete: boolean;
}

export class LspManager {
  private clients: LspClientInfo[] = [];
  private broken = new Set<string>();
  private spawning = new Map<string, Promise<LspClientInfo | undefined>>();
  private configs: LspServerConfig[];
  private projectRoot: string;
  private installer: LspInstaller;

  constructor(projectRoot: string, configs?: LspServerConfig[]) {
    this.projectRoot = projectRoot;
    this.configs = configs ?? ALL_SERVERS;
    this.installer = new LspInstaller();
    // Make whichBin() aware of the managed bin directory
    setManagedBinDir(this.installer.getBinDir().getPath());
  }

  async touchFile(filePath: string, waitForDiagnostics: boolean): Promise<void> {
    const logger = getLogger();
    logger.info("[LSP] touching file", { file: filePath });

    const clients = await this.getClients(filePath);
    await Promise.all(
      clients.map(async (client) => {
        const wait = waitForDiagnostics
          ? client.waitForDiagnostics({ path: filePath })
          : Promise.resolve();
        await client.notify.open({ path: filePath });
        return wait;
      }),
    ).catch((err) => {
      logger.error("[LSP] failed to touch file", { error: err, file: filePath });
    });
  }

  async diagnostics(): Promise<Record<string, Diagnostic[]>> {
    const results: Record<string, Diagnostic[]> = {};
    for (const client of this.clients) {
      for (const [filePath, diags] of client.diagnostics.entries()) {
        const arr = results[filePath] || [];
        arr.push(...diags);
        results[filePath] = arr;
      }
    }
    return results;
  }

  async getClients(file: string): Promise<LspClientInfo[]> {
    const logger = getLogger();
    const extension = path.parse(file).ext || file;
    const result: LspClientInfo[] = [];

    for (const server of this.configs) {
      if (
        server.extensions.length &&
        !server.extensions.includes(extension)
      ) {
        continue;
      }

      const root = await server.root(file, this.projectRoot);
      if (!root) continue;

      const key = root + server.id;
      if (this.broken.has(key)) continue;

      // Check if client already exists
      const existing = this.clients.find(
        (c) => c.root === root && c.serverID === server.id,
      );
      if (existing) {
        result.push(existing);
        continue;
      }

      // Check if spawning in progress
      const inflight = this.spawning.get(key);
      if (inflight) {
        const client = await inflight;
        if (client) result.push(client);
        continue;
      }

      // Spawn new server
      const task = this.spawnClient(server, root, key);
      this.spawning.set(key, task);
      task.finally(() => {
        if (this.spawning.get(key) === task) {
          this.spawning.delete(key);
        }
      });

      const client = await task;
      if (client) result.push(client);
    }

    return result;
  }

  private async spawnClient(
    server: LspServerConfig,
    root: string,
    key: string,
  ): Promise<LspClientInfo | undefined> {
    const logger = getLogger();

    let handle = await server
      .spawn(root)
      .catch((err) => {
        logger.error(`[LSP] Failed to spawn ${server.id}`, { error: err });
        return undefined;
      });

    // Auto-install: if binary not found and install spec exists, try installing
    if (!handle && server.install && !this.installer.isDisabled()) {
      logger.info(`[LSP] ${server.id} not found, attempting auto-install...`);
      const result = await this.installer.install(server.id, server.install);
      if (result.success) {
        // Retry spawn after install
        handle = await server.spawn(root).catch((err) => {
          logger.error(`[LSP] Failed to spawn ${server.id} after install`, { error: err });
          return undefined;
        });
      }
    }

    if (!handle) {
      this.broken.add(key);
      return undefined;
    }
    logger.info(`[LSP] spawned server ${server.id}`);

    const client = await createLspClient({
      serverID: server.id,
      server: handle,
      root,
      projectRoot: this.projectRoot,
    }).catch((err) => {
      this.broken.add(key);
      handle.process.kill();
      logger.error(`[LSP] Failed to initialize ${server.id}`, { error: err });
      return undefined;
    });

    if (!client) {
      handle.process.kill();
      return undefined;
    }

    // Prevent duplicate if another getClients call resolved first
    const duplicate = this.clients.find(
      (c) => c.root === root && c.serverID === server.id,
    );
    if (duplicate) {
      handle.process.kill();
      return duplicate;
    }

    this.clients.push(client);
    return client;
  }

  /**
   * Wait for all active LSP clients to finish workspace indexing.
   * If any spawning is still in progress, waits for it first (up to timeoutMs).
   * Falls back gracefully if gopls doesn't send $/progress (older versions).
   */
  async waitForReady(timeoutMs = 15_000): Promise<void> {
    const deadline = Date.now() + timeoutMs;

    // Wait for any in-flight spawning tasks
    const spawningPromises = [...this.spawning.values()];
    if (spawningPromises.length > 0) {
      const remaining = deadline - Date.now();
      await Promise.race([
        Promise.allSettled(spawningPromises),
        new Promise<void>((r) => setTimeout(r, Math.max(0, remaining))),
      ]);
    }

    // Wait for all clients to report ready
    const remaining = deadline - Date.now();
    if (this.clients.length > 0 && remaining > 0) {
      await Promise.allSettled(
        this.clients.map((c) => c.waitForReady(Math.max(0, remaining))),
      );
    }
  }

  /**
   * Pre-spawn LSP servers applicable to the project.
   * Scans projectRoot for marker files (go.mod, package.json, etc.)
   * and starts only matching servers in background.
   * Best-effort: failures are logged but don't propagate.
   */
  async warmup(): Promise<void> {
    const logger = getLogger();
    logger.info("[LSP] warmup: scanning project for applicable servers");

    const serverIds = await this.detectApplicableServers();
    if (serverIds.size === 0) {
      logger.info("[LSP] warmup: no applicable servers found");
      return;
    }
    logger.info("[LSP] warmup: detected servers", { servers: [...serverIds] });

    const seen = new Set<string>();
    const tasks: Promise<void>[] = [];

    for (const server of this.configs) {
      if (!serverIds.has(server.id)) continue;
      if (server.extensions.length === 0) continue;

      const ext = server.extensions[0];
      const dummyFile = path.join(this.projectRoot, `_${ext}`);

      tasks.push(
        (async () => {
          const root = await server.root(dummyFile, this.projectRoot).catch(() => undefined);
          if (!root) return;

          const key = root + server.id;
          if (this.broken.has(key) || seen.has(key)) return;
          seen.add(key);

          if (this.clients.find((c) => c.root === root && c.serverID === server.id)) return;
          if (this.spawning.has(key)) return;

          const task = this.spawnClient(server, root, key);
          this.spawning.set(key, task);
          task.finally(() => {
            if (this.spawning.get(key) === task) this.spawning.delete(key);
          });

          const client = await task;
          if (client) {
            logger.info(`[LSP] warmup: started ${server.id}`, { root });
          }
        })(),
      );
    }

    await Promise.allSettled(tasks);
    logger.info("[LSP] warmup complete", { servers: this.clients.length });
  }

  /**
   * Detect which LSP servers are applicable based on root marker files.
   */
  private async detectApplicableServers(): Promise<Set<string>> {
    let rootFiles: string[];
    try {
      rootFiles = await fs.readdir(this.projectRoot);
    } catch {
      return new Set();
    }

    const has = new Set(rootFiles);
    const ids = new Set<string>();

    // Deno takes priority over TypeScript (checked first)
    const isDeno = has.has("deno.json") || has.has("deno.jsonc");
    if (isDeno) {
      ids.add("deno");
    } else if (has.has("package.json") || has.has("bun.lockb") || has.has("bun.lock") || has.has("package-lock.json") || has.has("pnpm-lock.yaml") || has.has("yarn.lock")) {
      ids.add("typescript");
    }

    if (has.has("go.mod") || has.has("go.sum") || has.has("go.work")) ids.add("gopls");
    if (has.has("Cargo.toml") || has.has("Cargo.lock")) ids.add("rust");
    if (has.has("pyproject.toml") || has.has("setup.py") || has.has("setup.cfg") || has.has("requirements.txt") || has.has("Pipfile") || has.has("pyrightconfig.json")) ids.add("pyright");
    if (has.has("CMakeLists.txt") || has.has("compile_commands.json") || has.has("compile_flags.txt") || has.has(".clangd") || has.has("Makefile")) ids.add("clangd");

    return ids;
  }

  /**
   * Check if any active LSP clients exist (without triggering on-demand spawn).
   * Used by LspTool to skip retry delays when no servers are running.
   */
  hasActiveClients(): boolean {
    return this.clients.length > 0;
  }

  getHealthStatus(): LspHealthStatus {
    return {
      applicableServers: this.configs.map(c => c.id),
      activeServers: this.clients.map(c => c.serverID),
      brokenServers: [...this.broken].map(key => {
        const config = this.configs.find(c => key.endsWith(c.id));
        return config?.id ?? key;
      }),
      warmupComplete: this.spawning.size === 0,
    };
  }

  async dispose(): Promise<void> {
    const logger = getLogger();
    logger.info("[LSP] disposing all clients");

    // Wait for in-flight spawning tasks (with 5s timeout) so they don't
    // finish after dispose and leak orphan LSP processes.
    const inflight = [...this.spawning.values()];
    if (inflight.length > 0) {
      await Promise.race([
        Promise.allSettled(inflight),
        new Promise<void>((r) => setTimeout(r, 5_000)),
      ]);
    }

    // Shutdown all clients (including any that just finished spawning)
    await Promise.allSettled(this.clients.map((c) => c.shutdown()));
    this.clients = [];
    this.broken.clear();
    this.spawning.clear();
  }
}
