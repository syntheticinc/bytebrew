// Indexer service - coordinates the indexing process
import { TreeSitterParser } from './parser.js';
import { ASTChunker } from './chunker.js';
import { FileScanner, ScanOptions } from './scanner.js';
import { EmbeddingsClient, EmbeddingsConfig } from './embeddings.js';
import { ChunkStore, ChunkStoreConfig } from './store.js';
import { CodeChunk, IndexStatus } from '../domain/chunk.js';
import { FileIgnoreFactory } from '../infrastructure/file-ignore/FileIgnoreFactory.js';

export interface IndexerConfig {
  rootPath: string;
  scanOptions?: ScanOptions;
  embeddingsConfig?: EmbeddingsConfig;
  storeConfig?: ChunkStoreConfig;
  onProgress?: (progress: IndexProgress) => void;
}

export interface IndexProgress {
  phase: 'scanning' | 'parsing' | 'embedding' | 'storing' | 'complete';
  filesScanned?: number;
  totalFiles?: number;
  chunksProcessed?: number;
  totalChunks?: number;
  currentFile?: string;
  error?: string;
}

export class Indexer {
  private config: IndexerConfig;
  private parser: TreeSitterParser;
  private chunker: ASTChunker;
  private scanner: FileScanner | null = null;
  private embeddings: EmbeddingsClient;
  private store: ChunkStore;

  constructor(config: IndexerConfig) {
    this.config = config;
    this.parser = new TreeSitterParser();
    this.chunker = new ASTChunker(this.parser);
    // scanner created lazily (needs async FileIgnore)
    this.embeddings = new EmbeddingsClient(config.embeddingsConfig);
    this.store = new ChunkStore(config.rootPath, this.embeddings, config.storeConfig);
  }

  private async getScanner(): Promise<FileScanner> {
    if (!this.scanner) {
      const fileIgnore = await FileIgnoreFactory.create(this.config.rootPath);
      this.scanner = new FileScanner(this.config.rootPath, fileIgnore, this.config.scanOptions);
    }
    return this.scanner;
  }

  async index(reindex: boolean = false): Promise<IndexStatus> {
    const report = this.config.onProgress || (() => {});

    try {
      // Initialize parser
      await this.parser.init();

      // Check dependencies
      const ollamaOk = await this.embeddings.ping();
      if (!ollamaOk) {
        throw new Error(
          'Cannot connect to Ollama. Make sure Ollama is running on localhost:11434. ' +
          'Install with: https://ollama.ai and run: ollama pull nomic-embed-text'
        );
      }

      // Ensure collection exists
      if (reindex) {
        await this.store.clear();
      }
      await this.store.ensureCollection();

      // Scan files
      report({ phase: 'scanning' });
      const scanner = await this.getScanner();
      const files = await scanner.scan();
      report({ phase: 'scanning', filesScanned: files.length, totalFiles: files.length });

      if (files.length === 0) {
        return await this.store.getStatus();
      }

      // Parse and chunk files
      report({ phase: 'parsing', totalFiles: files.length });
      const allChunks: CodeChunk[] = [];
      let filesProcessed = 0;

      for (const file of files) {
        try {
          const content = await scanner.readFile(file.filePath);
          const chunks = await this.chunker.chunkFile(file.filePath, content);
          allChunks.push(...chunks);

          filesProcessed++;
          report({
            phase: 'parsing',
            filesScanned: filesProcessed,
            totalFiles: files.length,
            currentFile: file.relativePath,
            chunksProcessed: allChunks.length,
          });
        } catch (error) {
          // Log error but continue with other files
          console.error(`Error processing ${file.filePath}:`, error);
        }
      }

      if (allChunks.length === 0) {
        return await this.store.getStatus();
      }

      // Store chunks in batches
      report({
        phase: 'embedding',
        totalChunks: allChunks.length,
        chunksProcessed: 0,
      });

      const batchSize = 50;
      for (let i = 0; i < allChunks.length; i += batchSize) {
        const batch = allChunks.slice(i, i + batchSize);
        await this.store.store(batch);

        report({
          phase: 'storing',
          chunksProcessed: Math.min(i + batchSize, allChunks.length),
          totalChunks: allChunks.length,
        });
      }

      report({ phase: 'complete' });
      return await this.store.getStatus();
    } catch (error: any) {
      report({ phase: 'complete', error: error.message });
      throw error;
    }
  }

  async getStatus(): Promise<IndexStatus> {
    return await this.store.getStatus();
  }

  async updateFile(filePath: string): Promise<void> {
    // Delete existing chunks for this file
    await this.store.deleteByFilePath(filePath);

    // Re-parse and store
    const scanner = await this.getScanner();
    const content = await scanner.readFile(filePath);
    const chunks = await this.chunker.chunkFile(filePath, content);
    await this.store.store(chunks);
  }

  async removeFile(filePath: string): Promise<void> {
    await this.store.deleteByFilePath(filePath);
  }
}
