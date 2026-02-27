// ToolExecutorAdapter - adapts existing ToolExecutor to IToolExecutor interface
import path from 'path';
import { IToolExecutor, ToolExecutionResult } from '../../domain/ports/IToolExecutor.js';
import { ToolCallInfo } from '../../domain/entities/Message.js';
import { ToolExecutor, ToolExecutorOptions } from '../../tools/executor.js';
import { IToolRegistry } from '../../domain/ports/IToolRegistry.js';
import { PermissionService } from '../../application/services/PermissionService.js';
import { PermissionRequest } from '../../domain/permission/Permission.js';
import { getLogger } from '../../lib/logger.js';
import { debugLog } from '../../lib/debugLog.js';
import type { DiagnosticsService } from '../lsp/DiagnosticsService.js';
import type { LspService } from '../lsp/LspService.js';
import type { ShellSessionManager } from '../shell/ShellSessionManager.js';
import type { AskUserCallback } from '../../tools/askUser.js';

export interface ToolExecutorAdapterOptions {
  headlessMode?: boolean;
  askUserCallback?: AskUserCallback;
}

/**
 * Adapter that wraps the existing ToolExecutor to implement the
 * IToolExecutor port interface.
 *
 * Uses lazy initialization to avoid creating ToolExecutor (and its storeFactory)
 * until first tool execution. This prevents concurrent access issues with
 * BackgroundIndexer which also uses the store.
 *
 * Also consults IToolRegistry to handle server-side tools that shouldn't be
 * executed locally.
 *
 * Permission checks are applied before tool execution via PermissionService.
 */
export class ToolExecutorAdapter implements IToolExecutor {
  private executor: ToolExecutor | null = null;
  private projectRoot: string;
  private toolRegistry?: IToolRegistry;
  private permissionService: PermissionService;
  private diagnosticsService?: DiagnosticsService;
  private lspService?: LspService;
  private shellSessionManager?: ShellSessionManager;
  private adapterOptions?: ToolExecutorAdapterOptions;

  constructor(
    projectRoot: string,
    toolRegistry?: IToolRegistry,
    diagnosticsService?: DiagnosticsService,
    lspService?: LspService,
    shellSessionManager?: ShellSessionManager,
    adapterOptions?: ToolExecutorAdapterOptions,
  ) {
    this.projectRoot = projectRoot;
    this.toolRegistry = toolRegistry;
    this.permissionService = new PermissionService(projectRoot);
    this.diagnosticsService = diagnosticsService;
    this.lspService = lspService;
    this.shellSessionManager = shellSessionManager;
    this.adapterOptions = adapterOptions;
  }

  private getExecutor(): ToolExecutor {
    if (!this.executor) {
      this.executor = new ToolExecutor({
        projectRoot: this.projectRoot,
        headlessMode: this.adapterOptions?.headlessMode,
        askUserCallback: this.adapterOptions?.askUserCallback,
        lspService: this.lspService,
        shellSessionManager: this.shellSessionManager,
      });
    }
    return this.executor;
  }

