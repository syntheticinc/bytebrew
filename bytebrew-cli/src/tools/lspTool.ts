// LSP tool - navigate code using Language Server Protocol
import { fileURLToPath } from 'url';
import fs from 'fs/promises';
import { Tool, ToolResult } from './registry.js';
import { LspService, LspLocation } from '../infrastructure/lsp/LspService.js';
import { IChunkStoreFactory } from '../domain/store.js';
import { symbolSearch } from '../infrastructure/search/symbolSearch.js';
import { toRelativePath } from '../lib/pathUtils.js';
import { getLogger } from '../lib/logger.js';

const VALID_OPERATIONS = ['definition', 'references', 'implementation'] as const;
type LspOperation = (typeof VALID_OPERATIONS)[number];

export interface LspToolConfig {
  lspService: LspService;
  storeFactory: IChunkStoreFactory;
  projectRoot: string;
  /**
   * Timeout for waiting LSP readiness (ms). Defaults to 15000.
   * Set to 0 in tests to skip the wait.
   */
  readyTimeoutMs?: number;
}

/**
 * LSP tool: resolves a symbol's position via symbol search,
 * then calls the LSP server to get definition / references / implementation.
 */
export class LspTool implements Tool {
  readonly name = 'lsp';

  private lspService: LspService;
  private storeFactory: IChunkStoreFactory;
  private projectRoot: string;
  private readyTimeoutMs: number;

  constructor(config: LspToolConfig) {
    this.lspService = config.lspService;
    this.storeFactory = config.storeFactory;
    this.projectRoot = config.projectRoot;
    this.readyTimeoutMs = config.readyTimeoutMs ?? 15_000;
  }

  async execute(args: Record<string, string>): Promise<ToolResult> {
    const logger = getLogger();

    try {
      const symbolName = args.symbol_name;
      const operation = args.operation as LspOperation;

      if (!symbolName) {
        return { result: '', error: new Error('symbol_name argument is required') };
      }

      if (!operation) {
        return { result: '', error: new Error('operation argument is required') };
      }

      if (!(VALID_OPERATIONS as readonly string[]).includes(operation)) {
        return {
          result: '',
          error: new Error(
            `Invalid operation "${operation}". Must be one of: ${VALID_OPERATIONS.join(', ')}`,
          ),
        };
      }

      // Resolve symbol → position via symbol search
      let store;
      try {
        store = await this.storeFactory.getStore();
      } catch (err) {
        return {
          result: `Symbol search unavailable. Make sure the codebase is indexed (run 'vector index').`,
          summary: 'index unavailable',
        };
      }

      // Exact match only — no fuzzy/embedding fallback (avoids Ollama dependency)
      const matches = await symbolSearch(store, symbolName, { exactMatch: true });

      if (matches.length === 0) {
        return {
          result: `No symbols found matching "${symbolName}". Make sure codebase is indexed.`,
          summary: 'symbol not found',
        };
      }

      const match = matches[0];
      const { filePath, startLine } = match;

      // Resolve exact position by reading the actual file.
      // Chunk content may differ from the file (e.g., chunker strips keywords
      // like "type " or "func "), so we read the real file lines near startLine.
      const { line, character, resolvedLine } = await this.resolvePosition(
        filePath,
        startLine,
        symbolName,
      );

      const symbolSource = `${toRelativePath(filePath, this.projectRoot)}:${resolvedLine}`;

      logger.info('[LspTool] resolved position', {
        symbolName, operation, filePath,
        line, character, resolvedLine,
        matchStartLine: startLine,
      });

      // Wait for LSP servers to finish workspace indexing before querying.
      // gopls sends $/progress end when ready; waitForReady() resolves then.
      // Falls back after readyTimeoutMs if gopls doesn't send $/progress.
      if (this.readyTimeoutMs > 0) {
        await this.lspService.waitForReady(this.readyTimeoutMs);
      }

      let locations: LspLocation[];
      try {
        locations = await this.callLsp(operation, filePath, line, character);

        // Single short retry: gopls may need a didOpen for the specific file
        // even after workspace indexing is complete.
        // Only retry if LSP servers are actually running — retrying without
        // active servers just wastes time (2s × N calls adds up).
        if (locations.length === 0 && this.lspService.hasActiveClients()) {
          logger.info('[LspTool] empty result, retrying after short delay', {
            operation, symbolName,
          });
          await new Promise((r) => setTimeout(r, 2_000));
          locations = await this.callLsp(operation, filePath, line, character);
        }
      } catch (err) {
        logger.warn('[LspTool] LSP operation failed', { operation, symbolName, error: err });
        const health = this.lspService.getHealthInfo(filePath);
        const hint = health ? ` Hint: ${health}` : '';
        return {
          result: `LSP operation failed for "${operation}" on symbol "${symbolName}".${hint}`,
          summary: 'LSP error',
        };
      }

      if (locations.length === 0) {
        const health = this.lspService.getHealthInfo(filePath);
        const hint = health ? ` Hint: ${health}` : '';
        return {
          result: `No results found for operation "${operation}" on symbol "${symbolName}".${hint}`,
          summary: 'no results',
        };
      }

      const formatted = this.formatLocations(operation, symbolName, symbolSource, locations);
      return {
        result: formatted,
        summary: `${locations.length} result${locations.length === 1 ? '' : 's'}`,
      };
    } catch (error) {
      const err = error as Error;
      getLogger().error('[LspTool] Unexpected error', { error: err.message });
      return { result: '', error: err };
    }
  }

