// Store interfaces for dependency injection (DIP - Dependency Inversion Principle)
import { CodeChunk, IndexStatus, SearchResult } from './chunk.js';

/** Row returned by getChunksWithoutEmbeddings — key + metadata for embedding */
export interface ChunkMetadataRow {
  key: number;
  name: string;
  signature: string | null;
  content: string;
}

/**
 * Interface for chunk storage operations
 * Tools depend on this interface, not on concrete implementations
 */
export interface IChunkStore {
  ensureCollection(): Promise<void>;
  store(chunks: CodeChunk[], fileMtime?: number): Promise<void>;
  storeMetadataOnly(chunks: CodeChunk[], fileMtime?: number): Promise<void>;
  search(query: string, limit?: number): Promise<SearchResult[]>;
  searchWithFilter(
    query: string,
    limit?: number,
    filter?: { language?: string; chunkType?: string; filePath?: string }
  ): Promise<SearchResult[]>;
  getByName(name: string): Promise<CodeChunk[]>;
  getByFilePath(filePath: string): Promise<CodeChunk[]>;
  deleteByFilePath(filePath: string): Promise<void>;
  getStatus(): Promise<IndexStatus>;
  getIndexedFiles(): Promise<Map<string, number | null>>;
  clear(): Promise<void>;
  close(): void;
  getChunksWithoutEmbeddings(limit?: number): Promise<ChunkMetadataRow[]>;
  addEmbeddingForKey(key: number, embedding: number[]): void;
  markEmbeddings(keys: number[]): void;
  saveIndex(): void;
}

/**
 * Interface for embeddings generation
 */
export interface IEmbeddingsClient {
  embed(text: string): Promise<number[]>;
  embedBatch(texts: string[]): Promise<(number[] | null)[]>;
  ping(): Promise<boolean>;
  getDimension(): number;
  getModel(): string;
}

/**
 * Factory interface for creating store instances
 * Implements lazy initialization pattern
 */
export interface IChunkStoreFactory {
  getStore(): Promise<IChunkStore>;
}
