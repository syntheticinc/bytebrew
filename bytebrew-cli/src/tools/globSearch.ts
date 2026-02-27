// Glob search tool - find files by name pattern
import { Tool, ToolResult } from './registry.js';
import { globSearch } from '../infrastructure/search/globSearch.js';
import { getLogger } from '../lib/logger.js';

export interface GlobSearchToolConfig {
  projectRoot: string;
}

export class GlobSearchTool implements Tool {
  readonly name = 'glob';
  private projectRoot: string;

  constructor(config: GlobSearchToolConfig) {
    this.projectRoot = config.projectRoot;
  }

  async execute(args: Record<string, string>): Promise<ToolResult> {
    const logger = getLogger();
    const pattern = args.pattern;

    if (!pattern) {
      return { result: '', error: new Error('pattern argument is required') };
    }

    const limit = parseInt(args.limit || '100', 10);

    try {
      logger.debug('Glob search', { pattern, limit });

      // Request limit + 1 to detect truncation
      const matches = await globSearch(this.projectRoot, pattern, {
        maxResults: limit + 1,
      });

      if (matches.length === 0) {
        return {
          result: `No files found matching pattern: "${pattern}"`,
          summary: 'no results',
        };
      }

      const truncated = matches.length > limit;
      const displayMatches = truncated ? matches.slice(0, limit) : matches;

      let output = displayMatches.map((m) => m.filePath).join('\n');
      if (truncated) {
        output += '\n\n(Results truncated. Consider using a more specific pattern or path.)';
      }

      logger.debug('Glob search completed', { resultsCount: displayMatches.length, truncated });
      return {
        result: output,
        summary: `${displayMatches.length} files${truncated ? ' (truncated)' : ''}`,
      };
    } catch (error) {
      const err = error as Error;
      logger.error('Glob search failed', { error: err.message });
      return { result: '', error: err };
    }
  }
}
