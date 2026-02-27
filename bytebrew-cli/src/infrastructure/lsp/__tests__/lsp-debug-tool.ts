/**
 * Debug LspTool logic — test symbol search + position resolution + LSP call
 * Usage: cd test-project && bun ../vector-cli-node/src/infrastructure/lsp/__tests__/lsp-debug-tool.ts
 */
import path from "path";
import fs from "fs/promises";
import { ChunkStoreFactory } from "../../../indexing/storeFactory.js";
import { symbolSearch } from "../../search/symbolSearch.js";
import { LspManager } from "../LspManager.js";
import { LspService } from "../LspService.js";

const PROJECT_ROOT = process.cwd();
console.log(`Project: ${PROJECT_ROOT}`);

async function main() {
  // 1. Symbol search
  console.log("\n=== SYMBOL SEARCH ===");
  const storeFactory = new ChunkStoreFactory(PROJECT_ROOT);
  const store = await storeFactory.getStore();

  let matches = await symbolSearch(store, "AgentEvent", { exactMatch: true });
  console.log(`Exact matches: ${matches.length}`);
  if (matches.length === 0) {
    matches = await symbolSearch(store, "AgentEvent");
    console.log(`Fuzzy matches: ${matches.length}`);
  }

  if (matches.length === 0) {
    console.log("NO MATCHES FOUND");
    process.exit(1);
  }

  const match = matches[0];
  console.log(`Match: ${match.filePath}:${match.startLine}`);

  // 2. Position resolution (same logic as LspTool.resolvePosition)
  console.log("\n=== POSITION RESOLUTION ===");
  const symbolName = "AgentEvent";
  const { filePath, startLine } = match;

  try {
    const fileContent = await fs.readFile(filePath, "utf-8");
    const fileLines = fileContent.split("\n");

    let resolved = false;
    for (let offset = 0; offset <= 10; offset++) {
      const candidates = offset === 0 ? [0] : [offset, -offset];
      for (const delta of candidates) {
        const lineIdx = startLine - 1 + delta;
        if (lineIdx < 0 || lineIdx >= fileLines.length) continue;
        const line = fileLines[lineIdx];
        const trimmed = line.trimStart();
        const isComment = trimmed.startsWith("//") || trimmed.startsWith("/*") || trimmed.startsWith("*");

        // Word boundary search
        let col = -1;
        let searchStart = 0;
        while (searchStart < line.length) {
          const idx = line.indexOf(symbolName, searchStart);
          if (idx === -1) break;
          const before = idx > 0 ? line[idx - 1] : " ";
          const after = idx + symbolName.length < line.length ? line[idx + symbolName.length] : " ";
          if (!/[a-zA-Z0-9_]/.test(before) && !/[a-zA-Z0-9_]/.test(after)) {
            col = idx;
            break;
          }
          searchStart = idx + 1;
        }

        console.log(`  offset=${delta} lineIdx=${lineIdx} (1-based: ${lineIdx + 1}) comment=${isComment} wholeWord=${col} text="${line.slice(0, 60)}..."`);

        if (!isComment && col !== -1) {
          console.log(`\n  RESOLVED: line=${lineIdx} (0-based), character=${col}, 1-based=${lineIdx + 1}:${col}`);
          resolved = true;

          // 3. Test LSP
          console.log("\n=== LSP TEST ===");
          const manager = new LspManager(PROJECT_ROOT);
          console.log("Warming up LSP...");
          await manager.warmup();
          console.log("Warmup done. Waiting 5s for gopls to load...");
          await new Promise(r => setTimeout(r, 5000));

          const lspService = new LspService(manager);
          console.log(`Calling definition(${filePath}, ${lineIdx}, ${col})...`);
          const defResult = await lspService.definition(filePath, lineIdx, col);
          console.log(`Definition results: ${defResult.length}`);
          for (const loc of defResult) {
            console.log(`  ${loc.uri} line ${loc.range.start.line + 1}`);
          }

          console.log(`\nCalling references(${filePath}, ${lineIdx}, ${col})...`);
          const refResult = await lspService.references(filePath, lineIdx, col);
          console.log(`References results: ${refResult.length}`);
          for (const loc of refResult.slice(0, 10)) {
            console.log(`  ${loc.uri.replace(/.*test-project/, "")} line ${loc.range.start.line + 1}`);
          }
          if (refResult.length > 10) console.log(`  ... and ${refResult.length - 10} more`);

          await manager.dispose();
          break;
        }
      }
      if (resolved) break;
    }

    if (!resolved) {
      console.log("FAILED TO RESOLVE POSITION");
    }
  } catch (err) {
    console.error("Error:", err);
  }

  process.exit(0);
}

main().catch(err => {
  console.error("FATAL:", err);
  process.exit(1);
});
