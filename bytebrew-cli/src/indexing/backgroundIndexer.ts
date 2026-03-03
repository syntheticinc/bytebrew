// Background indexer for automatic file monitoring and reindexing
import path from 'path';
import fs from 'fs';
import { TreeSitterParser } from './parser.js';
import { ASTChunker } from './chunker.js';
import { FileScanner, ScanOptions } from './scanner.js';
import { IChunkStore, IEmbeddingsClient, ChunkMetadataRow } from '../domain/store.js';
import { FileIgnoreFactory } from '../infrastructure/file-ignore/FileIgnoreFactory.js';
import { getLogger } from '../lib/logger.js';

/** Write crash diagnostic to .bytebrew/crash-diag.log (sync, survives crashes) */
function diagWrite(projectRoot: string, msg: string): void {
  try {
    const diagPath = path.join(projectRoot, '.bytebrew', 'crash-diag.log');
    fs.appendFileSync(diagPath, `[indexer] ${msg}\n`);
  } catch { /* ignore */ }
}

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
    const diag = (msg: string) => diagWrite(this.config.projectRoot, msg);

    try {
      diag('start: parser.init()');
      await this.parser.init();
      diag('start: store.ensureCollection()');
      await this.config.store.ensureCollection();

      // Phase 1: metadata sync (ALWAYS runs, no Ollama needed)
      diag('start: metadataSync begin');
      report({ phase: 'syncing', filesChecked: 0, filesUpdated: 0 });
      const filesChanged = await this.metadataSync(report);
      diag(`start: metadataSync done, filesChanged=${filesChanged}`);

      // Auto-compact: rebuild USearch index only when orphaned vectors exceed 50%
      // of valid embeddings. This avoids re-embedding 15k+ chunks on every file
      // change while keeping the index from growing unbounded.
      // Orphans are harmless (keys are monotonic, never reused) but waste memory.
      if (filesChanged) {
        const needsCompact = await this.config.store.shouldCompactIndex();
        if (needsCompact) {
          diag('start: compacting USearch index (too many orphans)');
          this.config.store.rebuildIndex();
        }
      }

      // Phase 2: embedding sync (only if Ollama available)
      diag('start: ping ollama');
      const ollamaOk = await this.config.embeddingsClient.ping();
      diag(`start: ollama ping=${ollamaOk}`);
      if (ollamaOk) {
        diag('start: embeddingSync begin');
        await this.embeddingSync(report);
        diag('start: embeddingSync done');
        report({ phase: 'watching', ollamaAvailable: true });
      } else {
        this.logger.info('Ollama not available, metadata indexed. Will recheck periodically.');
        report({ phase: 'watching', ollamaAvailable: false });
        this.startOllamaRecheck();
      }
    } catch (error: any) {
      diag(`start: CAUGHT ERROR: ${error.message}\n${error.stack}`);
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

  /**
   * Sync file metadata with the index. Returns true if any files were changed.
   */
  private async metadataSync(report: (status: IndexingStatus) => void): Promise<boolean> {
    const store = this.config.store;
    const diag = (msg: string) => diagWrite(this.config.projectRoot, msg);
    const indexedFiles = await store.getIndexedFiles();
    const scanner = await this.getScanner();
    const currentFiles = await scanner.scan();
    const currentPaths = new Set(currentFiles.map(f => f.filePath));

    let processed = 0;
    let deleted = 0;
    let updated = 0;
    const total = currentFiles.length;
    diag(`metadataSync: indexed=${indexedFiles.size} current=${total}`);

    // Delete removed files
    for (const [filePath] of indexedFiles) {
      if (!currentPaths.has(filePath)) {
        await store.deleteByFilePath(filePath);
        deleted++;
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
        updated++;
      } else if (storedMtime === null || currentMtime > storedMtime) {
        // Modified
        await store.deleteByFilePath(file.filePath);
        await this.indexFileMetadata(file.filePath, scanner, currentMtime);
        updated++;
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

    diag(`metadataSync: done, deleted=${deleted} updated=${updated} total=${total}`);
    this.logger.info('Metadata sync complete', { files: total });
    return deleted > 0 || updated > 0;
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
    const diag = (msg: string) => diagWrite(this.config.projectRoot, msg);

    // Count total chunks without embeddings
    diag('embeddingSync: counting chunks');
    const allWithout = await store.getChunksWithoutEmbeddings(100_000);
    const totalChunks = allWithout.length;
    diag(`embeddingSync: totalChunks=${totalChunks}`);

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

    // Process in batches. Wrapped in try-catch to prevent embedding failures
    // from crashing the entire process (e.g. Ollama connection drop mid-batch).
    try {
      while (this.isRunning) {
        diag(`embeddingSync: fetching batch, embedded=${embedded}`);
        const batch = await store.getChunksWithoutEmbeddings(EMBEDDING_BATCH_SIZE);
        if (batch.length === 0) break;

        const texts = batch.map((row: ChunkMetadataRow) => {
          let text = row.name;
          if (row.signature) text += '\n' + row.signature;
          text += '\n' + row.content;
          return text;
        });

        diag(`embeddingSync: calling embedBatch(${texts.length} texts)`);
        const batchEmbeddings = await embeddings.embedBatch(texts);
        diag(`embeddingSync: embedBatch returned ${batchEmbeddings.length} results`);

        const allKeys = batch.map((row: ChunkMetadataRow) => row.key);
        let batchEmbedded = 0;
        for (let i = 0; i < batch.length; i++) {
          const emb = batchEmbeddings[i];
          if (!emb) continue;
          store.addEmbeddingForKey(batch[i].key, emb);
          batchEmbedded++;
        }

        diag(`embeddingSync: marking ${allKeys.length} keys, batchEmbedded=${batchEmbedded}`);
        // Mark ALL keys as processed (including failed) to avoid infinite retry.
        // Failed chunks stay in SQLite for symbol search but won't have vectors.
        store.markEmbeddings(allKeys);
        if (batchEmbedded > 0) {
          diag('embeddingSync: saveIndex');
          store.saveIndex();
        }

        embedded += batchEmbedded;
        report({
          phase: 'embedding',
          chunksEmbedded: embedded,
          chunksTotal: totalChunks,
        });
      }
    } catch (error: any) {
      diag(`embeddingSync: CAUGHT ERROR: ${error.message}\n${error.stack}`);
      this.logger.error('Embedding sync failed', { error: error.message });
      report({ phase: 'error', error: error.message });
      return;
    }

    diag(`embeddingSync: complete, embedded=${embedded}/${totalChunks}`);
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
