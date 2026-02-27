# Frontend Developer Memory

## Windows-specific: EBUSY on temp directory cleanup in tests

When tests create a ChunkStore (bun:sqlite + USearch) in a temp directory,
`rm(dir, { recursive: true })` throws EBUSY on Windows because SQLite WAL
files and USearch index files are not released synchronously after `store.close()`.

Fix: retry deletion with delay in afterEach:

```typescript
async function deleteWithRetry(dir: string, attempts = 5, delayMs = 50): Promise<void> {
  for (let i = 0; i < attempts; i++) {
    try {
      await rm(dir, { recursive: true, force: true });
      return;
    } catch {
      if (i === attempts - 1) return; // give up silently
      await new Promise((r) => setTimeout(r, delayMs));
    }
  }
}
```

See: `bytebrew-cli/src/indexing/__tests__/metadataIndexer.test.ts`

## MetadataIndexer test pattern

- Use real ChunkStore with temp dir (not mocks) — storeMetadataOnly + SQLite
- `mockEmbeddingsClient` is sufficient (MetadataIndexer never calls embeddings)
- `ChunkStore(testDir, mockEmbeddingsClient)` — no config needed
- Access private `chunker` field via `(indexer as any).chunker` for spy injection
- go.mod file in testDir is NOT required for FileScanner to find .ts files
- go.mod IS needed to keep go.mod out of interference (scanner ignores non-code files)
