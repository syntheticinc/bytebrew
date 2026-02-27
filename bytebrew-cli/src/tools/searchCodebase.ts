// Semantic search tool - vector search in indexed code chunks
import { Tool, ToolResult } from './registry.js';
import { IChunkStoreFactory } from '../domain/store.js';
import { getLogger } from '../lib/logger.js';
import { toRelativePath } from '../lib/pathUtils.js';

export interface SearchCodebaseToolConfig {
  storeFactory: IChunkStoreFactory;
  /** Project root path for relative path conversion */
  projectRoot?: string;
  /** Tool name exposed to the server. Default: 'search_code' */
  name?: string;
}

export class SearchCodebaseTool implements Tool {
  readonly name: string;
  private storeFactory: IChunkStoreFactory;
  private projectRoot?: string;

  constructor(config: SearchCodebaseToolConfig) {
    this.storeFactory = config.storeFactory;
    this.projectRoot = config.projectRoot;
    this.name = config.name || 'search_code';
  }

  async execute(args: Record<string, string>): Promise<ToolResult> {
    const logger = getLogger();
    const query = args.query;

    if (!query) {
      return { result: '', error: new Error('query argument is required') };
    }

    const limit = parseInt(args.limit || '10', 10);
    const language = args.language;
    const chunkType = args.chunk_type;

    try {
      const store = await this.storeFactory.getStore();

      const filter = language || chunkType
        ? { language, chunkType }
        : undefined;

      logger.debug('Searching codebase', { query, limit, filter });

      const results = filter
        ? await store.searchWithFilter(query, limit, filter)
        : await store.search(query, limit);

      if (results.length === 0) {
        return {
          result: `No results found for query: "${query}". Make sure the codebase is indexed (run 'vector index').`,
          summary: 'no results',
        };
      }

      const output = results
        .map((r) => {
          const chunk = r.chunk;
          // Convert absolute path to relative for consistent output
          const relativePath = toRelativePath(chunk.filePath, this.projectRoot);
          // For file chunks (chunkType='other'), name is the file path - use relative version
          // For other chunks, name is the symbol name (function, class, etc.)
          const displayName = chunk.chunkType === 'other'
            ? toRelativePath(chunk.name, this.projectRoot)
            : chunk.name;
          return (
            `## ${chunk.chunkType}: ${displayName}\n` +
            `File: ${relativePath}:${chunk.startLine}-${chunk.endLine}\n` +
            `Score: ${r.score.toFixed(3)}\n` +
            `\`\`\`${chunk.language}\n${chunk.content}\n\`\`\``
          );
        })
        .join('\n\n');

      logger.debug('Search completed', { resultsCount: results.length });
      return {
        result: output,
        summary: `${results.length} results`,
      };
    } catch (error) {
      const err = error as Error;
      logger.error('Search failed', { error: err.message });

      if (this.isConnectionError(err)) {
        return {
          result: '',
          error: new Error(
            'Cannot connect to Ollama. Make sure Ollama is running on localhost:11434. ' +
            'Start it with: ollama serve'
          ),
        };
      }
      return { result: '', error: err };
    }
  }

  private isConnectionError(error: Error): boolean {
    const message = error.message?.toLowerCase() || '';
    return message.includes('econnrefused') ||
           message.includes('fetch failed') ||
           message.includes('network');
  }
}
