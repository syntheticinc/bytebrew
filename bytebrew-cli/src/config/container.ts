// DI Container - dependency injection setup for the application
import path from 'path';
import { v4 as uuidv4 } from 'uuid';
import { getStoreFactory, ChunkStoreFactory } from '../indexing/storeFactory.js';

// Domain ports
import { IMessageRepository } from '../domain/ports/IMessageRepository.js';
import { IStreamGateway } from '../domain/ports/IStreamGateway.js';
import { IToolExecutor } from '../domain/ports/IToolExecutor.js';
import { IEventBus } from '../domain/ports/IEventBus.js';
import { IToolRenderingService } from '../domain/ports/IToolRenderingService.js';
import { IChunkStore, IEmbeddingsClient } from '../domain/store.js';

// Application services
import { MessageAccumulatorService } from '../application/services/MessageAccumulatorService.js';
import { StreamProcessorService } from '../application/services/StreamProcessorService.js';

// Infrastructure implementations
import { InMemoryMessageRepository } from '../infrastructure/persistence/InMemoryMessageRepository.js';
import { SimpleEventBus } from '../infrastructure/events/SimpleEventBus.js';
import { GrpcStreamGateway } from '../infrastructure/grpc/GrpcStreamGateway.js';
import { getLogger } from '../lib/logger.js';
import { ToolExecutorAdapter } from '../infrastructure/tools/ToolExecutorAdapter.js';
import { DiagnosticsService } from '../infrastructure/lsp/DiagnosticsService.js';
import { LspManager } from '../infrastructure/lsp/LspManager.js';
import { LspService } from '../infrastructure/lsp/LspService.js';
import { AgentStateManager } from '../infrastructure/state/AgentStateManager.js';
import { ShellSessionManager } from '../infrastructure/shell/ShellSessionManager.js';
import type { AskUserCallback } from '../tools/askUser.js';
import { MobileProxyServer } from '../infrastructure/mobile/MobileProxyServer.js';

// Tools layer (singleton)
import { ToolManager } from '../tools/ToolManager.js';
import { initDebugLog } from '../lib/debugLog.js';

export interface ContainerConfig {
  projectRoot: string;
  serverAddress: string;
  projectKey: string;
  sessionId?: string; // Optional: reuse specific session, otherwise generate new
  headlessMode?: boolean;
  askUserCallback?: AskUserCallback;
  /** Port for WebSocket proxy to mobile clients. */
  mobileProxyPort?: number;
  /** Disable LSP server spawning (for tests that don't need real LSP servers). */
  disableLspServers?: boolean;
  overrides?: {
    streamGateway?: IStreamGateway;
    toolExecutor?: IToolExecutor;
  };
}

/**
 * Application container that manages dependency injection.
 * Creates and wires up all the components of the clean architecture.
 */
export class Container {
  private _sessionId: string;
  private _config: ContainerConfig;

  // Infrastructure layer
  private _eventBus: IEventBus;
  private _messageRepository: IMessageRepository;
  private _streamGateway: IStreamGateway;
  private _toolExecutor: IToolExecutor;
  private _lspManager: LspManager;
  private _diagnosticsService: DiagnosticsService;
  private _lspService: LspService;
  private _agentStateManager: AgentStateManager;
  private _shellSessionManager: ShellSessionManager;
  private _storeFactory: ChunkStoreFactory;
  private _chunkStore: IChunkStore | null = null;
  private _embeddingsClient: IEmbeddingsClient | null = null;
  private _mobileProxy: MobileProxyServer | null = null;

  // Application layer
  private _accumulator: MessageAccumulatorService;
  private _streamProcessor: StreamProcessorService;

  private _initialized = false;

  constructor(config: ContainerConfig) {
    this._config = config;
    this._sessionId = config.sessionId ?? uuidv4();

    // Initialize file-based debug logging (no-op unless BYTEBREW_DEBUG_LOG=1)
    initDebugLog(config.projectRoot);

    // Create infrastructure layer
    this._eventBus = new SimpleEventBus();
    this._messageRepository = new InMemoryMessageRepository();
    this._streamGateway = config.overrides?.streamGateway ?? new GrpcStreamGateway();
    this._lspManager = new LspManager(config.projectRoot, config.disableLspServers ? [] : undefined);
    this._diagnosticsService = new DiagnosticsService(this._lspManager);
    this._lspService = new LspService(this._lspManager);
    this._shellSessionManager = new ShellSessionManager();
    this._toolExecutor = config.overrides?.toolExecutor ?? new ToolExecutorAdapter(
      config.projectRoot,
      ToolManager,
      this._diagnosticsService,
      this._lspService,
      this._shellSessionManager,
      {
        headlessMode: config.headlessMode,
        askUserCallback: config.askUserCallback,
      },
    );
    this._agentStateManager = new AgentStateManager();
    this._storeFactory = getStoreFactory(config.projectRoot);

    // Create application layer
    this._accumulator = new MessageAccumulatorService();
    this._streamProcessor = new StreamProcessorService({
      streamGateway: this._streamGateway,
      messageRepository: this._messageRepository,
      toolExecutor: this._toolExecutor,
      accumulator: this._accumulator,
      eventBus: this._eventBus,
      agentStateManager: this._agentStateManager,
    });
  }

