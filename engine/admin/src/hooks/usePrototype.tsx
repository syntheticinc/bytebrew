import { createContext, useContext, useState, useEffect, type ReactNode } from 'react';

const STORAGE_KEY = 'bytebrew_prototype_mode';
const PROTOTYPE_ENABLED = import.meta.env.VITE_PROTOTYPE_ENABLED === 'true'
  || import.meta.env.DEV;

interface PrototypeContextValue {
  isPrototype: boolean;
  togglePrototype: () => void;
  prototypeEnabled: boolean;
}

const PrototypeContext = createContext<PrototypeContextValue>({
  isPrototype: false,
  togglePrototype: () => {},
  prototypeEnabled: false,
});

export function PrototypeProvider({ children }: { children: ReactNode }) {
  const [prototypeEnabled, setPrototypeEnabled] = useState(PROTOTYPE_ENABLED);
  const [isPrototype, setIsPrototype] = useState(() => {
    if (!PROTOTYPE_ENABLED) return false;
    return localStorage.getItem(STORAGE_KEY) === 'true';
  });

  // Check runtime setting from API (overrides build-time if available)
  useEffect(() => {
    if (!PROTOTYPE_ENABLED) return;
    fetch(`${import.meta.env.BASE_URL}../api/v1/settings`)
      .then((r) => r.ok ? r.json() : null)
      .then((settings: Array<{ key: string; value: string }> | null) => {
        if (!settings) return;
        const setting = settings.find((s) => s.key === 'prototype_mode_enabled');
        if (setting?.value === 'false') {
          setPrototypeEnabled(false);
          setIsPrototype(false);
        }
      })
      .catch(() => { /* API unavailable — use build-time default */ });
  }, []);

  const togglePrototype = () => {
    setIsPrototype((prev) => {
      const next = !prev;
      localStorage.setItem(STORAGE_KEY, String(next));
      return next;
    });
  };

  return (
    <PrototypeContext.Provider value={{ isPrototype, togglePrototype, prototypeEnabled }}>
      {children}
    </PrototypeContext.Provider>
  );
}

export function usePrototype() {
  return useContext(PrototypeContext);
}
