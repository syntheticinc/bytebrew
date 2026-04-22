import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import type { CreateModelRequest } from '../types';

// ────────────────────────────────────────────────────────────────────────────
// BYOK Onboarding Wizard (mandatory)
//
// 3 steps:
//   1. Add LLM API Key — required. User picks a provider, pastes key, tests
//      connection by creating a model. Can't proceed without success.
//   2. Choose a starter template (optional) — Support / Sales / Blank.
//   3. Done — confetti-free success screen with "Open Canvas" CTA and an
//      optional GitHub star invite.
//
// The wizard is rendered as a full-page route (/onboarding) — not a modal —
// because it blocks the rest of the admin surface until the user has at least
// one LLM configured. The gate logic lives in OnboardingGate (Layout wrapper).
// ────────────────────────────────────────────────────────────────────────────

type Provider = {
  id: string;
  label: string;
  description: string;
  baseUrl?: string;
  defaultModel: string;
  modelHint: string;
  apiKeyHint: string;
  requiresBaseUrl?: boolean;
};

const PROVIDERS: Provider[] = [
  {
    id: 'openai_compatible',
    label: 'OpenAI',
    description: 'GPT-4, GPT-4o, GPT-3.5 via official OpenAI API',
    baseUrl: 'https://api.openai.com/v1',
    defaultModel: 'gpt-4o-mini',
    modelHint: 'e.g. gpt-4o, gpt-4o-mini, gpt-4-turbo',
    apiKeyHint: 'Starts with sk-...',
  },
  {
    id: 'anthropic',
    label: 'Anthropic',
    description: 'Claude Opus, Sonnet, Haiku',
    defaultModel: 'claude-3-5-sonnet-latest',
    modelHint: 'e.g. claude-3-5-sonnet-latest, claude-3-5-haiku-latest',
    apiKeyHint: 'Starts with sk-ant-...',
  },
  {
    id: 'openrouter',
    label: 'OpenRouter',
    description: 'Unified gateway — 200+ models from many providers',
    baseUrl: 'https://openrouter.ai/api/v1',
    defaultModel: 'anthropic/claude-3.5-sonnet',
    modelHint: 'e.g. anthropic/claude-3.5-sonnet, openai/gpt-4o',
    apiKeyHint: 'Starts with sk-or-...',
  },
  {
    id: 'azure_openai',
    label: 'Azure OpenAI',
    description: 'Azure-hosted OpenAI deployments',
    defaultModel: '',
    modelHint: 'Your deployment name (not the underlying model)',
    apiKeyHint: 'Azure resource key',
    requiresBaseUrl: true,
  },
  {
    id: 'openai_compatible_custom',
    label: 'Custom',
    description: 'Any OpenAI-compatible endpoint (LM Studio, vLLM, etc.)',
    defaultModel: '',
    modelHint: 'Your model identifier',
    apiKeyHint: 'API key (optional for local endpoints)',
    requiresBaseUrl: true,
  },
];

// ────────────────────────────────────────────────────────────────────────────
// Templates (Step 2)
// ────────────────────────────────────────────────────────────────────────────

type TemplateId = 'support' | 'sales' | 'blank';

type Template = {
  id: TemplateId;
  label: string;
  description: string;
  schemaName: string;
  agentName: string;
  systemPrompt: string;
};

const TEMPLATES: Template[] = [
  {
    id: 'support',
    label: 'Support Bot',
    description:
      'A polite, fact-driven customer support agent. Answers from your docs, escalates when unsure.',
    schemaName: 'Support Bot',
    agentName: 'support-agent',
    systemPrompt:
      "You are a customer support agent. Be concise, empathetic, and accurate. If you don't know something, say so clearly and offer to escalate. Never invent product details.",
  },
  {
    id: 'sales',
    label: 'Sales Assistant',
    description:
      'A proactive sales assistant. Qualifies leads, answers pricing questions, books demos.',
    schemaName: 'Sales Assistant',
    agentName: 'sales-agent',
    systemPrompt:
      "You are a helpful sales assistant. Qualify leads (company size, use case, timeline), answer pricing/feature questions clearly, and suggest booking a demo when the fit is strong. Never pressure the customer.",
  },
  {
    id: 'blank',
    label: 'Blank canvas',
    description: 'Start from scratch. A single empty agent ready for you to shape.',
    schemaName: 'My First Workspace',
    agentName: 'assistant',
    systemPrompt: 'You are a helpful AI assistant.',
  },
];

