// Register all tool definitions
// Called once at app startup

import { ToolManager } from '../ToolManager.js';
import { planToolDefinition } from './planTool.js';
import { smartSearchToolDefinition } from './smartSearchTool.js';
import { webSearchToolDefinition } from './webSearchTool.js';
import { webFetchToolDefinition } from './webFetchTool.js';

/**
 * Register all tool definitions in ToolManager
 *
 * To add a new tool:
 * 1. Create definition file in tools/definitions/
 * 2. Import and add to registerToolDefinitions()
 */
export function registerToolDefinitions(): void {
  // Server-side tools with custom rendering
  ToolManager.register(planToolDefinition);

  // Server-side tools with standard rendering
  ToolManager.register(smartSearchToolDefinition);
  ToolManager.register(webSearchToolDefinition);
  ToolManager.register(webFetchToolDefinition);

  // Multi-agent tools (manage_tasks, manage_subtasks, spawn_code_agent are server-side)
  ToolManager.register({ name: 'manage_tasks', displayName: 'Tasks' });
  ToolManager.register({ name: 'manage_subtasks', displayName: 'Subtasks' });
  ToolManager.register({ name: 'spawn_code_agent', displayName: 'Agent' });
  ToolManager.register({ name: 'wait', displayName: 'Wait' });
  ToolManager.register({ name: 'lsp', displayName: 'LSP' });
}
