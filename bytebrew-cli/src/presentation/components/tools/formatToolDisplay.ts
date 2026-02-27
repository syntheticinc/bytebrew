// Helper functions for compact tool display

/**
 * Get a minimal prefix for the tool type
 */
export function getToolPrefix(toolName: string): string {
  const prefixMap: Record<string, string> = {
    // Search/find tools - specific prefixes for each
    smart_search: 'SmartSearch',
    search_code: 'VectorSearch',
    symbol_search: 'SymbolSearch',
    grep_search: 'GrepSearch',
    glob: 'Glob',
    search: 'Search',
    find: 'Find',
    grep: 'Grep',

    // File operations
    read_file: 'Read',
    read: 'Read',
    write_file: 'Write',
    write: 'Write',
    edit_file: 'Edit',
    edit: 'Edit',
    delete_file: 'Delete',

    // Directory operations
    list_dir: 'List',
    list_directory: 'List',
    create_dir: 'Mkdir',
    get_project_tree: 'Tree',

    // Execute/run
    execute: 'Exec',
    run: 'Run',
    shell: 'Shell',
    bash: 'Bash',
    command: 'Cmd',

    // Web/network
    web_search: 'WebSearch',
    web_fetch: 'WebFetch',
    fetch: 'Fetch',
    http: 'Http',
    api: 'Api',

    // Analysis
    analyze: 'Analyze',
    parse: 'Parse',
    lsp: 'LSP',

    // Planning
    manage_plan: 'Plan',
    plan: 'Plan',

    // Multi-agent
    manage_tasks: 'Tasks',
    manage_subtasks: 'Subtasks',
    spawn_code_agent: 'Agent',
    ask_user: 'AskUser',
  };

  const lowerName = toolName.toLowerCase();

  // Check for exact match first
  if (prefixMap[lowerName]) {
    return prefixMap[lowerName];
  }

  // Check for partial match
  for (const [key, prefix] of Object.entries(prefixMap)) {
    if (lowerName.includes(key)) {
      return prefix;
    }
  }

  // Return original name capitalized
  return toolName.charAt(0).toUpperCase() + toolName.slice(1);
}

/**
 * Extract the key argument to display for a tool
 */
export function getKeyArgument(
  toolName: string,
  args: Record<string, unknown>
): string {
  const lowerName = toolName.toLowerCase();

  // Action-based tools: show meaningful arg instead of "action" which duplicates summary
  const actionToolKeys: Record<string, string[]> = {
    'manage_tasks': ['title', 'task_id'],
    'manage_subtasks': ['title', 'subtask_id', 'task_id'],
    'spawn_code_agent': ['subtask_id', 'agent_id'],
  };

  const overrideKeys = actionToolKeys[lowerName];
  if (overrideKeys) {
    for (const key of overrideKeys) {
      if (args[key] !== undefined) {
        return formatValue(args[key], false);
      }
    }
    return ''; // No meaningful arg → show nothing
  }

  // Priority order of common argument names
  const priorityKeys = [
    'query',
    'symbol_name',
    'pattern',
    'path',
    'file',
    'file_path',
    'filepath',
    'filename',
    'content',
    'command',
    'cmd',
    'text',
    'message',
    'question',
    'action',
    'title',
    'task_id',
    'subtask_id',
    'name',
    'url',
    'project_key',
  ];

  // For execute_command, command should not be treated as a path
  const isCommandTool = lowerName.includes('execute') ||
                        lowerName.includes('command') ||
                        lowerName.includes('shell') ||
                        lowerName.includes('bash');

  // Find first matching key
  for (const key of priorityKeys) {
    if (args[key] !== undefined) {
      const value = args[key];
      // Don't apply path truncation to command arguments
      const treatAsPath = !isCommandTool || (key !== 'command' && key !== 'cmd');
      return formatValue(value, treatAsPath);
    }
  }

  // If no priority key found, use first argument
  const entries = Object.entries(args);
  if (entries.length > 0) {
    return formatValue(entries[0][1], !isCommandTool);
  }

  return '';
}

/**
 * Check if a string looks like a file path
 */
function isFilePath(value: string): boolean {
  return (
    value.includes('/') ||
    value.includes('\\') ||
    /\.[a-zA-Z0-9]{1,5}$/.test(value)
  );
}

/**
 * Format a value for display
 * @param treatAsPath - if true, apply path truncation logic (trim from start)
 */
function formatValue(value: unknown, treatAsPath: boolean = true): string {
  if (typeof value === 'string') {
    const maxLen = 45;
    if (value.length > maxLen) {
      // For file paths, trim from the start (end is more important)
      if (treatAsPath && isFilePath(value)) {
        const shortened = value.slice(-maxLen);
        // Find first path separator and cut there to start with full directory name
        const slashIdx = shortened.indexOf('/');
        const backslashIdx = shortened.indexOf('\\');
        const sepIndex = slashIdx >= 0 ? (backslashIdx >= 0 ? Math.min(slashIdx, backslashIdx) : slashIdx) : backslashIdx;
        if (sepIndex > 0) {
          // Return from separator onwards: /services/file.ts
          return shortened.slice(sepIndex);
        }
        return shortened;
      }
      // For other strings (including commands), trim from the end
      return `${value.slice(0, maxLen)}...`;
    }
    return value;
  }
  if (typeof value === 'number' || typeof value === 'boolean') {
    return String(value);
  }
  if (Array.isArray(value)) {
    return `[${value.length} items]`;
  }
  if (typeof value === 'object' && value !== null) {
    return '{...}';
  }
  return String(value);
}

