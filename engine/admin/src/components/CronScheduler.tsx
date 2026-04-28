import { useState, useEffect, useCallback } from 'react';

// ─── Cron presets ─────────────────────────────────────────────────────────────

interface CronPreset {
  label: string;
  template: string;
  hasTimePicker: boolean;
  hasMinutePicker?: boolean;
}

const CRON_PRESETS: CronPreset[] = [
  { label: 'Every 5 minutes',  template: '*/5 * * * *',  hasTimePicker: false },
  { label: 'Every 15 minutes', template: '*/15 * * * *', hasTimePicker: false },
  { label: 'Every 30 minutes', template: '*/30 * * * *', hasTimePicker: false },
  { label: 'Every hour',       template: '0 * * * *',    hasTimePicker: false, hasMinutePicker: true },
  { label: 'Every day',        template: '0 9 * * *',    hasTimePicker: true },
  { label: 'Every weekday (Mon-Fri)', template: '0 9 * * 1-5', hasTimePicker: true },
  { label: 'Every Monday',     template: '0 9 * * 1',    hasTimePicker: true },
  { label: 'Every Tuesday',    template: '0 9 * * 2',    hasTimePicker: true },
  { label: 'Every Wednesday',  template: '0 9 * * 3',    hasTimePicker: true },
  { label: 'Every Thursday',   template: '0 9 * * 4',    hasTimePicker: true },
  { label: 'Every Friday',     template: '0 9 * * 5',    hasTimePicker: true },
  { label: 'Every Saturday',   template: '0 9 * * 6',    hasTimePicker: true },
  { label: 'Every Sunday',     template: '0 9 * * 0',    hasTimePicker: true },
  { label: 'Custom',           template: '',              hasTimePicker: false },
];

const CUSTOM_PRESET = CRON_PRESETS[CRON_PRESETS.length - 1]!;

// ─── Cron ↔ Human helpers ────────────────────────────────────────────────────

function isValidCron(expr: string): boolean {
  const parts = expr.trim().split(/\s+/);
  if (parts.length !== 5) return false;
  const pattern = /^(\*|(\*\/\d+)|(\d+(-\d+)?(,\d+(-\d+)?)*))$/;
  return parts.every((p) => pattern.test(p));
}

function cronToHuman(cron: string): string {
  if (!cron || !cron.trim()) return '';
  const parts = cron.trim().split(/\s+/);
  if (parts.length !== 5) return 'Invalid cron expression';

  const [minute, hour, , , weekday] = parts;

  // Every N minutes
  if (minute?.startsWith('*/') && hour === '*') {
    return `Runs every ${minute.slice(2)} minutes`;
  }

  // Every hour at minute
  if (hour === '*' && !minute?.includes('*')) {
    return `Runs every hour at :${minute?.padStart(2, '0')}`;
  }

  // Daily / weekly
  if (!hour?.includes('*') && !minute?.includes('*')) {
    const h = parseInt(hour ?? '0', 10);
    const m = parseInt(minute ?? '0', 10);
    const ampm = h >= 12 ? 'PM' : 'AM';
    const h12 = h === 0 ? 12 : h > 12 ? h - 12 : h;
    const timeStr = `${h12}:${String(m).padStart(2, '0')} ${ampm}`;

    if (weekday === '*') return `Runs every day at ${timeStr}`;
    if (weekday === '1-5') return `Runs every weekday (Mon-Fri) at ${timeStr}`;

    const dayNames: Record<string, string> = {
      '0': 'Sunday', '1': 'Monday', '2': 'Tuesday', '3': 'Wednesday',
      '4': 'Thursday', '5': 'Friday', '6': 'Saturday', '7': 'Sunday',
    };
    if (dayNames[weekday ?? '']) return `Runs every ${dayNames[weekday ?? '']} at ${timeStr}`;
  }

  return `Cron: ${cron}`;
}

