import { createContainer, resetContainer } from '../config/container.js';
import { AppConfig } from '../config/index.js';
import { ConnectionStatus } from '../domain/ports/IStreamGateway.js';
import { toMessageViewModels } from '../presentation/mappers/MessageViewMapper.js';
import { filterMessagesForView } from '../presentation/mappers/MessageViewFilter.js';
import { getToolPrefix, getKeyArgument, formatResultSummary } from '../presentation/components/tools/formatToolDisplay.js';
import { parsePlanFromArgs, getStepStatusMarker, Plan } from '../domain/plan.js';
import { PermissionApproval, HeadlessBehavior } from '../infrastructure/permission/PermissionApproval.js';
import { VERSION } from '../version.js';
import { readTestingStrategy } from '../lib/testingStrategy.js';

export type Container = ReturnType<typeof createContainer>;

/**
 * Options for headless runner
 */
export interface HeadlessRunnerOptions {
  /** How to handle permission prompts: 'block' | 'allow-once' | 'allow-remember' */
  unknownCommandBehavior?: HeadlessBehavior;
}

/**
 * Base class for headless runners with common functionality.
 * Provides shared logic for container management, event subscriptions,
 * message handling, and output formatting.
 */
export abstract class BaseHeadlessRunner {
  protected config: AppConfig;
  protected options: HeadlessRunnerOptions;
  protected container: Container | null = null;
  protected unsubscribers: (() => void)[] = [];
  protected processedMessageIds = new Set<string>();
  private lastPlanHash: string = '';
  private lastAgentId: string = 'supervisor';

  constructor(config: AppConfig, options: HeadlessRunnerOptions = {}) {
    this.config = config;
    this.options = options;
    // Set headless mode for permission approval with specified behavior
    PermissionApproval.setHeadlessMode(options.unknownCommandBehavior || 'block');
  }

  /**
   * Create and initialize the container
   */
  protected initContainer(): Container {
    this.container = createContainer({
      projectRoot: this.config.projectRoot,
      serverAddress: this.config.serverAddress,
      projectKey: this.config.projectKey,
      sessionId: this.config.sessionId,
      headlessMode: true,
      bridgeAddress: this.config.bridgeAddress,
      bridgeEnabled: this.config.bridgeEnabled,
      bridgeAuthToken: this.config.bridgeAuthToken,
      serverId: this.config.serverId,
    });
    return this.container;
  }

  /**
   * Connect to the server
   */
  protected async connect(): Promise<void> {
    if (!this.container) {
      throw new Error('Container not initialized');
    }

    const testingStrategy = readTestingStrategy(this.config.projectRoot);

    await this.container.streamGateway.connect({
      serverAddress: this.config.serverAddress,
      sessionId: this.container.sessionId,
      userId: this.config.userId,
      projectKey: this.config.projectKey,
      projectRoot: this.config.projectRoot,
      clientVersion: VERSION,
      testingStrategy,
    });
  }

