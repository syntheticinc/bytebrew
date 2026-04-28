import { useState } from 'react';

export interface HeaderEntry {
  id: string;
  key: string;
  value: string;
}

interface HeadersEditorProps {
  headers: HeaderEntry[];
  onChange: (headers: HeaderEntry[]) => void;
}

export default function HeadersEditor({ headers, onChange }: HeadersEditorProps) {
  const [expanded, setExpanded] = useState(false);
  const [jsonError, setJsonError] = useState('');

  function addHeader() {
    onChange([...headers, { id: crypto.randomUUID(), key: '', value: '' }]);
    if (!expanded) setExpanded(true);
  }

  function updateHeader(id: string, field: 'key' | 'value', val: string) {
    onChange(headers.map((h) => (h.id === id ? { ...h, [field]: val } : h)));
  }

  function removeHeader(id: string) {
    onChange(headers.filter((h) => h.id !== id));
  }

  function importJson() {
    const raw = window.prompt('Paste JSON object (e.g. {"X-Api-Key": "abc123"}):');
    if (!raw) return;
    try {
      const parsed = JSON.parse(raw) as Record<string, unknown>;
      if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
        setJsonError('Expected a JSON object');
        return;
      }
      const entries: HeaderEntry[] = Object.entries(parsed).map(([k, v]) => ({
        id: crypto.randomUUID(),
        key: k,
        value: String(v),
      }));
      onChange([...headers, ...entries]);
      setJsonError('');
      if (!expanded) setExpanded(true);
    } catch {
      setJsonError('Invalid JSON');
    }
  }

  return (
    <div>
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-1.5 text-xs font-medium text-brand-shade3 uppercase tracking-wide hover:text-brand-light transition-colors w-full"
      >
        <svg
          width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
          className={`transition-transform ${expanded ? 'rotate-90' : ''}`}
        >
          <polyline points="9 18 15 12 9 6" />
        </svg>
        Headers
        {headers.length > 0 && (
          <span className="text-[10px] text-brand-shade3/60 font-normal normal-case">({headers.length})</span>
        )}
      </button>

      {expanded && (
        <div className="mt-2 space-y-2">
          {headers.map((h) => (
            <div key={h.id} className="flex gap-1.5 items-start">
              <input
                type="text"
                value={h.key}
                onChange={(e) => updateHeader(h.id, 'key', e.target.value)}
                placeholder="Header"
                className="flex-1 px-2 py-1 bg-brand-dark border border-brand-shade3/30 rounded text-xs text-brand-light placeholder-brand-shade3/50 focus:outline-none focus:border-brand-accent transition-colors"
              />
              <input
                type="text"
                value={h.value}
                onChange={(e) => updateHeader(h.id, 'value', e.target.value)}
                placeholder="Value"
                className="flex-1 px-2 py-1 bg-brand-dark border border-brand-shade3/30 rounded text-xs text-brand-light placeholder-brand-shade3/50 focus:outline-none focus:border-brand-accent transition-colors"
              />
              <button
                onClick={() => removeHeader(h.id)}
                className="p-1 text-brand-shade3 hover:text-red-400 transition-colors flex-shrink-0"
                title="Remove header"
              >
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <line x1="18" y1="6" x2="6" y2="18" />
                  <line x1="6" y1="6" x2="18" y2="18" />
                </svg>
              </button>
            </div>
          ))}

          <div className="flex items-center gap-2">
            <button
              onClick={addHeader}
              className="text-[11px] text-brand-shade3 hover:text-brand-light transition-colors"
            >
              + Add Header
            </button>
            <span className="text-brand-shade3/30">|</span>
            <button
              onClick={importJson}
              className="text-[11px] text-brand-shade3 hover:text-brand-light transition-colors"
            >
              Import JSON
            </button>
          </div>

          {jsonError && (
            <p className="text-[10px] text-red-400">{jsonError}</p>
          )}
        </div>
      )}
    </div>
  );
}
