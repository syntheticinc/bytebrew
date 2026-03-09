package indexing

import (
	"context"
	"fmt"
	"log/slog"
)

// IndexProgress reports the current state of an indexing operation.
type IndexProgress struct {
	Phase           string // "scanning", "parsing", "embedding", "storing", "complete"
	FilesScanned    int
	TotalFiles      int
	ChunksProcessed int
	TotalChunks     int
	CurrentFile     string
}

// Indexer orchestrates file scanning, chunking, embedding, and storage.
type Indexer struct {
	scanner    *FileScanner
	chunker    *Chunker
	embeddings *EmbeddingsClient
	store      *ChunkStore
	rootPath   string
}

// NewIndexer creates an indexer for the given project root.
// dbPath specifies the SQLite database file for chunk storage.
func NewIndexer(rootPath, dbPath string) (*Indexer, error) {
	store, err := NewChunkStore(dbPath, DefaultDimension)
	if err != nil {
		return nil, fmt.Errorf("create chunk store: %w", err)
	}

	return &Indexer{
		scanner:    NewFileScanner(rootPath),
		chunker:    NewChunker(),
		embeddings: NewEmbeddingsClient(DefaultOllamaURL, DefaultEmbedModel, DefaultDimension),
		store:      store,
		rootPath:   rootPath,
	}, nil
}

// Store returns the underlying ChunkStore for direct queries.
func (idx *Indexer) Store() *ChunkStore {
	return idx.store
}

// Close releases resources held by the indexer.
func (idx *Indexer) Close() error {
	return idx.store.Close()
}

// Index scans, parses, embeds, and stores code chunks.
// If reindex is false, only changed files (by mtime) are processed.
// onProgress is called with status updates (may be nil).
func (idx *Indexer) Index(ctx context.Context, reindex bool, onProgress func(IndexProgress)) error {
	report := func(p IndexProgress) {
		if onProgress != nil {
			onProgress(p)
		}
	}

	// Phase 1: Scan files
	report(IndexProgress{Phase: "scanning"})
	files, err := idx.scanner.Scan(ctx)
	if err != nil {
		return fmt.Errorf("scan files: %w", err)
	}

	slog.InfoContext(ctx, "indexing started", "files", len(files), "reindex", reindex)

	// Phase 2: Determine which files need processing
	filesToProcess := files
	if !reindex {
		filesToProcess, err = idx.filterChanged(ctx, files)
		if err != nil {
			return fmt.Errorf("filter changed files: %w", err)
		}
	}

	if len(filesToProcess) == 0 {
		slog.InfoContext(ctx, "no files to index")
		report(IndexProgress{Phase: "complete"})
		return nil
	}

	slog.InfoContext(ctx, "files to process", "count", len(filesToProcess), "total", len(files))

	// Phase 3: Parse files into chunks
	report(IndexProgress{Phase: "parsing", TotalFiles: len(filesToProcess)})

	var allChunks []CodeChunk
	var chunkMtimes []int64

	for i, file := range filesToProcess {
		if err := ctx.Err(); err != nil {
			return err
		}

		report(IndexProgress{
			Phase:        "parsing",
			FilesScanned: i + 1,
			TotalFiles:   len(filesToProcess),
			CurrentFile:  file.RelativePath,
		})

		content, err := idx.scanner.ReadFile(file.FilePath)
		if err != nil {
			slog.WarnContext(ctx, "skip file, read failed", "path", file.RelativePath, "error", err)
			continue
		}

		mtime, err := idx.scanner.GetFileMtime(file.FilePath)
		if err != nil {
			mtime = 0
		}

		// Delete old chunks for this file before inserting new ones
		if err := idx.store.DeleteByFilePath(ctx, file.FilePath); err != nil {
			slog.WarnContext(ctx, "delete old chunks failed", "path", file.RelativePath, "error", err)
		}

		chunks := idx.chunker.ChunkFile(file.FilePath, content, file.Language)
		for _, chunk := range chunks {
			allChunks = append(allChunks, chunk)
			chunkMtimes = append(chunkMtimes, mtime)
		}
	}

	if len(allChunks) == 0 {
		slog.InfoContext(ctx, "no chunks produced")
		report(IndexProgress{Phase: "complete"})
		return nil
	}

	slog.InfoContext(ctx, "chunks parsed", "count", len(allChunks))

	// Phase 4: Generate embeddings
	ollamaAvailable := idx.embeddings.Ping(ctx)
	var allEmbeddings [][]float32

	if ollamaAvailable {
		allEmbeddings, err = idx.embedChunks(ctx, allChunks, report)
		if err != nil {
			slog.WarnContext(ctx, "embedding failed, storing metadata only", "error", err)
			allEmbeddings = make([][]float32, len(allChunks))
		}
	} else {
		slog.WarnContext(ctx, "ollama not available, storing metadata only")
		allEmbeddings = make([][]float32, len(allChunks))
	}

	// Phase 5: Store chunks in batches
	report(IndexProgress{Phase: "storing", TotalChunks: len(allChunks)})

	batchSize := EmbedBatchSize
	for i := 0; i < len(allChunks); i += batchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := i + batchSize
		if end > len(allChunks) {
			end = len(allChunks)
		}

		batchChunks := allChunks[i:end]
		batchEmb := allEmbeddings[i:end]
		batchMtime := chunkMtimes[i] // use mtime of first chunk in batch

		if err := idx.store.Store(ctx, batchChunks, batchEmb, batchMtime); err != nil {
			slog.ErrorContext(ctx, "store batch failed", "offset", i, "error", err)
			continue
		}

		report(IndexProgress{
			Phase:           "storing",
			ChunksProcessed: end,
			TotalChunks:     len(allChunks),
		})
	}

	slog.InfoContext(ctx, "indexing complete",
		"files", len(filesToProcess),
		"chunks", len(allChunks),
		"embeddings", ollamaAvailable)

	report(IndexProgress{Phase: "complete", ChunksProcessed: len(allChunks), TotalChunks: len(allChunks)})
	return nil
}

// filterChanged returns only files whose mtime differs from what's stored.
func (idx *Indexer) filterChanged(ctx context.Context, files []ScanResult) ([]ScanResult, error) {
	indexed, err := idx.store.GetIndexedFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("get indexed files: %w", err)
	}

	var changed []ScanResult
	for _, file := range files {
		mtime, err := idx.scanner.GetFileMtime(file.FilePath)
		if err != nil {
			changed = append(changed, file)
			continue
		}

		storedMtime, exists := indexed[file.FilePath]
		if !exists || mtime != storedMtime {
			changed = append(changed, file)
		}
	}
	return changed, nil
}

// embedChunks generates embeddings for all chunks in batches.
func (idx *Indexer) embedChunks(ctx context.Context, chunks []CodeChunk, report func(IndexProgress)) ([][]float32, error) {
	allEmbeddings := make([][]float32, len(chunks))

	for i := 0; i < len(chunks); i += EmbedBatchSize {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		end := i + EmbedBatchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		report(IndexProgress{
			Phase:           "embedding",
			ChunksProcessed: i,
			TotalChunks:     len(chunks),
		})

		texts := make([]string, end-i)
		for j, chunk := range chunks[i:end] {
			texts[j] = chunk.Name + "\n" + chunk.Signature + "\n" + chunk.Content
		}

		embeddings, err := idx.embeddings.EmbedBatch(ctx, texts)
		if err != nil {
			return nil, fmt.Errorf("embed batch at offset %d: %w", i, err)
		}

		copy(allEmbeddings[i:end], embeddings)
	}

	return allEmbeddings, nil
}
