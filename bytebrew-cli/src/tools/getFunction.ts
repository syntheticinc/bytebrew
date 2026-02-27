// get_function tool - get function/method by name
import { Tool, ToolResult } from './registry.js';
import { IChunkStoreFactory } from '../domain/store.js';
import { getLogger } from '../lib/logger.js';

export class GetFunctionTool implements Tool {
  readonly name = 'get_function';
  private storeFactory: IChunkStoreFactory;

  constructor(storeFactory: IChunkStoreFactory) {
    this.storeFactory = storeFactory;
  }

  async execute(args: Record<string, string>): Promise<ToolResult> {
    const logger = getLogger();
    const name = args.name;

    if (!name) {
      return { result: '', error: new Error('name argument is required') };
    }

    const filePath = args.file_path;

    try {
      const store = await this.storeFactory.getStore();
      let chunks = await store.getByName(name);

      // Filter by file path if provided
      if (filePath && chunks.length > 0) {
        chunks = chunks.filter((c) => c.filePath.includes(filePath));
      }

      // Filter to only functions and methods
      chunks = chunks.filter((c) => c.chunkType === 'function' || c.chunkType === 'method');

      if (chunks.length === 0) {
        // Try semantic search as fallback
        const searchResults = await store.searchWithFilter(
          `function ${name}`,
          5,
          { chunkType: 'function' }
        );

        if (searchResults.length > 0) {
          const exactMatch = searchResults.find((r) =>
            r.chunk.name.toLowerCase() === name.toLowerCase()
          );

          if (exactMatch) {
            const chunk = exactMatch.chunk;
            return {
              result:
                `## Function: ${chunk.name}\n` +
                `File: ${chunk.filePath}:${chunk.startLine}-${chunk.endLine}\n` +
                `\`\`\`${chunk.language}\n${chunk.content}\n\`\`\``,
              summary: `1 function (${chunk.name})`,
            };
          }

          // Return similar functions
          return {
            result:
              `Function '${name}' not found. Similar functions:\n\n` +
              searchResults
                .slice(0, 3)
                .map((r) => `- ${r.chunk.name} (${r.chunk.filePath})`)
                .join('\n'),
            summary: `${searchResults.length} similar`,
          };
        }

        return {
          result: `Function '${name}' not found. Make sure the codebase is indexed.`,
          summary: 'not found',
        };
      }

      // Return all matching functions
      const output = chunks
        .map((chunk) => {
          return (
            `## ${chunk.chunkType}: ${chunk.name}\n` +
            `File: ${chunk.filePath}:${chunk.startLine}-${chunk.endLine}\n` +
            `\`\`\`${chunk.language}\n${chunk.content}\n\`\`\``
          );
        })
        .join('\n\n');

      logger.debug('Found function', { name, count: chunks.length });
      return {
        result: output,
        summary: `${chunks.length} functions`,
      };
    } catch (error) {
      const err = error as Error;
      logger.error('Get function failed', { error: err.message });

      if (this.isConnectionError(err)) {
        return {
          result: '',
          error: new Error(
            'Cannot connect to Ollama. Make sure Ollama is running on localhost:11434.'
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
