// React hook for background indexing
import { useState, useEffect, useRef, useCallback } from 'react';
import { BackgroundIndexer, IndexingStatus, BackgroundIndexerConfig } from '../../indexing/backgroundIndexer.js';
import { IChunkStore, IEmbeddingsClient } from '../../domain/store.js';

export interface UseBackgroundIndexerOptions {
  projectRoot: string;
  store: IChunkStore | null;
  embeddingsClient: IEmbeddingsClient | null;
  enabled?: boolean;
}

export interface UseBackgroundIndexerResult {
  status: IndexingStatus;
  isIndexing: boolean;
  restart: () => void;
}

export function useBackgroundIndexer(
  options: UseBackgroundIndexerOptions
): UseBackgroundIndexerResult {
  const { projectRoot, store, embeddingsClient, enabled = true } = options;
  const [status, setStatus] = useState<IndexingStatus>({ phase: 'idle' });
  const indexerRef = useRef<BackgroundIndexer | null>(null);
  const isStartedRef = useRef(false);

  const handleProgress = useCallback((newStatus: IndexingStatus) => {
    setStatus(newStatus);
  }, []);

  const handleError = useCallback((error: Error) => {
    setStatus({ phase: 'error', error: error.message });
  }, []);

  // Start/stop indexer based on enabled state
  useEffect(() => {
    if (!enabled || !store || !embeddingsClient) {
      if (indexerRef.current) {
        indexerRef.current.stop();
        indexerRef.current = null;
        isStartedRef.current = false;
        setStatus({ phase: 'idle' });
      }
      return;
    }

    // Prevent double start in React strict mode
    if (isStartedRef.current) {
      return;
    }
    isStartedRef.current = true;

    const config: BackgroundIndexerConfig = {
      projectRoot,
      store,
      embeddingsClient,
      onProgress: handleProgress,
      onError: handleError,
    };

    // Small delay to let React render first
    const timeoutId = setTimeout(() => {
      indexerRef.current = new BackgroundIndexer(config);
      indexerRef.current.start().catch((err) => {
        handleError(err instanceof Error ? err : new Error(String(err)));
      });
    }, 500);

    return () => {
      clearTimeout(timeoutId);
      if (indexerRef.current) {
        indexerRef.current.stop();
        indexerRef.current = null;
        isStartedRef.current = false;
      }
    };
  }, [enabled, projectRoot, store, embeddingsClient, handleProgress, handleError]);

  const restart = useCallback(() => {
    if (!store || !embeddingsClient) return;

    if (indexerRef.current) {
      indexerRef.current.stop();
    }

    const config: BackgroundIndexerConfig = {
      projectRoot,
      store,
      embeddingsClient,
      onProgress: handleProgress,
      onError: handleError,
    };

    indexerRef.current = new BackgroundIndexer(config);
    indexerRef.current.start().catch((err) => {
      handleError(err instanceof Error ? err : new Error(String(err)));
    });
  }, [projectRoot, store, embeddingsClient, handleProgress, handleError]);

  const isIndexing = status.phase === 'syncing';

  return {
    status,
    isIndexing,
    restart,
  };
}
