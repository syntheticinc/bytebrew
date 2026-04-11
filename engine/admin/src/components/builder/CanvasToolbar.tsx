import React, { useState } from 'react';
import { ExportButton, ImportButton } from './BuilderExportImport';
import ConfirmDialog from '../ConfirmDialog';

interface CanvasToolbarProps {
  isPrototype: boolean;
  savedIndicator: 'saved' | 'saving' | null;
  onAutoLayout: () => void;
  onRefetch: () => void;
  onAddAgent: () => void;
  onAddTrigger: (type: 'webhook' | 'cron' | 'chat') => void;
  // Schema name for production mode (from URL param)
  schemaName?: string;
  onBack?: () => void;
  isSystemSchema?: boolean;
  onRestoreDefaults?: () => Promise<void>;
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
  schemaName,
  onBack,
  isSystemSchema,
  onRestoreDefaults,
  protoSchema,
  protoSchemas,
  setProtoSchema,
  setProtoSchemas,
}: CanvasToolbarProps) {
  const [protoSchemaDropdown, setProtoSchemaDropdown] = useState(false);
  const [triggerDropdown, setTriggerDropdown] = useState(false);
  const [showRestoreConfirm, setShowRestoreConfirm] = useState(false);
  const [restoring, setRestoring] = useState(false);

  return (
    <div className="flex items-center gap-3 px-4 h-12 border-b border-brand-shade3/15 bg-brand-dark-alt flex-shrink-0 flex-wrap">
      {/* Back button + schema name in production mode */}
      {!isPrototype && schemaName && onBack && (
        <>
          <button
            onClick={onBack}
            className="flex items-center gap-1 text-xs text-brand-shade3 hover:text-brand-light transition-colors"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M19 12H5M12 19l-7-7 7-7" />
            </svg>
            Schemas
          </button>
          <div className="w-px h-4 bg-brand-shade3/20" />
        </>
      )}
      <span className="text-sm font-semibold text-brand-light">
        {!isPrototype && schemaName ? schemaName : 'Agent Builder'}
      </span>

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
      {isSystemSchema && onRestoreDefaults && (
        <button
          onClick={() => setShowRestoreConfirm(true)}
          className="px-3 py-1.5 text-xs text-amber-400 border border-amber-500/30 rounded-btn hover:bg-amber-500/10 hover:border-amber-500/50 transition-colors"
        >
          Restore Defaults
        </button>
      )}
      <div className="relative">
        <button
          onClick={() => setTriggerDropdown((v) => !v)}
          className="px-3 py-1.5 text-xs text-brand-shade2 border border-brand-shade3/30 rounded-btn hover:text-brand-light hover:border-brand-shade3 transition-colors"
        >
          + Add Trigger <span className="text-[10px]">&#9662;</span>
        </button>
        {triggerDropdown && (
          <div
            className="absolute top-full right-0 mt-1 bg-brand-dark-alt border border-brand-shade3/20 rounded-card z-50 min-w-[140px] shadow-lg py-1"
            onMouseLeave={() => setTriggerDropdown(false)}
          >
            {(['webhook', 'cron', 'chat'] as const).map((t) => (
              <button
                key={t}
                className="w-full px-3 py-1.5 text-left text-xs text-brand-light hover:bg-brand-accent/10 transition-colors capitalize"
                onClick={() => { onAddTrigger(t); setTriggerDropdown(false); }}
              >
                {t === 'webhook' ? 'Webhook' : t === 'cron' ? 'Cron Schedule' : 'Chat'}
              </button>
            ))}
          </div>
        )}
      </div>
      <button
        onClick={onAddAgent}
        className="px-3 py-1.5 text-xs bg-brand-accent text-brand-light rounded-btn hover:bg-brand-accent-hover transition-colors"
      >
        + Add Agent
      </button>

      <ConfirmDialog
        open={showRestoreConfirm}
        onClose={() => setShowRestoreConfirm(false)}
        onConfirm={async () => {
          if (!onRestoreDefaults) return;
          setRestoring(true);
          try {
            await onRestoreDefaults();
          } finally {
            setRestoring(false);
            setShowRestoreConfirm(false);
          }
        }}
        title="Restore Defaults"
        message="Reset builder-schema to factory defaults? This will restore the original agent settings, triggers, and connections."
        confirmLabel="Restore"
        loading={restoring}
        variant="warning"
      />
    </div>
  );
}
