import { useState, useEffect } from 'react';
import { api } from '../api/client';
import { useApi } from '../hooks/useApi';
import UsageDashboard from '../components/UsageDashboard';

// V2 §5.8 "Settings + BYOK": admin toggles for per-end-user BYOK. The
// values are stored in the `settings` table (jsonb) and read by the BYOK
// middleware on every request — flips here take effect without a
// restart. Provider-specific allow_* keys translate into a unified
// byok.allowed_providers array on the API side.
const BYOK_KEYS = [
  { key: 'byok.enabled', label: 'BYOK Enabled', description: 'Allow users to bring their own API keys' },
  { key: 'byok.allow_openai', label: 'Allow OpenAI', description: 'Users can use their OpenAI keys' },
  { key: 'byok.allow_anthropic', label: 'Allow Anthropic', description: 'Users can use their Anthropic keys' },
  { key: 'byok.allow_ollama', label: 'Allow Ollama', description: 'Users can connect to their Ollama instances' },
];

type SettingsTab = 'general' | 'usage';

export default function SettingsPage() {
  const [activeTab, setActiveTab] = useState<SettingsTab>('general');
  const { data: settings, loading, error, refetch } = useApi(() => api.listSettings());
  const [savingKey, setSavingKey] = useState<string | null>(null);
  const [localSettings, setLocalSettings] = useState<Record<string, string>>({});

  useEffect(() => {
    if (!settings) return;
    const map: Record<string, string> = {};
    if (Array.isArray(settings)) {
      for (const s of settings) {
        map[s.key] = s.value;
      }
    } else if (typeof settings === 'object') {
      // Handle flat object response from stub API
      const obj = settings as Record<string, unknown>;
      if (obj.byok_enabled !== undefined) map['byok.enabled'] = String(obj.byok_enabled);
      if (Array.isArray(obj.byok_allowed_providers)) {
        map['byok.allow_openai'] = (obj.byok_allowed_providers as string[]).includes('openai') ? 'true' : 'false';
        map['byok.allow_anthropic'] = (obj.byok_allowed_providers as string[]).includes('anthropic') ? 'true' : 'false';
        map['byok.allow_ollama'] = (obj.byok_allowed_providers as string[]).includes('ollama') ? 'true' : 'false';
      }
    }
    setLocalSettings(map);
  }, [settings]);

  async function handleToggle(key: string) {
    const current = localSettings[key] === 'true';
    const newValue = (!current).toString();
    setSavingKey(key);
    try {
      await api.updateSetting(key, newValue);
      setLocalSettings((prev) => ({ ...prev, [key]: newValue }));
      refetch();
    } catch {
      // visible in console
    } finally {
      setSavingKey(null);
    }
  }

  if (loading) return <div className="text-brand-shade3">Loading settings...</div>;
  if (error) return <div className="text-red-400">Error: {error}</div>;

  return (
    <div className="max-w-3xl">
      <h1 className="text-2xl font-bold text-brand-light mb-4">Settings</h1>

      {/* Tabs */}
      <div className="flex items-center gap-1 mb-6 border-b border-brand-shade3/15">
        {(['general', 'usage'] as const).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={[
              'px-4 py-2 text-sm font-medium font-mono transition-colors capitalize',
              activeTab === tab
                ? 'text-brand-light border-b-2 border-brand-accent'
                : 'text-brand-shade3 hover:text-brand-shade2',
            ].join(' ')}
          >
            {tab}
          </button>
        ))}
      </div>

      {/* Usage tab */}
      {activeTab === 'usage' && <UsageDashboard />}

      {/* General tab */}
      {activeTab === 'general' && <>

      {/* BYOK Configuration — wired into the request path (V2 §5.8). */}
      <section className="mb-8">
        <h2 className="text-lg font-semibold text-brand-light mb-4">BYOK (Bring Your Own Key)</h2>
        <p className="text-sm text-brand-shade3 mb-3">
          When enabled, end users can override the tenant model with their own credentials by sending
          <code className="mx-1 px-1 bg-brand-dark rounded">X-BYOK-Provider</code>,
          <code className="mx-1 px-1 bg-brand-dark rounded">X-BYOK-API-Key</code>,
          <code className="mx-1 px-1 bg-brand-dark rounded">X-BYOK-Model</code>
          headers on the chat endpoint.
        </p>
        <div className="bg-brand-dark-alt rounded-card border border-brand-shade3/15 divide-y divide-brand-shade3/10">
          {BYOK_KEYS.map((item) => {
            const enabled = localSettings[item.key] === 'true';
            return (
              <div key={item.key} className="flex items-center justify-between px-4 py-3">
                <div>
                  <div className="text-sm font-medium text-brand-light">{item.label}</div>
                  <div className="text-xs text-brand-shade3">{item.description}</div>
                </div>
                <button
                  onClick={() => handleToggle(item.key)}
                  disabled={savingKey === item.key}
                  className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                    enabled ? 'bg-brand-accent' : 'bg-brand-shade3/30'
                  } ${savingKey === item.key ? 'opacity-50' : ''}`}
                >
                  <span
                    className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                      enabled ? 'translate-x-6' : 'translate-x-1'
                    }`}
                  />
                </button>
              </div>
            );
          })}
        </div>
      </section>
      </>}
    </div>
  );
}
