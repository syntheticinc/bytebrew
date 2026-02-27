// USearch + SQLite vector store for code chunks
import { Index, MetricKind, ScalarKind } from 'usearch';
import { Database } from 'bun:sqlite';
import fs from 'fs';
import path from 'path';
import { CodeChunk, IndexStatus, SearchResult } from '../domain/chunk.js';
import { IChunkStore, IEmbeddingsClient, ChunkMetadataRow } from '../domain/store.js';
import { getLogger } from '../lib/logger.js';

const DEFAULT_BYTEBREW_DIR = '.bytebrew';
const DEFAULT_DIMENSION = 768;

export interface ChunkStoreConfig {
  bytebrewDir?: string;
  dimension?: number;
}

export class ChunkStore implements IChunkStore {
  private index: Index | null = null;
  private db: Database | null = null;
  private bytebrewDir: string;
  private indexPath: string;
  private dbPath: string;
  private embeddings: IEmbeddingsClient;
  private dimension: number;
  private nextKey: number = 1;

  constructor(projectRoot: string, embeddings: IEmbeddingsClient, config: ChunkStoreConfig = {}) {
    this.bytebrewDir = path.join(projectRoot, config.bytebrewDir || DEFAULT_BYTEBREW_DIR);
    this.indexPath = path.join(this.bytebrewDir, 'index.usearch');
    this.dbPath = path.join(this.bytebrewDir, 'metadata.db');
    this.embeddings = embeddings;
    this.dimension = config.dimension || embeddings.getDimension() || DEFAULT_DIMENSION;
  }