  /**
   * Setup common event subscriptions for tool execution display
   */
  protected setupToolEventSubscriptions(): void {
    if (!this.container) return;

    const { eventBus } = this.container;

    // Subscribe to errors
    this.unsubscribers.push(
      eventBus.subscribe('ErrorOccurred', (event) => {
        const errorMessage = event.error?.message || 'Unknown error';
        console.log(`\n⚠️ ${errorMessage}`);
      })
    );

    // Agent lifecycle events (spawned, completed, failed, restarted)
    this.unsubscribers.push(
      eventBus.subscribe('AgentLifecycle', (event) => {
        const shortId = event.agentId.replace('code-agent-', '');

        switch (event.lifecycleType) {
          case 'agent_spawned':
            console.log(`⊕ Code Agent [${shortId}] spawned: ${event.description || 'starting...'}`);
            break;
          case 'agent_completed': {
            // Parse enriched content: "Completed: <title>\n<truncated result>"
            const title = event.description.startsWith('Completed: ')
              ? event.description.split('\n')[0].replace('Completed: ', '')
              : event.description;
            console.log(`✓ Code Agent [${shortId}] completed: "${title}"`);
            break;
          }
          case 'agent_failed': {
            // Parse enriched content: "Failed: <title>\nReason: <reason>"
            const failTitle = event.description.startsWith('Failed: ')
              ? event.description.split('\n')[0].replace('Failed: ', '')
              : event.description;
            console.log(`✗ Code Agent [${shortId}] failed: "${failTitle}"`);
            break;
          }
          case 'agent_restarted':
            console.log(`↻ Code Agent [${shortId}] restarted: ${event.description}`);
            break;
          default:
            console.log(`[${event.lifecycleType}] ${shortId}: ${event.description}`);
        }
      })
    );

    this.unsubscribers.push(
      eventBus.subscribe('ToolExecutionStarted', (event) => {
        // Filter: only show supervisor tools (hide agent tools)
        const agentId = event.execution.agentId;
        if (agentId && agentId !== 'supervisor') {
          return; // Hide agent tools
        }

        if (event.execution.toolName === 'manage_plan') {
          // parsePlanFromArgs handles caching and filling missing descriptions
          const plan = parsePlanFromArgs(event.execution.arguments as Record<string, unknown>);
          if (plan) {
            // Deduplicate: only print plan if it changed
            const planHash = JSON.stringify(plan);
            if (planHash !== this.lastPlanHash) {
              this.lastPlanHash = planHash;
              this.printPlan(plan);
            }
            return;
          }
        }

        // Debug: show callId for tool call deduplication diagnostics
        if (this.config.debug) {
          console.error(`  [callId: ${event.execution.callId}]`);
        }

        const prefix = getToolPrefix(event.execution.toolName);
        const args = event.execution.arguments as Record<string, unknown>;

        // Make file paths clickable for file-related tools
        const isFileTool = ['read_file', 'read', 'write_file', 'write', 'edit_file', 'edit'].includes(
          event.execution.toolName.toLowerCase()
        );

        // Get the raw file path from arguments (before any shortening)
        const rawPath = (args.path || args.file_path || args.filepath || args.file) as string | undefined;

        let displayArg: string;
        if (isFileTool && rawPath && this.isFilePath(rawPath)) {
          // Use full path for link, shortened for display
          const fullPath = this.resolveToAbsolutePath(rawPath);
          const shortPath = this.shortenPath(rawPath);
          displayArg = this.makeClickableLink(fullPath, shortPath);
        } else {
          displayArg = getKeyArgument(event.execution.toolName, args);
        }

        const display = displayArg ? `● ${prefix}(${displayArg})` : `● ${prefix}`;
        console.log(display);
      })
    );

    this.unsubscribers.push(
      eventBus.subscribe('ToolExecutionCompleted', (event) => {
        // Filter: only show supervisor tools (hide agent tools)
        const agentId = event.execution.agentId;
        if (agentId && agentId !== 'supervisor') {
          return; // Hide agent tools
        }

        if (event.execution.toolName === 'manage_plan') {
          return;
        }

        // Special handling for smart_search - show detailed citations
        if (event.execution.toolName === 'smart_search' && event.execution.result) {
          this.printSmartSearchResults(event.execution.result);
          return;
        }

        // Special rendering for agent management tools
        if (['spawn_code_agent', 'manage_tasks', 'manage_subtasks'].includes(event.execution.toolName)) {
          const result = event.execution.result || '';
          const error = event.execution.error;

          // Error case
          if (result.startsWith('[ERROR]') || error) {
            const errorMsg = error || result.replace('[ERROR] ', '');
            // Truncate to first line
            const firstLine = errorMsg.split('\n')[0];
            console.log(` └ ✗ ${firstLine}`);
            return;
          }

          // Success: extract key info from result
          const firstLine = result.split('\n')[0];
          console.log(` └ ${firstLine}`);
          return;
        }

        const summary = formatResultSummary(
          event.execution.toolName,
          event.execution.result || '',
          event.execution.error,
          event.execution.summary
        );
        console.log(` └ ${summary}`);

        // Print diff lines if available
        const diffLines = event.execution.diffLines;
        if (diffLines && diffLines.length > 0) {
          for (const line of diffLines) {
            const prefix = line.type === ' ' ? '  ' : line.type + ' ';
            console.log(`   ${prefix}${line.content}`);
          }
        }
      })
    );
  }

  /**
   * Handle message completion - print new assistant messages
   */
  protected handleMessageCompleted(): void {
    if (!this.container) return;

    const messages = this.container.messageRepository.findComplete();
    const viewModels = toMessageViewModels(messages);
    const filtered = filterMessagesForView(viewModels, { type: 'supervisor' });

    for (const msg of filtered) {
      if (this.processedMessageIds.has(msg.id)) continue;
      this.processedMessageIds.add(msg.id);

      if (msg.role === 'assistant' && msg.content) {
        // Agent separators are now inserted as Messages by StreamProcessorService
        // (e.g., "─── Code Agent [abc]: Fix imports ───"), so printAgentPrefix
        // is no longer needed — it would duplicate the separator.

        if (msg.reasoning) {
          console.log(`[Thinking] ${msg.content}`);
        } else {
          console.log(msg.content);
        }
      }
    }
  }

