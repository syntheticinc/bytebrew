// Symbol search tool - find code symbols by name
import { Tool, ToolResult } from './registry.js';
import { symbolSearch, SymbolMatch } from '../infrastructure/search/symbolSearch.js';
import { IChunkStoreFactory } from '../domain/store.js';
import { ChunkType } from '../domain/chunk.js';
import { getLogger } from '../lib/logger.js';
import { toRelativePath } from '../lib/pathUtils.js';

export interface SymbolSearchToolConfig {
  storeFactory: IChunkStoreFactory;
  projectRoot?: string;
  name?: string;
}

export class SymbolSearchTool implements Tool {
  readonly name: string;
  private storeFactory: IChunkStoreFactory;
  private projectRoot?: string;

  constructor(config: SymbolSearchToolConfig) {
    this.storeFactory = config.storeFactory;
    this.projectRoot = config.projectRoot;
    this.name = config.name || 'symbol_search';
  }

  async execute(args: Record<string, string>): Promise<ToolResult> {
    const logger = getLogger();

    // Support multiple argument naming conventions:
    // - symbol_name (from server proxy) or name (from direct LLM calls)
    // - limit or max_results
    // - symbol_types or type
    const symbolName = args.symbol_name || args.name || args.query;

    if (!symbolName) {
      logger.warn('symbol_name is missing', { args: JSON.stringify(args) });
      return { result: '', error: new Error('symbol_name argument is required') };
    }

    const limit = parseInt(args.limit || args.max_results || '10', 10);
    const symbolTypesStr = args.symbol_types || args.type || '';
    const symbolTypes = symbolTypesStr
      ? (symbolTypesStr.split(',').map((t) => t.trim()).filter(Boolean) as ChunkType[])
      : undefined;

    logger.debug('SymbolSearch execute', { symbolName, limit, symbolTypes });

    try {
      const store = await this.storeFactory.getStore();

      logger.debug('Symbol search', { symbolName, limit, symbolTypes });

      const matches = await symbolSearch(store, symbolName, {
        maxResults: limit,
        symbolTypes,
      });

      if (matches.length === 0) {
        return {
          result: `No symbols found matching: "${symbolName}"`,
          summary: 'no results',
        };
      }

      const output = this.formatMatches(matches);
      logger.debug('Symbol search completed', { resultsCount: matches.length });
      return {
        result: output,
        summary: `${matches.length} results`,
      };
    } catch (error) {
      const err = error as Error;
      logger.error('Symbol search failed', { error: err.message });

      if (this.isStoreNotInitialized(err)) {
        return {
          result: `Symbol search unavailable. Make sure the codebase is indexed (run 'vector index').`,
        };
      }

      return { result: '', error: err };
    }
  }

  private formatMatches(matches: SymbolMatch[]): string {
    return matches
      .map((m) => {
        const relativePath = toRelativePath(m.filePath, this.projectRoot);
        const location = `${relativePath}:${m.startLine}-${m.endLine}`;
        const type = m.symbolType;
        const sig = m.signature ? ` - ${m.signature}` : '';
        return `[${type}] ${m.symbolName}${sig}\n  ${location}`;
      })
      .join('\n\n');
  }

  private isStoreNotInitialized(error: Error): boolean {
    const message = error.message?.toLowerCase() || '';
    return message.includes('not initialized') || message.includes('no such file');
  }
}
