// Background indexer for automatic file monitoring and reindexing
import path from 'path';
import { TreeSitterParser } from './parser.js';
import { ASTChunker } from './chunker.js';
import { FileScanner, ScanOptions } from './scanner.js';
import { IChunkStore, IEmbeddingsClient, ChunkMetadataRow } from '../domain/store.js';
import { FileIgnoreFactory } from '../infrastructure/file-ignore/FileIgnoreFactory.js';
import { getLogger } from '../lib/logger.js';

export interface IndexingStatus {
  phase: 'idle' | 'syncing' | 'embedding' | 'watching' | 'error';
  filesChecked?: number;
  filesTotal?: number;
  filesUpdated?: number;
  chunksEmbedded?: number;
  chunksTotal?: number;
  currentFile?: string;
  error?: string;
  ollamaAvailable?: boolean;
}

export interface BackgroundIndexerConfig {
  projectRoot: string;
  store: IChunkStore;
  embeddingsClient: IEmbeddingsClient;
  scanOptions?: ScanOptions;
  onProgress?: (status: IndexingStatus) => void;
  onError?: (error: Error) => void;
}

const EMBEDDING_BATCH_SIZE = 50;
const OLLAMA_RECHECK_INTERVAL_MS = 60_000;

export class BackgroundIndexer {
  private config: BackgroundIndexerConfig;
  private parser: TreeSitterParser;
  private chunker: ASTChunker;
  private scanner: FileScanner | null = null;
  private isRunning = false;
  private ollamaCheckTimer: ReturnType<typeof setInterval> | null = null;
  private logger = getLogger();

  constructor(config: BackgroundIndexerConfig) {
    this.config = config;
    this.parser = new TreeSitterParser();
    this.chunker = new ASTChunker(this.parser);
  }

  private async getScanner(): Promise<FileScanner> {
    if (!this.scanner) {
      const fileIgnore = await FileIgnoreFactory.create(this.config.projectRoot);
      this.scanner = new FileScanner(this.config.projectRoot, fileIgnore, this.config.scanOptions);
    }
    return this.scanner;
  }

  async start(): Promise<void> {
    if (this.isRunning) {
      this.logger.debug('BackgroundIndexer already running');
      return;
    }

    this.isRunning = true;
    const report = this.config.onProgress || (() => {});

    try {
      await this.parser.init();
      await this.config.store.ensureCollection();

      // Phase 1: metadata sync (ALWAYS runs, no Ollama needed)
      report({ phase: 'syncing', filesChecked: 0, filesUpdated: 0 });
      await this.metadataSync(report);

      // Phase 2: embedding sync (only if Ollama available)
      const ollamaOk = await this.config.embeddingsClient.ping();
      if (ollamaOk) {
        await this.embeddingSync(report);
        report({ phase: 'watching', ollamaAvailable: true });
      } else {
        this.logger.info('Ollama not available, metadata indexed. Will recheck periodically.');
        report({ phase: 'watching', ollamaAvailable: false });
        this.startOllamaRecheck();
      }
    } catch (error: any) {
      this.logger.error('BackgroundIndexer start failed', { error: error.message });
      report({ phase: 'error', error: error.message });
      this.config.onError?.(error);
    }
  }

  stop(): void {
    this.isRunning = false;
    if (this.ollamaCheckTimer) {
      clearInterval(this.ollamaCheckTimer);
      this.ollamaCheckTimer = null;
    }
    // NOTE: do NOT close store — it's shared
    this.logger.debug('BackgroundIndexer stopped');
  }

