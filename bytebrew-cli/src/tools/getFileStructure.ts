// get_file_structure tool - get symbols in a file
import { Tool, ToolResult } from './registry.js';
import { IChunkStoreFactory } from '../domain/store.js';
import { CodeChunk } from '../domain/chunk.js';
import { getLogger } from '../lib/logger.js';

interface GroupedChunks {
  classes: CodeChunk[];
  functions: CodeChunk[];
  variables: CodeChunk[];
}

export class GetFileStructureTool implements Tool {
  readonly name = 'get_file_structure';
  private storeFactory: IChunkStoreFactory;

  constructor(storeFactory: IChunkStoreFactory) {
    this.storeFactory = storeFactory;
  }

  async execute(args: Record<string, string>): Promise<ToolResult> {
    const logger = getLogger();
    const filePath = args.file_path;

    if (!filePath) {
      return { result: '', error: new Error('file_path argument is required') };
    }

    try {
      const store = await this.storeFactory.getStore();
      const chunks = await store.getByFilePath(filePath);

      if (chunks.length === 0) {
        return {
          result: `No indexed symbols found for file: ${filePath}. Make sure the codebase is indexed.`,
          summary: 'no symbols',
        };
      }

      // Sort by line number
      chunks.sort((a, b) => a.startLine - b.startLine);

      // Group by type
      const grouped = this.groupByType(chunks);

      let output = `# File Structure: ${filePath}\n\n`;

      // Classes/Structs/Interfaces
      if (grouped.classes.length > 0) {
        output += '## Classes/Structs/Interfaces\n';
        for (const chunk of grouped.classes) {
          output += `- **${chunk.chunkType}** \`${chunk.name}\` (lines ${chunk.startLine}-${chunk.endLine})\n`;
          if (chunk.signature) {
            output += `  \`${chunk.signature.split('\n')[0]}\`\n`;
          }
        }
        output += '\n';
      }

      // Functions/Methods
      if (grouped.functions.length > 0) {
        output += '## Functions/Methods\n';
        for (const chunk of grouped.functions) {
          const parent = chunk.parentName ? ` (${chunk.parentName})` : '';
          output += `- **${chunk.chunkType}** \`${chunk.name}\`${parent} (lines ${chunk.startLine}-${chunk.endLine})\n`;
          if (chunk.signature) {
            output += `  \`${chunk.signature.split('\n')[0]}\`\n`;
          }
        }
        output += '\n';
      }

      // Variables/Constants
      if (grouped.variables.length > 0) {
        output += '## Variables/Constants\n';
        for (const chunk of grouped.variables) {
          output += `- **${chunk.chunkType}** \`${chunk.name}\` (line ${chunk.startLine})\n`;
        }
        output += '\n';
      }

      // Summary
      output += `---\nTotal symbols: ${chunks.length} | `;
      output += `Classes: ${grouped.classes.length} | `;
      output += `Functions: ${grouped.functions.length} | `;
      output += `Variables: ${grouped.variables.length}`;

      logger.debug('Retrieved file structure', { filePath, symbolsCount: chunks.length });
      return {
        result: output,
        summary: `${chunks.length} symbols`,
      };
    } catch (error) {
      const err = error as Error;
      logger.error('Get file structure failed', { error: err.message });

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

  private groupByType(chunks: CodeChunk[]): GroupedChunks {
    const classes: CodeChunk[] = [];
    const functions: CodeChunk[] = [];
    const variables: CodeChunk[] = [];

    for (const chunk of chunks) {
      switch (chunk.chunkType) {
        case 'class':
        case 'struct':
        case 'interface':
        case 'type':
          classes.push(chunk);
          break;
        case 'function':
        case 'method':
          functions.push(chunk);
          break;
        case 'variable':
        case 'constant':
          variables.push(chunk);
          break;
      }
    }

    return { classes, functions, variables };
  }

  private isConnectionError(error: Error): boolean {
    const message = error.message?.toLowerCase() || '';
    return message.includes('econnrefused') ||
           message.includes('fetch failed') ||
           message.includes('network');
  }
}
