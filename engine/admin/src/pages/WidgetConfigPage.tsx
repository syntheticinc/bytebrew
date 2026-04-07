import { useState, useEffect, useCallback } from 'react';
import FormField from '../components/FormField';
import WidgetPreview from '../components/WidgetPreview';
import { api } from '../api/client';
import type { WidgetConfig, WidgetPosition, WidgetSize } from '../types';

const POSITION_OPTIONS = [
  { value: 'bottom-right', label: 'Bottom Right' },
  { value: 'bottom-left', label: 'Bottom Left' },
];

const SIZE_OPTIONS = [
  { value: 'compact', label: 'Compact' },
  { value: 'standard', label: 'Standard' },
  { value: 'full', label: 'Full' },
];

export default function WidgetConfigPage() {
  const [widgets, setWidgets] = useState<WidgetConfig[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [schemas, setSchemas] = useState<{ value: string; label: string }[]>([]);
  const [saving, setSaving] = useState(false);
  const [copiedEmbed, setCopiedEmbed] = useState(false);
  const [copiedSelfHosted, setCopiedSelfHosted] = useState(false);
  const [copiedId, setCopiedId] = useState(false);
  const [loading, setLoading] = useState(true);

  // Load widgets and schemas
  useEffect(() => {
    Promise.all([api.listWidgets(), api.listSchemas()])
      .then(([w, s]) => {
        setWidgets(w);
        if (w.length > 0 && !selectedId) setSelectedId(w[0]!.id);
        setSchemas(s.map((sc) => ({ value: sc.name, label: sc.name })));
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const selected = widgets.find((w) => w.id === selectedId) ?? null;

  const updateWidget = useCallback(
    <K extends keyof WidgetConfig>(key: K, value: WidgetConfig[K]) => {
      setWidgets((prev) =>
        prev.map((w) => (w.id === selectedId ? { ...w, [key]: value } : w)),
      );
    },
    [selectedId],
  );

  const handleCreate = useCallback(() => {
    const firstSchema = schemas[0]?.value ?? '';
    api
      .createWidget({
        name: 'New Widget',
        schema: firstSchema,
        status: 'disabled',
        primary_color: '#6366f1',
        position: 'bottom-right',
        size: 'standard',
        welcome_message: 'Hello! How can I help?',
        placeholder_text: 'Type your message...',
        avatar_url: '',
        domain_whitelist: '',
      })
      .then((w) => {
        setWidgets((prev) => [...prev, w]);
        setSelectedId(w.id);
      })
      .catch(() => {});
  }, [schemas]);

  const handleSave = useCallback(() => {
    if (!selected) return;
    setSaving(true);
    const { id, created_at, ...data } = selected;
    void created_at;
    api
      .updateWidget(id, data)
      .then((updated) => {
        setWidgets((prev) => prev.map((w) => (w.id === id ? updated : w)));
      })
      .catch(() => {})
      .finally(() => setSaving(false));
  }, [selected]);

  const handleDelete = useCallback(() => {
    if (!selected) return;
    api
      .deleteWidget(selected.id)
      .then(() => {
        setWidgets((prev) => {
          const next = prev.filter((w) => w.id !== selected.id);
          setSelectedId(next[0]?.id ?? null);
          return next;
        });
      })
      .catch(() => {});
  }, [selected]);

  function copyToClipboard(text: string, setter: (v: boolean) => void) {
    navigator.clipboard
      .writeText(text)
      .then(() => {
        setter(true);
        setTimeout(() => setter(false), 1500);
      })
      .catch(() => {});
  }

  const cloudEmbed = selected
    ? `<script src="https://bytebrew.ai/widget/${selected.id}.js"></script>`
    : '';
  const selfHostedEmbed = selected
    ? `<script src="https://your-domain.com/widget/${selected.id}.js"></script>`
    : '';

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16">
        <span className="text-sm text-brand-shade3 font-mono">Loading widgets...</span>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full min-h-0">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-lg font-semibold text-brand-light font-mono">Widget Configuration</h1>
        <button
          type="button"
          onClick={handleCreate}
          className="px-4 py-1.5 bg-brand-accent text-brand-light rounded-btn text-sm font-medium font-mono hover:bg-brand-accent-hover transition-colors"
        >
          + Create Widget
        </button>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="grid grid-cols-12 gap-6">
          {/* Widget list */}
          <div className="col-span-3 space-y-2">
            {widgets.map((w) => (
              <button
                key={w.id}
                type="button"
                onClick={() => setSelectedId(w.id)}
                className={[
                  'w-full text-left px-4 py-3 rounded-card border transition-colors',
                  w.id === selectedId
                    ? 'bg-brand-dark-surface border-brand-accent/50'
                    : 'bg-brand-dark-surface border-brand-shade3/10 hover:border-brand-shade3/30',
                ].join(' ')}
              >
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm font-medium text-brand-light font-mono truncate">{w.name}</span>
                  <span
                    className={[
                      'text-[10px] font-semibold px-2 py-0.5 rounded-card font-mono uppercase tracking-wider shrink-0 ml-2',
                      w.status === 'active'
                        ? 'bg-status-active/15 text-status-active'
                        : 'bg-brand-shade3/15 text-brand-shade3',
                    ].join(' ')}
                  >
                    {w.status}
                  </span>
                </div>
                <p className="text-xs text-brand-shade3 font-mono">{w.schema}</p>
                <div className="flex items-center gap-1.5 mt-1">
                  <div
                    className="w-3 h-3 rounded-full shrink-0 border border-white/10"
                    style={{ backgroundColor: w.primary_color }}
                  />
                  <span className="text-[10px] text-brand-shade3/60 font-mono">{w.id}</span>
                </div>
              </button>
            ))}
            {widgets.length === 0 && (
              <p className="text-xs text-brand-shade3 font-mono text-center py-8">
                No widgets yet. Create one to get started.
              </p>
            )}
          </div>

          {/* Widget config form */}
          {selected ? (
            <div className="col-span-5 space-y-4">
              {/* Identity */}
              <div className="bg-brand-dark-surface border border-brand-shade3/10 rounded-card p-4">
                <h2 className="text-xs font-semibold text-brand-shade3 uppercase tracking-widest mb-3 font-mono">
                  Identity
                </h2>
                <div>
                  <label className="block text-sm font-medium text-brand-light mb-1">Widget ID</label>
                  <div className="flex items-center gap-2">
                    <input
                      type="text"
                      value={selected.id}
                      readOnly
                      className="flex-1 px-3 py-2 bg-brand-dark border border-brand-shade3/30 rounded-card text-sm text-brand-shade2 font-mono cursor-default"
                    />
                    <button
                      type="button"
                      onClick={() => copyToClipboard(selected.id, setCopiedId)}
                      className="px-3 py-2 border border-brand-shade3/30 rounded-card text-xs text-brand-shade2 font-mono hover:text-brand-light hover:border-brand-shade3/60 transition-colors"
                    >
                      {copiedId ? 'Copied' : 'Copy'}
                    </button>
                  </div>
                </div>
                <div className="mt-3">
                  <FormField
                    label="Name"
                    value={selected.name}
                    onChange={(v) => updateWidget('name', v)}
                    hint="Display name for this widget"
                  />
                </div>
                <div className="mt-3">
                  <FormField
                    label="Schema"
                    type="select"
                    value={selected.schema}
                    onChange={(v) => updateWidget('schema', v)}
                    options={schemas}
                    hint="Agent schema handling conversations"
                  />
                </div>
                <div className="mt-3">
                  <FormField
                    label="Status"
                    type="select"
                    value={selected.status}
                    onChange={(v) => updateWidget('status', v as WidgetConfig['status'])}
                    options={[
                      { value: 'active', label: 'Active' },
                      { value: 'disabled', label: 'Disabled' },
                    ]}
                    hint="Disabled widgets won't load on client sites"
                  />
                </div>
              </div>

              {/* Styling */}
              <div className="bg-brand-dark-surface border border-brand-shade3/10 rounded-card p-4">
                <h2 className="text-xs font-semibold text-brand-shade3 uppercase tracking-widest mb-3 font-mono">
                  Appearance
                </h2>
                <div className="space-y-3">
                  <div>
                    <label className="block text-sm font-medium text-brand-light mb-1">Primary Color</label>
                    <div className="flex items-center gap-2">
                      <input
                        type="color"
                        value={selected.primary_color}
                        onChange={(e) => updateWidget('primary_color', e.target.value)}
                        className="w-9 h-9 rounded-card border border-brand-shade3/30 cursor-pointer bg-transparent p-0.5"
                      />
                      <input
                        type="text"
                        value={selected.primary_color}
                        onChange={(e) => updateWidget('primary_color', e.target.value)}
                        className="flex-1 px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light font-mono focus:outline-none focus:border-brand-accent transition-colors"
                      />
                    </div>
                  </div>
                  <FormField
                    label="Position"
                    type="select"
                    value={selected.position}
                    onChange={(v) => updateWidget('position', v as WidgetPosition)}
                    options={POSITION_OPTIONS}
                    hint="Widget placement on the page"
                  />
                  <FormField
                    label="Size"
                    type="select"
                    value={selected.size}
                    onChange={(v) => updateWidget('size', v as WidgetSize)}
                    options={SIZE_OPTIONS}
                    hint="Chat window dimensions"
                  />
                  <FormField
                    label="Avatar URL"
                    value={selected.avatar_url}
                    onChange={(v) => updateWidget('avatar_url', v)}
                    placeholder="https://example.com/avatar.png"
                    hint="Agent avatar in widget header (optional)"
                  />
                </div>
              </div>

              {/* Content */}
              <div className="bg-brand-dark-surface border border-brand-shade3/10 rounded-card p-4">
                <h2 className="text-xs font-semibold text-brand-shade3 uppercase tracking-widest mb-3 font-mono">
                  Content
                </h2>
                <div className="space-y-3">
                  <FormField
                    label="Welcome Message"
                    value={selected.welcome_message}
                    onChange={(v) => updateWidget('welcome_message', v)}
                    hint="Greeting shown when the widget opens"
                  />
                  <FormField
                    label="Placeholder Text"
                    value={selected.placeholder_text}
                    onChange={(v) => updateWidget('placeholder_text', v)}
                    hint="Input placeholder text"
                  />
                </div>
              </div>

              {/* Security */}
              <div className="bg-brand-dark-surface border border-brand-shade3/10 rounded-card p-4">
                <h2 className="text-xs font-semibold text-brand-shade3 uppercase tracking-widest mb-3 font-mono">
                  Security
                </h2>
                <FormField
                  label="Domain Whitelist"
                  value={selected.domain_whitelist}
                  onChange={(v) => updateWidget('domain_whitelist', v)}
                  placeholder="example.com, app.example.com"
                  hint="Comma-separated list of allowed embed domains (empty = allow all)"
                />
              </div>

              {/* Embed code */}
              <div className="bg-brand-dark-surface border border-brand-shade3/10 rounded-card p-4">
                <h2 className="text-xs font-semibold text-brand-shade3 uppercase tracking-widest mb-3 font-mono">
                  Embed Code
                </h2>
                <div className="space-y-3">
                  <div>
                    <p className="text-xs text-brand-shade3 mb-1.5 font-mono">Cloud (bytebrew.ai)</p>
                    <div className="relative">
                      <pre className="bg-brand-dark-alt px-4 py-3 rounded-card text-xs text-brand-shade2 font-mono overflow-x-auto border border-brand-shade3/20">
                        <code>{cloudEmbed}</code>
                      </pre>
                      <button
                        type="button"
                        onClick={() => copyToClipboard(cloudEmbed, setCopiedEmbed)}
                        className="absolute top-2 right-2 px-2.5 py-1 bg-brand-dark border border-brand-shade3/30 rounded-btn text-[11px] text-brand-shade2 font-mono hover:text-brand-light transition-colors"
                      >
                        {copiedEmbed ? 'Copied' : 'Copy'}
                      </button>
                    </div>
                  </div>
                  <div>
                    <p className="text-xs text-brand-shade3 mb-1.5 font-mono">Self-hosted</p>
                    <div className="relative">
                      <pre className="bg-brand-dark-alt px-4 py-3 rounded-card text-xs text-brand-shade2 font-mono overflow-x-auto border border-brand-shade3/20">
                        <code>{selfHostedEmbed}</code>
                      </pre>
                      <button
                        type="button"
                        onClick={() => copyToClipboard(selfHostedEmbed, setCopiedSelfHosted)}
                        className="absolute top-2 right-2 px-2.5 py-1 bg-brand-dark border border-brand-shade3/30 rounded-btn text-[11px] text-brand-shade2 font-mono hover:text-brand-light transition-colors"
                      >
                        {copiedSelfHosted ? 'Copied' : 'Copy'}
                      </button>
                    </div>
                  </div>
                </div>
              </div>

              {/* Actions */}
              <div className="flex items-center gap-3">
                <button
                  type="button"
                  onClick={handleSave}
                  disabled={saving}
                  className="px-6 py-2 bg-brand-accent hover:bg-brand-accent-hover text-brand-light rounded-btn text-sm font-medium font-mono transition-colors disabled:opacity-60"
                >
                  {saving ? 'Saving...' : 'Save'}
                </button>
                <button
                  type="button"
                  onClick={handleDelete}
                  className="px-4 py-2 bg-red-500/10 hover:bg-red-500/20 text-red-400 rounded-btn text-sm font-medium font-mono border border-red-500/20 transition-colors"
                >
                  Delete Widget
                </button>
              </div>
            </div>
          ) : (
            <div className="col-span-5 flex items-center justify-center text-sm text-brand-shade3 font-mono py-16">
              Select a widget to configure
            </div>
          )}

          {/* Live preview */}
          <div className="col-span-4">
            {selected && (
              <div className="sticky top-0 pt-2">
                <WidgetPreview
                  primaryColor={selected.primary_color}
                  position={selected.position}
                  welcomeMessage={selected.welcome_message}
                  placeholderText={selected.placeholder_text}
                  size={selected.size}
                  avatarUrl={selected.avatar_url || undefined}
                  name={selected.name}
                />
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
