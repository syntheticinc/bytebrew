// Web fetch tool definition - web_fetch
// Server-side tool for fetching web page content via Tavily API

import { ToolDefinition } from '../ToolManager.js';

/**
 * web_fetch tool definition
 * - No executor (executes on server via Tavily API)
 * - Standard rendering (uses formatToolDisplay helpers)
 */
export const webFetchToolDefinition: ToolDefinition = {
  name: 'web_fetch',
  displayName: 'Web Fetch',
  executor: undefined,
};