/**
 * Format a result summary for display - MUST be deterministic (no re-computation)
 */
export function formatResultSummary(
  toolName: string,
  result: string,
  error?: string,
  summary?: string  // Server-provided summary — used for server-side tools
): string {
  if (error) {
    const shortError = error.length > 40 ? error.slice(0, 40) + '...' : error;
    return shortError;
  }
  // Use server-provided summary if available (for server-side tools)
  if (summary) {
    return summary;
  }

  // Tools that return errors as result text (not via error field)
  if (result.startsWith('[ERROR]') || result.startsWith('[SECURITY]') || result.startsWith('[CANCELLED]')) {
    const msg = result.replace(/^\[(ERROR|SECURITY|CANCELLED)\]\s*/, '');
    return msg.length > 45 ? msg.slice(0, 45) + '...' : msg;
  }

  const lowerName = toolName.toLowerCase();

  // manage_tasks — contextual summary
  if (lowerName === 'manage_tasks') {
    if (result.includes('Task created')) {
      const titleMatch = result.match(/Title: (.+)/);
      if (titleMatch) return `created: ${titleMatch[1].slice(0, 35)}`;
      return 'created';
    }
    if (result.includes('No tasks')) return 'no tasks';
    if (result.includes('approved')) return 'approved';
    if (result.includes('started')) return 'started';
    if (result.includes('completed')) return 'completed';
    if (result.includes('failed')) return 'failed';
    if (result.includes('Tasks (')) {
      const countMatch = result.match(/Tasks \((\d+)\)/);
      if (countMatch) return `${countMatch[1]} tasks`;
    }
    return 'done';
  }

  // manage_subtasks — contextual summary
  if (lowerName === 'manage_subtasks') {
    if (result.includes('Subtask created')) {
      const idMatch = result.match(/ID: (\S+)/);
      const titleMatch = result.match(/Title: (.+)/);
      const shortId = idMatch ? idMatch[1].slice(0, 8) : '';
      const title = titleMatch ? titleMatch[1] : '';
      if (title) return `created ${shortId} (${title})`;
      if (shortId) return `created ${shortId}`;
      return 'created';
    }
    if (result.includes('Subtasks (')) {
      const countMatch = result.match(/Subtasks \((\d+)\)/);
      if (countMatch) return `${countMatch[1]} subtasks`;
    }
    if (result.includes('completed')) return 'completed';
    if (result.includes('failed')) return 'failed';
    if (result.includes('No subtasks')) return 'no subtasks';
    if (result.includes('Ready subtasks')) {
      const countMatch = result.match(/Ready subtasks \((\d+)\)/);
      if (countMatch) return `${countMatch[1]} ready`;
      return 'ready';
    }
    // Subtask details (get action)
    if (result.includes('Subtask:') && result.includes('Status:')) {
      const statusMatch = result.match(/Status: (\w+)/);
      const titleMatch = result.match(/Title: (.+)/);
      if (statusMatch && titleMatch) return `${statusMatch[1]}: ${titleMatch[1].slice(0, 30)}`;
      if (statusMatch) return statusMatch[1];
    }
    return 'done';
  }

  // spawn_code_agent — agent status
  if (lowerName === 'spawn_code_agent') {
    if (result.includes('spawned')) {
      const idMatch = result.match(/Agent ID: (\S+)/);
      if (idMatch) return `spawned ${idMatch[1]}`;
      return 'spawned';
    }
    if (result.includes('Status: ')) {
      const statusMatch = result.match(/Status: (\w+)/);
      if (statusMatch) return statusMatch[1];
    }
    if (result.includes('Agents (')) {
      const countMatch = result.match(/Agents \((\d+)\)/);
      if (countMatch) return `${countMatch[1]} agents`;
    }
    if (result.includes('stopped')) return 'stopped';
    if (result.includes('restarted') || result.includes('Restarted')) return 'restarted';
    return 'done';
  }

  // ask_user — show response
  if (lowerName === 'ask_user') {
    if (result.length <= 30) return result;
    return result.slice(0, 30) + '...';
  }

  // Web search - count numbered results in markdown output
  if (lowerName === 'web_search') {
    if (result.startsWith('No results found')) {
      return '0 results';
    }
    const matches = result.match(/^\d+\./gm);
    if (matches) {
      return `${matches.length} result${matches.length !== 1 ? 's' : ''}`;
    }
    return 'done';
  }

  // Glob - count files (one per line)
  if (lowerName === 'glob') {
    if (result.startsWith('[ERROR]')) {
      return 'error';
    }
    const files = result.split('\n').filter(l => l.trim()).length;
    return `${files} file${files !== 1 ? 's' : ''}`;
  }

  // Search results - count items
  if (lowerName.includes('search') || lowerName.includes('find') || lowerName.includes('grep')) {
    const count = countResults(result);
    if (count !== null) {
      return `${count} result${count !== 1 ? 's' : ''}`;
    }
    return 'done';
  }

  // Web fetch - show line count
  if (lowerName === 'web_fetch') {
    if (result.startsWith('[ERROR]')) {
      return 'error';
    }
    const lines = result.split('\n').length;
    return `${lines} line${lines !== 1 ? 's' : ''}`;
  }

  // File read - show line count
  if (lowerName.includes('read')) {
    // Check for error marker — only check prefix, NOT file content
    // File content may contain strings like "not found" in source code
    if (result.startsWith('[ERROR]') || result.startsWith('[PERMISSION]')) {
      return 'error';
    }
    const lines = result.split('\n').length;
    return `${lines} line${lines !== 1 ? 's' : ''}`;
  }

  // File write/edit — parse result from tool (e.g. "File written: path (5 lines)", "Edit applied: path (+2 lines)")
  if (lowerName.includes('write') || lowerName.includes('edit')) {
    if (result.startsWith('[ERROR]') || result.startsWith('[PERMISSION]')) {
      return result.length > 40 ? result.slice(0, 40) + '...' : result;
    }
    const linesMatch = result.match(/\(([^)]+lines?)\)/);
    if (linesMatch) {
      return linesMatch[1];
    }
    return 'saved';
  }

  // Tree/list directory
  if (lowerName.includes('tree') || lowerName.includes('list') || lowerName.includes('dir')) {
    // Count actual tree entries (lines with ├── or └──), excluding depth limit markers
    const treeLines = result.split('\n').filter(l => {
      const trimmed = l.trim();
      if (!trimmed) return false;
      if (trimmed.includes('(depth limit reached)')) return false;
      return trimmed.includes('├──') || trimmed.includes('└──');
    });
    if (treeLines.length > 0) {
      return `${treeLines.length} item${treeLines.length !== 1 ? 's' : ''}`;
    }
    // JSON tree output — count nodes by "name" fields
    const jsonCount = countJsonTreeNodes(result);
    if (jsonCount > 0) {
      return `${jsonCount} item${jsonCount !== 1 ? 's' : ''}`;
    }
    // Fallback: non-empty lines
    const lineCount = result.split('\n').filter(l => l.trim()).length;
    return `${lineCount} item${lineCount !== 1 ? 's' : ''}`;
  }

  // Command execution
  if (lowerName.includes('execute') || lowerName.includes('shell') ||
      lowerName.includes('bash') || lowerName.includes('command') || lowerName.includes('run')) {
    // Check for error indicators first
    if (result.includes('[Command timed out]')) {
      return 'timeout';
    }
    // Check for non-zero exit code: [Exit code: N] where N != 0
    const exitCodeMatch = result.match(/\[Exit code: (\d+)\]/);
    if (exitCodeMatch && exitCodeMatch[1] !== '0') {
      return `exit ${exitCodeMatch[1]}`;
    }
    // Check for error markers
    if (result.startsWith('[ERROR]') || result.includes('error:') || result.includes('Error:')) {
      return 'error';
    }
    const lines = result.split('\n').filter(l => l.trim()).length;
    if (lines > 0) {
      return `${lines} line${lines !== 1 ? 's' : ''}`;
    }
    return 'done';
  }

  // Default - show line count or done
  const lines = result.split('\n').filter(l => l.trim()).length;
  if (lines > 1) {
    return `${lines} lines`;
  }

  return 'done';
}

