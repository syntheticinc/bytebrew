// Input history hook
import { useState, useCallback } from 'react';

const MAX_HISTORY_SIZE = 100;

export function useInputHistory(initialHistory: string[] = []) {
  const [history, setHistory] = useState<string[]>(initialHistory);

  const addToHistory = useCallback((input: string) => {
    const trimmed = input.trim();
    if (!trimmed) return;

    setHistory((prev) => {
      // Don't add duplicates of the last entry
      if (prev.length > 0 && prev[prev.length - 1] === trimmed) {
        return prev;
      }

      const newHistory = [...prev, trimmed];

      // Limit history size
      if (newHistory.length > MAX_HISTORY_SIZE) {
        return newHistory.slice(-MAX_HISTORY_SIZE);
      }

      return newHistory;
    });
  }, []);

  const clearHistory = useCallback(() => {
    setHistory([]);
  }, []);

  return {
    history,
    addToHistory,
    clearHistory,
  };
}