  async ensureCollection(): Promise<void> {
    // Already initialized - skip
    if (this.db && this.index) {
      return;
    }

    // Create directory if not exists
    if (!fs.existsSync(this.bytebrewDir)) {
      fs.mkdirSync(this.bytebrewDir, { recursive: true });
    }

    // Check if index file exists before loading
    const indexFileExisted = fs.existsSync(this.indexPath);

    // Initialize SQLite database
    this.db = new Database(this.dbPath);
    this.db.exec('PRAGMA journal_mode = WAL');

    // Create metadata table (base schema without file_mtime for compatibility)
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS chunks (
        key INTEGER PRIMARY KEY,
        id TEXT NOT NULL,
        file_path TEXT NOT NULL,
        content TEXT NOT NULL,
        start_line INTEGER NOT NULL,
        end_line INTEGER NOT NULL,
        language TEXT NOT NULL,
        chunk_type TEXT NOT NULL,
        name TEXT NOT NULL,
        parent_name TEXT,
        signature TEXT,
        created_at TEXT NOT NULL
      );

      CREATE INDEX IF NOT EXISTS idx_chunks_name ON chunks(name);
      CREATE INDEX IF NOT EXISTS idx_chunks_file_path ON chunks(file_path);
      CREATE INDEX IF NOT EXISTS idx_chunks_chunk_type ON chunks(chunk_type);
    `);

    // Migration: add file_mtime column if not exists (for existing databases)
    try {
      this.db.exec(`ALTER TABLE chunks ADD COLUMN file_mtime INTEGER`);
    } catch {
      // Column already exists, ignore
    }

    // Migration: add has_embedding column if not exists
    try {
      this.db.exec(`ALTER TABLE chunks ADD COLUMN has_embedding INTEGER DEFAULT 0`);
    } catch {
      // Column already exists, ignore
    }

    // Create index for file_mtime (after column exists)
    this.db.exec(`
      CREATE INDEX IF NOT EXISTS idx_chunks_file_mtime ON chunks(file_path, file_mtime);
    `);

    // Get next key
    const maxKey = this.db.prepare('SELECT MAX(key) as max_key FROM chunks').get() as { max_key: number | null };
    this.nextKey = (maxKey?.max_key || 0) + 1;

    // Initialize or load USearch index (native addon — can crash on corruption)
    try {
      this.index = new Index({
        metric: MetricKind.Cos,
        quantization: ScalarKind.F32,
        connectivity: 16,
        dimensions: this.dimension,
        expansion_add: 128,
        expansion_search: 64,
        multi: false,
      });

      // Load existing index if exists (with fallback on corruption)
      if (fs.existsSync(this.indexPath)) {
        try {
          this.index.load(this.indexPath);
        } catch {
          const logger = getLogger();
          logger.warn('USearch index corrupted, recreating...');
          try { fs.unlinkSync(this.indexPath); } catch {}
        }
      }
    } catch (error) {
      const logger = getLogger();
      logger.error('USearch index initialization failed, vector search disabled', { error });
      this.index = null;
    }

    // After DB and index are ready: if index file was missing, reset has_embedding
    if (!indexFileExisted) {
      this.db.exec(`UPDATE chunks SET has_embedding = 0`);
    }

    // If index has data, mark existing chunks as having embeddings
    try {
      if (this.index && this.index.size() > 0) {
        this.db.exec(`UPDATE chunks SET has_embedding = 1 WHERE has_embedding = 0`);
      }
    } catch {
      const logger = getLogger();
      logger.warn('USearch index.size() failed, resetting embeddings');
      this.db.exec(`UPDATE chunks SET has_embedding = 0`);
    }
  }

  async store(chunks: CodeChunk[], fileMtime?: number): Promise<void> {
    if (chunks.length === 0) return;

    await this.ensureCollection();
    if (!this.index || !this.db) throw new Error('Store not initialized');

    // Generate embeddings for chunks
    const texts = chunks.map((chunk) => {
      let text = chunk.name;
      if (chunk.signature) {
        text += '\n' + chunk.signature;
      }
      text += '\n' + chunk.content;
      return text;
    });

    const embeddings = await this.embeddings.embedBatch(texts);

    // Prepare SQLite insert statement (has_embedding = 1 for chunks with embeddings)
    const insertStmt = this.db.prepare(`
      INSERT INTO chunks (key, id, file_path, content, start_line, end_line, language, chunk_type, name, parent_name, signature, created_at, file_mtime, has_embedding)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
    `);

    // Insert chunks (skip those where embedding failed)
    const insertMany = this.db.transaction((chunksToInsert: CodeChunk[], embeddingsToInsert: (number[] | null)[]) => {
      for (let i = 0; i < chunksToInsert.length; i++) {
        const embedding = embeddingsToInsert[i];
        if (!embedding) continue; // Skip failed embeddings

        const chunk = chunksToInsert[i];
        const key = this.nextKey++;

        // Insert metadata into SQLite
        insertStmt.run(
          key,
          chunk.id,
          chunk.filePath,
          chunk.content,
          chunk.startLine,
          chunk.endLine,
          chunk.language,
          chunk.chunkType,
          chunk.name,
          chunk.parentName || null,
          chunk.signature || null,
          new Date().toISOString(),
          fileMtime || null
        );

        // Add vector to USearch index
        this.index!.add(BigInt(key), new Float32Array(embedding));
      }
    });

    insertMany(chunks, embeddings);

    // Save index to disk
    this.index.save(this.indexPath);
  }

  async storeMetadataOnly(chunks: CodeChunk[], fileMtime?: number): Promise<void> {
    if (chunks.length === 0) return;
    await this.ensureCollection();
    if (!this.db) throw new Error('Store not initialized');

    const insertStmt = this.db.prepare(`
      INSERT INTO chunks (key, id, file_path, content, start_line, end_line,
        language, chunk_type, name, parent_name, signature, created_at, file_mtime, has_embedding)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)
    `);

    const insertMany = this.db.transaction((chunksToInsert: CodeChunk[]) => {
      for (const chunk of chunksToInsert) {
        const key = this.nextKey++;
        insertStmt.run(
          key,
          chunk.id,
          chunk.filePath,
          chunk.content,
          chunk.startLine,
          chunk.endLine,
          chunk.language,
          chunk.chunkType,
          chunk.name,
          chunk.parentName || null,
          chunk.signature || null,
          new Date().toISOString(),
          fileMtime || null
        );
      }
    });

    insertMany(chunks);
    // Intentionally NOT calling this.index.add() or this.index.save() — metadata only
  }


  async search(query: string, limit: number = 10): Promise<SearchResult[]> {
    await this.ensureCollection();
    if (!this.index || !this.db) return []; // Vector search disabled

    try {
      if (this.index.size() === 0) {
        return [];
      }
    } catch {
      return []; // USearch index corrupted
    }

    const queryEmbedding = await this.embeddings.embed(query);
    let results;
    try {
      results = this.index.search(new Float32Array(queryEmbedding), limit, 0);
    } catch {
      const logger = getLogger();
      logger.warn('USearch index.search() failed');
      return [];
    }

    const searchResults: SearchResult[] = [];
    const getChunkStmt = this.db.prepare('SELECT * FROM chunks WHERE key = ?');

    for (let i = 0; i < results.keys.length; i++) {
      const key = Number(results.keys[i]);
      const distance = results.distances[i];
      const row = getChunkStmt.get(key) as ChunkRow | undefined;

      if (row) {
        searchResults.push({
          chunk: this.rowToChunk(row),
          score: 1 - distance, // Convert distance to similarity score
        });
      }
    }

    return searchResults;
  }

  async searchWithFilter(
    query: string,
    limit: number = 10,
    filter?: {
      language?: string;
      chunkType?: string;
      filePath?: string;
    }
  ): Promise<SearchResult[]> {
    // USearch doesn't support filtering, so we search more and filter after
    const results = await this.search(query, limit * 3);

    let filtered = results;

    if (filter?.language) {
      filtered = filtered.filter((r) => r.chunk.language === filter.language);
    }
    if (filter?.chunkType) {
      filtered = filtered.filter((r) => r.chunk.chunkType === filter.chunkType);
    }
    if (filter?.filePath) {
      filtered = filtered.filter((r) => r.chunk.filePath.includes(filter.filePath!));
    }

    return filtered.slice(0, limit);
  }

  async getByName(name: string): Promise<CodeChunk[]> {
    await this.ensureCollection();
    if (!this.db) throw new Error('Store not initialized');

    const rows = this.db.prepare('SELECT * FROM chunks WHERE name = ?').all(name) as ChunkRow[];
    return rows.map((row) => this.rowToChunk(row));
  }

  async getByFilePath(filePath: string): Promise<CodeChunk[]> {
    await this.ensureCollection();
    if (!this.db) throw new Error('Store not initialized');

    const rows = this.db.prepare('SELECT * FROM chunks WHERE file_path = ?').all(filePath) as ChunkRow[];
    return rows.map((row) => this.rowToChunk(row));
  }

  async deleteByFilePath(filePath: string): Promise<void> {
    await this.ensureCollection();
    if (!this.index || !this.db) throw new Error('Store not initialized');

    // Get keys to delete
    const rows = this.db.prepare('SELECT key FROM chunks WHERE file_path = ?').all(filePath) as { key: number }[];

    if (rows.length === 0) return;

    // Delete from USearch index
    for (const row of rows) {
      try {
        this.index.remove(BigInt(row.key));
      } catch {
        // Key might not exist in index
      }
    }

    // Delete from SQLite
    this.db.prepare('DELETE FROM chunks WHERE file_path = ?').run(filePath);

    // Save index
    this.index.save(this.indexPath);
  }

  async getStatus(): Promise<IndexStatus> {
    await this.ensureCollection();
    if (!this.db) throw new Error('Store not initialized');

    const countResult = this.db.prepare('SELECT COUNT(*) as count FROM chunks').get() as { count: number };
    const languagesResult = this.db.prepare('SELECT DISTINCT language FROM chunks').all() as { language: string }[];
    const filesResult = this.db.prepare('SELECT COUNT(DISTINCT file_path) as count FROM chunks').get() as { count: number };

    return {
      totalChunks: countResult.count,
      filesCount: filesResult.count,
      languages: languagesResult.map((r) => r.language),
      lastUpdated: new Date(),
      isStale: countResult.count === 0,
    };
  }

  /**
   * Get all indexed files with their stored mtime for comparison
   */
  async getIndexedFiles(): Promise<Map<string, number | null>> {
    await this.ensureCollection();
    if (!this.db) throw new Error('Store not initialized');

    const rows = this.db.prepare(
      'SELECT DISTINCT file_path, file_mtime FROM chunks'
    ).all() as { file_path: string; file_mtime: number | null }[];

    const result = new Map<string, number | null>();
    for (const row of rows) {
      result.set(row.file_path, row.file_mtime);
    }
    return result;
  }

  async clear(): Promise<void> {
    // Close connections
    if (this.db) {
      this.db.close();
      this.db = null;
    }
    this.index = null;

    // Delete files
    if (fs.existsSync(this.indexPath)) {
      fs.unlinkSync(this.indexPath);
    }
    if (fs.existsSync(this.dbPath)) {
      fs.unlinkSync(this.dbPath);
    }
    // Also delete WAL files
    const walPath = this.dbPath + '-wal';
    const shmPath = this.dbPath + '-shm';
    if (fs.existsSync(walPath)) fs.unlinkSync(walPath);
    if (fs.existsSync(shmPath)) fs.unlinkSync(shmPath);

    // Reinitialize
    this.nextKey = 1;
    await this.ensureCollection();
  }

  private rowToChunk(row: ChunkRow): CodeChunk {
    return {
      id: row.id,
      filePath: row.file_path,
      content: row.content,
      startLine: row.start_line,
      endLine: row.end_line,
      language: row.language,
      chunkType: row.chunk_type as CodeChunk['chunkType'],
      name: row.name,
      parentName: row.parent_name || undefined,
      signature: row.signature || undefined,
    };
  }

  getVectorDir(): string {
    return this.bytebrewDir;
  }

  async getChunksWithoutEmbeddings(limit: number = 100): Promise<ChunkMetadataRow[]> {
    await this.ensureCollection();
    if (!this.db) throw new Error('Store not initialized');

    return this.db.prepare(
      'SELECT key, name, signature, content FROM chunks WHERE has_embedding = 0 LIMIT ?'
    ).all(limit) as ChunkMetadataRow[];
  }

  addEmbeddingForKey(key: number, embedding: number[]): void {
    if (!this.index) return; // Vector search disabled (USearch init failed)
    try {
      this.index.add(BigInt(key), new Float32Array(embedding));
    } catch (error) {
      const logger = getLogger();
      logger.warn('USearch index.add() failed', { key, error });
    }
  }

  markEmbeddings(keys: number[]): void {
    if (!this.db) throw new Error('Store not initialized');

    const stmt = this.db.prepare('UPDATE chunks SET has_embedding = 1 WHERE key = ?');
    const updateMany = this.db.transaction((keysToMark: number[]) => {
      for (const key of keysToMark) {
        stmt.run(key);
      }
    });
    updateMany(keys);
  }

  saveIndex(): void {
    if (!this.index) return;
    try {
      this.index.save(this.indexPath);
    } catch (error) {
      const logger = getLogger();
      logger.warn('USearch index.save() failed', { error });
    }
  }

  close(): void {
    if (this.db) {
      this.db.close();
      this.db = null;
    }
    this.index = null;
  }
}

interface ChunkRow {
  key: number;
  id: string;
  file_path: string;
  content: string;
  start_line: number;
  end_line: number;
  language: string;
  chunk_type: string;
  name: string;
  parent_name: string | null;
  signature: string | null;
  created_at: string;
  file_mtime: number | null;
  has_embedding: number; // 0 or 1
}