/**
 * Count nodes in a JSON tree structure (recursive).
 * Counts children nodes only (excludes root).
 */
function countJsonTreeNodes(result: string): number {
  try {
    const tree = JSON.parse(result);
    return countChildren(tree);
  } catch {
    return 0;
  }
}

function countChildren(node: any): number {
  if (!node || !node.children || !Array.isArray(node.children)) {
    return 0;
  }
  let count = node.children.length;
  for (const child of node.children) {
    count += countChildren(child);
  }
  return count;
}

/**
 * Try to count results in JSON or newline-separated output
 */
function countResults(result: string): number | null {
  try {
    const parsed = JSON.parse(result);
    if (Array.isArray(parsed)) {
      return parsed.length;
    }
    if (parsed.results && Array.isArray(parsed.results)) {
      return parsed.results.length;
    }
    if (parsed.matches && Array.isArray(parsed.matches)) {
      return parsed.matches.length;
    }
    if (parsed.items && Array.isArray(parsed.items)) {
      return parsed.items.length;
    }
    if (typeof parsed.count === 'number') {
      return parsed.count;
    }
  } catch {
    // Not JSON - count non-empty lines
    const lines = result.split('\n').filter(l => l.trim());
    if (lines.length > 0 && lines.length < 1000) {
      return lines.length;
    }
  }
  return null;
}
