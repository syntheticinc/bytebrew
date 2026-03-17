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
    for (const s of settings) {
      map[s.key] = s.value;
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

  if (loading) return <div className="text-gray-500">Loading settings...</div>;
  if (error) return <div className="text-red-600">Error: {error}</div>;

  return (
    <div className="max-w-3xl">
      <h1 className="text-2xl font-bold text-gray-900 mb-6">Settings</h1>

      {/* BYOK Configuration */}
      <section className="mb-8">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">BYOK (Bring Your Own Key)</h2>
        <div className="bg-white rounded-lg shadow divide-y">
          {BYOK_KEYS.map((item) => {
            const enabled = localSettings[item.key] === 'true';
            return (
              <div key={item.key} className="flex items-center justify-between px-4 py-3">
                <div>
                  <div className="text-sm font-medium text-gray-900">{item.label}</div>
                  <div className="text-xs text-gray-500">{item.description}</div>
                </div>
                <button
                  onClick={() => handleToggle(item.key)}
                  disabled={savingKey === item.key}
                  className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                    enabled ? 'bg-blue-600' : 'bg-gray-200'
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
        <h2 className="text-lg font-semibold text-gray-900 mb-4">General</h2>
        <div className="bg-white rounded-lg shadow divide-y">
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
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Security (Environment Variables)</h2>
        <p className="text-sm text-gray-500 mb-3">
          These values are set via environment variables and cannot be changed from the dashboard.
        </p>
        <div className="bg-white rounded-lg shadow divide-y">
          {ENV_VARS.map((env) => (
            <div key={env.name} className="flex items-center justify-between px-4 py-3">
              <div>
                <div className="text-sm font-mono text-gray-900">{env.name}</div>
                <div className="text-xs text-gray-500">{env.description}</div>
              </div>
              <span className="text-sm text-gray-400 font-mono">*****</span>
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
      <div className="text-sm font-medium text-gray-900">{label}</div>
      <div className="flex items-center gap-2">
        {type === 'select' && options ? (
          <select
            value={localValue}
            onChange={(e) => {
              setLocalValue(e.target.value);
            }}
            className="px-2 py-1 border border-gray-300 rounded text-sm"
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
            className="px-2 py-1 border border-gray-300 rounded text-sm w-48"
          />
        )}
        {localValue !== value && (
          <button
            onClick={save}
            disabled={saving}
            className="px-3 py-1 text-xs text-white bg-blue-600 rounded hover:bg-blue-700 disabled:opacity-50"
          >
            {saving ? '...' : 'Save'}
          </button>
        )}
      </div>
    </div>
  );
}
