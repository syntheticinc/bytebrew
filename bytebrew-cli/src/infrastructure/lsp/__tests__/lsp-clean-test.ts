/**
 * LSP test on a CLEAN Go project that compiles without errors.
 * Tests: definition, references, implementation on a project with cross-file usage.
 * Usage: bun src/infrastructure/lsp/__tests__/lsp-clean-test.ts
 */
import path from "path";
import { pathToFileURL } from "url";
import {
  createMessageConnection,
  StreamMessageReader,
  StreamMessageWriter,
} from "vscode-jsonrpc/node";
import { spawn } from "child_process";

const TEST_PROJECT = "C:\\Users\\busul\\AppData\\Local\\Temp\\lsp-test-project";

async function main() {
  console.log(`Project: ${TEST_PROJECT}`);

  const proc = spawn("C:\\Users\\busul\\go\\bin\\gopls.exe", [], { cwd: TEST_PROJECT });
  const connection = createMessageConnection(
    new StreamMessageReader(proc.stdout as any),
    new StreamMessageWriter(proc.stdin as any),
  );

  // Track progress
  let progressDone = false;
  connection.onNotification((method: string, params: any) => {
    if (method === "textDocument/publishDiagnostics") {
      const uri = params.uri || "";
      const count = params.diagnostics?.length || 0;
      if (count > 0) {
        console.log(`[DIAG] ${uri.replace(/.*lsp-test-project/, "")} (${count} errors)`);
        for (const d of params.diagnostics) {
          console.log(`  ${d.range.start.line}:${d.range.start.character} ${d.message.slice(0, 100)}`);
        }
      }
    } else if (method === "$/progress") {
      const kind = params.value?.kind;
      const msg = params.value?.message || params.value?.title || "";
      console.log(`[PROGRESS] kind=${kind} msg="${msg}"`);
      if (kind === "end") progressDone = true;
    }
  });

  connection.onRequest("window/workDoneProgress/create", () => null);
  connection.onRequest("workspace/configuration", () => [{}]);
  connection.onRequest("client/registerCapability", () => ({}));
  connection.onRequest("client/unregisterCapability", () => {});
  connection.onRequest("workspace/workspaceFolders", () => [
    { name: "workspace", uri: pathToFileURL(TEST_PROJECT).href },
  ]);

  connection.listen();

  // Initialize
  await connection.sendRequest("initialize", {
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
  await connection.sendNotification("initialized", {});
  console.log("Initialized. Waiting for gopls...");

  // Wait for gopls to finish loading
  await new Promise((r) => setTimeout(r, 8000));
  console.log(`Progress done: ${progressDone}`);

  // === TEST: References of MyStruct (defined in pkg/types.go, used in pkg/impl.go and main.go) ===
  const typesFile = path.join(TEST_PROJECT, "pkg", "types.go");
  const typesUri = pathToFileURL(typesFile).href;
  const typesText = await Bun.file(typesFile).text();

  // Open the file
  await connection.sendNotification("textDocument/didOpen", {
    textDocument: { uri: typesUri, languageId: "go", version: 0, text: typesText },
  });
  await new Promise((r) => setTimeout(r, 2000));

  // Find MyStruct position
  const lines = typesText.split("\n");
  let structLine = -1, structCol = -1;
  for (let i = 0; i < lines.length; i++) {
    if (lines[i].includes("type MyStruct struct")) {
      structCol = lines[i].indexOf("MyStruct");
      structLine = i;
      break;
    }
  }
  console.log(`\nMyStruct at types.go:${structLine + 1}:${structCol}`);

  // Test 1: References of MyStruct
  console.log("\n=== TEST 1: References of MyStruct (from definition file) ===");
  const refs: any = await connection
    .sendRequest("textDocument/references", {
      textDocument: { uri: typesUri },
      position: { line: structLine, character: structCol },
      context: { includeDeclaration: true },
    })
    .catch((e: any) => { console.log(`ERROR: ${e.message}`); return []; });

  console.log(`Results: ${Array.isArray(refs) ? refs.length : 0}`);
  if (Array.isArray(refs)) {
    for (const r of refs) {
      const file = r.uri.replace(/.*lsp-test-project/, "");
      console.log(`  ${file}:${r.range.start.line + 1}:${r.range.start.character}`);
    }
  }

  // Test 2: References of MyService (interface, defined in types.go, implemented in impl.go)
  let ifaceLine = -1, ifaceCol = -1;
  for (let i = 0; i < lines.length; i++) {
    if (lines[i].includes("type MyService interface")) {
      ifaceCol = lines[i].indexOf("MyService");
      ifaceLine = i;
      break;
    }
  }
  console.log(`\nMyService at types.go:${ifaceLine + 1}:${ifaceCol}`);

  console.log("\n=== TEST 2: References of MyService (interface) ===");
  const ifaceRefs: any = await connection
    .sendRequest("textDocument/references", {
      textDocument: { uri: typesUri },
      position: { line: ifaceLine, character: ifaceCol },
      context: { includeDeclaration: true },
    })
    .catch((e: any) => { console.log(`ERROR: ${e.message}`); return []; });

  console.log(`Results: ${Array.isArray(ifaceRefs) ? ifaceRefs.length : 0}`);
  if (Array.isArray(ifaceRefs)) {
    for (const r of ifaceRefs) {
      const file = r.uri.replace(/.*lsp-test-project/, "");
      console.log(`  ${file}:${r.range.start.line + 1}:${r.range.start.character}`);
    }
  }

  // Test 3: Implementation of MyService interface
  console.log("\n=== TEST 3: Implementation of MyService ===");
  const impls: any = await connection
    .sendRequest("textDocument/implementation", {
      textDocument: { uri: typesUri },
      position: { line: ifaceLine, character: ifaceCol },
    })
    .catch((e: any) => { console.log(`ERROR: ${e.message}`); return []; });

  console.log(`Results: ${Array.isArray(impls) ? impls.length : 0}`);
  if (Array.isArray(impls)) {
    for (const r of impls) {
      const file = r.uri.replace(/.*lsp-test-project/, "");
      console.log(`  ${file}:${r.range.start.line + 1}:${r.range.start.character}`);
    }
  }

  // Test 4: Definition from consumer file (main.go) WITHOUT opening it first
  console.log("\n=== TEST 4: Definition from main.go (NOT opened via didOpen) ===");
  const mainFile = path.join(TEST_PROJECT, "main.go");
  const mainUri = pathToFileURL(mainFile).href;
  const mainText = await Bun.file(mainFile).text();
  const mainLines = mainText.split("\n");

  let myStructUseLine = -1, myStructUseCol = -1;
  for (let i = 0; i < mainLines.length; i++) {
    const col = mainLines[i].indexOf("NewMyStruct");
    if (col !== -1 && !mainLines[i].trimStart().startsWith("//")) {
      myStructUseLine = i;
      myStructUseCol = col;
      break;
    }
  }

  if (myStructUseLine >= 0) {
    console.log(`NewMyStruct usage at main.go:${myStructUseLine + 1}:${myStructUseCol}`);
    // Try definition WITHOUT opening main.go
    const def: any = await connection
      .sendRequest("textDocument/definition", {
        textDocument: { uri: mainUri },
        position: { line: myStructUseLine, character: myStructUseCol },
      })
      .catch((e: any) => { console.log(`ERROR (not opened): ${e.message}`); return null; });

    if (def) {
      const defs = Array.isArray(def) ? def : [def];
      console.log(`Definition (file NOT opened): ${defs.length} results`);
      for (const d of defs) {
        console.log(`  ${d.uri.replace(/.*lsp-test-project/, "")}:${d.range.start.line + 1}`);
      }
    } else {
      console.log("Definition returned null (file not opened)");
    }

    // Now open it and retry
    await connection.sendNotification("textDocument/didOpen", {
      textDocument: { uri: mainUri, languageId: "go", version: 0, text: mainText },
    });
    await new Promise((r) => setTimeout(r, 2000));

    console.log("\n=== TEST 5: Definition from main.go (AFTER didOpen) ===");
    const def2: any = await connection
      .sendRequest("textDocument/definition", {
        textDocument: { uri: mainUri },
        position: { line: myStructUseLine, character: myStructUseCol },
      })
      .catch((e: any) => { console.log(`ERROR (opened): ${e.message}`); return null; });

    if (def2) {
      const defs = Array.isArray(def2) ? def2 : [def2];
      console.log(`Definition (file opened): ${defs.length} results`);
      for (const d of defs) {
        console.log(`  ${d.uri.replace(/.*lsp-test-project/, "")}:${d.range.start.line + 1}`);
      }
    }

    // Test 6: References from main.go after opening it
    console.log("\n=== TEST 6: References of MyStruct from main.go ===");
    // Find MyStruct in main.go
    let mStructLine = -1, mStructCol = -1;
    for (let i = 0; i < mainLines.length; i++) {
      const col = mainLines[i].indexOf("MyStruct");
      if (col !== -1 && mainLines[i].includes("pkg.")) {
        mStructLine = i;
        mStructCol = col;
        break;
      }
    }

    if (mStructLine >= 0) {
      console.log(`MyStruct at main.go:${mStructLine + 1}:${mStructCol}`);
      const refs2: any = await connection
        .sendRequest("textDocument/references", {
          textDocument: { uri: mainUri },
          position: { line: mStructLine, character: mStructCol },
          context: { includeDeclaration: true },
        })
        .catch((e: any) => { console.log(`ERROR: ${e.message}`); return []; });

      console.log(`Results: ${Array.isArray(refs2) ? refs2.length : 0}`);
      if (Array.isArray(refs2)) {
        for (const r of refs2) {
          const file = r.uri.replace(/.*lsp-test-project/, "");
          console.log(`  ${file}:${r.range.start.line + 1}:${r.range.start.character}`);
        }
      }
    }
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
