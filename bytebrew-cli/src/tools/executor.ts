// Tool executor - dispatches tool calls to appropriate handlers
import { ToolRegistry, ToolResult, Tool } from './registry.js';
import { ReadFileTool } from './readFile.js';
import { ProjectTreeTool } from './projectTree.js';
import { SearchCodebaseTool } from './searchCodebase.js';
import { GetFunctionTool } from './getFunction.js';
import { GetClassTool } from './getClass.js';
import { GetFileStructureTool } from './getFileStructure.js';
import { GrepSearchTool } from './grepSearch.js';
import { GlobSearchTool } from './globSearch.js';
import { SymbolSearchTool } from './symbolSearch.js';
import { LspTool } from './lspTool.js';
import { ExecuteCommandTool } from './executeCommand.js';
import type { LspService } from '../infrastructure/lsp/LspService.js';
import { WriteFileTool } from './writeFile.js';
import { EditFileTool } from './editFile.js';
import { AskUserTool, AskUserCallback } from './askUser.js';
import { IChunkStoreFactory } from '../domain/store.js';
import { getStoreFactory } from '../indexing/storeFactory.js';
import { FileIgnoreFactory } from '../infrastructure/file-ignore/FileIgnoreFactory.js';
import type { ShellSessionManager } from '../infrastructure/shell/ShellSessionManager.js';
import { getLogger } from '../lib/logger.js';

export interface ToolExecutionContext {
  agentId?: string;
}

export interface ToolExecutorOptions {
  projectRoot: string;
  storeFactory?: IChunkStoreFactory;
  additionalTools?: Tool[];
  headlessMode?: boolean;
  askUserCallback?: AskUserCallback;
  lspService?: LspService;
  shellSessionManager?: ShellSessionManager;
}

/**
 * Tool executor with plugin pattern for Open/Closed principle compliance.
 * Default tools are registered automatically, additional tools can be passed via options.
 */
export class ToolExecutor {
  private registry: ToolRegistry;
  private projectRoot: string;
  private storeFactory: IChunkStoreFactory;
  private asyncInitPromise: Promise<void> | null = null;
  private headlessMode: boolean;
  private askUserCallback: AskUserCallback | undefined;
  private lspService: LspService | undefined;
  private shellSessionManager: ShellSessionManager | undefined;

  constructor(projectRoot: string);
  constructor(options: ToolExecutorOptions);
  constructor(arg: string | ToolExecutorOptions) {
    if (typeof arg === 'string') {
      this.projectRoot = arg;
      this.storeFactory = getStoreFactory(arg);
      this.headlessMode = false;
      this.askUserCallback = undefined;
      this.shellSessionManager = undefined;
      this.registry = new ToolRegistry();
      this.registerDefaultTools();
    } else {
      this.projectRoot = arg.projectRoot;
      this.storeFactory = arg.storeFactory || getStoreFactory(arg.projectRoot);
      this.headlessMode = arg.headlessMode || false;
      this.askUserCallback = arg.askUserCallback;
      this.lspService = arg.lspService;
      this.shellSessionManager = arg.shellSessionManager;
      this.registry = new ToolRegistry();
      this.registerDefaultTools();

      // Register additional tools (Open for extension)
      if (arg.additionalTools) {
        for (const tool of arg.additionalTools) {
          this.registry.register(tool);
        }
      }
    }
  }

  private registerDefaultTools(): void {
    const logger = getLogger();
    logger.debug('Registering default tools');

    // File tools
    this.registry.register(new ReadFileTool(this.projectRoot));
    // ProjectTreeTool registered via async init (needs FileIgnore)

    // Indexing tools (semantic search via USearch + SQLite)
    // search_code is the name expected by the server
    this.registry.register(new SearchCodebaseTool({
      storeFactory: this.storeFactory,
      projectRoot: this.projectRoot,
      name: 'search_code',
    }));
    this.registry.register(new GetFunctionTool(this.storeFactory));
    this.registry.register(new GetClassTool(this.storeFactory));
    this.registry.register(new GetFileStructureTool(this.storeFactory));

    // Search tools for smart search
    this.registry.register(new GrepSearchTool({
      projectRoot: this.projectRoot,
    }));
    this.registry.register(new GlobSearchTool({
      projectRoot: this.projectRoot,
    }));
    this.registry.register(new SymbolSearchTool({
      storeFactory: this.storeFactory,
      projectRoot: this.projectRoot,
    }));

    if (this.lspService) {
      this.registry.register(new LspTool({
        lspService: this.lspService,
        storeFactory: this.storeFactory,
        projectRoot: this.projectRoot,
      }));
    }

    // Shell execution tool
    if (this.shellSessionManager) {
      this.registry.register(new ExecuteCommandTool(this.projectRoot, this.shellSessionManager));
    } else {
      // Fallback for tests that don't pass shellSessionManager
      this.registry.register(new ExecuteCommandTool(this.projectRoot));
    }

    // File modification tools
    this.registry.register(new WriteFileTool(this.projectRoot));
    this.registry.register(new EditFileTool(this.projectRoot));

    // Multi-agent tools
    this.registry.register(new AskUserTool(
      this.headlessMode,
      this.askUserCallback,
    ));
  }

  /**
   * Lazy async initialization for tools that need async setup (FileIgnore).
   * Called once on first execute, subsequent calls return cached promise.
   */
  private ensureAsyncInit(): Promise<void> {
    if (!this.asyncInitPromise) {
      this.asyncInitPromise = this.initAsyncTools();
    }
    return this.asyncInitPromise;
  }

  private async initAsyncTools(): Promise<void> {
    const logger = getLogger();
    try {
      const fileIgnore = await FileIgnoreFactory.create(this.projectRoot);
      this.registry.register(new ProjectTreeTool(this.projectRoot, fileIgnore));
      logger.debug('Async tools initialized (FileIgnore loaded)');
    } catch (error: any) {
      logger.error('Failed to init async tools', { error: error.message });
      // Fallback: register ProjectTreeTool with default FileIgnore (no gitignore)
      const { FileIgnore } = await import('../domain/file-ignore/FileIgnore.js');
      this.registry.register(new ProjectTreeTool(this.projectRoot, new FileIgnore()));
    }
  }

  /**
   * Register a custom tool (Open/Closed principle - open for extension)
   */
  registerTool(tool: Tool): void {
    this.registry.register(tool);
  }

  async execute(toolName: string, args: Record<string, string>, context?: ToolExecutionContext): Promise<ToolResult> {
    const logger = getLogger();

    // Ensure async tools are initialized before execution
    await this.ensureAsyncInit();

    const tool = this.registry.get(toolName);

    if (!tool) {
      logger.warn('Unknown tool requested', { toolName });
      return {
        result: '',
        error: new Error(`Unknown tool: ${toolName}`),
      };
    }

    logger.debug('Executing tool', { toolName, args: Object.keys(args) });

    // Inject agentId for tools that declare needsContext (e.g. ExecuteCommandTool for session pool)
    const finalArgs = (tool.needsContext && context?.agentId)
      ? { ...args, _agent_id: context.agentId }
      : args;

    try {
      const result = await tool.execute(finalArgs);
      logger.debug('Tool execution completed', { toolName, hasError: !!result.error });
      return result;
    } catch (error) {
      logger.error('Tool execution failed', { toolName, error: (error as Error).message });
      return {
        result: '',
        error: error as Error,
      };
    }
  }

  hasTool(name: string): boolean {
    return this.registry.has(name);
  }

  listTools(): string[] {
    return this.registry.list();
  }
}