  /**
   * Initialize the container - must be called before using services
   */
  initialize(): void {
    if (this._initialized) return;

    this._streamProcessor.initialize();

    // Fire-and-forget: pre-spawn LSP servers so they're warm for first write/edit
    void this._diagnosticsService.warmup();
    // Pre-warm metadata index (fire-and-forget) and populate chunk store
    void this._storeFactory.getStore().then(store => {
      this._chunkStore = store;
      this._embeddingsClient = this._storeFactory.getEmbeddings();
    }).catch((error) => {
      getLogger().error('Store initialization failed', { error: error?.message || error });
    });

    // Start mobile proxy if port is configured
    if (this._config.mobileProxyPort) {
      this._mobileProxy = new MobileProxyServer(
        this._messageRepository,
        this._eventBus,
        {
          projectName: path.basename(this._config.projectRoot),
          projectPath: this._config.projectRoot,
          sessionId: this._sessionId,
        },
        this._streamProcessor,
      );
      try {
        this._mobileProxy.start(this._config.mobileProxyPort);
      } catch (error) {
        getLogger().error('Failed to start mobile proxy', {
          port: this._config.mobileProxyPort,
          error: (error as Error).message,
        });
        this._mobileProxy = null;
      }
    }

    this._initialized = true;
  }

  /**
   * Synchronously close the SQLite database to release WAL file locks.
   * Must be called BEFORE process.exit() to prevent stale locks on next launch.
   * Safe to call multiple times.
   */
  closeDatabaseSync(): void {
    if (this._chunkStore && 'close' in this._chunkStore) {
      try {
        (this._chunkStore as { close(): void }).close();
      } catch {
        // Already closed or failed — ignore
      }
    }
    this._chunkStore = null;
    this._embeddingsClient = null;
  }

  /**
   * Dispose of all resources
   */
  async dispose(): Promise<void> {
    this._mobileProxy?.stop();
    this._mobileProxy = null;
    this._streamProcessor.dispose();
    this._streamGateway.disconnect();
    // Close SQLite FIRST (synchronous) — before slow async operations
    this.closeDatabaseSync();
    await this._diagnosticsService.dispose();
    await this._shellSessionManager.disposeAll();
    this._eventBus.clear();
    this._initialized = false;
  }

  // Getters for all components

  get sessionId(): string {
    return this._sessionId;
  }

  get config(): ContainerConfig {
    return this._config;
  }

  get eventBus(): IEventBus {
    return this._eventBus;
  }

  get messageRepository(): IMessageRepository {
    return this._messageRepository;
  }

  get streamGateway(): IStreamGateway {
    return this._streamGateway;
  }

  get toolExecutor(): IToolExecutor {
    return this._toolExecutor;
  }

  get accumulator(): MessageAccumulatorService {
    return this._accumulator;
  }

  get streamProcessor(): StreamProcessorService {
    return this._streamProcessor;
  }

  get toolRenderingService(): IToolRenderingService {
    return ToolManager;
  }

  get agentStateManager(): AgentStateManager {
    return this._agentStateManager;
  }

  get chunkStore(): IChunkStore | null {
    return this._chunkStore;
  }

  get embeddingsClient(): IEmbeddingsClient | null {
    return this._embeddingsClient;
  }

  get shellSessionManager(): ShellSessionManager {
    return this._shellSessionManager;
  }
}

// Singleton container instance
let containerInstance: Container | null = null;

/**
 * Create and configure the application container
 */
export function createContainer(config: ContainerConfig): Container {
  if (containerInstance) {
    // Fire-and-forget: LSP shutdown is best-effort
    void containerInstance.dispose();
  }
  containerInstance = new Container(config);
  containerInstance.initialize();
  return containerInstance;
}

/**
 * Get the current container instance
 */
export function getContainer(): Container {
  if (!containerInstance) {
    throw new Error('Container not initialized. Call createContainer first.');
  }
  return containerInstance;
}

/**
 * Reset the container (for testing)
 */
export function resetContainer(): void {
  if (containerInstance) {
    // Fire-and-forget: LSP shutdown is best-effort
    void containerInstance.dispose();
    containerInstance = null;
  }
}
