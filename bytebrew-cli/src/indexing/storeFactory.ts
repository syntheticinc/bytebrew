// Store factory for lazy initialization and singleton access
import { IChunkStore, IChunkStoreFactory, IEmbeddingsClient } from '../domain/store.js';
import { ChunkStore, ChunkStoreConfig } from './store.js';
import { EmbeddingsClient, EmbeddingsConfig } from './embeddings.js';
import { getLogger } from '../lib/logger.js';

/**
 * Default factory that creates ChunkStore with EmbeddingsClient
 * Implements lazy initialization and singleton pattern per projectRoot
 */
export class ChunkStoreFactory implements IChunkStoreFactory {
  private projectRoot: string;
  private store: IChunkStore | null = null;
  private embeddings: IEmbeddingsClient | null = null;
  private embeddingsConfig?: EmbeddingsConfig;
  private storeConfig?: ChunkStoreConfig;
  private initPromise: Promise<IChunkStore> | null = null;

  constructor(
    projectRoot: string,
    embeddingsConfig?: EmbeddingsConfig,
    storeConfig?: ChunkStoreConfig
  ) {
    this.projectRoot = projectRoot;
    this.embeddingsConfig = embeddingsConfig;
    this.storeConfig = storeConfig;
  }

  async getStore(): Promise<IChunkStore> {
    // Return existing promise if initialization is in progress
    if (this.initPromise) {
      return this.initPromise;
    }

    // Return existing store if already initialized
    if (this.store) {
      return this.store;
    }

    // Create and cache the initialization promise
    this.initPromise = this.createStore();

    try {
      this.store = await this.initPromise;
      return this.store;
    } finally {
      this.initPromise = null;
    }
  }

  private async createStore(): Promise<IChunkStore> {
    const logger = getLogger();
    logger.debug('Creating ChunkStore', { projectRoot: this.projectRoot });

    this.embeddings = new EmbeddingsClient(this.embeddingsConfig);
    const store = new ChunkStore(
      this.projectRoot,
      this.embeddings,
      this.storeConfig
    );
    await store.ensureCollection();

    logger.debug('ChunkStore initialized');
    return store;
  }

  getEmbeddings(): IEmbeddingsClient | null {
    return this.embeddings;
  }
}

// Global factory registry for singleton access
const factoryRegistry = new Map<string, ChunkStoreFactory>();

/**
 * Get or create a ChunkStoreFactory for the given project root
 */
export function getStoreFactory(
  projectRoot: string,
  embeddingsConfig?: EmbeddingsConfig,
  storeConfig?: ChunkStoreConfig
): ChunkStoreFactory {
  const existing = factoryRegistry.get(projectRoot);
  if (existing) {
    return existing;
  }

  const factory = new ChunkStoreFactory(projectRoot, embeddingsConfig, storeConfig);
  factoryRegistry.set(projectRoot, factory);
  return factory;
}

/**
 * Clear factory registry (for testing)
 */
export function clearFactoryRegistry(): void {
  factoryRegistry.clear();
}
