import { useState } from 'react';
import FormField from '../components/FormField';
import { usePrototype } from '../hooks/usePrototype';

interface Widget {
  id: string;
  name: string;
  schema: string;
  status: 'active' | 'disabled';
  primary_color: string;
  position: 'bottom-right' | 'bottom-left' | 'top-right' | 'top-left';
  welcome_message: string;
}

const MOCK_WIDGETS: Widget[] = [
  {
    id: 'wid_abc123',
    name: 'Support Chat',
    schema: 'Support Flow',
    status: 'active',
    primary_color: '#D7513E',
    position: 'bottom-right',
    welcome_message: 'Hi! How can we help you today?',
  },
  {
    id: 'wid_def456',
    name: 'Sales Bot',
    schema: 'Sales Flow',
    status: 'disabled',
    primary_color: '#3B82F6',
    position: 'bottom-left',
    welcome_message: 'Welcome! Looking for a demo?',
  },
];

const SCHEMA_OPTIONS = [
  { value: 'Support Flow', label: 'Support Flow' },
  { value: 'Sales Flow', label: 'Sales Flow' },
  { value: 'Onboarding Flow', label: 'Onboarding Flow' },
];

const POSITION_OPTIONS = [
  { value: 'bottom-right', label: 'Bottom Right' },
  { value: 'bottom-left', label: 'Bottom Left' },
  { value: 'top-right', label: 'Top Right' },
  { value: 'top-left', label: 'Top Left' },
];