  /**
   * Print agent prefix when switching between agents
   */
  protected printAgentPrefix(agentId: string): void {
    if (agentId === this.lastAgentId) return;
    this.lastAgentId = agentId;

    const label = agentId === 'supervisor'
      ? 'Supervisor'
      : `Code Agent ${agentId.replace('code-agent-', '')}`;
    console.log(`\n[Agent: ${label}]`);
  }

  /**
   * Cleanup resources
   */
  protected cleanup(): void {
    for (const unsub of this.unsubscribers) {
      unsub();
    }
    this.unsubscribers = [];
    resetContainer();
    this.container = null;
  }

  /**
   * Shorten a file path for display
   * Example: /services/StreamProcessorService.ts:60
   */
  protected shortenPath(path: string, maxLength: number = 50): string {
    if (path.length <= maxLength) return path;
    // Trim from the start, keep the end (file name is most important)
    const shortened = path.slice(-maxLength);
    // Find first path separator and cut there to start with full directory name
    const slashIdx = shortened.indexOf('/');
    const backslashIdx = shortened.indexOf('\\');
    const sepIndex = slashIdx >= 0 ? (backslashIdx >= 0 ? Math.min(slashIdx, backslashIdx) : slashIdx) : backslashIdx;
    if (sepIndex >= 0) {
      // Skip past the separator to avoid leading slash
      return shortened.slice(sepIndex + 1);
    }
    return shortened;
  }

  /**
   * Check if a string looks like a file path
   */
  protected isFilePath(value: string): boolean {
    return (
      value.includes('/') ||
      value.includes('\\') ||
      /\.[a-zA-Z0-9]{1,5}$/.test(value)
    );
  }

  /**
   * Resolve a relative path to absolute using projectRoot
   */
  protected resolveToAbsolutePath(path: string): string {
    const isWindows = process.platform === 'win32';

    // Check if already absolute
    // Windows: C:/ or C:\
    // Unix: /path (but NOT on Windows where /path is relative)
    const isWindowsAbsolute = /^[A-Za-z]:/.test(path);
    const isUnixAbsolute = !isWindows && path.startsWith('/');

    if (isWindowsAbsolute || isUnixAbsolute) {
      return path;
    }

    // Resolve relative to projectRoot
    const projectRoot = (this.config.projectRoot || process.cwd()).replace(/\\/g, '/');

    // Remove leading slash if present (common in vector search results)
    let cleanPath = path.startsWith('/') ? path.slice(1) : path;

    // Handle case where path starts with project directory name
    // e.g., projectRoot = ".../vector-cli-node" and path = "vector-cli-node/src/..."
    // This happens when vector search stores paths relative to parent directory
    const projectDirName = projectRoot.split('/').pop();
    if (projectDirName && cleanPath.startsWith(projectDirName + '/')) {
      cleanPath = cleanPath.slice(projectDirName.length + 1);
    }

    return `${projectRoot}/${cleanPath}`;
  }

  /**
   * Check if terminal supports hyperlinks (TTY and not piped)
   */
  protected supportsHyperlinks(): boolean {
    // Only use hyperlinks if stdout is a TTY (interactive terminal)
    // Piped output or redirected output should not include escape codes
    return process.stdout.isTTY === true;
  }