function findPresetForCron(cron: string): CronPreset | null {
  if (!cron) return null;
  const parts = cron.trim().split(/\s+/);
  if (parts.length !== 5) return null;

  for (const preset of CRON_PRESETS) {
    if (preset.label === 'Custom') continue;
    const tParts = preset.template.split(/\s+/);

    if (preset.hasTimePicker) {
      // Match everything except minute and hour
      if (tParts[2] === parts[2] && tParts[3] === parts[3] && tParts[4] === parts[4]) {
        return preset;
      }
    } else if (preset.hasMinutePicker) {
      // "Every hour" — match hour=* and day/month/weekday
      if (parts[1] === '*' && tParts[2] === parts[2] && tParts[3] === parts[3] && tParts[4] === parts[4]) {
        return preset;
      }
    } else {
      if (preset.template === cron.trim()) return preset;
    }
  }
  return null;
}

function extractTime(cron: string): { hour: string; minute: string } {
  const parts = cron.trim().split(/\s+/);
  return {
    minute: parts[0] ?? '0',
    hour: parts[1] ?? '9',
  };
}

// ─── Component ───────────────────────────────────────────────────────────────

interface CronSchedulerProps {
  value: string;
  onChange: (cron: string) => void;
  error?: string;
}

export default function CronScheduler({ value, onChange, error }: CronSchedulerProps) {
  const [advanced, setAdvanced] = useState(false);
  const [rawInput, setRawInput] = useState(value);
  const [selectedPreset, setSelectedPreset] = useState<CronPreset | null>(null);
  const [hour, setHour] = useState('09');
  const [minute, setMinute] = useState('00');

  // Sync from external value on mount or value change
  useEffect(() => {
    setRawInput(value);
    const preset = findPresetForCron(value);
    if (preset) {
      setSelectedPreset(preset);
      if (preset.hasTimePicker || preset.hasMinutePicker) {
        const time = extractTime(value);
        setHour(time.hour.padStart(2, '0'));
        setMinute(time.minute.padStart(2, '0'));
      }
      setAdvanced(false);
    } else if (value) {
      setSelectedPreset(CUSTOM_PRESET); // Custom
      setAdvanced(true);
    }
  }, [value]);

  const buildCron = useCallback((preset: CronPreset, h: string, m: string): string => {
    if (preset.label === 'Custom') return rawInput;
    if (preset.hasTimePicker) {
      const tParts = preset.template.split(/\s+/);
      return `${parseInt(m, 10)} ${parseInt(h, 10)} ${tParts[2]} ${tParts[3]} ${tParts[4]}`;
    }
    if (preset.hasMinutePicker) {
      return `${parseInt(m, 10)} * * * *`;
    }
    return preset.template;
  }, [rawInput]);

  function handlePresetChange(label: string) {
    const preset = CRON_PRESETS.find((p) => p.label === label) ?? CUSTOM_PRESET;
    setSelectedPreset(preset);

    if (preset.label === 'Custom') {
      setAdvanced(true);
      return;
    }

    const cron = buildCron(preset, hour, minute);
    setRawInput(cron);
    onChange(cron);
  }

  function handleTimeChange(h: string, m: string) {
    setHour(h);
    setMinute(m);
    if (selectedPreset && selectedPreset.label !== 'Custom') {
      const cron = buildCron(selectedPreset, h, m);
      setRawInput(cron);
      onChange(cron);
    }
  }

  function handleRawChange(raw: string) {
    setRawInput(raw);
    if (isValidCron(raw)) {
      onChange(raw);
      // Try to detect a matching preset
      const preset = findPresetForCron(raw);
      if (preset) {
        setSelectedPreset(preset);
        if (preset.hasTimePicker || preset.hasMinutePicker) {
          const time = extractTime(raw);
          setHour(time.hour.padStart(2, '0'));
          setMinute(time.minute.padStart(2, '0'));
        }
      } else {
        setSelectedPreset(CUSTOM_PRESET);
      }
    }
  }

  const preview = cronToHuman(value);
  const isInvalid = !!error || (rawInput.trim() !== '' && !isValidCron(rawInput));

  return (
    <div className="space-y-2">
      {/* Preset selector + time picker */}
      <div className="flex items-center gap-2 flex-wrap">
        <label className="text-xs text-brand-shade3 shrink-0">Repeat:</label>
        <select
          value={selectedPreset?.label ?? 'Custom'}
          onChange={(e) => handlePresetChange(e.target.value)}
          className="flex-1 min-w-[140px] px-2 py-1.5 bg-brand-dark border border-brand-shade3/30 rounded-card text-xs text-brand-light focus:outline-none focus:border-brand-accent transition-colors"
        >
          {CRON_PRESETS.map((p) => (
            <option key={p.label} value={p.label}>{p.label}</option>
          ))}
        </select>

        {selectedPreset?.hasTimePicker && (
          <>
            <label className="text-xs text-brand-shade3 shrink-0">at</label>
            <input
              type="time"
              value={`${hour.padStart(2, '0')}:${minute.padStart(2, '0')}`}
              onChange={(e) => {
                const [h, m] = e.target.value.split(':');
                handleTimeChange(h ?? '09', m ?? '00');
              }}
              className="px-2 py-1.5 bg-brand-dark border border-brand-shade3/30 rounded-card text-xs text-brand-light font-mono focus:outline-none focus:border-brand-accent transition-colors"
            />
          </>
        )}

        {selectedPreset?.hasMinutePicker && (
          <>
            <label className="text-xs text-brand-shade3 shrink-0">at minute:</label>
            <select
              value={minute}
              onChange={(e) => handleTimeChange(hour, e.target.value)}
              className="px-2 py-1.5 bg-brand-dark border border-brand-shade3/30 rounded-card text-xs text-brand-light font-mono focus:outline-none focus:border-brand-accent transition-colors"
            >
              {['00', '05', '10', '15', '20', '25', '30', '35', '40', '45', '50', '55'].map((m) => (
                <option key={m} value={m}>:{m}</option>
              ))}
            </select>
          </>
        )}
      </div>

      {/* Preview */}
      {preview && (
        <p className={`text-[11px] ${isInvalid ? 'text-red-400' : 'text-brand-shade2'}`}>
          {isInvalid ? 'Invalid cron expression' : preview}
        </p>
      )}

      {/* Advanced toggle */}
      <button
        type="button"
        onClick={() => setAdvanced((v) => !v)}
        className="flex items-center gap-1 text-[11px] text-brand-shade3 hover:text-brand-shade2 transition-colors"
      >
        <svg
          width="8" height="8" viewBox="0 0 24 24" fill="currentColor"
          className={`transition-transform ${advanced ? 'rotate-90' : ''}`}
        >
          <path d="M8 5l10 7-10 7V5z" />
        </svg>
        Advanced
      </button>

      {/* Raw cron input */}
      {advanced && (
        <div>
          <label className="block text-[10px] text-brand-shade3 mb-1 font-mono">Cron expression</label>
          <input
            type="text"
            value={rawInput}
            onChange={(e) => handleRawChange(e.target.value)}
            placeholder="0 9 * * *"
            className={`w-full px-3 py-2 bg-brand-dark border rounded-card text-sm text-brand-light placeholder-brand-shade3 font-mono focus:outline-none focus:ring-1 transition-colors ${
              isInvalid
                ? 'border-red-500/60 focus:border-red-500 focus:ring-red-500/30'
                : 'border-brand-shade3/30 focus:border-brand-accent focus:ring-brand-accent'
            }`}
          />
          <p className="text-[10px] text-brand-shade3/60 mt-0.5 font-mono">minute hour day month weekday</p>
        </div>
      )}

      {error && <p className="text-[10px] text-red-400">{error}</p>}
    </div>
  );
}

export { cronToHuman, isValidCron };
