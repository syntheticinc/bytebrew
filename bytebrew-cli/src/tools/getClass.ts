// get_class tool - get class/struct/interface by name
import { Tool, ToolResult } from './registry.js';
import { IChunkStoreFactory } from '../domain/store.js';
import { getLogger } from '../lib/logger.js';

const CLASS_TYPES = ['class', 'struct', 'interface', 'type'] as const;

export class GetClassTool implements Tool {
  readonly name = 'get_class';
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

      // Filter to only classes, structs, and interfaces
      chunks = chunks.filter((c) =>
        CLASS_TYPES.includes(c.chunkType as typeof CLASS_TYPES[number])
      );

      if (chunks.length === 0) {
        // Try semantic search as fallback
        const searchResults = await store.searchWithFilter(
          `class ${name} struct interface type`,
          5,
          { chunkType: 'class' }
        );

        if (searchResults.length > 0) {
          const exactMatch = searchResults.find((r) =>
            r.chunk.name.toLowerCase() === name.toLowerCase()
          );

          if (exactMatch) {
            const chunk = exactMatch.chunk;
            return {
              result:
                `## ${chunk.chunkType}: ${chunk.name}\n` +
                `File: ${chunk.filePath}:${chunk.startLine}-${chunk.endLine}\n` +
                `\`\`\`${chunk.language}\n${chunk.content}\n\`\`\``,
              summary: `1 ${chunk.chunkType} (${chunk.name})`,
            };
          }

          // Return similar types
          return {
            result:
              `Class/struct '${name}' not found. Similar types:\n\n` +
              searchResults
                .slice(0, 3)
                .map((r) => `- ${r.chunk.name} (${r.chunk.chunkType}) in ${r.chunk.filePath}`)
                .join('\n'),
            summary: `${searchResults.length} similar`,
          };
        }

        return {
          result: `Class/struct '${name}' not found. Make sure the codebase is indexed.`,
          summary: 'not found',
        };
      }

      // Return all matching types
      const output = chunks
        .map((chunk) => {
          return (
            `## ${chunk.chunkType}: ${chunk.name}\n` +
            `File: ${chunk.filePath}:${chunk.startLine}-${chunk.endLine}\n` +
            `\`\`\`${chunk.language}\n${chunk.content}\n\`\`\``
          );
        })
        .join('\n\n');

      logger.debug('Found class/type', { name, count: chunks.length });
      return {
        result: output,
        summary: `${chunks.length} types`,
      };
    } catch (error) {
      const err = error as Error;
      logger.error('Get class failed', { error: err.message });

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
