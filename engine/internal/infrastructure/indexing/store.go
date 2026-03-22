package indexing

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"
)

const createChunksTable = `
CREATE TABLE IF NOT EXISTS chunks (
    id TEXT PRIMARY KEY,
    file_path TEXT NOT NULL,
    content TEXT NOT NULL,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    language TEXT NOT NULL,
    chunk_type TEXT NOT NULL,
    name TEXT NOT NULL,
    parent_name TEXT DEFAULT '',
    signature TEXT DEFAULT '',
    embedding BLOB,
    file_mtime INTEGER DEFAULT 0,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_chunks_name ON chunks(name);
CREATE INDEX IF NOT EXISTS idx_chunks_file_path ON chunks(file_path);
CREATE INDEX IF NOT EXISTS idx_chunks_chunk_type ON chunks(chunk_type);
`

// ChunkStore persists code chunks and their embeddings in SQLite.
type ChunkStore struct {
	db        *sql.DB
	dimension int
}

// NewChunkStore opens or creates a SQLite database for chunk storage.
func NewChunkStore(dbPath string, dimension int) (*ChunkStore, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open db %s: %w", dbPath, err)
	}

	if _, err := db.Exec(createChunksTable); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &ChunkStore{db: db, dimension: dimension}, nil
}

// Close closes the database connection.
func (s *ChunkStore) Close() error {
	return s.db.Close()
}

// Store inserts or replaces chunks with their embeddings.
func (s *ChunkStore) Store(ctx context.Context, chunks []CodeChunk, embeddings [][]float32, fileMtime int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT OR REPLACE INTO chunks (id, file_path, content, start_line, end_line, language, chunk_type, name, parent_name, signature, embedding, file_mtime, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC().Format(time.RFC3339)

	for i, chunk := range chunks {
		var embBlob []byte
		if i < len(embeddings) && embeddings[i] != nil {
			embBlob = float32sToBytes(embeddings[i])
		}

		_, err := stmt.ExecContext(ctx,
			chunk.ID, chunk.FilePath, chunk.Content,
			chunk.StartLine, chunk.EndLine, chunk.Language,
			string(chunk.ChunkType), chunk.Name,
			chunk.ParentName, chunk.Signature,
			embBlob, fileMtime, now,
		)
		if err != nil {
			return fmt.Errorf("insert chunk %s: %w", chunk.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// SearchResult pairs a chunk with its similarity score.
type SearchResult struct {
	Chunk CodeChunk
	Score float32
}

// Search finds the most similar chunks to the query embedding via brute-force cosine similarity.
func (s *ChunkStore) Search(ctx context.Context, queryEmbedding []float32, limit int) ([]SearchResult, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, file_path, content, start_line, end_line, language, chunk_type, name, parent_name, signature, embedding
		 FROM chunks WHERE embedding IS NOT NULL`)
	if err != nil {
		return nil, fmt.Errorf("query chunks: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var chunk CodeChunk
		var chunkType string
		var embBlob []byte

		if err := rows.Scan(
			&chunk.ID, &chunk.FilePath, &chunk.Content,
			&chunk.StartLine, &chunk.EndLine, &chunk.Language,
			&chunkType, &chunk.Name, &chunk.ParentName, &chunk.Signature,
			&embBlob,
		); err != nil {
			slog.WarnContext(ctx, "scan chunk row failed", "error", err)
			continue
		}

		chunk.ChunkType = ChunkType(chunkType)
		emb := bytesToFloat32s(embBlob)
		if len(emb) == 0 {
			continue
		}

		score := cosineSimilarity(queryEmbedding, emb)
		results = append(results, SearchResult{Chunk: chunk, Score: score})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// GetByName returns chunks matching the given symbol name.
func (s *ChunkStore) GetByName(ctx context.Context, name string) ([]CodeChunk, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, file_path, content, start_line, end_line, language, chunk_type, name, parent_name, signature
		 FROM chunks WHERE name = ?`, name)
	if err != nil {
		return nil, fmt.Errorf("query by name: %w", err)
	}
	defer rows.Close()

	return scanChunks(rows)
}

// GetByFilePath returns all chunks for a given file path.
func (s *ChunkStore) GetByFilePath(ctx context.Context, filePath string) ([]CodeChunk, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, file_path, content, start_line, end_line, language, chunk_type, name, parent_name, signature
		 FROM chunks WHERE file_path = ?`, filePath)
	if err != nil {
		return nil, fmt.Errorf("query by file path: %w", err)
	}
	defer rows.Close()

	return scanChunks(rows)
}

// GetIndexedFiles returns a map of file paths to their stored modification times.
func (s *ChunkStore) GetIndexedFiles(ctx context.Context) (map[string]int64, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT file_path, file_mtime FROM chunks`)
	if err != nil {
		return nil, fmt.Errorf("query indexed files: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var fp string
		var mtime int64
		if err := rows.Scan(&fp, &mtime); err != nil {
			continue
		}
		result[fp] = mtime
	}
	return result, rows.Err()
}

// DeleteByFilePath removes all chunks for a given file.
func (s *ChunkStore) DeleteByFilePath(ctx context.Context, filePath string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM chunks WHERE file_path = ?`, filePath)
	if err != nil {
		return fmt.Errorf("delete by file path: %w", err)
	}
	return nil
}

// Clear removes all chunks from the store.
func (s *ChunkStore) Clear(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM chunks`)
	if err != nil {
		return fmt.Errorf("clear chunks: %w", err)
	}
	return nil
}

// scanChunks reads CodeChunk rows from a query result (without embedding column).
func scanChunks(rows *sql.Rows) ([]CodeChunk, error) {
	var chunks []CodeChunk
	for rows.Next() {
		var chunk CodeChunk
		var chunkType string
		if err := rows.Scan(
			&chunk.ID, &chunk.FilePath, &chunk.Content,
			&chunk.StartLine, &chunk.EndLine, &chunk.Language,
			&chunkType, &chunk.Name, &chunk.ParentName, &chunk.Signature,
		); err != nil {
			continue
		}
		chunk.ChunkType = ChunkType(chunkType)
		chunks = append(chunks, chunk)
	}
	return chunks, rows.Err()
}

// float32sToBytes encodes a float32 slice as little-endian binary.
func float32sToBytes(fs []float32) []byte {
	buf := make([]byte, len(fs)*4)
	for i, f := range fs {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

// bytesToFloat32s decodes little-endian binary into a float32 slice.
func bytesToFloat32s(b []byte) []float32 {
	if len(b)%4 != 0 {
		return nil
	}
	fs := make([]float32, len(b)/4)
	for i := range fs {
		fs[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return fs
}

// cosineSimilarity computes the cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / float32(math.Sqrt(float64(normA))*math.Sqrt(float64(normB)))
}
