// Smart search tool definition - smart_search
// Server-side only tool that combines vector, grep, and symbol search

import React from 'react';
import { Box, Text } from 'ink';
import { ToolDefinition, ToolRendererProps } from '../ToolManager.js';

/**
 * Parse smart_search result into citations
 */
interface Citation {
  location: string;
  source: string;
  symbol?: string;
  preview?: string;
}

/**
 * Shorten a file path for display
 * Example: /services/StreamProcessorService.ts:60
 */
function shortenPath(path: string, maxLength: number = 50): string {
  if (path.length <= maxLength) return path;
  // Trim from the start, keep the end (file name is most important)
  const shortened = path.slice(-maxLength);
  // Find first path separator and cut there to start with full directory name
  const slashIdx = shortened.indexOf('/');
  const backslashIdx = shortened.indexOf('\\');
  const sepIndex = slashIdx >= 0 ? (backslashIdx >= 0 ? Math.min(slashIdx, backslashIdx) : slashIdx) : backslashIdx;
  if (sepIndex > 0) {
    return shortened.slice(sepIndex);
  }
  return shortened;
}

function parseSmartSearchResult(result: string): Citation[] {
  if (!result) return [];

  const citations: Citation[] = [];
  const lines = result.split('\n');

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    // Match numbered results: "1. path:start-end [source] ..." - use (.+?) for paths with special chars
    const match = line.match(/^\d+\.\s+(.+?)\s+\[(\w+)\](.*)$/);
    if (match) {
      const citation: Citation = {
        location: match[1],
        source: match[2],
      };

      // Extract symbol/type info if present
      const rest = match[3].trim();
      if (rest) {
        // Format: (type) name: signature or just name
        const typeMatch = rest.match(/^\((\w+)\)\s+(.+)/);
        if (typeMatch) {
          citation.symbol = typeMatch[2].split(':')[0].trim();
        } else {
          citation.symbol = rest.split(':')[0].trim();
        }
      }

      // Check next line for preview (indented)
      if (i + 1 < lines.length && lines[i + 1].startsWith('   ')) {
        citation.preview = lines[i + 1].trim();
        i++; // Skip preview line
      }

      citations.push(citation);
    }
  }

  return citations;
}

/**
 * Custom renderer for smart_search tool
 * Shows compact citations format:
 * SmartSearch("query") -> N results
 *   src/auth/handler.go:45-78 handleAuth()
 */
function SmartSearchRenderer(props: ToolRendererProps): React.ReactNode {
  const { arguments: args, result, error, isExecuting } = props;
  const query = (args.query as string) || '';
  const displayQuery = query.length > 30 ? query.slice(0, 30) + '...' : query;

  // Header
  const header = React.createElement(
    Box,
    { key: 'header' },
    React.createElement(Text, { color: isExecuting ? 'gray' : 'green' }, '\u25cf'),
    React.createElement(Text, { color: 'white', bold: true }, ' SmartSearch'),
    React.createElement(Text, { color: 'gray' }, `("${displayQuery}")`)
  );

  if (isExecuting) {
    return React.createElement(Box, { flexDirection: 'column', marginBottom: 1 }, header);
  }

  if (error) {
    return React.createElement(
      Box,
      { flexDirection: 'column', marginBottom: 1 },
      header,
      React.createElement(
        Box,
        { marginLeft: 1, key: 'error' },
        React.createElement(Text, { color: 'red' }, '\u2514 ', error.slice(0, 50))
      )
    );
  }

  const citations = parseSmartSearchResult(result || '');

  if (citations.length === 0) {
    return React.createElement(
      Box,
      { flexDirection: 'column', marginBottom: 1 },
      header,
      React.createElement(
        Box,
        { marginLeft: 1, key: 'empty' },
        React.createElement(Text, { color: 'gray' }, '\u2514 no results')
      )
    );
  }

  // Show header with count, then first few citations
  const maxDisplay = 5;
  const displayCitations = citations.slice(0, maxDisplay);

  const elements: React.ReactNode[] = [
    React.createElement(
      Box,
      { key: 'header-with-count' },
      React.createElement(Text, { color: 'green' }, '\u25cf'),
      React.createElement(Text, { color: 'white', bold: true }, ' SmartSearch'),
      React.createElement(Text, { color: 'gray' }, `("${displayQuery}")`),
      React.createElement(Text, { color: 'cyan' }, ` \u2192 ${citations.length} result${citations.length !== 1 ? 's' : ''}`)
    ),
  ];

  // Add citations
  for (let i = 0; i < displayCitations.length; i++) {
    const c = displayCitations[i];
    const isLast = i === displayCitations.length - 1 && citations.length <= maxDisplay;
    const prefix = isLast ? '\u2514' : '\u251c';

    const symbolText = c.symbol ? ` ${c.symbol}()` : '';
    const sourceColor = c.source === 'vector' ? 'blue' : c.source === 'grep' ? 'yellow' : 'magenta';

    elements.push(
      React.createElement(
        Box,
        { marginLeft: 1, key: `citation-${i}` },
        React.createElement(Text, { color: 'gray' }, `${prefix} `),
        React.createElement(Text, { color: 'white' }, shortenPath(c.location)),
        React.createElement(Text, { color: sourceColor }, ` [${c.source}]`),
        React.createElement(Text, { color: 'cyan' }, symbolText)
      )
    );
  }

  // Show "and N more" if truncated
  if (citations.length > maxDisplay) {
    elements.push(
      React.createElement(
        Box,
        { marginLeft: 1, key: 'more' },
        React.createElement(Text, { color: 'gray' }, `\u2514 ...and ${citations.length - maxDisplay} more`)
      )
    );
  }

  return React.createElement(Box, { flexDirection: 'column', marginBottom: 1 }, ...elements);
}

/**
 * smart_search tool definition
 * - No executor (executes on server)
 * - Custom renderer for detailed citations display
 * - Combines vector search, grep search, and symbol search
 */
export const smartSearchToolDefinition: ToolDefinition = {
  name: 'smart_search',
  displayName: 'Smart Search',

  // Server-side execution - no client executor
  executor: undefined,

  // Custom renderer for citations display
  renderer: SmartSearchRenderer,

  // Render separately to show full citations
  renderSeparately: true,
};
