import { useState, useCallback } from 'react';
import type { Edge } from '@xyflow/react';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface FieldMapping {
  source: string;
  target: string;
}

interface EdgeConfigPanelProps {
  edge: Edge;
  onClose: () => void;
  onSave?: (edge: Edge, config: EdgeConfig) => void;
  onDelete?: (edgeId: string) => void;
}

export interface EdgeConfig {
  mode: ConfigMode;
  fieldMappings: FieldMapping[];
  promptTemplate: string;
}

type ConfigMode = 'full' | 'field' | 'prompt';

const EDGE_TYPE_COLORS: Record<string, { bg: string; text: string; border: string }> = {
  can_spawn: { bg: 'bg-red-500/15', text: 'text-red-400', border: 'border-red-500/30' },
  triggers:  { bg: 'bg-purple-500/15', text: 'text-purple-400', border: 'border-purple-500/30' },
  flow:      { bg: 'bg-green-500/15', text: 'text-green-400', border: 'border-green-500/30' },
  transfer:  { bg: 'bg-blue-500/15', text: 'text-blue-400', border: 'border-blue-500/30' },
};

const MODE_OPTIONS: { value: ConfigMode; label: string; description: string }[] = [
  { value: 'full', label: 'Full Output', description: 'Next agent receives the complete output' },
  { value: 'field', label: 'Field Mapping', description: 'Map specific fields from source to target' },
  { value: 'prompt', label: 'Custom Prompt', description: 'Transform output with a template' },
];

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function EdgeConfigPanel({ edge, onClose, onSave, onDelete }: EdgeConfigPanelProps) {
  const edgeData = (edge.data ?? {}) as Record<string, unknown>;

  const [mode, setMode] = useState<ConfigMode>(
    (edgeData.configMode as ConfigMode) ?? 'full',
  );
  const [fieldMappings, setFieldMappings] = useState<FieldMapping[]>(
    (edgeData.fieldMappings as FieldMapping[]) ?? [{ source: '', target: '' }],
  );
  const [promptTemplate, setPromptTemplate] = useState(
    (edgeData.promptTemplate as string) ?? '{{output}}',
  );

  const edgeType = (edge.label as string) ?? 'flow';
  const defaultColors = { bg: 'bg-green-500/15', text: 'text-green-400', border: 'border-green-500/30' };
  const resolved = EDGE_TYPE_COLORS[edgeType];
  const colors = resolved !== undefined ? resolved : defaultColors;

  const addMapping = useCallback(() => {
    setFieldMappings((prev) => [...prev, { source: '', target: '' }]);
  }, []);

  const removeMapping = useCallback((index: number) => {
    setFieldMappings((prev) => prev.filter((_, i) => i !== index));
  }, []);

  const updateMapping = useCallback((index: number, field: 'source' | 'target', value: string) => {
    setFieldMappings((prev) =>
      prev.map((m, i) => (i === index ? { ...m, [field]: value } : m)),
    );
  }, []);

  const handleSave = () => {
    onSave?.(edge, { mode, fieldMappings, promptTemplate });
  };

  const handleDelete = () => {
    onDelete?.(edge.id);
  };

  // Preview
  const preview = (() => {
    switch (mode) {
      case 'full':
        return '{ ...full agent output }';
      case 'field': {
        const mapped = fieldMappings.filter((m) => m.source && m.target);
        if (mapped.length === 0) return '(add field mappings)';
        return mapped.map((m) => `${m.target}: output.${m.source}`).join('\n');
      }
      case 'prompt':
        return promptTemplate || '(empty template)';
    }
  })();

  return (
    <div className="w-80 bg-brand-dark-surface border-l border-brand-shade3/10 flex flex-col h-full flex-shrink-0 animate-slide-in-right">
      {/* Header */}
      <div className="px-4 py-3 border-b border-brand-shade3/15 flex items-center justify-between">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-xs text-brand-shade2 font-mono truncate">{edge.source}</span>
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="text-brand-shade3 flex-shrink-0">
            <path d="M5 12h14M12 5l7 7-7 7" />
          </svg>
          <span className="text-xs text-brand-shade2 font-mono truncate">{edge.target}</span>
        </div>
        <button
          onClick={onClose}
          className="p-1 text-brand-shade3 hover:text-brand-light transition-colors flex-shrink-0"
          title="Close"
          aria-label="Close edge config panel"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M18 6L6 18M6 6l12 12" />
          </svg>
        </button>
      </div>

      {/* Edge type badge (read-only) */}
      <div className="px-4 py-3 border-b border-brand-shade3/15">
        <span className="text-[10px] text-brand-shade3 uppercase tracking-wider">Edge type</span>
        <div className="mt-1.5">
          <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium border ${colors.bg} ${colors.text} ${colors.border}`}>
            {edgeType}
          </span>
        </div>
      </div>

      {/* Configuration mode */}
      <div className="px-4 py-3 flex-1 overflow-y-auto">
        <span className="text-[10px] text-brand-shade3 uppercase tracking-wider">Configuration</span>
        <div className="mt-2 space-y-2">
          {MODE_OPTIONS.map((opt) => (
            <label
              key={opt.value}
              className={`flex items-start gap-2.5 p-2 rounded-card border cursor-pointer transition-colors ${
                mode === opt.value
                  ? 'border-brand-accent/40 bg-brand-accent/5'
                  : 'border-brand-shade3/15 hover:border-brand-shade3/30'
              }`}
            >
              <input
                type="radio"
                name="edge-config-mode"
                value={opt.value}
                checked={mode === opt.value}
                onChange={() => setMode(opt.value)}
                className="mt-0.5 accent-brand-accent"
              />
              <div className="min-w-0">
                <div className="text-xs text-brand-light font-medium">{opt.label}</div>
                <div className="text-[11px] text-brand-shade3 leading-snug mt-0.5">{opt.description}</div>
              </div>
            </label>
          ))}
        </div>

        {/* Field Mapping — key-value pairs */}
        {mode === 'field' && (
          <div className="mt-3">
            <div className="flex items-center justify-between mb-2">
              <span className="text-[10px] text-brand-shade3 uppercase tracking-wider">Field Mappings</span>
              <button
                onClick={addMapping}
                className="text-[10px] text-brand-accent hover:text-brand-accent-hover transition-colors font-medium"
              >
                + Add row
              </button>
            </div>
            <div className="space-y-2">
              {fieldMappings.map((mapping, i) => (
                <div key={i} className="flex items-center gap-1.5">
                  <input
                    type="text"
                    value={mapping.source}
                    onChange={(e) => updateMapping(i, 'source', e.target.value)}
                    placeholder="source field"
                    className="flex-1 px-2 py-1.5 bg-brand-dark border border-brand-shade3/30 rounded-card text-xs text-brand-light font-mono focus:outline-none focus:border-brand-accent placeholder-brand-shade3/50 transition-colors min-w-0"
                  />
                  <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="text-brand-shade3 flex-shrink-0">
                    <path d="M5 12h14M12 5l7 7-7 7" />
                  </svg>
                  <input
                    type="text"
                    value={mapping.target}
                    onChange={(e) => updateMapping(i, 'target', e.target.value)}
                    placeholder="target field"
                    className="flex-1 px-2 py-1.5 bg-brand-dark border border-brand-shade3/30 rounded-card text-xs text-brand-light font-mono focus:outline-none focus:border-brand-accent placeholder-brand-shade3/50 transition-colors min-w-0"
                  />
                  <button
                    onClick={() => removeMapping(i)}
                    className="p-1 text-brand-shade3 hover:text-red-400 transition-colors flex-shrink-0"
                    title="Remove mapping"
                  >
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                      <path d="M18 6L6 18M6 6l12 12" />
                    </svg>
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Custom prompt textarea */}
        {mode === 'prompt' && (
          <div className="mt-3">
            <span className="text-[10px] text-brand-shade3 uppercase tracking-wider">Prompt template</span>
            <textarea
              value={promptTemplate}
              onChange={(e) => setPromptTemplate(e.target.value)}
              placeholder="Use {{output}} or {{output.field}} to reference agent output"
              rows={4}
              className="mt-1 w-full px-2 py-1.5 bg-brand-dark border border-brand-shade3/30 rounded-card text-xs text-brand-light font-mono focus:outline-none focus:border-brand-accent placeholder-brand-shade3/50 transition-colors resize-y leading-relaxed"
            />
            <p className="mt-1 text-[10px] text-brand-shade3/70">
              Variables: <code className="text-brand-shade2">{'{{output}}'}</code>, <code className="text-brand-shade2">{'{{output.field_name}}'}</code>
            </p>
          </div>
        )}

        {/* Preview */}
        <div className="mt-4">
          <span className="text-[10px] text-brand-shade3 uppercase tracking-wider">Preview</span>
          <div className="mt-1.5 px-2 py-2 bg-brand-dark border border-brand-shade3/20 rounded-card">
            <pre className="text-[11px] text-brand-shade2 font-mono whitespace-pre-wrap break-all leading-relaxed">
              {preview}
            </pre>
          </div>
        </div>
      </div>

      {/* Action buttons */}
      <div className="px-4 py-3 border-t border-brand-shade3/15 flex items-center gap-2">
        <button
          onClick={handleSave}
          className="flex-1 px-3 py-1.5 bg-brand-accent hover:bg-brand-accent-hover text-brand-light text-xs font-medium rounded-card transition-colors"
        >
          Save
        </button>
        <button
          onClick={handleDelete}
          className="px-3 py-1.5 bg-red-500/10 hover:bg-red-500/20 text-red-400 text-xs font-medium rounded-card border border-red-500/20 transition-colors"
        >
          Delete Edge
        </button>
      </div>
    </div>
  );
}
