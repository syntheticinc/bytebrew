// DI Container - dependency injection setup for the application
import { v4 as uuidv4 } from 'uuid';

// Domain ports
import { IMessageRepository } from '../domain/ports/IMessageRepository.js';
import { IStreamGateway } from '../domain/ports/IStreamGateway.js';
import { IEventBus } from '../domain/ports/IEventBus.js';
import { IToolRenderingService } from '../domain/ports/IToolRenderingService.js';

// Application services
import { MessageAccumulatorService } from '../application/services/MessageAccumulatorService.js';
import { StreamProcessorService } from '../application/services/StreamProcessorService.js';

// Infrastructure implementations
import { InMemoryMessageRepository } from '../infrastructure/persistence/InMemoryMessageRepository.js';
import { SimpleEventBus } from '../infrastructure/events/SimpleEventBus.js';
import { WsStreamGateway } from '../infrastructure/ws/WsStreamGateway.js';
import { AgentStateManager } from '../infrastructure/state/AgentStateManager.js';
import type { AskUserCallback } from '../tools/askUser.js';

// Tools layer (rendering only — no execution, singleton)
import { ToolManager } from '../tools/ToolManager.js';
import { initDebugLog } from '../lib/debugLog.js';

export interface ContainerConfig {
  projectRoot: string;
  serverAddress: string;
  wsAddress?: string; // WebSocket address (host:port)
  projectKey: string;
  sessionId?: string; // Optional: reuse specific session, otherwise generate new
  agentName?: string; // Optional: agent name to use for this session
  headlessMode?: boolean;
  askUserCallback?: AskUserCallback;
  overrides?: {
    streamGateway?: IStreamGateway;
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
  private _agentStateManager: AgentStateManager;

  // Application layer
  private _accumulator: MessageAccumulatorService;
  private _streamProcessor: StreamProcessorService;

  private _initialized = false;

  constructor(config: ContainerConfig) {
    // Gateway connects to wsAddress if provided, otherwise serverAddress
    if (config.wsAddress && !config.overrides?.streamGateway) {
      config = { ...config, serverAddress: config.wsAddress };
    }
    this._config = config;
    this._sessionId = config.sessionId ?? uuidv4();

    // Initialize file-based debug logging (no-op unless BYTEBREW_DEBUG_LOG=1)
    initDebugLog(config.projectRoot);

    // Create infrastructure layer
    this._eventBus = new SimpleEventBus();
    this._messageRepository = new InMemoryMessageRepository();
    this._streamGateway = config.overrides?.streamGateway ?? new WsStreamGateway();
    this._agentStateManager = new AgentStateManager();

    // Create application layer (no toolExecutor — server executes tools)
    this._accumulator = new MessageAccumulatorService();
    this._streamProcessor = new StreamProcessorService({
      streamGateway: this._streamGateway,
      messageRepository: this._messageRepository,
      toolExecutor: null,
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

    this._initialized = true;
  }

  /**
   * Dispose of all resources
   */
  async dispose(): Promise<void> {
    this._streamProcessor.dispose();
    this._streamGateway.disconnect();
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