  async execute(toolCall: ToolCallInfo): Promise<ToolExecutionResult> {
    const logger = getLogger();
    logger.info('ToolExecutorAdapter.execute start', { callId: toolCall.callId, toolName: toolCall.toolName });
    debugLog('ADAPTER', 'execute start', { callId: toolCall.callId, toolName: toolCall.toolName });

    // Check if this is a server-side tool (registered in ToolManager but no client executor).
    // A tool is server-side ONLY if ToolManager knows it AND ToolManager has no executor for it
    // AND ToolExecutor also doesn't have it. This prevents mistakenly treating proxied tools
    // (like lsp) as server-side when they have a client executor in ToolExecutor.
    if (this.toolRegistry?.has(toolCall.toolName)
        && !this.toolRegistry?.executesOnClient(toolCall.toolName)
        && !this.getExecutor().hasTool(toolCall.toolName)) {
      logger.info('ToolExecutorAdapter: server-side tool, skipping local execution', { toolName: toolCall.toolName });
      return {
        result: 'Server-side tool executed',
        error: undefined,
      };
    }

    // Check tool-type permission (read, edit, bash, list)
    const permRequest = buildPermissionRequest(toolCall);
    if (permRequest) {
      debugLog('ADAPTER', 'checking permission', { type: permRequest.type, value: permRequest.value?.slice(0, 80) });
      // Propagate agentId for auto-approval logic
      permRequest.agentId = toolCall.agentId;
      const permResult = await this.permissionService.check(permRequest);
      logger.info('Permission check result', {
        callId: toolCall.callId,
        toolName: toolCall.toolName,
        allowed: permResult.allowed,
        reason: permResult.reason,
      });
      debugLog('ADAPTER', 'permission result', { allowed: permResult.allowed, reason: permResult.reason });
      if (!permResult.allowed) {
        logger.warn('Tool blocked by permission', {
          tool: toolCall.toolName,
          reason: permResult.reason,
        });
        return {
          result: `[PERMISSION] ${permResult.reason}`,
          error: new Error(`PERMISSION_DENIED: ${permResult.reason}`),
        };
      }
    }

    debugLog('ADAPTER', 'calling executor', { toolName: toolCall.toolName });
    const result = await this.getExecutor().execute(
      toolCall.toolName,
      toolCall.arguments,
      { agentId: toolCall.agentId },
    );
    debugLog('ADAPTER', 'executor returned', { hasError: !!result.error, resultLen: result.result?.length });

    // Post-write LSP diagnostics
    if (!result.error && this.diagnosticsService) {
      const name = toolCall.toolName.toLowerCase();
      if (name === 'write_file' || name === 'edit_file') {
        const filePath = toolCall.arguments?.file_path || toolCall.arguments?.path;
        if (filePath) {
          const resolved = path.resolve(this.projectRoot, filePath);
          const diagnostics = await this.diagnosticsService.runAfterWrite(resolved);
          if (diagnostics) {
            return {
              result: result.result + diagnostics,
              error: result.error,
              summary: result.summary,
              diffLines: result.diffLines,
            };
          }
        }
      }
    }

    return {
      result: result.result,
      error: result.error,
      summary: result.summary,
      diffLines: result.diffLines,
    };
  }

  hasTool(name: string): boolean {
    return (this.toolRegistry?.has(name) ?? false) || this.getExecutor().hasTool(name);
  }

  listTools(): string[] {
    const executorTools = this.getExecutor().listTools();
    const registryTools = this.toolRegistry?.list() ?? [];
    return [...new Set([...executorTools, ...registryTools])];
  }
}

/**
 * Map tool call to a permission request.
 * Returns null for tools that don't need permission checks (search, etc.)
 */
function buildPermissionRequest(toolCall: ToolCallInfo): PermissionRequest | null {
  const name = toolCall.toolName.toLowerCase();
  let args = toolCall.arguments;

  // Unwrap _json wrapper (server sends complex args wrapped in _json field)
  if (args._json && typeof args._json === 'string') {
    try { args = JSON.parse(args._json); } catch { /* keep original */ }
  }

  // execute_command → bash permission (skip for bg_action management calls)
  if (name === 'execute_command') {
    if (args.bg_action) return null; // list/read/kill don't need permission
    return {
      type: 'bash',
      value: args.command || '',
    };
  }

  // read_file → read permission
  if (name === 'read_file') {
    return {
      type: 'read',
      value: args.file_path || args.path || '',
    };
  }

  // write_file, edit_file → edit permission
  if (name === 'write_file' || name === 'edit_file') {
    return {
      type: 'edit',
      value: args.file_path || args.path || '',
    };
  }

  // get_project_tree → list permission
  if (name === 'get_project_tree') {
    return {
      type: 'list',
      value: args.path || args.root || '',
    };
  }

  // No permission needed for search tools, etc.
  return null;
}
