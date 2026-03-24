import { useMemo } from 'react';
import { api } from '../api/client';
import { useApi } from './useApi';
import type { ModelRegistryEntry } from '../types';

interface UseModelRegistryResult {
  registry: ModelRegistryEntry[];
  registryByModelName: Map<string, ModelRegistryEntry>;
  loading: boolean;
  error: string | null;
}

/**
 * Fetches model registry once and provides a lookup map by model id (provider/model format).
 * Graceful degradation: if registry API fails, returns empty data without blocking UI.
 */
export function useModelRegistry(): UseModelRegistryResult {
  const { data, loading, error } = useApi<ModelRegistryEntry[]>(
    () => api.getModelRegistry().catch(() => []),
  );

  const registry = data ?? [];

  const registryByModelName = useMemo(() => {
    const map = new Map<string, ModelRegistryEntry>();
    for (const entry of registry) {
      map.set(entry.id, entry);
    }
    return map;
  }, [registry]);

  return { registry, registryByModelName, loading, error };
}