  /**
   * Find the exact 0-based line and column of a symbol by reading the file.
   * Searches near startLine (±10 lines) to handle chunker offset differences.
   * Chunk startLine may point to the beginning of a chunk that contains the symbol
   * several lines later (e.g. a struct field vs the struct declaration).
   * Uses word-boundary matching to avoid partial matches (e.g. "Foo" in "FooBar").
   */
  private async resolvePosition(
    filePath: string,
    startLine: number,
    symbolName: string,
  ): Promise<{ line: number; character: number; resolvedLine: number }> {
    try {
      const fileContent = await fs.readFile(filePath, 'utf-8');
      const fileLines = fileContent.split('\n');

      // Search near startLine, expanding outward (±10 lines to cover chunk offsets)
      for (let offset = 0; offset <= 10; offset++) {
        const candidates = offset === 0 ? [0] : [offset, -offset];
        for (const delta of candidates) {
          const lineIdx = startLine - 1 + delta; // 1-based to 0-based
          if (lineIdx < 0 || lineIdx >= fileLines.length) continue;
          const trimmed = fileLines[lineIdx].trimStart();
          if (trimmed.startsWith('//') || trimmed.startsWith('/*') || trimmed.startsWith('*')) continue;
          const col = this.findWholeWord(fileLines[lineIdx], symbolName);
          if (col !== -1) {
            return { line: lineIdx, character: col, resolvedLine: lineIdx + 1 };
          }
        }
      }
    } catch {
      // File read failed, use defaults
    }

    // Fallback: use startLine with character 0
    return { line: startLine - 1, character: 0, resolvedLine: startLine };
  }

  /**
   * Find a whole-word match of `word` in `line`.
   * Returns the column index or -1 if not found.
   * Ensures the match is not part of a larger identifier (e.g. "Foo" won't match "FooBar").
   */
  private findWholeWord(line: string, word: string): number {
    let start = 0;
    while (start < line.length) {
      const idx = line.indexOf(word, start);
      if (idx === -1) return -1;

      const before = idx > 0 ? line[idx - 1] : ' ';
      const after = idx + word.length < line.length ? line[idx + word.length] : ' ';
      const isWordBoundaryBefore = !/[a-zA-Z0-9_]/.test(before);
      const isWordBoundaryAfter = !/[a-zA-Z0-9_]/.test(after);

      if (isWordBoundaryBefore && isWordBoundaryAfter) {
        return idx;
      }
      start = idx + 1;
    }
    return -1;
  }

  private async callLsp(
    operation: LspOperation,
    filePath: string,
    line: number,
    character: number,
  ): Promise<LspLocation[]> {
    switch (operation) {
      case 'definition':
        return this.lspService.definition(filePath, line, character);
      case 'references':
        return this.lspService.references(filePath, line, character);
      case 'implementation':
        return this.lspService.implementation(filePath, line, character);
    }
  }

  private formatLocations(
    operation: LspOperation,
    symbolName: string,
    symbolSource: string,
    locations: LspLocation[],
  ): string {
    const lines = locations.map((loc) => {
      const filePath = this.uriToRelative(loc.uri);
      // LSP lines are 0-based, convert to 1-based for human readability
      const lineNum = loc.range.start.line + 1;
      return `  ${filePath}:${lineNum}`;
    });

    const header = this.buildHeader(operation, symbolName, symbolSource, locations.length);
    return `${header}\n${lines.join('\n')}`;
  }

  private buildHeader(
    operation: LspOperation,
    symbolName: string,
    symbolSource: string,
    count: number,
  ): string {
    const opLabel = operation.charAt(0).toUpperCase() + operation.slice(1);
    const fromPart = `from symbol search: ${symbolSource}`;

    if (operation === 'references') {
      return `${opLabel} of "${symbolName}" (${count} found, ${fromPart}):`;
    }
    return `${opLabel} of "${symbolName}" (${fromPart}):`;
  }

  private uriToRelative(uri: string): string {
    try {
      const absPath = fileURLToPath(uri);
      return toRelativePath(absPath, this.projectRoot);
    } catch {
      // If URI conversion fails, return the uri as-is
      return uri;
    }
  }
}
