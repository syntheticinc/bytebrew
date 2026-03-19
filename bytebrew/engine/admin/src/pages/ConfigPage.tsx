import { useState, useRef } from 'react';
import { api } from '../api/client';

export default function ConfigPage() {
  const [reloading, setReloading] = useState(false);
  const [reloadResult, setReloadResult] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);
  const [importing, setImporting] = useState(false);
  const [importResult, setImportResult] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  async function handleReload() {
    setReloading(true);
    setError(null);
    setReloadResult(null);
    try {
      const res = await api.reloadConfig();
      setReloadResult(`Config reloaded. ${res.agents_count} agents loaded.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Reload failed');
    } finally {
      setReloading(false);
    }
  }

  async function handleExport() {
    setExporting(true);
    setError(null);
    try {
      const yaml = await api.exportConfig();
      const blob = new Blob([yaml], { type: 'application/x-yaml' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'bytebrew-config.yaml';
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Export failed');
    } finally {
      setExporting(false);
    }
  }

  async function handleImport() {
    const file = fileInputRef.current?.files?.[0];
    if (!file) return;

    setImporting(true);
    setError(null);
    setImportResult(null);
    try {
      const text = await file.text();
      const res = await api.importConfig(text);
      setImportResult(`Config imported. ${res.agents_count} agents loaded.`);
      if (fileInputRef.current) fileInputRef.current.value = '';
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Import failed');
    } finally {
      setImporting(false);
    }
  }

  return (
    <div className="max-w-2xl">
      <h1 className="text-2xl font-bold text-brand-light mb-6">Configuration</h1>

      {error && (
        <div className="mb-4 p-3 bg-red-500/10 border border-red-500/30 rounded-btn text-sm text-red-400">
          {error}
        </div>
      )}
      {reloadResult && (
        <div className="mb-4 p-3 bg-status-active/10 border border-status-active/30 rounded-btn text-sm text-status-active">
          {reloadResult}
        </div>
      )}
      {importResult && (
        <div className="mb-4 p-3 bg-status-active/10 border border-status-active/30 rounded-btn text-sm text-status-active">
          {importResult}
        </div>
      )}

      {/* Reload */}
      <section className="mb-8">
        <div className="bg-brand-dark-alt rounded-card border border-brand-shade3/15 p-5">
          <h2 className="text-lg font-semibold text-brand-light mb-2">Hot Reload</h2>
          <p className="text-sm text-brand-shade3 mb-4">
            Reload agent configuration from the database. Changes take effect immediately.
          </p>
          <button
            onClick={handleReload}
            disabled={reloading}
            className="px-4 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover disabled:opacity-50 transition-colors"
          >
            {reloading ? 'Reloading...' : 'Reload Config'}
          </button>
        </div>
      </section>

      {/* Export */}
      <section className="mb-8">
        <div className="bg-brand-dark-alt rounded-card border border-brand-shade3/15 p-5">
          <h2 className="text-lg font-semibold text-brand-light mb-2">Export</h2>
          <p className="text-sm text-brand-shade3 mb-4">
            Download current configuration as YAML file.
          </p>
          <button
            onClick={handleExport}
            disabled={exporting}
            className="px-4 py-2 bg-brand-dark border border-brand-shade3/30 text-brand-shade2 rounded-btn text-sm font-medium hover:bg-brand-dark hover:text-brand-light disabled:opacity-50 transition-colors"
          >
            {exporting ? 'Exporting...' : 'Export YAML'}
          </button>
        </div>
      </section>

      {/* Import */}
      <section>
        <div className="bg-brand-dark-alt rounded-card border border-brand-shade3/15 p-5">
          <h2 className="text-lg font-semibold text-brand-light mb-2">Import</h2>
          <p className="text-sm text-brand-shade3 mb-4">
            Upload a YAML config file. It will be parsed and saved to the database, then reloaded.
          </p>
          <div className="flex items-center gap-3">
            <input
              ref={fileInputRef}
              type="file"
              accept=".yaml,.yml"
              className="text-sm text-brand-shade3 file:mr-4 file:py-2 file:px-4 file:rounded-btn file:border-0 file:text-sm file:font-medium file:bg-brand-dark file:text-brand-shade2 hover:file:bg-brand-shade3/20"
            />
            <button
              onClick={handleImport}
              disabled={importing}
              className="px-4 py-2 bg-brand-dark border border-brand-shade3/30 text-brand-shade2 rounded-btn text-sm font-medium hover:text-brand-light disabled:opacity-50 transition-colors"
            >
              {importing ? 'Importing...' : 'Import'}
            </button>
          </div>
        </div>
      </section>
    </div>
  );
}
