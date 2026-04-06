import { useState, useEffect } from 'react';
import { usePrototype } from './usePrototype';

interface PrototypeDataResult<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
}

/**
 * Returns mock data when in prototype mode, otherwise calls the API.
 * This is a wrapper — existing hooks (useApi, useAuth) are NOT modified.
 */
export function usePrototypeData<T>(
  apiCall: () => Promise<T>,
  mockData: T,
): PrototypeDataResult<T> {
  const { isPrototype } = usePrototype();
  const [data, setData] = useState<T | null>(isPrototype ? mockData : null);
  const [loading, setLoading] = useState(!isPrototype);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isPrototype) {
      setData(mockData);
      setLoading(false);
      setError(null);
      return;
    }

    setLoading(true);
    apiCall()
      .then((result) => {
        setData(result);
        setError(null);
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : 'Failed to load');
      })
      .finally(() => setLoading(false));
  }, [isPrototype]); // eslint-disable-line react-hooks/exhaustive-deps

  return { data, loading, error };
}
