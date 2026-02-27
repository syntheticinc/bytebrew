// Grep search tool - pattern-based search using ripgrep
import { Tool, ToolResult } from './registry.js';
import { grepSearch, GrepMatch } from '../infrastructure/search/grepSearch.js';
import { getLogger } from '../lib/logger.js';

export interface GrepSearchToolConfig {
  projectRoot: string;
  name?: string;
}

export class GrepSearchTool implements Tool {
  readonly name: string;
  private projectRoot: string;

  constructor(config: GrepSearchToolConfig) {
    this.projectRoot = config.projectRoot;
    this.name = config.name || 'grep_search';
  }

  async execute(args: Record<string, string>): Promise<ToolResult> {
    const logger = getLogger();
    const pattern = args.pattern;

    if (!pattern) {
      return { result: '', error: new Error('pattern argument is required') };
    }

    const limit = parseInt(args.limit || '100', 10);
    const fileTypesStr = args.file_types || '';
    const fileTypes = fileTypesStr ? fileTypesStr.split(',').map((t) => t.trim()).filter(Boolean) : undefined;
    const ignoreCase = args.ignore_case === 'true';

    try {
      logger.debug('Grep search', { pattern, limit, fileTypes, ignoreCase });

      // Request limit + 1 to detect truncation
      const matches = await grepSearch(this.projectRoot, pattern, {
        maxResults: limit + 1,
        fileTypes,
        ignoreCase,
      });

      if (matches.length === 0) {
        return {
          result: `No matches found for pattern: "${pattern}"`,
          summary: 'no results',
        };
      }

      const truncated = matches.length > limit;
      const displayMatches = truncated ? matches.slice(0, limit) : matches;

      let output = this.formatMatches(displayMatches);
      if (truncated) {
        output += '\n\n(Results truncated. Consider using a more specific pattern or include filter.)';
      }

      logger.debug('Grep search completed', { resultsCount: displayMatches.length, truncated });
      return {
        result: output,
        summary: `${displayMatches.length} results${truncated ? ' (truncated)' : ''}`,
      };
    } catch (error) {
      const err = error as Error;
      logger.error('Grep search failed', { error: err.message });
      return { result: '', error: err };
    }
  }

  private formatMatches(matches: GrepMatch[]): string {
    return matches
      .map((m) => {
        const location = `${m.filePath}:${m.line}`;
        const content = m.content.trim();
        return `${location}\n  ${content}`;
      })
      .join('\n\n');
  }
}
