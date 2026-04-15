import { useState, useEffect, useMemo } from 'react';
import FormField from '../components/FormField';
import { api } from '../api/client';
import type { Trigger, WidgetPosition, WidgetSize, WidgetSnippetConfig } from '../types';

/**
 * Widget snippet generator.
 *
 * V2: a widget is a *client*, not a domain entity — there is no server-side
 * widgets table (docs/architecture/agent-first-runtime.md §4.3). This page
 * renders a <script> tag that loads the static widget.js bundle with the
 * chosen chat-trigger's agent name and styling baked in as data-* attributes.
 *
 * The admin picks an existing chat trigger, configures visual options, and
 * copies the resulting snippet to paste into the host page.
 */

const POSITION_OPTIONS = [
  { value: 'bottom-right', label: 'Bottom Right' },
  { value: 'bottom-left', label: 'Bottom Left' },
];

const SIZE_OPTIONS = [
  { value: 'compact', label: 'Compact' },
  { value: 'standard', label: 'Standard' },
  { value: 'full', label: 'Full' },
];

const DEFAULT_CONFIG: WidgetSnippetConfig = {
  triggerId: '',
  primaryColor: '#6366f1',
  position: 'bottom-right',
  size: 'standard',
  welcomeMessage: 'Hi! How can I help?',
  placeholderText: 'Type a message...',
  title: 'Chat',
};

