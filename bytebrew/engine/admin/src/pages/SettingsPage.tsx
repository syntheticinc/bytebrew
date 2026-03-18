import { useState, useEffect } from 'react';
import { api } from '../api/client';
import { useApi } from '../hooks/useApi';
const BYOK_KEYS = [
  { key: 'byok.enabled', label: 'BYOK Enabled', description: 'Allow users to bring their own API keys' },
  { key: 'byok.allow_openai', label: 'Allow OpenAI', description: 'Users can use their OpenAI keys' },
  { key: 'byok.allow_anthropic', label: 'Allow Anthropic', description: 'Users can use their Anthropic keys' },
  { key: 'byok.allow_ollama', label: 'Allow Ollama', description: 'Users can connect to their Ollama instances' },
];

const ENV_VARS = [
  { name: 'BYTEBREW_ADMIN_USER', description: 'Admin username' },
  { name: 'BYTEBREW_ADMIN_PASSWORD', description: 'Admin password' },
  { name: 'BYTEBREW_JWT_SECRET', description: 'JWT signing secret' },
  { name: 'BYTEBREW_DB_URL', description: 'PostgreSQL connection string' },
];

export default function SettingsPage() {
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
  if (error) return <div className="text-red-600">Error: {error}</div>;

  return (
    <div className="max-w-3xl">
      <h1 className="text-2xl font-bold text-brand-dark mb-6">Settings</h1>

      {/* BYOK Configuration */}
      <section className="mb-8">
        <h2 className="text-lg font-semibold text-brand-dark mb-4">BYOK (Bring Your Own Key)</h2>
        <div className="bg-white rounded-card border border-brand-shade1 divide-y divide-brand-shade1">
          {BYOK_KEYS.map((item) => {
            const enabled = localSettings[item.key] === 'true';
            return (
              <div key={item.key} className="flex items-center justify-between px-4 py-3">
                <div>
                  <div className="text-sm font-medium text-brand-dark">{item.label}</div>
                  <div className="text-xs text-brand-shade3">{item.description}</div>
                </div>
                <button
                  onClick={() => handleToggle(item.key)}
                  disabled={savingKey === item.key}
                  className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                    enabled ? 'bg-brand-accent' : 'bg-brand-shade2'
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

      {/* General Settings */}
      <section className="mb-8">
        <h2 className="text-lg font-semibold text-brand-dark mb-4">General</h2>
        <div className="bg-white rounded-card border border-brand-shade1 divide-y divide-brand-shade1">
          <SettingRow
            label="Logging Level"
            settingKey="logging.level"
            value={localSettings['logging.level'] ?? 'info'}
            type="select"
            options={['debug', 'info', 'warn', 'error']}
            onSave={async (val) => {
              await api.updateSetting('logging.level', val);
              refetch();
            }}
          />
        </div>
      </section>

      {/* Environment Variables (read-only, masked) */}
      <section>
        <h2 className="text-lg font-semibold text-brand-dark mb-4">Security (Environment Variables)</h2>
        <p className="text-sm text-brand-shade3 mb-3">
          These values are set via environment variables and cannot be changed from the dashboard.
        </p>
        <div className="bg-white rounded-card border border-brand-shade1 divide-y divide-brand-shade1">
          {ENV_VARS.map((env) => (
            <div key={env.name} className="flex items-center justify-between px-4 py-3">
              <div>
                <div className="text-sm font-mono text-brand-dark">{env.name}</div>
                <div className="text-xs text-brand-shade3">{env.description}</div>
              </div>
              <span className="text-sm text-brand-shade3 font-mono">*****</span>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}

function SettingRow({
  label,
  settingKey: _settingKey,
  value,
  type,
  options,
  onSave,
}: {
  label: string;
  settingKey: string;
  value: string;
  type: 'select' | 'text';
  options?: string[];
  onSave: (value: string) => Promise<void>;
}) {
  const [localValue, setLocalValue] = useState(value);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setLocalValue(value);
  }, [value]);

  async function save() {
    if (localValue === value) return;
    setSaving(true);
    try {
      await onSave(localValue);
    } catch {
      // visible in console
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="flex items-center justify-between px-4 py-3">
      <div className="text-sm font-medium text-brand-dark">{label}</div>
      <div className="flex items-center gap-2">
        {type === 'select' && options ? (
          <select
            value={localValue}
            onChange={(e) => {
              setLocalValue(e.target.value);
            }}
            className="px-2 py-1 bg-white border border-brand-shade1 rounded-btn text-sm focus:outline-none focus:border-brand-accent"
          >
            {options.map((o) => (
              <option key={o} value={o}>
                {o}
              </option>
            ))}
          </select>
        ) : (
          <input
            type="text"
            value={localValue}
            onChange={(e) => setLocalValue(e.target.value)}
            className="px-2 py-1 bg-white border border-brand-shade1 rounded-btn text-sm w-48 focus:outline-none focus:border-brand-accent"
          />
        )}
        {localValue !== value && (
          <button
            onClick={save}
            disabled={saving}
            className="px-3 py-1 text-xs text-brand-light bg-brand-accent rounded-btn hover:bg-brand-accent-hover disabled:opacity-50"
          >
            {saving ? '...' : 'Save'}
          </button>
        )}
      </div>
    </div>
  );
}