  private async metadataSync(report: (status: IndexingStatus) => void): Promise<void> {
    const store = this.config.store;
    const indexedFiles = await store.getIndexedFiles();
    const scanner = await this.getScanner();
    const currentFiles = await scanner.scan();
    const currentPaths = new Set(currentFiles.map(f => f.filePath));

    let processed = 0;
    const total = currentFiles.length;

    // Delete removed files
    for (const [filePath] of indexedFiles) {
      if (!currentPaths.has(filePath)) {
        await store.deleteByFilePath(filePath);
      }
    }

    // New or modified files → storeMetadataOnly
    for (const file of currentFiles) {
      const storedMtime = indexedFiles.get(file.filePath);
      const stats = await scanner.getFileStats(file.filePath);
      if (!stats) {
        processed++;
        continue;
      }

      const currentMtime = Math.floor(stats.mtime.getTime());

      if (storedMtime === undefined) {
        // New file
        await this.indexFileMetadata(file.filePath, scanner, currentMtime);
      } else if (storedMtime === null || currentMtime > storedMtime) {
        // Modified
        await store.deleteByFilePath(file.filePath);
        await this.indexFileMetadata(file.filePath, scanner, currentMtime);
      }

      processed++;
      if (processed % 50 === 0 || processed === total) {
        report({
          phase: 'syncing',
          filesChecked: processed,
          filesTotal: total,
          filesUpdated: processed,
          currentFile: path.basename(file.filePath),
        });
      }
    }

    this.logger.info('Metadata sync complete', { files: total });
  }

  private async indexFileMetadata(
    filePath: string,
    scanner: FileScanner,
    mtime: number
  ): Promise<void> {
    try {
      const content = await scanner.readFile(filePath);
      const chunks = await this.chunker.chunkFile(filePath, content);
      if (chunks.length > 0) {
        await this.config.store.storeMetadataOnly(chunks, mtime);
      }
    } catch (err: any) {
      this.logger.warn('Failed to index file metadata', { filePath, error: err.message });
    }
  }

  private async embeddingSync(report: (status: IndexingStatus) => void): Promise<void> {
    const store = this.config.store;
    const embeddings = this.config.embeddingsClient;

    // Count total chunks without embeddings
    const allWithout = await store.getChunksWithoutEmbeddings(100_000);
    const totalChunks = allWithout.length;

    if (totalChunks === 0) {
      this.logger.debug('All chunks have embeddings');
      return;
    }

    report({
      phase: 'embedding',
      chunksEmbedded: 0,
      chunksTotal: totalChunks,
    });

    let embedded = 0;

    // Process in batches
    while (this.isRunning) {
      const batch = await store.getChunksWithoutEmbeddings(EMBEDDING_BATCH_SIZE);
      if (batch.length === 0) break;

      const texts = batch.map((row: ChunkMetadataRow) => {
        let text = row.name;
        if (row.signature) text += '\n' + row.signature;
        text += '\n' + row.content;
        return text;
      });

      const batchEmbeddings = await embeddings.embedBatch(texts);

      const allKeys = batch.map((row: ChunkMetadataRow) => row.key);
      let batchEmbedded = 0;
      for (let i = 0; i < batch.length; i++) {
        const emb = batchEmbeddings[i];
        if (!emb) continue;
        store.addEmbeddingForKey(batch[i].key, emb);
        batchEmbedded++;
      }

      // Mark ALL keys as processed (including failed) to avoid infinite retry.
      // Failed chunks stay in SQLite for symbol search but won't have vectors.
      store.markEmbeddings(allKeys);
      if (batchEmbedded > 0) {
        store.saveIndex();
      }

      embedded += batchEmbedded;
      report({
        phase: 'embedding',
        chunksEmbedded: embedded,
        chunksTotal: totalChunks,
      });
    }

    this.logger.info('Embedding sync complete', { embedded, total: totalChunks });
  }

  private startOllamaRecheck(): void {
    this.ollamaCheckTimer = setInterval(async () => {
      if (!this.isRunning) return;

      const ollamaOk = await this.config.embeddingsClient.ping();
      if (!ollamaOk) return;

      this.logger.info('Ollama became available, starting embedding sync');

      // Stop the recheck timer
      if (this.ollamaCheckTimer) {
        clearInterval(this.ollamaCheckTimer);
        this.ollamaCheckTimer = null;
      }

      const report = this.config.onProgress || (() => {});
      try {
        await this.embeddingSync(report);
        report({ phase: 'watching', ollamaAvailable: true });
      } catch (err: any) {
        this.logger.error('Embedding sync failed', { error: err.message });
        report({ phase: 'watching', ollamaAvailable: false });
        // Restart recheck
        this.startOllamaRecheck();
      }
    }, OLLAMA_RECHECK_INTERVAL_MS);
  }

  isActive(): boolean {
    return this.isRunning;
  }
}