export default function WidgetConfigPage() {
  usePrototype();
  const [widgets, setWidgets] = useState<Widget[]>(MOCK_WIDGETS);
  const [selectedId, setSelectedId] = useState<string | null>(MOCK_WIDGETS[0]?.id ?? null);
  const [copiedEmbed, setCopiedEmbed] = useState(false);
  const [copiedId, setCopiedId] = useState(false);

  const selected = widgets.find((w) => w.id === selectedId) ?? null;

  function updateWidget<K extends keyof Widget>(key: K, value: Widget[K]) {
    setWidgets((prev) =>
      prev.map((w) => (w.id === selectedId ? { ...w, [key]: value } : w)),
    );
  }

  function handleCreate() {
    const id = `wid_${Math.random().toString(36).slice(2, 8)}`;
    const newWidget: Widget = {
      id,
      name: 'New Widget',
      schema: SCHEMA_OPTIONS[0]?.value ?? '',
      status: 'disabled',
      primary_color: '#D7513E',
      position: 'bottom-right',
      welcome_message: 'Hello! How can I help?',
    };
    setWidgets((prev) => [...prev, newWidget]);
    setSelectedId(id);
  }

  function copyToClipboard(text: string, setter: (v: boolean) => void) {
    navigator.clipboard.writeText(text).then(() => {
      setter(true);
      setTimeout(() => setter(false), 1500);
    }).catch(() => {});
  }

  const embedCode = selected
    ? `<script src="https://bytebrew.ai/widget/${selected.id}.js"></script>`
    : '';

  return (
    <div className="flex flex-col h-full min-h-0">
      {/* Header */}
      <div className="flex items-center justify-between px-6 py-3 border-b border-brand-shade3/10 bg-brand-dark-surface flex-shrink-0">
        <h1 className="text-lg font-semibold text-brand-light font-mono">Widget Configuration</h1>
        <button
          type="button"
          onClick={handleCreate}
          className="px-4 py-1.5 bg-brand-accent text-brand-light rounded-btn text-sm font-medium font-mono hover:bg-brand-accent/90 transition-colors"
        >
          + Create Widget
        </button>
      </div>

      <div className="flex-1 overflow-y-auto px-6 py-6">
        <div className="grid grid-cols-3 gap-6">
          {/* Widget list */}
          <div className="col-span-1 space-y-2">
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
                  <span className="text-sm font-medium text-brand-light font-mono">{w.name}</span>
                  <span
                    className={[
                      'text-[10px] font-semibold px-2 py-0.5 rounded-card font-mono uppercase tracking-wider',
                      w.status === 'active'
                        ? 'bg-status-active/15 text-status-active'
                        : 'bg-brand-shade3/15 text-brand-shade3',
                    ].join(' ')}
                  >
                    {w.status}
                  </span>
                </div>
                <p className="text-xs text-brand-shade3 font-mono">{w.id}</p>
                <p className="text-xs text-brand-shade3 font-mono mt-0.5">{w.schema}</p>
              </button>
            ))}
          </div>

          {/* Widget detail panel */}
          {selected ? (
            <div className="col-span-2 space-y-4">
              {/* Widget ID */}
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
                    label="Status"
                    type="select"
                    value={selected.status}
                    onChange={(v) => updateWidget('status', v as Widget['status'])}
                    options={[
                      { value: 'active', label: 'Active' },
                      { value: 'disabled', label: 'Disabled' },
                    ]}
                    hint="Disabled widgets won't load on client sites"
                  />
                </div>
              </div>

              {/* Schema */}
              <div className="bg-brand-dark-surface border border-brand-shade3/10 rounded-card p-4">
                <h2 className="text-xs font-semibold text-brand-shade3 uppercase tracking-widest mb-3 font-mono">
                  Schema
                </h2>
                <FormField
                  label="Schema"
                  type="select"
                  value={selected.schema}
                  onChange={(v) => updateWidget('schema', v)}
                  options={SCHEMA_OPTIONS}
                  hint="Agent schema that handles this widget's conversations"
                />
              </div>

              {/* Styling */}
              <div className="bg-brand-dark-surface border border-brand-shade3/10 rounded-card p-4">
                <h2 className="text-xs font-semibold text-brand-shade3 uppercase tracking-widest mb-3 font-mono">
                  Styling
                </h2>
                <div className="space-y-3">
                  <div>
                    <label className="block text-sm font-medium text-brand-light mb-1">Primary Color</label>
                    <div className="flex items-center gap-2">
                      <input
                        type="text"
                        value={selected.primary_color}
                        onChange={(e) => updateWidget('primary_color', e.target.value)}
                        className="flex-1 px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light font-mono focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent transition-colors"
                      />
                      <div
                        className="w-9 h-9 rounded-card border border-brand-shade3/30 shrink-0"
                        style={{ backgroundColor: selected.primary_color }}
                      />
                    </div>
                  </div>
                  <FormField
                    label="Position"
                    type="select"
                    value={selected.position}
                    onChange={(v) => updateWidget('position', v as Widget['position'])}
                    options={POSITION_OPTIONS}
                    hint="Widget placement on the host page"
                  />
                  <FormField
                    label="Welcome Message"
                    value={selected.welcome_message}
                    onChange={(v) => updateWidget('welcome_message', v)}
                    hint="Greeting shown when the widget opens"
                  />
                </div>
              </div>

              {/* Embed code */}
              <div className="bg-brand-dark-surface border border-brand-shade3/10 rounded-card p-4">
                <h2 className="text-xs font-semibold text-brand-shade3 uppercase tracking-widest mb-3 font-mono">
                  Embed Code
                </h2>
                <p className="text-xs text-brand-shade3 mb-2">
                  Add this snippet to your website to embed the chat widget.
                </p>
                <div className="relative">
                  <pre className="bg-brand-dark-alt px-4 py-3 rounded-card text-xs text-brand-shade2 font-mono overflow-x-auto border border-brand-shade3/20">
                    <code>{embedCode}</code>
                  </pre>
                  <button
                    type="button"
                    onClick={() => copyToClipboard(embedCode, setCopiedEmbed)}
                    className="absolute top-2 right-2 px-2.5 py-1 bg-brand-dark border border-brand-shade3/30 rounded-btn text-[11px] text-brand-shade2 font-mono hover:text-brand-light hover:border-brand-shade3/60 transition-colors"
                  >
                    {copiedEmbed ? 'Copied' : 'Copy'}
                  </button>
                </div>
              </div>
            </div>
          ) : (
            <div className="col-span-2 flex items-center justify-center text-sm text-brand-shade3 font-mono">
              Select a widget to configure
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
