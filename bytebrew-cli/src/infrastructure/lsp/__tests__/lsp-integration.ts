/**
 * LSP protocol debugging — log all notifications from gopls and test references
 * Usage: bun src/infrastructure/lsp/__tests__/lsp-integration.ts
 */
import path from "path";
import { pathToFileURL } from "url";
import {
  createMessageConnection,
  StreamMessageReader,
  StreamMessageWriter,
} from "vscode-jsonrpc/node";
import { spawn } from "child_process";

const TEST_PROJECT = path.resolve(import.meta.dir, "../../../../..", "test-project");

async function main() {
  console.log(`Project: ${TEST_PROJECT}`);

  // Spawn gopls directly
  const proc = spawn("gopls", [], { cwd: TEST_PROJECT });
  const connection = createMessageConnection(
    new StreamMessageReader(proc.stdout as any),
    new StreamMessageWriter(proc.stdin as any),
  );

  // Log ALL notifications from gopls
  connection.onNotification((method: string, params: any) => {
    if (method === "textDocument/publishDiagnostics") {
      const uri = params.uri || "";
      console.log(`[NOTIFICATION] ${method} — ${uri.replace(/.*test-project/, "")} (${params.diagnostics?.length || 0} diags)`);
    } else if (method === "$/progress") {
      const token = params.token;
      const kind = params.value?.kind;
      const msg = params.value?.message || params.value?.title || "";
      console.log(`[PROGRESS] token=${token} kind=${kind} msg="${msg}"`);
    } else {
      console.log(`[NOTIFICATION] ${method}`, JSON.stringify(params).slice(0, 200));
    }
  });

  // Handle all requests from gopls
  connection.onRequest("window/workDoneProgress/create", (params: any) => {
    console.log(`[REQUEST] window/workDoneProgress/create token=${params.token}`);
    return null;
  });
  connection.onRequest("workspace/configuration", () => [{}]);
  connection.onRequest("client/registerCapability", (params: any) => {
    console.log(`[REQUEST] client/registerCapability`, JSON.stringify(params).slice(0, 300));
    return {};
  });
  connection.onRequest("client/unregisterCapability", () => {});
  connection.onRequest("workspace/workspaceFolders", () => [
    { name: "workspace", uri: pathToFileURL(TEST_PROJECT).href },
  ]);

  connection.listen();

  // Initialize (exactly like opencode)
  console.log("\n=== INITIALIZING ===");
  const initResult = await connection.sendRequest("initialize", {
    rootUri: pathToFileURL(TEST_PROJECT).href,
    processId: proc.pid,
    workspaceFolders: [
      { name: "workspace", uri: pathToFileURL(TEST_PROJECT).href },
    ],
    capabilities: {
      window: { workDoneProgress: true },
      workspace: {
        configuration: true,
        didChangeWatchedFiles: { dynamicRegistration: true },
      },
      textDocument: {
        synchronization: { didOpen: true, didChange: true },
        publishDiagnostics: { versionSupport: true },
      },
    },
  });
  console.log("Initialize done, server capabilities:", Object.keys((initResult as any).capabilities || {}).join(", "));

  await connection.sendNotification("initialized", {});
  console.log("Sent 'initialized' notification");

  // Wait and watch for progress notifications
  console.log("\n=== WAITING 10s FOR GOPLS TO LOAD WORKSPACE ===");
  await new Promise((r) => setTimeout(r, 10000));

  // Open the target file
  const targetFile = path.join(TEST_PROJECT, "internal", "domain", "agent_event.go");
  const targetUri = pathToFileURL(targetFile).href;
  const text = await Bun.file(targetFile).text();

  console.log("\n=== OPENING TARGET FILE ===");
  await connection.sendNotification("textDocument/didOpen", {
    textDocument: {
      uri: targetUri,
      languageId: "go",
      version: 0,
      text,
    },
  });

  // Wait for gopls to process
  console.log("Waiting 3s after didOpen...");
  await new Promise((r) => setTimeout(r, 3000));

  // Test 1: References right after didOpen
  console.log("\n=== TEST 1: References after didOpen ===");
  const ref1: any = await connection
    .sendRequest("textDocument/references", {
      textDocument: { uri: targetUri },
      position: { line: 28, character: 5 },
      context: { includeDeclaration: true },
    })
    .catch((e: any) => { console.log(`ERROR: ${e.message}`); return []; });
  console.log(`Results: ${Array.isArray(ref1) ? ref1.length : 0}`);
  if (Array.isArray(ref1)) {
    const files = new Set(ref1.map((r: any) => r.uri.replace(/.*test-project/, "")));
    console.log(`Unique files: ${files.size}`);
    for (const f of files) console.log(`  ${f}`);
  }

  // Test 2: Try WITHOUT opening the file first (different gopls instance would be needed)
  // Instead: try on a consumer file
  console.log("\n=== TEST 2: Definition from consumer file ===");
  const consumerFile = path.join(TEST_PROJECT, "internal", "delivery", "grpc", "flow_handler.go");
  const consumerUri = pathToFileURL(consumerFile).href;
  const consumerText = await Bun.file(consumerFile).text();

  await connection.sendNotification("textDocument/didOpen", {
    textDocument: {
      uri: consumerUri,
      languageId: "go",
      version: 0,
      text: consumerText,
    },
  });
  await new Promise((r) => setTimeout(r, 2000));

  // Find AgentEvent in consumer file
  const lines = consumerText.split("\n");
  let aeLine = -1, aeCol = -1;
  for (let i = 0; i < lines.length; i++) {
    const col = lines[i].indexOf("AgentEvent");
    if (col !== -1 && !lines[i].trimStart().startsWith("//")) {
      aeLine = i;
      aeCol = col;
      break;
    }
  }

  if (aeLine >= 0) {
    console.log(`Found AgentEvent at flow_handler.go:${aeLine + 1}:${aeCol}`);

    // Definition from consumer
    const def: any = await connection
      .sendRequest("textDocument/definition", {
        textDocument: { uri: consumerUri },
        position: { line: aeLine, character: aeCol },
      })
      .catch(() => null);
    if (def) {
      const defs = Array.isArray(def) ? def : [def];
      console.log(`Definition: ${defs.length} results`);
      for (const d of defs) console.log(`  ${d.uri.replace(/.*test-project/, "")}:${d.range.start.line + 1}`);
    }

    // References from consumer
    console.log("\n=== TEST 3: References from CONSUMER file ===");
    const ref2: any = await connection
      .sendRequest("textDocument/references", {
        textDocument: { uri: consumerUri },
        position: { line: aeLine, character: aeCol },
        context: { includeDeclaration: true },
      })
      .catch((e: any) => { console.log(`ERROR: ${e.message}`); return []; });
    console.log(`Results: ${Array.isArray(ref2) ? ref2.length : 0}`);
    if (Array.isArray(ref2)) {
      const files = new Set(ref2.map((r: any) => r.uri.replace(/.*test-project/, "")));
      console.log(`Unique files: ${files.size}`);
      for (const f of files) console.log(`  ${f}`);
    }
  }

  // Test 4: References from DEFINITION file AGAIN after opening consumer
  console.log("\n=== TEST 4: References from definition file AGAIN (after consumer opened) ===");
  const ref3: any = await connection
    .sendRequest("textDocument/references", {
      textDocument: { uri: targetUri },
      position: { line: 28, character: 5 },
      context: { includeDeclaration: true },
    })
    .catch((e: any) => { console.log(`ERROR: ${e.message}`); return []; });
  console.log(`Results: ${Array.isArray(ref3) ? ref3.length : 0}`);
  if (Array.isArray(ref3)) {
    const files = new Set(ref3.map((r: any) => r.uri.replace(/.*test-project/, "")));
    console.log(`Unique files: ${files.size}`);
    for (const f of files) console.log(`  ${f}`);
  }

  connection.end();
  connection.dispose();
  proc.kill();
  process.exit(0);
}

main().catch((err) => {
  console.error("FATAL:", err);
  process.exit(1);
});
