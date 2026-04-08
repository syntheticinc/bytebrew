import React, { useState } from 'react';
import { ExportButton, ImportButton } from './BuilderExportImport';

interface CanvasToolbarProps {
  isPrototype: boolean;
  savedIndicator: 'saved' | 'saving' | null;
  onAutoLayout: () => void;
  onRefetch: () => void;
  onAddAgent: () => void;
  onAddTrigger: () => void;
  // Prototype schema props
  protoSchema: string;
  protoSchemas: string[];
  setProtoSchema: (s: string) => void;
  setProtoSchemas: React.Dispatch<React.SetStateAction<string[]>>;
}

export default function CanvasToolbar({
  isPrototype,
  savedIndicator,
  onAutoLayout,
  onRefetch,
  onAddAgent,
  onAddTrigger,
  protoSchema,
  protoSchemas,
  setProtoSchema,
  setProtoSchemas,
}: CanvasToolbarProps) {
  const [protoSchemaDropdown, setProtoSchemaDropdown] = useState(false);

  return (
    <div className="flex items-center gap-3 px-4 h-12 border-b border-brand-shade3/15 bg-brand-dark-alt flex-shrink-0 flex-wrap">
      <span className="text-sm font-semibold text-brand-light">Agent Builder</span>

      {/* Prototype: schema switcher */}
      {isPrototype && (
        <>
          <div className="w-px h-4 bg-brand-shade3/20" />
          <div className="relative">
            <button
              onClick={() => setProtoSchemaDropdown(v => !v)}
              className="bg-brand-dark border border-brand-shade3/20 rounded-btn text-brand-light text-xs px-2.5 py-1 cursor-pointer flex items-center gap-1.5 font-mono"
            >
              {protoSchema}
              <span className="text-brand-shade3 text-[10px]">&#9662;</span>
            </button>
            {protoSchemaDropdown && (
              <div className="absolute top-full left-0 mt-1 bg-brand-dark-alt border border-brand-shade3/20 rounded-card z-50 min-w-[180px] shadow-lg">
                {protoSchemas.map(s => (
                  <div
                    key={s}
                    className={`flex items-center justify-between text-xs px-3 py-[7px] font-mono transition-colors ${
                      s === protoSchema ? 'bg-brand-accent/[0.13] text-brand-accent' : 'text-brand-light hover:bg-brand-shade3/20'
                    }`}
                  >
                    <button
                      className="flex-1 text-left cursor-pointer"
                      onClick={() => { setProtoSchema(s); setProtoSchemaDropdown(false); }}
                    >
                      {s}
                    </button>
                    <div className="flex items-center gap-1 ml-2 shrink-0">
                      <button
                        title="Rename"
                        className="text-brand-shade3 hover:text-brand-light transition-colors"
                        onClick={(e) => {
                          e.stopPropagation();
                          const newName = window.prompt('Rename schema:', s);
                          if (newName && newName.trim() && newName.trim() !== s) {
                            setProtoSchemas(prev => prev.map(n => n === s ? newName.trim() : n));
                            if (protoSchema === s) setProtoSchema(newName.trim());
                          }
                        }}
                      >
                        <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M17 3a2.85 2.85 0 0 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/></svg>
                      </button>
                      {s !== protoSchema && (
                        <button
                          title="Delete"
                          className="text-brand-shade3 hover:text-red-400 transition-colors"
                          onClick={(e) => {
                            e.stopPropagation();
                            setProtoSchemas(prev => prev.filter(n => n !== s));
                          }}
                        >
                          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 6L6 18M6 6l12 12" /></svg>
                        </button>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
          <button
            className="px-2.5 py-1 text-xs text-brand-shade2 border border-brand-shade3/20 rounded-btn font-mono hover:text-brand-light transition-colors"
            onClick={() => {
              const name = window.prompt('New schema name:');
              if (name && name.trim()) {
                const trimmed = name.trim();
                setProtoSchemas(prev => [...prev, trimmed]);
                setProtoSchema(trimmed);
              }
            }}
          >
            + Schema
          </button>
        </>
      )}

      <div className="flex-1" />

      {/* Saved indicator — production only */}
      {!isPrototype && savedIndicator && (
        <span className={`text-[10px] transition-opacity ${savedIndicator === 'saving' ? 'text-brand-shade3' : 'text-green-400'}`}>
          {savedIndicator === 'saving' ? 'Saving\u2026' : 'All changes saved'}
        </span>
      )}

      <button
        onClick={onAutoLayout}
        className="px-3 py-1.5 text-xs text-brand-shade2 border border-brand-shade3/30 rounded-btn hover:text-brand-light hover:border-brand-shade3 transition-colors"
      >
        Auto Layout
      </button>
      {!isPrototype && <ExportButton />}
      {!isPrototype && <ImportButton onImported={onRefetch} />}
      <button
        onClick={onAddTrigger}
        className="px-3 py-1.5 text-xs text-brand-shade2 border border-brand-shade3/30 rounded-btn hover:text-brand-light hover:border-brand-shade3 transition-colors"
      >
        + Add Trigger
      </button>
      <button
        onClick={onAddAgent}
        className="px-3 py-1.5 text-xs bg-brand-accent text-brand-light rounded-btn hover:bg-brand-accent-hover transition-colors"
      >
        + Add Agent
      </button>
    </div>
  );
}
