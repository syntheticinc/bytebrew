// Web search tool definition - web_search
// Server-side tool for searching the web via Tavily API

import { ToolDefinition } from '../ToolManager.js';

/**
 * web_search tool definition
 * - No executor (executes on server via Tavily API)
 * - Standard rendering (uses formatToolDisplay helpers)
 */
export const webSearchToolDefinition: ToolDefinition = {
  name: 'web_search',
  displayName: 'Web Search',
  executor: undefined,
};
