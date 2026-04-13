import { useState, useEffect, useCallback, useRef } from 'react';

interface UseApiResult<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
  refetch: () => void;
}

export function useApi<T>(fetcher: () => Promise<T>, deps: unknown[] = []): UseApiResult<T> {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [refetchCount, setRefetchCount] = useState(0);
  const hasLoadedOnce = useRef(false);
  const prevRefetchRef = useRef(0);

  const refetch = useCallback(() => {
    setRefetchCount((c) => c + 1);
  }, []);

  useEffect(() => {
    let cancelled = false;
    // Distinguish deps change (filter/page) from refetch() call (auto-refresh).
    // Show loading spinner on initial load or deps change; skip on background
    // refetch to prevent table flicker every 5s.
    const isRefetchTriggered = refetchCount !== prevRefetchRef.current;
    prevRefetchRef.current = refetchCount;
    if (!(hasLoadedOnce.current && isRefetchTriggered)) {
      setLoading(true);
    }
    setError(null);

    fetcher()
      .then((result) => {
        if (!cancelled) {
          setData(result);
          setLoading(false);
          hasLoadedOnce.current = true;
        }
      })
      .catch((err: Error) => {
        if (!cancelled) {
          setError(err.message);
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refetchCount, ...deps]);

  return { data, loading, error, refetch };
}