  /**
   * Create a clickable file link using OSC 8 hyperlinks
   * Format: \x1b]8;;file:///path\x07display_text\x1b]8;;\x07
   * Works in most modern terminals (Windows Terminal, iTerm2, etc.)
   * Falls back to plain text when not in a TTY
   */
  protected makeClickableLink(fullPath: string, displayText: string): string {
    // If not in TTY, return plain text to avoid escape codes in output
    if (!this.supportsHyperlinks()) {
      return displayText;
    }

    // Normalize path (forward slashes for URL)
    let normalizedPath = fullPath.replace(/\\/g, '/');

    // Extract line number if present (path:line or path:line-endline)
    const lineMatch = normalizedPath.match(/^(.+?):(\d+)(?:-\d+)?$/);
    const filePath = lineMatch ? lineMatch[1] : normalizedPath;
    const line = lineMatch ? lineMatch[2] : '';

    // Build file:// URL
    // Windows: file:///C:/path/file.ts
    // Unix: file:///path/file.ts
    let fileUrl: string;
    if (/^[A-Za-z]:/.test(filePath)) {
      // Windows absolute path: C:/path -> file:///C:/path
      fileUrl = `file:///${filePath}`;
    } else if (filePath.startsWith('/')) {
      // Unix absolute path: /path -> file:///path
      fileUrl = `file://${filePath}`;
    } else {
      // Relative path - shouldn't happen after resolve, but handle it
      fileUrl = `file:///${filePath}`;
    }

    // Append line number if present (some terminals support this)
    if (line) {
      fileUrl += `#L${line}`;  // GitHub-style line anchor
    }

    // OSC 8 hyperlink: \x1b]8;;URL\x07TEXT\x1b]8;;\x07
    return `\x1b]8;;${fileUrl}\x07${displayText}\x1b]8;;\x07`;
  }

  /**
   * Print smart_search results in detailed citation format
   */
  protected printSmartSearchResults(result: string): void {
    // Parse citations from result format:
    // "Found N results:\n\n1. path:start-end [source] (type) name\n   preview"
    const lines = result.split('\n');
    const citations: { location: string; source: string; symbol?: string; symbolType?: string }[] = [];

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      // Match: "1. path:start-end [source] ..." - use (.+?) for paths with special chars
      const match = line.match(/^\d+\.\s+(.+?)\s+\[(\w+)\](.*)$/);
      if (match) {
        const citation: { location: string; source: string; symbol: string; symbolType: string } = {
          location: match[1],
          source: match[2],
          symbol: '',
          symbolType: '',
        };
        // Extract symbol if present: (type) name or just name
        const rest = match[3].trim();
        if (rest) {
          const typeMatch = rest.match(/^\((\w+)\)\s+(.+)/);
          if (typeMatch) {
            citation.symbolType = typeMatch[1]; // class, function, method, interface, etc.
            citation.symbol = typeMatch[2].split(':')[0].trim();
          } else {
            citation.symbol = rest.split(':')[0].trim();
          }
        }
        citations.push(citation);
      }
    }

    if (citations.length === 0) {
      console.log(' └ no results');
      return;
    }

    // Print header with count
    console.log(` → ${citations.length} result${citations.length !== 1 ? 's' : ''}`);

    // Print up to 5 citations
    const maxDisplay = 5;
    const displayCitations = citations.slice(0, maxDisplay);

    for (let i = 0; i < displayCitations.length; i++) {
      const c = displayCitations[i];
      const isLast = i === displayCitations.length - 1 && citations.length <= maxDisplay;
      const prefix = isLast ? '└' : '├';

      // Only add () for function/method types, show symbol name as-is for others
      let symbolText = '';
      if (c.symbol) {
        const isFunctionLike = c.symbolType === 'function' || c.symbolType === 'method';
        symbolText = isFunctionLike ? ` ${c.symbol}()` : ` ${c.symbol}`;
      }

      // Make path clickable: resolve to absolute FIRST (before shortening)
      const fullPath = this.resolveToAbsolutePath(c.location);
      // Shorten for display (but link uses full path)
      const displayPath = this.shortenPath(c.location);
      const clickablePath = this.makeClickableLink(fullPath, displayPath);

      console.log(`   ${prefix} ${clickablePath} [${c.source}]${symbolText}`);
    }

    if (citations.length > maxDisplay) {
      console.log(`   └ ...and ${citations.length - maxDisplay} more`);
    }
  }

  /**
   * Print plan in formatted way
   */
  protected printPlan(plan: Plan): void {
    const hasInProgress = plan.steps.some(s => s.status === 'in_progress');
    const allComplete = plan.steps.every(s => s.status === 'completed');
    const planMarker = allComplete ? '✓' : hasInProgress ? '○' : ' ';

    console.log(`● [${planMarker}] Plan - ${plan.goal}`);

    for (const step of plan.steps) {
      const marker = getStepStatusMarker(step.status);
      console.log(`   └ [${marker}] step ${step.index + 1} - ${step.description}`);
    }
  }

  /**
   * Log status change in debug mode
   */
  protected logStatus(status: ConnectionStatus): void {
    if (this.config.debug) {
      console.error(`[Status] ${status}`);
    }
  }
}