// ────────────────────────────────────────────────────────────────────────────
// SVG icons (inline — lucide-react is not in package.json)
// ────────────────────────────────────────────────────────────────────────────

function CheckIcon({ className = 'w-5 h-5' }: { className?: string }) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <polyline points="20 6 9 17 4 12" />
    </svg>
  );
}

function StarIcon({ className = 'w-4 h-4' }: { className?: string }) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" className={className} viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" />
    </svg>
  );
}

function KeyIcon({ className = 'w-4 h-4' }: { className?: string }) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4" />
    </svg>
  );
}

function SpinnerIcon({ className = 'w-4 h-4' }: { className?: string }) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" className={`${className} animate-spin`} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
      <circle cx="12" cy="12" r="10" strokeOpacity="0.25" />
      <path d="M22 12a10 10 0 0 1-10 10" strokeLinecap="round" />
    </svg>
  );
}

function XIcon({ className = 'w-4 h-4' }: { className?: string }) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <line x1="18" y1="6" x2="6" y2="18" />
      <line x1="6" y1="6" x2="18" y2="18" />
    </svg>
  );
}

// ────────────────────────────────────────────────────────────────────────────
// Progress header
// ────────────────────────────────────────────────────────────────────────────

function ProgressHeader({ step }: { step: 1 | 2 | 3 }) {
  const labels = ['Connect LLM', 'Starter template', 'Done'];
  return (
    <div className="w-full max-w-3xl mx-auto mb-8">
      <div className="flex items-center justify-between mb-3 text-xs text-brand-shade3 font-mono">
        <span>Step {step} of 3</span>
        <span>{Math.round((step / 3) * 100)}%</span>
      </div>
      <div className="flex items-center gap-2">
        {labels.map((label, idx) => {
          const n = (idx + 1) as 1 | 2 | 3;
          const done = n < step;
          const active = n === step;
          return (
            <div key={label} className="flex-1 flex items-center gap-2">
              <div
                className={`flex items-center justify-center w-7 h-7 rounded-full text-xs font-semibold shrink-0 transition-colors ${
                  done
                    ? 'bg-brand-accent text-brand-light'
                    : active
                    ? 'bg-brand-accent text-brand-light ring-2 ring-brand-accent/30'
                    : 'bg-brand-dark-alt text-brand-shade3 border border-brand-shade3/30'
                }`}
              >
                {done ? <CheckIcon className="w-4 h-4" /> : n}
              </div>
              <span
                className={`text-sm truncate ${
                  active ? 'text-brand-light font-medium' : done ? 'text-brand-shade2' : 'text-brand-shade3'
                }`}
              >
                {label}
              </span>
              {idx < labels.length - 1 && (
                <div
                  className={`flex-1 h-px mx-1 ${
                    done ? 'bg-brand-accent' : 'bg-brand-shade3/20'
                  }`}
                />
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

// ────────────────────────────────────────────────────────────────────────────
// Step 1 — Connect LLM
// ────────────────────────────────────────────────────────────────────────────

type TestStatus =
  | { kind: 'idle' }
  | { kind: 'testing' }
  | { kind: 'success'; modelName: string }
  | { kind: 'error'; message: string };

function Step1ConnectLLM({
  onSuccess,
}: {
  onSuccess: () => void;
}) {
  const [providerId, setProviderId] = useState<string>(PROVIDERS[0]!.id);
  const [apiKey, setApiKey] = useState('');
  const [modelName, setModelName] = useState(PROVIDERS[0]!.defaultModel);
  const [baseUrl, setBaseUrl] = useState(PROVIDERS[0]!.baseUrl ?? '');
  const [displayName, setDisplayName] = useState('default');
  const [status, setStatus] = useState<TestStatus>({ kind: 'idle' });

  const provider = PROVIDERS.find((p) => p.id === providerId)!;

  function selectProvider(id: string) {
    const next = PROVIDERS.find((p) => p.id === id)!;
    setProviderId(id);
    setModelName(next.defaultModel);
    setBaseUrl(next.baseUrl ?? '');
    setStatus({ kind: 'idle' });
  }

  // Map our onboarding provider id to the backend's provider type enum used
  // in POST /models. "openai_compatible_custom" is an onboarding-only alias
  // so we can show the Custom card separately; backend sees it as
  // openai_compatible.
  function backendType(id: string): string {
    if (id === 'openai_compatible_custom') return 'openai_compatible';
    return id;
  }

  async function handleNext(e: FormEvent) {
    e.preventDefault();
    if (status.kind === 'testing') return;

    if (!apiKey.trim() && providerId !== 'openai_compatible_custom') {
      setStatus({ kind: 'error', message: 'API key is required.' });
      return;
    }
    if (!modelName.trim()) {
      setStatus({ kind: 'error', message: 'Model name is required.' });
      return;
    }
    if (provider.requiresBaseUrl && !baseUrl.trim()) {
      setStatus({ kind: 'error', message: 'Base URL is required for this provider.' });
      return;
    }
    if (!displayName.trim()) {
      setStatus({ kind: 'error', message: 'Display name is required.' });
      return;
    }

    setStatus({ kind: 'testing' });

    const payload: CreateModelRequest = {
      // Onboarding wizard only configures *chat* models — the embedding
      // flow is a separate admin surface. Without `kind` the server
      // rejects the create call with "kind is required", which surfaced
      // as an unactionable error on step 1 of the wizard.
      kind: 'chat',
      name: displayName.trim(),
      type: backendType(providerId),
      model_name: modelName.trim(),
      api_key: apiKey.trim() || undefined,
      base_url: baseUrl.trim() || undefined,
    };

    try {
      // POST /models is the only synchronous validation path today — backend
      // rejects malformed payloads (missing kind, empty name, etc.) and 201
      // means "good enough to persist". Bad API keys surface on the first
      // real chat call, not here; adding a provider-ping validate endpoint
      // is a separate backend change tracked in the playwright-smoke plan.
      //
      // On success we advance immediately — there is no value in showing a
      // separate "Connected" state before the user clicks again.
      await api.createModel(payload);
      setStatus({ kind: 'success', modelName: payload.name });
      onSuccess();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Connection failed.';
      setStatus({ kind: 'error', message });
    }
  }

  return (
    <form onSubmit={handleNext} className="w-full max-w-3xl mx-auto">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-brand-light mb-1">Connect your LLM</h1>
        <p className="text-sm text-brand-shade2">
          ByteBrew is bring-your-own-key. Paste an API key from your LLM provider to continue —
          keys live on your Engine, not in our cloud.
        </p>
      </div>

      {/* Provider cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 mb-6">
        {PROVIDERS.map((p) => {
          const selected = providerId === p.id;
          return (
            <button
              type="button"
              key={p.id}
              onClick={() => selectProvider(p.id)}
              className={`text-left p-4 rounded-card border transition-colors ${
                selected
                  ? 'border-brand-accent bg-brand-accent/5 ring-1 ring-brand-accent/40'
                  : 'border-brand-shade3/20 bg-brand-dark-alt hover:border-brand-shade3/40'
              }`}
            >
              <div className="flex items-center justify-between mb-1">
                <span className="font-semibold text-brand-light text-sm">{p.label}</span>
                {selected && <CheckIcon className="w-4 h-4 text-brand-accent" />}
              </div>
              <p className="text-xs text-brand-shade2 leading-relaxed">{p.description}</p>
            </button>
          );
        })}
      </div>

      {/* Credentials */}
      <div className="bg-brand-dark-alt rounded-card border border-brand-shade3/15 p-5 space-y-4">
        <div>
          <label className="block text-sm font-medium text-brand-light mb-1">Display name</label>
          <input
            type="text"
            value={displayName}
            onChange={(e) => {
              setDisplayName(e.target.value);
              setStatus({ kind: 'idle' });
            }}
            placeholder="default"
            className="w-full px-3 py-2 bg-brand-dark border border-brand-shade3/30 rounded-btn text-sm text-brand-light focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent"
          />
          <p className="mt-1 text-xs text-brand-shade3">Internal label for this model connection.</p>
        </div>

        <div>
          <label className="block text-sm font-medium text-brand-light mb-1">
            Model name
          </label>
          <input
            type="text"
            value={modelName}
            onChange={(e) => {
              setModelName(e.target.value);
              setStatus({ kind: 'idle' });
            }}
            placeholder={provider.defaultModel || provider.modelHint}
            className="w-full px-3 py-2 bg-brand-dark border border-brand-shade3/30 rounded-btn text-sm text-brand-light font-mono focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent"
          />
          <p className="mt-1 text-xs text-brand-shade3">{provider.modelHint}</p>
        </div>

        {(provider.requiresBaseUrl || provider.baseUrl) && (
          <div>
            <label className="block text-sm font-medium text-brand-light mb-1">Base URL</label>
            <input
              type="text"
              value={baseUrl}
              onChange={(e) => {
                setBaseUrl(e.target.value);
                setStatus({ kind: 'idle' });
              }}
              placeholder="https://api.example.com/v1"
              disabled={!provider.requiresBaseUrl}
              className="w-full px-3 py-2 bg-brand-dark border border-brand-shade3/30 rounded-btn text-sm text-brand-light font-mono focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent disabled:opacity-60"
            />
            {!provider.requiresBaseUrl && (
              <p className="mt-1 text-xs text-brand-shade3">Auto-configured for {provider.label}.</p>
            )}
          </div>
        )}

        <div>
          <label className="flex items-center gap-1.5 text-sm font-medium text-brand-light mb-1">
            <KeyIcon className="w-4 h-4" />
            API key
          </label>
          <input
            type="password"
            value={apiKey}
            onChange={(e) => {
              setApiKey(e.target.value);
              setStatus({ kind: 'idle' });
            }}
            placeholder={provider.apiKeyHint}
            className="w-full px-3 py-2 bg-brand-dark border border-brand-shade3/30 rounded-btn text-sm text-brand-light font-mono focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent"
            autoComplete="off"
          />
          <p className="mt-1 text-xs text-brand-shade3">{provider.apiKeyHint}</p>
        </div>
      </div>

      {/* Status */}
      {status.kind === 'success' && (
        <div className="mt-4 flex items-center gap-2 p-3 bg-green-500/10 border border-green-500/30 rounded-btn text-sm text-green-400">
          <CheckIcon className="w-4 h-4 shrink-0" />
          <span>
            Connected. Model <strong className="font-mono">{status.modelName}</strong> is ready.
          </span>
        </div>
      )}
      {status.kind === 'error' && (
        <div className="mt-4 flex items-start gap-2 p-3 bg-red-500/10 border border-red-500/30 rounded-btn text-sm text-red-400">
          <XIcon className="w-4 h-4 mt-0.5 shrink-0" />
          <span className="break-words">{status.message}</span>
        </div>
      )}

      {/* Actions */}
      <div className="mt-6 flex items-center justify-between">
        <p className="text-xs text-brand-shade3">
          Your key is stored on your Engine's database, never transmitted to bytebrew.ai.
        </p>
        <div className="flex items-center gap-3">
          <button
            type="submit"
            disabled={status.kind === 'testing'}
            className="flex items-center gap-2 px-5 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover transition-colors disabled:opacity-60 disabled:cursor-not-allowed"
          >
            {status.kind === 'testing' ? (
              <>
                <SpinnerIcon className="w-4 h-4" />
                Connecting…
              </>
            ) : (
              'Next'
            )}
          </button>
        </div>
      </div>
    </form>
  );
}

// ────────────────────────────────────────────────────────────────────────────
// Step 2 — Template picker
// ────────────────────────────────────────────────────────────────────────────

function Step2Template({
  onDone,
}: {
  onDone: () => void;
}) {
  const [selected, setSelected] = useState<TemplateId | null>(null);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function createFromTemplate(template: Template) {
    setCreating(true);
    setError(null);
    try {
      // Schema first — then single agent wired as the entry. This matches the
      // minimum shape the canvas needs. Users add delegation later.
      const schema = await api.createSchema({
        name: template.schemaName,
        description: `Created from ${template.label} template during onboarding`,
      });

      try {
        // Best-effort agent creation. If the schemas endpoint on this engine
        // already auto-creates an entry agent, the second call will fail with
        // a conflict — we swallow that and continue. Anything else bubbles up.
        await api.createAgent({
          name: template.agentName,
          system_prompt: template.systemPrompt,
        });
      } catch (err) {
        const message = err instanceof Error ? err.message.toLowerCase() : '';
        if (!message.includes('exists') && !message.includes('duplicate') && !message.includes('conflict')) {
          throw err;
        }
      }

      // Swallow "already a member" errors for the same reason as above.
      try {
        await api.createAgentRelation(schema.id, template.agentName, template.agentName);
      } catch {
        // non-fatal — relation wiring is a nice-to-have here; the schema +
        // agent already exist, which is enough for the canvas to open.
      }

      onDone();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create template.');
    } finally {
      setCreating(false);
    }
  }

  async function handleContinue() {
    if (!selected) return;
    const template = TEMPLATES.find((t) => t.id === selected);
    if (!template) return;
    await createFromTemplate(template);
  }

  return (
    <div className="w-full max-w-3xl mx-auto">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-brand-light mb-1">Pick a starter (optional)</h1>
        <p className="text-sm text-brand-shade2">
          Start from a pre-built workspace or skip to a blank canvas. You can always add more
          later.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mb-6">
        {TEMPLATES.map((t) => {
          const active = selected === t.id;
          return (
            <button
              key={t.id}
              type="button"
              onClick={() => setSelected(t.id)}
              disabled={creating}
              className={`text-left p-4 rounded-card border transition-colors ${
                active
                  ? 'border-brand-accent bg-brand-accent/5 ring-1 ring-brand-accent/40'
                  : 'border-brand-shade3/20 bg-brand-dark-alt hover:border-brand-shade3/40'
              } disabled:opacity-60`}
            >
              <div className="flex items-center justify-between mb-2">
                <span className="font-semibold text-brand-light text-sm">{t.label}</span>
                {active && <CheckIcon className="w-4 h-4 text-brand-accent" />}
              </div>
              <p className="text-xs text-brand-shade2 leading-relaxed">{t.description}</p>
            </button>
          );
        })}
      </div>

      {selected && (
        <div className="mb-6 p-4 bg-brand-dark-alt border border-brand-shade3/15 rounded-card">
          <p className="text-xs text-brand-shade3 mb-1">You'll get:</p>
          <p className="text-sm text-brand-light">
            A schema named{' '}
            <strong className="font-mono">
              {TEMPLATES.find((t) => t.id === selected)?.schemaName}
            </strong>{' '}
            with one entry agent (
            <span className="font-mono text-brand-shade2">
              {TEMPLATES.find((t) => t.id === selected)?.agentName}
            </span>
            ).
          </p>
        </div>
      )}

      {error && (
        <div className="mb-4 flex items-start gap-2 p-3 bg-red-500/10 border border-red-500/30 rounded-btn text-sm text-red-400">
          <XIcon className="w-4 h-4 mt-0.5 shrink-0" />
          <span className="break-words">{error}</span>
        </div>
      )}

      <div className="flex items-center justify-end gap-3">
        <button
          type="button"
          onClick={onDone}
          disabled={creating}
          className="px-4 py-2 bg-brand-dark border border-brand-shade3/30 text-brand-light rounded-btn text-sm font-medium hover:border-brand-shade3/60 transition-colors disabled:opacity-60"
        >
          Skip
        </button>
        <button
          type="button"
          onClick={handleContinue}
          disabled={!selected || creating}
          className="flex items-center gap-2 px-5 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {creating ? (
            <>
              <SpinnerIcon className="w-4 h-4" />
              Creating…
            </>
          ) : (
            'Create & continue'
          )}
        </button>
      </div>
    </div>
  );
}

// ────────────────────────────────────────────────────────────────────────────
// Step 3 — Success
// ────────────────────────────────────────────────────────────────────────────

const STAR_DISMISSED_KEY = 'bytebrew_onboarding_star_dismissed';

function Step3Done() {
  const navigate = useNavigate();
  const [starDismissed, setStarDismissed] = useState(
    () => localStorage.getItem(STAR_DISMISSED_KEY) === 'true',
  );

  function dismissStar() {
    localStorage.setItem(STAR_DISMISSED_KEY, 'true');
    setStarDismissed(true);
  }

  return (
    <div className="w-full max-w-2xl mx-auto text-center">
      <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-green-500/10 border border-green-500/30 mb-5">
        <CheckIcon className="w-8 h-8 text-green-400" />
      </div>
      <h1 className="text-3xl font-bold text-brand-light mb-2">Your workspace is ready!</h1>
      <p className="text-base text-brand-shade2 mb-8">
        Open the canvas to start wiring agents, tools, and MCP servers.
      </p>

      <div className="flex items-center justify-center gap-3 mb-8">
        <button
          type="button"
          onClick={() => navigate('/schemas')}
          className="px-6 py-3 bg-brand-accent text-brand-light rounded-btn text-sm font-semibold hover:bg-brand-accent-hover transition-colors"
        >
          Open Canvas
        </button>
        <button
          type="button"
          onClick={() => navigate('/overview')}
          className="px-6 py-3 bg-brand-dark-alt border border-brand-shade3/30 text-brand-light rounded-btn text-sm font-medium hover:border-brand-shade3/60 transition-colors"
        >
          Go to Overview
        </button>
      </div>

      {!starDismissed && (
        <div className="relative p-5 bg-brand-dark-alt border border-brand-shade3/15 rounded-card text-left">
          <button
            type="button"
            onClick={dismissStar}
            aria-label="Dismiss"
            className="absolute top-2 right-2 p-1 text-brand-shade3 hover:text-brand-light transition-colors"
          >
            <XIcon className="w-4 h-4" />
          </button>
          <div className="flex items-start gap-3">
            <span className="inline-flex items-center justify-center w-10 h-10 rounded-full bg-yellow-400/10 text-yellow-400 shrink-0">
              <StarIcon className="w-5 h-5" />
            </span>
            <div className="min-w-0">
              <p className="text-sm font-semibold text-brand-light mb-1">
                Love ByteBrew? Star us on GitHub
              </p>
              <p className="text-xs text-brand-shade2 mb-3">
                It helps more teams discover the project — takes five seconds.
              </p>
              <a
                href="https://github.com/syntheticinc/bytebrew"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-brand-dark border border-brand-shade3/30 text-brand-light rounded-btn text-xs font-medium hover:border-brand-shade3/60 transition-colors"
              >
                <StarIcon className="w-3.5 h-3.5 text-yellow-400" />
                Star on GitHub
              </a>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// ────────────────────────────────────────────────────────────────────────────
// Wizard container
// ────────────────────────────────────────────────────────────────────────────

export default function OnboardingWizard() {
  const [step, setStep] = useState<1 | 2 | 3>(1);

  return (
    <div className="fixed inset-0 z-50 bg-brand-dark overflow-auto">
      <div className="min-h-full flex flex-col">
        <div className="px-6 py-4 border-b border-brand-shade3/10 bg-brand-dark-surface">
          <div className="max-w-3xl mx-auto flex items-center justify-between">
            <div className="text-sm font-semibold text-brand-light">ByteBrew setup</div>
            <div className="text-xs text-brand-shade3">BYOK — bring your own key</div>
          </div>
        </div>

        <div className="flex-1 px-6 py-10">
          <ProgressHeader step={step} />
          {step === 1 && <Step1ConnectLLM onSuccess={() => setStep(2)} />}
          {step === 2 && <Step2Template onDone={() => setStep(3)} />}
          {step === 3 && <Step3Done />}
        </div>
      </div>
    </div>
  );
}