function escapeAttr(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

export default function WidgetConfigPage() {
  const [triggers, setTriggers] = useState<Trigger[]>([]);
  const [loading, setLoading] = useState(true);
  const [config, setConfig] = useState<WidgetSnippetConfig>(DEFAULT_CONFIG);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    api
      .listTriggers()
      .then((list) => {
        const chatOnly = (Array.isArray(list) ? list : []).filter((t) => t.type === 'chat');
        setTriggers(chatOnly);
        if (chatOnly.length > 0 && !config.triggerId) {
          setConfig((c) => ({ ...c, triggerId: chatOnly[0]!.id }));
        }
      })
      .catch(() => {})
      .finally(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const selectedTrigger = useMemo(
    () => triggers.find((t) => t.id === config.triggerId) ?? null,
    [triggers, config.triggerId],
  );

  const agentName = selectedTrigger?.agent_name ?? '';

  const snippet = useMemo(() => {
    if (!agentName) return '';
    const attrs = [
      'src="https://your-engine.example.com/widget.js"',
      `data-agent="${escapeAttr(agentName)}"`,
      `data-position="${config.position}"`,
      `data-primary-color="${escapeAttr(config.primaryColor)}"`,
      `data-title="${escapeAttr(config.title)}"`,
      `data-welcome="${escapeAttr(config.welcomeMessage)}"`,
      `data-placeholder="${escapeAttr(config.placeholderText)}"`,
    ];
    return `<script ${attrs.join('\n        ')}></script>`;
  }, [agentName, config]);

  function update<K extends keyof WidgetSnippetConfig>(key: K, value: WidgetSnippetConfig[K]) {
    setConfig((c) => ({ ...c, [key]: value }));
  }

  function handleCopy() {
    if (!snippet) return;
    navigator.clipboard
      .writeText(snippet)
      .then(() => {
        setCopied(true);
        setTimeout(() => setCopied(false), 1500);
      })
      .catch(() => {});
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16">
        <span className="text-sm text-brand-shade3 font-mono">Loading triggers...</span>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full min-h-0">
      <div className="mb-6">
        <h1 className="text-lg font-semibold text-brand-light font-mono">Widget Snippet Generator</h1>
        <p className="mt-1 text-xs text-brand-shade3 font-mono max-w-2xl">
          Pick a chat trigger, choose how the widget looks, and copy the{' '}
          <code className="text-brand-shade2">&lt;script&gt;</code> tag into your site. Widgets are clients — the engine
          does not store widget configuration, so every embed is self-contained.
        </p>
      </div>

      {triggers.length === 0 ? (
        <div className="bg-brand-dark-surface border border-brand-shade3/10 rounded-card p-6 text-sm text-brand-shade3 font-mono">
          No chat triggers found. Create a trigger of type <strong>chat</strong> bound to an agent, then return here to
          generate an embed snippet.
        </div>
      ) : (
        <div className="flex-1 overflow-y-auto">
          <div className="grid grid-cols-12 gap-6">
            {/* Config form */}
            <div className="col-span-7 space-y-4">
              {/* Identity */}
              <div className="bg-brand-dark-surface border border-brand-shade3/10 rounded-card p-4">
                <h2 className="text-xs font-semibold text-brand-shade3 uppercase tracking-widest mb-3 font-mono">
                  Chat Trigger
                </h2>
                <FormField
                  label="Trigger"
                  type="select"
                  value={config.triggerId}
                  onChange={(v) => update('triggerId', v)}
                  options={triggers.map((t) => ({
                    value: t.id,
                    label: t.agent_name ? `${t.title} → ${t.agent_name}` : t.title,
                  }))}
                  hint="Each chat trigger routes incoming widget chats to a specific agent."
                />
                {agentName && (
                  <div className="mt-3 text-xs text-brand-shade3 font-mono">
                    Binds to agent: <span className="text-brand-light">{agentName}</span>
                  </div>
                )}
              </div>

              {/* Appearance */}
              <div className="bg-brand-dark-surface border border-brand-shade3/10 rounded-card p-4">
                <h2 className="text-xs font-semibold text-brand-shade3 uppercase tracking-widest mb-3 font-mono">
                  Appearance
                </h2>
                <div className="space-y-3">
                  <FormField
                    label="Title"
                    value={config.title}
                    onChange={(v) => update('title', v)}
                    hint="Shown in the widget header"
                  />
                  <div>
                    <label className="block text-sm font-medium text-brand-light mb-1">Primary Color</label>
                    <div className="flex items-center gap-2">
                      <input
                        type="color"
                        value={config.primaryColor}
                        onChange={(e) => update('primaryColor', e.target.value)}
                        className="w-9 h-9 rounded-card border border-brand-shade3/30 cursor-pointer bg-transparent p-0.5"
                      />
                      <input
                        type="text"
                        value={config.primaryColor}
                        onChange={(e) => update('primaryColor', e.target.value)}
                        className="flex-1 px-3 py-2 bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light font-mono focus:outline-none focus:border-brand-accent transition-colors"
                      />
                    </div>
                  </div>
                  <FormField
                    label="Position"
                    type="select"
                    value={config.position}
                    onChange={(v) => update('position', v as WidgetPosition)}
                    options={POSITION_OPTIONS}
                    hint="Widget placement on the page"
                  />
                  <FormField
                    label="Size"
                    type="select"
                    value={config.size}
                    onChange={(v) => update('size', v as WidgetSize)}
                    options={SIZE_OPTIONS}
                    hint="Chat window dimensions"
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
                    value={config.welcomeMessage}
                    onChange={(v) => update('welcomeMessage', v)}
                    hint="Greeting shown when the widget opens"
                  />
                  <FormField
                    label="Placeholder Text"
                    value={config.placeholderText}
                    onChange={(v) => update('placeholderText', v)}
                    hint="Input placeholder text"
                  />
                </div>
              </div>
            </div>

            {/* Snippet output */}
            <div className="col-span-5">
              <div className="sticky top-0 pt-2 space-y-4">
                <div className="bg-brand-dark-surface border border-brand-shade3/10 rounded-card p-4">
                  <div className="flex items-center justify-between mb-3">
                    <h2 className="text-xs font-semibold text-brand-shade3 uppercase tracking-widest font-mono">
                      Embed Snippet
                    </h2>
                    <button
                      type="button"
                      onClick={handleCopy}
                      disabled={!snippet}
                      className="px-3 py-1.5 bg-brand-accent hover:bg-brand-accent-hover text-brand-light rounded-btn text-xs font-medium font-mono transition-colors disabled:opacity-50"
                    >
                      {copied ? 'Copied' : 'Copy'}
                    </button>
                  </div>
                  {snippet ? (
                    <pre className="bg-brand-dark-alt px-4 py-3 rounded-card text-xs text-brand-shade2 font-mono overflow-x-auto border border-brand-shade3/20 whitespace-pre-wrap break-all">
                      <code>{snippet}</code>
                    </pre>
                  ) : (
                    <p className="text-xs text-brand-shade3 font-mono">Select a chat trigger to generate a snippet.</p>
                  )}
                  <p className="mt-3 text-[11px] text-brand-shade3 font-mono">
                    Replace <code className="text-brand-shade2">your-engine.example.com</code> with the hostname where
                    your ByteBrew engine is reachable from the browser.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
