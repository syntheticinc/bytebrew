import path from "path";
import { pathToFileURL, fileURLToPath } from "url";
import {
  createMessageConnection,
  StreamMessageReader,
  StreamMessageWriter,
} from "vscode-jsonrpc/node";
import type { Diagnostic } from "vscode-languageserver-types";
import { execSync } from "child_process";
import { LANGUAGE_EXTENSIONS } from "./languageExtensions.js";
import type { LspServerHandle } from "./LspServerConfigs.js";
import { getLogger } from "../../lib/logger.js";

const DIAGNOSTICS_DEBOUNCE_MS = 150;
const INITIALIZE_TIMEOUT_MS = 45_000;
const DIAGNOSTICS_TIMEOUT_MS = 10_000;

export interface LspClientInfo {
  readonly serverID: string;
  readonly root: string;
  readonly diagnostics: Map<string, Diagnostic[]>;
  readonly connection: ReturnType<typeof createMessageConnection>;
  notify: {
    open(input: { path: string }): Promise<void>;
  };
  waitForDiagnostics(input: { path: string }): Promise<void>;
  waitForReady(timeoutMs?: number): Promise<boolean>;
  shutdown(): Promise<void>;
}

export async function createLspClient(input: {
  serverID: string;
  server: LspServerHandle;
  root: string;
  projectRoot: string;
  onDiagnostics?: (filePath: string) => void;
}): Promise<LspClientInfo> {
  const logger = getLogger();
  logger.info(`[LSP] starting client for ${input.serverID}`);

  const connection = createMessageConnection(
    new StreamMessageReader(input.server.process.stdout as any),
    new StreamMessageWriter(input.server.process.stdin as any),
  );

  const diagnosticsMap = new Map<string, Diagnostic[]>();
  let diagnosticsVersion = 0;

  // Track diagnostics events for waitForDiagnostics
  type DiagnosticsListener = (filePath: string) => void;
  const listeners = new Set<DiagnosticsListener>();

  // Track LSP server readiness via $/progress notifications
  let ready = false;
  let readyResolve: (() => void) | null = null;
  const readyPromise = new Promise<void>((resolve) => {
    readyResolve = resolve;
  });

  connection.onNotification(
    "textDocument/publishDiagnostics",
    (params: any) => {
      const filePath = path.normalize(fileURLToPath(params.uri));
      const exists = diagnosticsMap.has(filePath);
      diagnosticsMap.set(filePath, params.diagnostics);

      // Many LSP servers send empty diagnostics right after didOpen before real analysis.
      // Skip first empty diagnostics so waitForDiagnostics doesn't resolve prematurely.
      if (!exists && params.diagnostics.length === 0) return;

      diagnosticsVersion++;
      input.onDiagnostics?.(filePath);
      for (const listener of listeners) {
        listener(filePath);
      }
    },
  );

  connection.onRequest("window/workDoneProgress/create", () => null);

  // Track gopls workspace indexing via $/progress notifications.
  // When gopls finishes loading the workspace it sends kind: "end".
  // For LSP servers that don't send $/progress, readyPromise is resolved
  // by the timeout in waitForReady() as a fallback.
  connection.onNotification("$/progress", (params: any) => {
    if (params.value?.kind === "end" && !ready) {
      logger.info(`[LSP] ${input.serverID} workspace ready ($/progress end)`);
      ready = true;
      readyResolve?.();
    }
  });

  connection.onRequest("workspace/configuration", async () => [
    input.server.initialization ?? {},
  ]);
  connection.onRequest("client/registerCapability", async () => {});
  connection.onRequest("client/unregisterCapability", async () => {});
  connection.onRequest("workspace/workspaceFolders", async () => [
    {
      name: "workspace",
      uri: pathToFileURL(input.root).href,
    },
  ]);
  connection.listen();

  // Initialize LSP server
  await Promise.race([
    connection.sendRequest("initialize", {
      rootUri: pathToFileURL(input.root).href,
      processId: input.server.process.pid,
      workspaceFolders: [
        {
          name: "workspace",
          uri: pathToFileURL(input.root).href,
        },
      ],
      initializationOptions: {
        ...input.server.initialization,
      },
      capabilities: {
        window: {
          workDoneProgress: true,
        },
        workspace: {
          configuration: true,
          didChangeWatchedFiles: {
            dynamicRegistration: true,
          },
        },
        textDocument: {
          synchronization: {
            didOpen: true,
            didChange: true,
          },
          publishDiagnostics: {
            versionSupport: true,
          },
        },
      },
    }),
    sleep(INITIALIZE_TIMEOUT_MS).then(() => {
      throw new Error(`LSP initialize timeout for ${input.serverID}`);
    }),
  ]);

  await connection.sendNotification("initialized", {});

  if (input.server.initialization) {
    await connection.sendNotification("workspace/didChangeConfiguration", {
      settings: input.server.initialization,
    });
  }

  const files: Record<string, number> = {};

  const client: LspClientInfo = {
    root: input.root,
    get serverID() {
      return input.serverID;
    },
    get connection() {
      return connection;
    },
    get diagnostics() {
      return diagnosticsMap;
    },
    notify: {
      async open(openInput: { path: string }) {
        const filePath = path.isAbsolute(openInput.path)
          ? openInput.path
          : path.resolve(input.projectRoot, openInput.path);

        const text = await Bun.file(filePath).text();
        const extension = path.extname(filePath);
        const languageId = LANGUAGE_EXTENSIONS[extension] ?? "plaintext";

        const version = files[filePath];
        if (version !== undefined) {
          // File already opened — send change notification
          await connection.sendNotification(
            "workspace/didChangeWatchedFiles",
            {
              changes: [
                { uri: pathToFileURL(filePath).href, type: 2 }, // Changed
              ],
            },
          );

          const next = version + 1;
          files[filePath] = next;
          await connection.sendNotification("textDocument/didChange", {
            textDocument: {
              uri: pathToFileURL(filePath).href,
              version: next,
            },
            contentChanges: [{ text }],
          });
          return;
        }

        // First time opening — send didOpen
        await connection.sendNotification("workspace/didChangeWatchedFiles", {
          changes: [
            { uri: pathToFileURL(filePath).href, type: 1 }, // Created
          ],
        });

        diagnosticsMap.delete(filePath);
        await connection.sendNotification("textDocument/didOpen", {
          textDocument: {
            uri: pathToFileURL(filePath).href,
            languageId,
            version: 0,
            text,
          },
        });
        files[filePath] = 0;
      },
    },

    async waitForReady(timeoutMs = 15_000): Promise<boolean> {
      if (ready) return true;
      return Promise.race([
        readyPromise.then(() => true as const),
        sleep(timeoutMs).then(() => false as const),
      ]);
    },

    async waitForDiagnostics(waitInput: { path: string }) {
      const normalizedPath = path.normalize(
        path.isAbsolute(waitInput.path)
          ? waitInput.path
          : path.resolve(input.projectRoot, waitInput.path),
      );
      let unsubscribe: (() => void) | undefined;
      let debounceTimer: ReturnType<typeof setTimeout> | undefined;

      await Promise.race([
        new Promise<void>((resolve) => {
          const handler: DiagnosticsListener = (filePath) => {
            if (filePath !== normalizedPath) return;
            // Debounce: allow LSP to send follow-up diagnostics (e.g. semantic after syntax)
            if (debounceTimer) clearTimeout(debounceTimer);
            debounceTimer = setTimeout(() => {
              unsubscribe?.();
              resolve();
            }, DIAGNOSTICS_DEBOUNCE_MS);
          };
          listeners.add(handler);
          unsubscribe = () => listeners.delete(handler);
        }),
        sleep(DIAGNOSTICS_TIMEOUT_MS),
      ]).finally(() => {
        if (debounceTimer) clearTimeout(debounceTimer);
        unsubscribe?.();
      });
    },

    async shutdown() {
      logger.info(`[LSP] shutting down ${input.serverID}`);
      connection.end();
      connection.dispose();

      if (process.platform === "win32") {
        // Kill process tree on Windows
        try {
          execSync(`taskkill /T /F /PID ${input.server.process.pid}`, {
            stdio: "ignore",
          });
        } catch {
          /* ignore — process may already be dead */
        }
      } else {
        input.server.process.kill();
      }
    },
  };

  logger.info(`[LSP] initialized ${input.serverID}`);
  return client;
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
