import { useState } from 'react';
import { api } from '../../api/client';
import { storeExportHash } from './DriftNotification';

// ─── Export Button ──────────────────────────────────────────────────────────

interface ExportButtonProps {
  className?: string;
  onExportSuccess?: () => void;
}

export function ExportButton({ className, onExportSuccess }: ExportButtonProps) {
  const [exporting, setExporting] = useState(false);
  const [error, setError] = useState('');

  async function handleExport() {
    setError('');
    setExporting(true);
    try {
      const yaml = await api.exportConfig();
      const yamlStr = typeof yaml === 'string' ? yaml : String(yaml);

      // Store hash for drift detection
      await storeExportHash(yamlStr);

      const blob = new Blob([yamlStr], { type: 'application/x-yaml' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'bytebrew-config.yaml';
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
      onExportSuccess?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Export failed');
    } finally {
      setExporting(false);
    }
  }

  return (
    <div className="relative">
      <button
        onClick={handleExport}
        disabled={exporting}
        className={`px-3 py-1.5 text-xs text-brand-shade2 border border-brand-shade3/30 rounded-btn hover:text-brand-light hover:border-brand-shade3 disabled:opacity-40 transition-colors inline-flex items-center gap-1.5 ${className ?? ''}`}
      >
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4" />
          <polyline points="7 10 12 15 17 10" />
          <line x1="12" y1="15" x2="12" y2="3" />
        </svg>
        {exporting ? 'Exporting...' : 'Export'}
      </button>
      {error && (
        <div className="absolute top-full left-0 mt-1 px-2 py-1 bg-red-900/80 border border-red-500/30 rounded text-[10px] text-red-300 whitespace-nowrap z-50">
          {error}
        </div>
      )}
    </div>
  );
}

// ─── Import Button ──────────────────────────────────────────────────────────

interface ImportButtonProps {
  onImported: () => void;
  className?: string;
}

interface ImportResult {
  imported: boolean;
  agents_count: number;
}

export function ImportButton({ onImported, className }: ImportButtonProps) {
  const [showPreview, setShowPreview] = useState(false);
  const [yamlContent, setYamlContent] = useState('');
  const [fileName, setFileName] = useState('');
  const [importing, setImporting] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState<ImportResult | null>(null);

  function handleFileSelect() {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.yaml,.yml';
    input.onchange = async () => {
      const file = input.files?.[0];
      if (!file) return;

      try {
        const text = await file.text();
        setYamlContent(text);
        setFileName(file.name);
        setError('');
        setResult(null);
        setShowPreview(true);
      } catch {
        setError('Failed to read file');
      }
    };
    input.click();
  }

  async function handleApply() {
    setError('');
    setImporting(true);
    try {
      const res = await api.importConfig(yamlContent);
      setResult(res);
      onImported();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Import failed');
    } finally {
      setImporting(false);
    }
  }

  function handleClose() {
    setShowPreview(false);
    setYamlContent('');
    setFileName('');
    setError('');
    setResult(null);
  }

  return (
    <>
      <button
        onClick={handleFileSelect}
        className={`px-3 py-1.5 text-xs text-brand-shade2 border border-brand-shade3/30 rounded-btn hover:text-brand-light hover:border-brand-shade3 transition-colors inline-flex items-center gap-1.5 ${className ?? ''}`}
      >
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4" />
          <polyline points="17 8 12 3 7 8" />
          <line x1="12" y1="3" x2="12" y2="15" />
        </svg>
        Import
      </button>

      {/* Preview Modal */}
      {showPreview && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={handleClose} />
          <div className="relative bg-brand-dark-alt border border-brand-shade3/20 rounded-lg shadow-xl w-[560px] max-h-[80vh] flex flex-col animate-modal-in">
            {/* Header */}
            <div className="px-5 py-4 border-b border-brand-shade3/15 flex items-center justify-between flex-shrink-0">
              <div>
                <h3 className="text-sm font-semibold text-brand-light">Import Configuration</h3>
                <p className="text-[11px] text-brand-shade3 mt-0.5">{fileName}</p>
              </div>
              <button onClick={handleClose} className="text-brand-shade3 hover:text-brand-light p-1 transition-colors">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            {/* Preview */}
            <div className="flex-1 overflow-y-auto p-5 min-h-0">
              <p className="text-xs text-brand-shade3 mb-3">Review the configuration before applying. Existing entities will be updated by name, new ones created. Entities not in the file will remain unchanged (upsert, not full overwrite).</p>
              <pre className="text-[11px] font-mono text-brand-shade2 bg-brand-dark border border-brand-shade3/20 rounded-card p-3 overflow-auto whitespace-pre leading-relaxed max-h-[340px]">
                {yamlContent}
              </pre>
            </div>

            {/* Result / Error */}
            {result && (
              <div className="mx-5 mb-3 px-3 py-2 bg-emerald-900/20 border border-emerald-500/20 rounded text-xs text-emerald-400">
                Configuration imported successfully. {result.agents_count} agent{result.agents_count !== 1 ? 's' : ''} loaded.
              </div>
            )}
            {error && (
              <div className="mx-5 mb-3 px-3 py-2 bg-red-900/20 border border-red-500/20 rounded text-xs text-red-400">
                {error}
              </div>
            )}

            {/* Footer */}
            <div className="px-5 py-4 border-t border-brand-shade3/15 flex gap-3 flex-shrink-0">
              {result ? (
                <button
                  onClick={handleClose}
                  className="flex-1 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover transition-colors"
                >
                  Done
                </button>
              ) : (
                <>
                  <button
                    onClick={handleApply}
                    disabled={importing}
                    className="flex-1 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover disabled:opacity-40 transition-colors"
                  >
                    {importing ? 'Applying...' : 'Apply Configuration'}
                  </button>
                  <button
                    onClick={handleClose}
                    className="px-4 py-2 border border-brand-shade3/30 text-brand-shade2 rounded-btn text-sm hover:text-brand-light transition-colors"
                  >
                    Cancel
                  </button>
                </>
              )}
            </div>
          </div>
        </div>
      )}
    </>
  );
}
